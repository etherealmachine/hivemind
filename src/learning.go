package main

import (
	"container/vector"
	"fmt"
	"github.com/ajstarks/svgo"
	"gob"
	"log"
	"math"
	"os"
	"rand"
	"sort"
	"time"
)

type Swarm struct {
	Mu, P, Lambda uint
	Samples       uint
	Generation    uint
	Particles     Particles
	config        *Config
	evals         *vector.Vector
}

func NewSwarm(config *Config) *Swarm {
	if config.Mu >= config.Lambda {
		panic("mu must be less than lambda")
	}
	if config.Parents > config.Mu {
		panic("parents must be less than or equal to mu")
	}
	s := new(Swarm)
	s.config = config
	s.Lambda = s.config.Lambda
	s.Mu = s.config.Mu
	s.P = s.config.Parents
	s.Samples = s.config.Samples
	s.Generation = 0
	s.Particles = make(Particles, s.Mu)
	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = NewParticle(s, 0, 100)
	}
	return s
}

type Particle struct {
	Strategy float64
	Position map[uint32]float64
	Min, Max float64
	Fitness  float64
	swarm    *Swarm
}

func NewParticle(swarm *Swarm, min, max float64) *Particle {
	p := new(Particle)
	p.swarm = swarm
	p.Strategy = rand.Float64() * 0.05
	p.Position = make(map[uint32]float64)
	p.Min = min
	p.Max = max
	p.Fitness = 0
	return p
}

func (p *Particle) Copy() *Particle {
	cp := new(Particle)
	cp.Strategy = p.Strategy
	cp.Position = make(map[uint32]float64)
	for i := range p.Position {
		cp.Position[i] = p.Position[i]
	}
	cp.Min = p.Min
	cp.Max = p.Max
	cp.Fitness = p.Fitness
	cp.swarm = p.swarm
	return cp
}

type Particles []*Particle

func (s Particles) Len() int {
	return len(s)
}

func (s Particles) Less(i, j int) bool {
	return s[i].Fitness > s[j].Fitness
}

func (s Particles) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (p *Particle) Get(i uint32) float64 {
	if _, exists := p.Position[i]; !exists && p.swarm != nil {
		p.Init(i)
	}
	return p.Position[i]
}

func (p *Particle) Init(i uint32) {
	p.Position[i] = p.Min + (p.Max-p.Min)*rand.Float64()
}

func (s *Swarm) evalPlay(p1 *Particle, p2 *Particle) {
	config := new(Config)
	*config = *s.config
	for sample := uint(0); sample < s.Samples; sample++ {
		log.Printf("game %d / %d\n", sample, s.Samples)
		t := NewTracker(config)
		color := BLACK
		target := BLACK
		if rand.Float64() < 0.5 {
			target = WHITE
		}
		for {
			if color == target {
				config.policy_weights = p1
			} else {
				config.policy_weights = p2
			}
			root := NewRoot(color, t, config)
			genmove(root, t)
			t.Play(color, root.Best().Vertex)
			if config.Verbose {
				log.Println(t.String())
				log.Println(Ctoa(color), t.Vtoa(root.Best().Vertex))
				log.Println(Ctoa(Reverse(color)), "to play")
			}
			if t.Winner() != EMPTY {
				break
			}
			color = Reverse(color)
		}
		if t.Winner() == target {
			p1.Fitness++
			p2.Fitness--
		} else {
			p2.Fitness++
			p1.Fitness--
		}
	}
}

/**
Evolution Strategies update
(mu/p ,+ lambda)-ES
fitness function F
individual a_k = (y_k, s_k, F(y_k))
B_p parents, mu = |B_p|
B_o offspring, lambda = |B_o|

		1. generate lambda (lambda < mu for comma) offspring B_o
		  a. select randomly p parents from B_p
		2. evaluate either B_o (,) or B_o + B_p (+) for fitness
		3. select mu parents from either B_o (,) or B_o + B_p (+)
*/
func (s *Swarm) step() {

	parents := s.Particles

	// generate lambda (lambda >= mu for comma) children
	s.Particles = make(Particles, s.Lambda)
	for i := range s.Particles {
		// select randomly p parents from parents
		p := make(Particles, s.P)
		for j := uint(0); j < s.P; j++ {
			p[j] = randParticle(parents, p)
		}
		s.Particles[i] = s.recombine(p)
		s.mutate(s.Particles[i])
		s.Particles[i].Fitness = 0
	}

	// propagate the last best particles without change
	for i := uint(0); i < s.config.Propagate; i++ {
		s.Particles[i] = parents[i].Copy()
	}

	// evaluate either children (,) or children + parents (+) for fitness
	for i := range s.Particles {
		log.Printf("evaluating %d/%d\n", i, len(s.Particles))
		s.evalPlay(s.Particles[i], randParticle(s.Particles, []*Particle{s.Particles[i]}))
		log.Printf("fitness of %d: %.4f\n", i, s.Particles[i].Fitness)
	}

	// select mu parents from either children (,) or children + parents (+)
	sort.Sort(s.Particles)

	s.Particles = s.Particles[:s.Mu]
}

/*
	return a random particle from p that is not in e
*/
func randParticle(p Particles, e Particles) (r *Particle) {
	contains := true
	for contains {
		r = p[rand.Int31n(int32(len(p)))]
		contains = false
		for i := 0; e != nil && i < len(e); i++ {
			if e[i] == r {
				contains = true
			}
		}
	}
	return
}

/*
	return a new particle that is the average of the given p particles
*/
func (s *Swarm) recombine(parents Particles) (r *Particle) {
	r = NewParticle(parents[0].swarm, parents[0].Min, parents[0].Max)
	superset := make(map[uint32]bool)
	for i := range parents {
		for j := range parents[i].Position {
			superset[j] = true
		}
	}
	for i := range superset {
		count := 0
		for j := range parents {
			if _, exists := parents[j].Position[i]; exists {
				r.Position[i] += parents[j].Position[i]
				count++
			}
		}
		r.Position[i] /= float64(count)
	}
	return
}

/*
	randomly permute particle's position and strategy using evolution strategies method
*/
func (s *Swarm) mutate(p *Particle) {
	dim := float64(len(p.Position))
	tau := (1 / math.Sqrt(2*dim)) * (1.0 - float64(s.Generation)/float64(s.config.Generations))
	p.Strategy *= math.Exp(tau * rand.NormFloat64())
	for i := range p.Position {
		p.Position[i] += p.Strategy * rand.NormFloat64()
		if p.Position[i] > p.Max {
			p.Position[i] = p.Max
		} else if p.Position[i] < p.Min {
			p.Position[i] = p.Min
		}
	}
}

func (s *Swarm) Best() (best *Particle) {
	if s.config.Combine {
		best = NewParticle(s, s.Particles[0].Min, s.Particles[0].Max)
		best.Fitness = 0
		superset := make(map[uint32]bool)
		for i := range s.Particles {
			for j := range s.Particles[i].Position {
				superset[j] = true
			}
			best.Fitness += s.Particles[i].Fitness
		}
		best.Fitness /= float64(len(s.Particles))
		for i := range superset {
			count := 0.0
			for j := range s.Particles {
				if _, exists := s.Particles[j].Position[i]; exists {
					best.Position[i] += s.Particles[j].Fitness * s.Particles[j].Position[i]
					count += s.Particles[j].Fitness
				}
			}
			best.Position[i] /= float64(count)
		}
	} else {
		best = s.Particles[0]
	}
	return
}

func (s *Swarm) SaveSwarm() {
	var filename string
	if s.config.Prefix != "" {
		filename = fmt.Sprintf(s.config.Prefix+".swarm.%d.gob", s.Generation)
	} else {
		filename = fmt.Sprintf("swarm.%d.gob", s.Generation)
	}
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer func() { f.Close() }()
	e := gob.NewEncoder(f)
	err = e.Encode(s)
	if err != nil {
		panic(err)
	}
}

func (s *Swarm) LoadSwarm(filename string, config *Config) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer func() { f.Close() }()
	d := gob.NewDecoder(f)
	err = d.Decode(s)
	if err != nil {
		panic(err)
	}
	s.config = config
	for i := range s.Particles {
		s.Particles[i].swarm = s
	}
}

func LoadBest(filename string, config *Config) *Particle {
	s := new(Swarm)
	s.LoadSwarm(filename, config)
	if s.Best() == nil {
		panic("swarm has no particles")
	}
	return s.Best()
}

func Train(config *Config) {
	var s *Swarm
	s = NewSwarm(config)
	if config.Sfile != "" {
		s.LoadSwarm(config.Sfile, config)
	}
	for s.Generation < s.config.Generations {
		start := time.Nanoseconds()
		s.step()
		s.Generation++
		s.SaveSwarm()
		log.Printf("generation %d/%d, best: %.4f, took %d seconds",
			s.Generation, s.config.Generations, s.Best().Fitness,
			(time.Nanoseconds()-start)/1e9)
	}
}

func drawHex(xoff, yoff, width float64, pos int, style1, style2 string, s *svg.SVG) {
	if style1 == "" {
		return
	}
	c := width
	a := 0.5 * c
	b := math.Sin(1.04719755) * c
	switch pos {
	case 0:
	case 1:
		xoff += 1.5 * c
		yoff -= b
	case 2:
		xoff += 3 * c
	case 3:
		xoff += 3 * c
		yoff += 2 * b
	case 4:
		xoff += 1.5 * c
		yoff += 3 * b
	case 5:
		yoff += 2 * b
	case 6:
		xoff += 1.5 * c
		yoff += b
	}
	x := []int{int(xoff), int(a + xoff), int(a + c + xoff), int(2*c + xoff), int(a + c + xoff), int(a + xoff)}
	y := []int{int(b + yoff), int(yoff), int(yoff), int(b + yoff), int(2*b + yoff), int(2*b + yoff)}
	s.Polygon(x, y, style1)
	if style2 != "" {
		s.Circle(int(xoff+c), int(yoff+b), int(width/2), style2)
	}
}

func PrintBestWeights(config *Config) {
	f, err := os.Create(config.Sfile + ".svg")
	if err != nil {
		panic(err)
	}
	p := LoadBest(config.Sfile, config)
	s := svg.New(f)
	if config.Hex {
		v := new(vector.Vector)
		a := []byte{EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY}
		for odometer(a, len(a)-1) {
			hash := hex_hash(BLACK, a, []int{0, 1, 2, 3, 4, 5, 6})
			if min_hash, exists := hex_min_hash[hash]; a[6] == EMPTY && exists && hash == min_hash {
				v.Push(mkcp(a))
			}
		}
		w := 20
		width := 3 * w
		height := 2 * width
		s.Start(5*width, height*v.Len())
		for i := 0; i < v.Len(); i++ {
			a = v.At(i).([]byte)
			x := float64(width) / 2.0
			y := float64(20 + i*height)
			for j := range a {
				s1 := "fill:#d6d6d6;stroke:#464646;stroke-width:2"
				s2 := ""
				switch a[j] {
				case EMPTY:
				case BLACK:
					s2 = "fill:black;stroke:black;stroke-width:2"
				case WHITE:
					s2 = "fill:white;stroke:black;stroke-width:2"
				case ILLEGAL:
					s1 = ""
				}
				drawHex(x, y, float64(w), j, s1, s2, s)
			}
			black_weight := p.Get(hex_min_hash[hex_hash(BLACK, a, []int{0, 1, 2, 3, 4, 5, 6})])
			s.Text(int(x)+2*width, int(y)+2*w-10, fmt.Sprintf("B: %f", black_weight), "font-family:monospace")
			white_weight := p.Get(hex_min_hash[hex_hash(WHITE, a, []int{0, 1, 2, 3, 4, 5, 6})])
			s.Text(int(x)+2*width, int(y)+2*w+10, fmt.Sprintf("W: %f", white_weight), "font-family:monospace")
		}
	}
	s.End()
}

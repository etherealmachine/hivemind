package main

import (
	"math"
	"gob"
	"os"
	"rand"
	"sort"
	"time"
	"fmt"
	"log"
	"container/vector"
	"json"
	"github.com/ajstarks/svgo"
)

type Swarm struct {
	Mu, P, Lambda uint
	Samples       uint
	Generation    uint
	Particles     Particles
	GBest         *Particle
	GBestGen      uint
	evals         *vector.Vector
	config        *Config
}

func NewSwarm(config *Config) *Swarm {
	var min, max, vmax float64
	if !config.Eval {
		min = 0.001
		max = 10.0
		vmax = 5.0
	} else if config.Eval {
		min = 0
		max = 1
		vmax = 0.5
	}
	if config.ESswarm && config.Mu >= config.Lambda {
		panic("ES swarm: mu must be less than lambda")
	}
	if config .ESswarm && config.Parents > config.Mu {
		panic("ES swarm: parents must be less than or equal to mu")
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
		s.Particles[i] = NewParticle(s, min, max, vmax)
	}
	s.GBest = s.Particles[0].Copy()
	return s
}

type Particle struct {
	Strategy       float64
	Position       map[uint32]float64
	Velocity       map[uint32]float64
	Min, Max, VMax float64
	PBest          *Particle
	PBestGen       uint
	Fitness        float64
	swarm          *Swarm
}

func NewParticle(swarm *Swarm, min, max, vMax float64) *Particle {
	p := new(Particle)
	p.swarm = swarm
	p.Strategy = rand.Float64() * 0.05
	p.Position = make(map[uint32]float64)
	p.Min = min
	p.Max = max
	p.Velocity = make(map[uint32]float64)
	p.VMax = vMax
	p.PBest = p.Copy()
	p.Fitness = math.MaxFloat64
	return p
}

func (p *Particle) Copy() *Particle {
	cp := new(Particle)
	cp.Strategy = p.Strategy
	cp.Position = make(map[uint32]float64)
	for i := range p.Position {
		cp.Position[i] = p.Position[i]
	}
	if p.swarm.config.Pswarm {
		cp.Velocity = make(map[uint32]float64)
		for i := range p.Velocity {
			cp.Velocity[i] = p.Velocity[i]
		}
	}
	cp.Fitness = p.Fitness
	cp.Min = p.Min
	cp.Max = p.Max
	cp.VMax = p.VMax
	cp.swarm = p.swarm
	return cp
}

type Particles []*Particle

func (s Particles) Len() int {
	return len(s)
}

func (s Particles) Less(i, j int) bool {
	return s[i].Fitness < s[j].Fitness
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
	if p.swarm.config.Pswarm {
		p.Velocity[i] = -p.VMax + 2*p.VMax*rand.Float64()
	}
}

func (s *Swarm) playOneGame() (moves *vector.IntVector, evals *vector.Vector) {
	moves = new(vector.IntVector)
	evals = new(vector.Vector)
	t := NewTracker(s.config)
	var br, wr *Node
	var vertex int
	for {
		br = NewRoot(BLACK, t, s.config)
		genmove(br, t)
		vertex = br.Best().Vertex
		moves.Push(vertex)
		evals.Push(br.Best().Mean)
		t.Play(BLACK, vertex)
		log.Println(t.String())
		if t.Winner() != EMPTY || moves.Len() >= 2 * t.Sqsize() {
			break
		}
		wr = NewRoot(WHITE, t, s.config)
		genmove(wr, t)
		vertex = wr.Best().Vertex
		moves.Push(vertex)
		evals.Push(wr.Best().Mean)
		t.Play(WHITE, vertex)
		log.Println(t.String())
		if t.Winner() != EMPTY || moves.Len() >= 2 * t.Sqsize() {
			break
		}
	}
	return moves, evals
}

func (s *Swarm) evaluate(p *Particle) {
	p.Fitness = 0
	for i := 0; i < s.evals.Len(); i++ {
		s.eval(p, s.evals.At(i).(*Eval))
	}
	p.Fitness /= float64(s.evals.Len())
}

func (s *Swarm) eval(p *Particle, eval *Eval) {
	s.config.policy_weights = p
	t := NewTracker(s.config)
	color := BLACK
	for i := range eval.Moves {
		t.Play(color, eval.Moves[i])
		color = Reverse(color)
	}
	t.Play(color, eval.Next)
	wins := 0
	for j := 0; j < 10; j++ {
		cp := t.Copy()
		cp.Playout(Reverse(color))
		if cp.Winner() == color {
			wins++
		}
	}
	err := eval.Mean - (float64(wins) / 10.0)
	err = err * err
	p.Fitness += err
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
func (s *Swarm) ESStep() {

	// generate lambda (lambda >= mu for comma) children
	children := make(Particles, s.Lambda)
	for i := uint(0); i < s.Lambda; i++ {
		// select randomly p parents from parents
		p := make(Particles, s.P)
		for j := uint(0); j < s.P; j++ {
			p[j] = randParticle(s.Particles, p)
		}
		children[i] = s.recombine(p)
		s.mutate(children[i])
	}
	// propagate the 2 last best particles without change
	children[0] = s.Particles[0].Copy()
	children[1] = s.Particles[1].Copy()

	// evaluate either children (,) or children + parents (+) for fitness
	for i := uint(0); i < s.Lambda; i++ {
		log.Printf("evaluating %d/%d\n", i, s.Lambda)
		s.evaluate(children[i])
		log.Printf("fitness of %d: %.4f\n", i, children[i].Fitness)
	}

	// select mu parents from either children (,) or children + parents (+)
	sort.Sort(children)

	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = children[i]
	}
	if s.Particles[0].Fitness < s.GBest.Fitness {
		s.GBest = s.Particles[0].Copy()
	}
}

/**
Particle swarm update
*/
func (s *Swarm) PSStep() {
	for i := uint(0); i < s.Mu; i++ {
		s.update_particle(i)
	}
}

func (s *Swarm) update_particle(i uint) {
	s.ps_update(s.Particles[i])
	s.Particles[i].Fitness = 0
	log.Printf("evaluating %d\n", i)
	s.evaluate(s.Particles[i])
	log.Printf("fitness of %d: %.4f\n", i, s.Particles[i].Fitness)
	s.update_gbest(i)
	s.update_pbest(i)
}

func (s *Swarm) update_gbest(i uint) {
	if s.Generation - s.GBestGen > 10 {
		log.Println("re-evaluating gbest")
		old_fitness := s.GBest.Fitness
		s.evaluate(s.GBest)
		s.GBest.Fitness += old_fitness
		s.GBest.Fitness /= 2.0
	}
	if s.Particles[i].Fitness < s.GBest.Fitness {
		log.Printf("updated gbest, old: %.4f, new: %.4f\n", s.GBest.Fitness, s.Particles[i].Fitness)
		s.GBest = s.Particles[i].Copy()
		s.GBestGen = s.Generation
	}
}

func (s *Swarm) update_pbest(i uint) {
	if s.Generation - s.Particles[i].PBestGen > 10 {
		log.Printf("re-evaluating pbest %d\n", i)
		old_fitness := s.Particles[i].PBest.Fitness
		s.evaluate(s.Particles[i].PBest)
		s.Particles[i].PBest.Fitness += old_fitness
		s.Particles[i].PBest.Fitness /= 2.0
	}
	if s.Particles[i].Fitness < s.Particles[i].PBest.Fitness {
		log.Printf("updated pbest of particle %d, old: %.4f, new: %.4f\n", i, s.Particles[i].PBest.Fitness, s.Particles[i].Fitness)
		s.Particles[i].PBest = s.Particles[i].Copy()
		s.Particles[i].PBestGen = s.Generation
	}
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
	r = NewParticle(parents[0].swarm, parents[0].Min, parents[0].Max, parents[0].VMax)
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

/*
	update position and velocity of particle using particle swarm method
*/
func (s *Swarm) ps_update(p *Particle) {
	w := 0.4 + 0.5*(1.0-float64(s.Generation)/float64(s.config.Generations))
	superset := make(map[uint32]bool)
	for i := range p.Position {
		superset[i] = true
	}
	for i := range p.PBest.Position {
		superset[i] = true
	}
	for i := range s.GBest.Position {
		superset[i] = true
	}
	for i := range superset {
		if _, exists := p.Velocity[i]; !exists {
			p.Init(i)
		}
		delta_pbest := 0.0
		if _, exists := p.PBest.Position[i]; exists {
			delta_pbest = 2 * rand.Float64() * (p.PBest.Position[i] - p.Position[i])
		}
		delta_gbest := 0.0
		if _, exists := s.GBest.Position[i]; exists {
			delta_gbest = 2 * rand.Float64() * (s.GBest.Position[i] - p.Position[i])
		}
		p.Velocity[i] = w * (p.Velocity[i] + delta_pbest + delta_gbest)
		if p.Velocity[i] < -p.VMax {
			p.Velocity[i] = -p.VMax
		} else if p.Velocity[i] > p.VMax {
			p.Velocity[i] = p.VMax
		}
		p.Position[i] += p.Velocity[i]
		if p.Position[i] < p.Min {
			p.Position[i] = p.Min
		} else if p.Position[i] > p.Max {
			p.Position[i] = p.Max
		}
	}
}

func (s *Swarm) Best() (best *Particle) {
	return s.GBest
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
	s.evals = new(vector.Vector)
	f, err := os.OpenFile("evals.json", os.O_RDONLY, 0666)
	if err != nil {
		panic(err)
	}
	defer func() { f.Close() }()
	decoder := json.NewDecoder(f)
	for {
		var eval Eval
		err := decoder.Decode(&eval)
		if err != nil {
			if s.evals.Len() == 0 {
				panic(err)
			}
			break
		}
		s.evals.Push(&eval)
	}
	for s.Generation < s.config.Generations {
		start := time.Nanoseconds()
		if config.ESswarm {
			s.ESStep()
		} else if config.Pswarm {
			s.PSStep()
		}
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
	x := []int{int(xoff), int(a + xoff), int(a + c + xoff), int(2 * c + xoff), int(a + c + xoff), int(a + xoff)}
	y := []int{int(b + yoff), int(yoff), int(yoff), int(b + yoff), int(2 * b + yoff), int(2 * b + yoff)}
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
		s.Start(5 * width, height * v.Len())
		for i := 0; i < v.Len(); i++ {
			a = v.At(i).([]byte)
			x := float64(width) / 2.0
			y := float64(20 + i * height)
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

package main

import (
	"math"
	"gob"
	"os"
	"rand"
	"sort"
	"time"
	"fmt"
	"strings"
	"strconv"
)

type Swarm struct {
	Mu, P, Lambda uint
	Samples       uint
	Generation    uint
	Generations   uint
	Particles     Particles
	Arch          []int
	Max, Min      float64
}

func NewSwarm(mu, p, lambda, samples, generations uint, min, max float64, dim int, arch []int) *Swarm {
	s := new(Swarm)
	s.Mu = mu
	s.P = p
	s.Lambda = lambda
	if mu >= lambda {
		panic("illegal argument to NewSwarm - mu must be less than lambda")
	}
	if p > mu {
		panic("illegal argument to NewSwarm - p must be less than or equal to mu")
	}
	s.Samples = samples
	s.Generations = generations
	s.Max = max
	s.Min = min
	s.Particles = make(Particles, s.Mu)
	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = NewParticle(dim)
	}
	s.Arch = arch
	return s
}

type Particle struct {
	Dim      int
	Strategy float64
	Position []float64
	Fitness  float64
}

func NewParticle(dim int) *Particle {
	p := new(Particle)
	p.Dim = dim
	p.Strategy = rand.Float64() * 0.05
	p.Position = make([]float64, dim)
	for i := 0; i < p.Dim; i++ {
		if *tablepat {
			p.Position[i] = rand.Float64()
		} else {
			p.Position[i] = rand.Float64()*0.4 - 0.2
		}
	}
	return p
}

func (p *Particle) Copy() *Particle {
	cp := new(Particle)
	cp.Dim = p.Dim
	cp.Strategy = p.Strategy
	cp.Position = make([]float64, len(p.Position))
	copy(cp.Position, p.Position)
	cp.Fitness = 0
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

func (s *Swarm) evaluate(p *Particle) {
	defer func() { recover() }()
	var m PatternMatcher
	if *tablepat {
		m = p
	} else {
		m = &NeuralNet{s.Arch, p.Position}
	}
	t := NewTracker(*size)
	passes := 0
	var vertex int
	for {
		br := NewRoot(BLACK, t)
		genmove(br, t, m)
		if br == nil || br.Best() == nil {
			vertex = -1
			passes++
		} else {
			passes = 0
			vertex = br.Best().vertex
		}
		t.Play(BLACK, vertex)
		if (*hex && t.Winner() != EMPTY) || passes == 2 {
			break
		}
		wr := NewRoot(WHITE, t)
		genmove(wr, t, m)
		if wr == nil || wr.Best() == nil {
			vertex = -1
			passes++
		} else {
			passes = 0
			vertex = wr.Best().vertex
		}
		t.Play(WHITE, vertex)
		if (*hex && t.Winner() != EMPTY) || passes == 2 {
			break
		}
	}
	if t.Winner() == BLACK {
		p.Fitness += 1
	} else if t.Winner() == WHITE {
		p.Fitness -= 1
	}
}

/**
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
func (s *Swarm) Step() {

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
	children[0] = s.Particles[0].Copy()
	children[1] = s.Particles[1].Copy()

	// evaluate either children (,) or children + parents (+) for fitness
	for i := uint(0); i < s.Lambda; i++ {
		for j := uint(0); j < s.Samples; j++ {
			s.evaluate(children[i])
			log.Printf("%d / %d\n", i*s.Samples+j, s.Lambda*s.Samples)
		}
	}

	// select mu parents from either children (,) or children + parents (+)
	sort.Sort(children)
	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = children[i]
	}

	s.Generation++
}

func randParticle(p Particles, e Particles) (r *Particle) {
	contains := true
	for contains {
		r = p[rand.Int31n(int32(len(p)))]
		contains = false
		for i := 0; i < len(e); i++ {
			if e[i] == r {
				contains = true
			}
		}
	}
	return
}

func (s *Swarm) recombine(p Particles) (r *Particle) {
	r = p[0].Copy()
	for i := 0; i < r.Dim; i++ {
		r.Strategy = 0
		r.Position[i] = 0
		for j := 0; j < len(p); j++ {
			r.Strategy += p[j].Strategy
			r.Position[i] += p[j].Position[i]
		}
		r.Strategy /= float64(len(p))
		r.Position[i] /= float64(len(p))
	}
	return
}

func (s *Swarm) mutate(p *Particle) {
	tau := (1 / math.Sqrt(2*float64(p.Dim))) * (1.0 - float64(s.Generation)/float64(s.Generations))
	p.Strategy *= math.Exp(tau*rand.NormFloat64())
	for i := 0; i < p.Dim; i++ {
		if p.Position[i] > 0 {
			p.Position[i] += p.Strategy * rand.NormFloat64()
			if p.Position[i] > s.Max {
				p.Position[i] = s.Max
			} else if p.Position[i] < s.Min {
				p.Position[i] = s.Min
			}
		}
	}
}

func (s *Swarm) Best() (best *Particle) {
	for _, p := range s.Particles {
		if best == nil || p.Fitness > best.Fitness {
			best = p
		}
	}
	return best
}

func (s *Swarm) SaveSwarm(filename string) {
	f, err := os.Open(filename, os.O_RDWR|os.O_TRUNC|os.O_CREAT, 0666)
	if err != nil {
		log.Println("failed to save swarm")
		return
	}
	defer func() { f.Close() }()
	e := gob.NewEncoder(f)
	e.Encode(s)
}

func (s *Swarm) LoadSwarm(filename string) os.Error {
	f, err := os.Open(filename, os.O_RDONLY, 0)
	if err != nil {
		log.Println("failed to load swarm")
		return err
	}
	defer func() { f.Close() }()
	d := gob.NewDecoder(f)
	d.Decode(s)
	return nil
}

func LoadBest(filename string) *Particle {
	s := new(Swarm)
	err := s.LoadSwarm(filename)
	if err != nil {
		return nil
	}
	return s.Best()
}

func Train() {
	var s *Swarm
	if *tablepat {
		if *hex {
			s = NewSwarm(*mu, *parents, *lambda, *samples, 1000, 0.01, 100, 114688, nil)
		} else {
			s = NewSwarm(*mu, *parents, *lambda, *samples, 1000, 0.01, 100, 2359296, nil)
		}
	} else {
		net := NewNeuralNet([]int{inputsize + 1, inputsize, 1})
		s = NewSwarm(*mu, *parents, *lambda, *samples, 1000, -10, 10, len(net.Config), net.Arch)
	}
	f, err := os.Open(".", os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	names, err := f.Readdirnames(-1)
	if err != nil {
		panic(err)
	}
	max := 1
	for i := range names {
		if strings.HasPrefix(names[i], "swarm") && strings.HasSuffix(names[i], "gob") {
			j, err := strconv.Atoi(strings.Split(strings.Split(names[i], "-", -1)[1], ".", -1)[0])
			if err != nil {
				panic(err)
			}
			if j > max {
				max = j
			}
		}
	}
	s.LoadSwarm(fmt.Sprintf("swarm-%d.gob", max))
	for s.Generation < s.Generations {
		start := time.Nanoseconds()
		s.Step()
		s.SaveSwarm(fmt.Sprintf("swarm-%d.gob", s.Generation))
		log.Printf("generation %d/%d, best %.0f wins, took %d seconds",
			s.Generation, s.Generations, s.Best().Fitness,
			(time.Nanoseconds()-start)/1e9)
	}
}

func ShowSwarm(filename string) {
	s := new(Swarm)
	s.LoadSwarm(filename)
	log.Println(s.Generation)
}

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
)

type Swarm struct {
	Mu, P, Lambda uint
	Samples       uint
	Generation    uint
	Particles     Particles
	GBest         *Particle
	games         *vector.Vector
	results       *vector.Vector
	config        *Config
}

func NewSwarm(config *Config) *Swarm {
	var min, max, vmax float64
	if config.Pat {
		min = 0
		max = 10
		vmax = 5
	} else if config.Eval {
		min = 0
		max = 1
		vmax = 0.5
	}
	if config.ESswarm && config.Mu >= config.Lambda {
		panic("illegal argument to NewSwarm - mu must be less than lambda")
	}
	if config.Parents > config.Mu {
		panic("illegal argument to NewSwarm - parents must be less than or equal to mu")
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
	Fitness        float64
	swarm          *Swarm
	log            map[uint32]bool
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
	p.log = make(map[uint32]bool)
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
	return s[i].Fitness > s[j].Fitness
}

func (s Particles) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (p *Particle) Get(i uint32) float64 {
	if _, exists := p.Position[i]; !exists {
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

func (s *Swarm) playOneGame() (moves *vector.IntVector, result byte) {
	moves = new(vector.IntVector)
	t := NewTracker(s.config)
	result = EMPTY
	var br, wr *Node
	var vertex int
	for {
		br = NewRoot(BLACK, t, s.config)
		genmove(br, t)
		vertex = br.Best().Vertex
		moves.Push(vertex)
		t.Play(BLACK, vertex)
		log.Println(t.String())
		result = t.Winner()
		if result != EMPTY || moves.Len() >= 2 * t.Sqsize() {
			result = WHITE
			break
		}
		wr = NewRoot(WHITE, t, s.config)
		genmove(wr, t)
		vertex = wr.Best().Vertex
		moves.Push(vertex)
		t.Play(WHITE, vertex)
		log.Println(t.String())
		result = t.Winner()
		if result != EMPTY || moves.Len() >= 2 * t.Sqsize() {
			result = BLACK
			break
		}
	}
	return moves, result
}

type Visits float64

func (f1 Visits) Less(f2 interface{}) bool {
	return f1 > f2.(Visits)
}

func (s *Swarm) evaluate(p *Particle, moves *vector.IntVector, result byte) {
	p.log = make(map[uint32]bool)
	gamma := 0.95
	discount := gamma
	s.config.matcher = &ComboMatcher{s.config.expert_patterns, p}
	t := NewTracker(s.config)
	color := BLACK
	fitness := 0.0
	for i := 0; i < moves.Len(); i++ {
		vertex := moves.At(i)
		t.Play(color, vertex)
		if vertex != -1 {
			cp := t.Copy()
			cp.Playout(Reverse(color))
			if cp.Winner() == result {
				fitness += discount
			}
		}
		color = Reverse(color)
		discount *= gamma
	}
	p.Fitness += fitness / float64(moves.Len())
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
	for i := uint(0); i < s.Lambda; i++ {
		children[i].Fitness = 0
	}

	// evaluate either children (,) or children + parents (+) for fitness
	for i := uint(0); i < s.Lambda; i++ {
		for j := 0; j < s.games.Len(); j++ {
			log.Printf("evaluating %d/%d\n", i, s.Lambda)
			s.evaluate(children[i], s.games.At(j).(*vector.IntVector), s.results.At(j).(uint8))
			log.Printf("fitness of %d: %.2f\n", i, children[i].Fitness)
		}
		children[i].Fitness /= float64(s.games.Len())
	}

	// select mu parents from either children (,) or children + parents (+)
	sort.Sort(children)

	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = children[i]
	}
}

/**
Particle swarm update
*/
func (s *Swarm) PSStep() {

	s.GBest.Fitness *= 0.95

	for i := uint(0); i < s.Mu; i++ {
		s.update_particle(i)
	}
}

func (s *Swarm) update_particle(i uint) {
	s.Particles[i].PBest.Fitness *= 0.95
	s.Particles[i].Fitness = 0
	for j := 0; j < s.games.Len(); j++ {
		log.Printf("evaluating %d/%d\n", i, s.Lambda)
		s.evaluate(s.Particles[i], s.games.At(j).(*vector.IntVector), s.results.At(j).(uint8))
		log.Printf("fitness of %d: %.2f\n", i, s.Particles[i].Fitness)
	}
	s.Particles[i].Fitness /= float64(s.games.Len())
	s.update_gbest(i)
	s.update_pbest(i)
}

func (s *Swarm) update_gbest(i uint) {
	if s.Particles[i].Fitness > s.GBest.Fitness {
		log.Printf("updated gbest, old: %.2f, new: %.2f\n", s.GBest.Fitness, s.Particles[i].Fitness)
		s.GBest = s.Particles[i].Copy()
	}
}

func (s *Swarm) update_pbest(i uint) {
	if s.Particles[i].Fitness > s.Particles[i].PBest.Fitness {
		log.Printf("updated pbest of particle %d, old: %.2f, new: %.2f\n", i, s.Particles[i].PBest.Fitness, s.Particles[i].Fitness)
		s.Particles[i].PBest = s.Particles[i].Copy()
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
		for pos := range parents[i].log {
			r.log[pos] = true
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
	for _, p := range s.Particles {
		if best == nil || p.Fitness > best.Fitness {
			best = p
		}
	}
	return best
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
	s.games = new(vector.Vector)
	s.results = new(vector.Vector)
	for i := uint(0); i < s.Samples; i++ {
		game, result := s.playOneGame()
		s.games.Push(game)
		s.results.Push(result)
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
		log.Printf("generation %d/%d, best: %.2f, took %d seconds",
			s.Generation, s.config.Generations, s.Best().Fitness,
			(time.Nanoseconds()-start)/1e9)
	}
}

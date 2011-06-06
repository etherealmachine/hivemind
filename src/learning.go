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
	"runtime"
)

type Swarm struct {
	Mu, P, Lambda uint
	Samples       uint
	Generation    uint
	Generations   uint
	Particles     Particles
	GBest *Particle
	config Config
}

func NewSwarm(config *Config) *Swarm {
	var stride int
	var min, max, vmax float64
	if config.pat {
		stride = len(NewTracker(config).Neighbors(0, 2))
		if config.tenuki { stride++ }
		min = 0.01
		max = 100
		vmax = 20
	} else if config.eval {
		stride = 1
		min = 0
		max = 1
		vmax = 0.5
	}
	if config.mu >= config.lambda {
		panic("illegal argument to NewSwarm - mu must be less than lambda")
	}
	if config.parents > config.mu {
		panic("illegal argument to NewSwarm - p must be less than or equal to mu")
	}
	s := new(Swarm)
	s.config = *config
	s.Lambda = s.config.lambda
	s.Mu = s.config.mu
	s.P = s.config.parents
	s.Samples = s.config.samples
	s.Generations = s.config.generations
	s.Particles = make(Particles, s.Mu)
	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = NewParticle(min, max, vmax, stride, s.config)
	}
	s.GBest = s.Particles[0].Copy()
	return s
}

type Particle struct {
	Strategy float64
	Position map[uint32][]float64
	Velocity map[uint32][]float64
	Min, Max, VMax	float64
	PBest		 *Particle
	Fitness  float64
	Stride	int
	config Config
}

func NewParticle(min, max, vMax float64, stride int, config Config) *Particle {
	p := new(Particle)
	p.Strategy = rand.Float64() * 0.05
	p.Position = make(map[uint32][]float64)
	p.Min = min
	p.Max = max
	p.Velocity = make(map[uint32][]float64)
	p.VMax = vMax
	p.config = config
	p.PBest = p.Copy()
	p.Stride = stride
	return p
}

func (p *Particle) Copy() *Particle {
	cp := new(Particle)
	cp.Strategy = p.Strategy
	cp.Position = make(map[uint32][]float64)
	for i := range p.Position {
		cp.Position[i] = make([]float64, p.Stride)
		copy(cp.Position[i], p.Position[i])
	}
	if p.config.pswarm {
		cp.Velocity = make(map[uint32][]float64)
		for i := range p.Velocity {
			cp.Velocity[i] = make([]float64, p.Stride)
			copy(cp.Velocity[i], p.Velocity[i])
		}
	}
	cp.Fitness = p.Fitness
	cp.Min = p.Min
	cp.Max = p.Max
	cp.VMax = p.VMax
	cp.Stride = p.Stride
	cp.config = p.config
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

func swap(i, j uint32, b *uint32) {
	x := ((*b >> i) ^ (*b >> j)) & 1 // XOR temporary
	*b ^= ((x << i) | (x << j))
}

// set the ith bit of b to j
func set(i, j uint32, b *uint32) {
	*b ^= j << i
}

// get the ith bit of b
func get(i, b uint32) uint32 {
 return b >> i & 0x00000001
}

func HashVertices(board []uint8, neighbors []int, offset int) uint32 {
	index := uint32(0)
	for i := 0; i < len(neighbors); i++ {
		// set the 2*i, 2*i+1 bits of the index
		m0 := uint32(0)
		m1 := uint32(0)
		if neighbors[i] == -1 {
			m0, m1 = 1, 1
		} else if board[neighbors[i]] == BLACK {
			m0, m1 = 0, 1
		} else if board[neighbors[i]] == WHITE {
			m0, m1 = 1, 0
		}
		set(uint32(2*i+offset), m0, &index)
		set(uint32(2*i+1+offset), m1, &index)
	}
	return index
}

func (p *Particle) Get(i uint32) []float64 {
	if p.Position[i] == nil {
		p.Init(i)
	}
	return p.Position[i]
}

func (p *Particle) Init(i uint32) {
	p.Position[i] = make([]float64, p.Stride)
	for j := 0; j < p.Stride; j++ {
		p.Position[i][j] = p.Min + (p.Max - p.Min) * rand.Float64()
	}
	if p.config.pswarm {
		p.Velocity[i] = make([]float64, p.Stride)
		for j := 0; j < p.Stride; j++ {
			p.Velocity[i][j] = -p.VMax + 2 * p.VMax * rand.Float64()
		}
	}
}

func (s *Swarm) evaluate(p *Particle) {
	bc := &p.config
	bc.pat = false
	bc.matcher = nil
	bc.eval = false
	bc.evaluator = nil
	wc := &p.config
	t := NewTracker(&p.config)
	t.SetKomi(7.5)
	move := 0
	maxMoves := 2 * t.Sqsize()
	var vertex int
	for {
		if move == 0 && p.config.hex {
			vertex = p.config.size + 2
		} else {
			br := NewRoot(BLACK, t, bc)
			genmove(br, t)
			vertex = br.Best().vertex
		}
		t.Play(BLACK, vertex)
		move++
		if t.Winner() != EMPTY || move >= maxMoves { break }
		wr := NewRoot(WHITE, t, wc)
		genmove(wr, t)
		vertex = wr.Best().vertex
		t.Play(WHITE, vertex)
		move++
		if t.Winner() != EMPTY || move >= maxMoves { break }
	}
	if t.Winner() == WHITE {
		p.Fitness++
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
	
	eval := make(chan uint, s.Lambda)
	done := make(chan bool)
	
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func(s *Swarm) {
			for j := range eval {
				for k := uint(0); k < s.Samples; k++ {
					s.evaluate(children[j])
					done <- true
				}
			}
		}(s)
	}

	// evaluate either children (,) or children + parents (+) for fitness
	for i := uint(0); i < s.Lambda; i++ {
		eval <- i
	}
	for i := uint(0); i < s.Lambda; i++ {
		for j := uint(0); j < s.Samples; j++ {
			<-done
			log.Printf("%d / %d\n", i*s.Samples+j, s.Lambda*s.Samples)
		}
	}
	close(eval)

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

	s.GBest.Fitness *= 0.9

	for i := uint(0); i < s.Mu; i++ {
		s.update_particle(i)
	}
}

func (s *Swarm) update_particle(i uint) {
	s.Particles[i].PBest.Fitness *= 0.9
	s.Particles[i].Fitness = 0
	for j := uint(0); j < s.Samples; j++ {
		s.evaluate(s.Particles[i])
		log.Printf("%d / %d\n", i*s.Samples+j, s.Mu*s.Samples)
	}
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
	r = NewParticle(parents[0].Min, parents[0].Max, parents[0].VMax, parents[0].Stride, parents[0].config)
	superset := make(map[uint32]bool)
	for i := range parents {
		for j := range parents[i].Position {
			superset[j] = true
		}
	}
	for i := range superset {
		r.Position[i] = make([]float64, r.Stride)
		count := 0
		for j := range parents {
			if parents[j].Position[i] != nil {
				for k := 0; k < r.Stride; k++ {
					r.Position[i][k] += parents[j].Position[i][k]
				}
				count++
			}
		}
		for k := 0; k < r.Stride; k++ {
			r.Position[i][k] /= float64(count)
		}
	}
	return
}

/*
	randomly permute particle's position and strategy using evolution strategies method
*/
func (s *Swarm) mutate(p *Particle) {
	dim := float64(len(p.Position) * p.Stride)
	tau := (1 / math.Sqrt(2*dim)) * (1.0 - float64(s.Generation)/float64(s.Generations))
	p.Strategy *= math.Exp(tau * rand.NormFloat64())
	for i := range p.Position {
		for j := range p.Position[i] {
			p.Position[i][j] += p.Strategy * rand.NormFloat64()
			if p.Position[i][j] > p.Max {
				p.Position[i][j] = p.Max
			} else if p.Position[i][j] < p.Min {
				p.Position[i][j] = p.Min
			}
		}
	}
}

/*
	update position and velocity of particle using particle swarm method
*/
func (s *Swarm) ps_update(p *Particle) {
	w := 0.4 + 0.5 * (1.0 - float64(s.Generation)/float64(s.Generations))
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
		if p.Velocity[i] == nil {
			p.Init(i)
		}
		for j := range p.Velocity[i] {
			delta_pbest := 0.0
			if p.PBest.Position[i] != nil {
				delta_pbest = 2 * rand.Float64() * (p.PBest.Position[i][j] - p.Position[i][j])
			}
			delta_gbest := 0.0
			if s.GBest.Position[i] != nil {
				delta_gbest = 2 * rand.Float64() * (s.GBest.Position[i][j] - p.Position[i][j])
			}
			p.Velocity[i][j] = w * (p.Velocity[i][j] + delta_pbest + delta_gbest)
			if p.Velocity[i][j] < -p.VMax {
				p.Velocity[i][j] = -p.VMax
			} else if (p.Velocity[i][j] > p.VMax) {
				p.Velocity[i][j] = p.VMax
			}
			p.Position[i][j] += p.Velocity[i][j]
			if p.Position[i][j] < p.Min {
				p.Position[i][j] = p.Min
			} else if (p.Position[i][j] > p.Max) {
				p.Position[i][j] = p.Max
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
	f, err := os.Create(filename)
	if err != nil { panic(err) }
	defer func() { f.Close() }()
	e := gob.NewEncoder(f)
	err = e.Encode(s)
	if err != nil { panic(err) }
}

func (s *Swarm) LoadSwarm(filename string) {
	f, err := os.Open(filename)
	if err != nil { panic(err) }
	defer func() { f.Close() }()
	d := gob.NewDecoder(f)
	err = d.Decode(s)
	if err != nil { panic(err) }
}

func LoadBest(filename string) *Particle {
	s := new(Swarm)
	s.LoadSwarm(filename)
	return s.Best()
}

func Train(config *Config) {
	var s *Swarm
	s = NewSwarm(config)
	if config.file != "" {
		s.LoadSwarm(config.file)
	}
	for s.Generation < s.Generations {
		start := time.Nanoseconds()
		if config.esswarm {
			s.ESStep()
		} else if config.pswarm {
			s.PSStep();
		}
		s.Generation++
		s.SaveSwarm(fmt.Sprintf(s.config.prefix+".%d.gob", s.Generation))
		log.Printf("generation %d/%d, best %.0f wins, took %d seconds",
			s.Generation, s.Generations, s.Best().Fitness,
			(time.Nanoseconds()-start)/1e9)
	}
}

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
	Particles     Particles
	GBest         *Particle
	config        *Config
}

func NewSwarm(config *Config) *Swarm {
	var stride int
	var min, max, vmax float64
	if config.Pat {
		stride = len(NewTracker(config).Neighbors(0, 2))
		if config.Tenuki {
			stride++
		}
		min = 0.01
		max = 100
		vmax = 20
	} else if config.Eval {
		stride = 1
		min = 0
		max = 1
		vmax = 0.5
	}
	if config.Mu >= config.Lambda {
		panic("illegal argument to NewSwarm - mu must be less than lambda")
	}
	if config.Parents > config.Mu {
		panic("illegal argument to NewSwarm - p must be less than or equal to mu")
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
		s.Particles[i] = NewParticle(s, min, max, vmax, stride)
	}
	s.GBest = s.Particles[0].Copy()
	return s
}

type Particle struct {
	Strategy       float64
	Position       map[uint32][]float64
	Velocity       map[uint32][]float64
	Min, Max, VMax float64
	PBest          *Particle
	Fitness        float64
	Stride         int
	swarm          *Swarm
}

func NewParticle(swarm *Swarm, min, max, vMax float64, stride int) *Particle {
	p := new(Particle)
	p.swarm = swarm
	p.Strategy = rand.Float64() * 0.05
	p.Position = make(map[uint32][]float64)
	p.Min = min
	p.Max = max
	p.Velocity = make(map[uint32][]float64)
	p.VMax = vMax
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
	if p.swarm.config.Pswarm {
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

func HashVertices(board []uint8, Neighbors []int, offset int) uint32 {
	index := uint32(0)
	for i := 0; i < len(Neighbors); i++ {
		// set the 2*i, 2*i+1 bits of the index
		m0 := uint32(0)
		m1 := uint32(0)
		if Neighbors[i] == -1 {
			m0, m1 = 1, 1
		} else if board[Neighbors[i]] == BLACK {
			m0, m1 = 0, 1
		} else if board[Neighbors[i]] == WHITE {
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
		p.Position[i][j] = p.Min + (p.Max-p.Min)*rand.Float64()
	}
	if p.swarm.config.Pswarm {
		p.Velocity[i] = make([]float64, p.Stride)
		for j := 0; j < p.Stride; j++ {
			p.Velocity[i][j] = -p.VMax + 2*p.VMax*rand.Float64()
		}
	}
}

func (s *Swarm) evaluate(p *Particle) {
	bc := *p.swarm.config
	bc.Pat = false
	bc.matcher = nil
	bc.Eval = false
	bc.evaluator = nil
	wc := *p.swarm.config
	if wc.Eval && wc.Pat {
		panic("evaluator and pattern matching are mutually exclusive")
	}
	if wc.Eval {
		wc.evaluator = p
	}
	if wc.Pat {
		wc.matcher = p
	}
	t := NewTracker(p.swarm.config)
	move := 0
	maxMoves := 3 * t.Sqsize()
	winner := EMPTY
	var vertex int
	for {
		if move == 0 && p.swarm.config.Hex {
			vertex = p.swarm.config.Size + 2
		} else {
			br := NewRoot(BLACK, t, &bc)
			genmove(br, t)
			vertex = br.Best().vertex
			if br.wins/br.visits < 0.01 {
				winner = WHITE
				break
			}
		}
		t.Play(BLACK, vertex)
		winner = t.Winner()
		move++
		if winner != EMPTY || move >= maxMoves {
			break
		}
		wr := NewRoot(WHITE, t, &wc)
		genmove(wr, t)
		vertex = wr.Best().vertex
		if wr.wins/wr.visits < 0.01 {
			winner = BLACK
			break
		}
		t.Play(WHITE, vertex)
		winner = t.Winner()
		move++
		if winner != EMPTY || move >= maxMoves {
			break
		}
	}
	if winner != EMPTY && p.swarm.config.Go {
		gotracker := t.(*GoTracker)
		dead := gotracker.dead()
		for i := range dead {
			gotracker.board[dead[i]] = EMPTY
		}
		bc, wc := gotracker.Score(gotracker.GetKomi())
		winner = WHITE
		if bc > wc {
			winner = BLACK
		}
	}
	log.Println(t.String())
	log.Println(Ctoa(winner))
	if winner == WHITE {
		log.Println("win for white")
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

	Eval := make(chan uint, s.Lambda)
	done := make(chan bool)

	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func(s *Swarm) {
			runtime.LockOSThread()
			for j := range Eval {
				for k := uint(0); k < s.Samples; k++ {
					s.evaluate(children[j])
					done <- true
				}
			}
		}(s)
	}

	// evaluate either children (,) or children + parents (+) for fitness
	for i := uint(0); i < s.Lambda; i++ {
		Eval <- i
	}
	for i := uint(0); i < s.Lambda; i++ {
		for j := uint(0); j < s.Samples; j++ {
			<-done
			log.Printf("%d / %d\n", i*s.Samples+j, s.Lambda*s.Samples)
		}
	}
	close(Eval)

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
func (s *Swarm) recombine(Parents Particles) (r *Particle) {
	r = NewParticle(Parents[0].swarm, Parents[0].Min, Parents[0].Max, Parents[0].VMax, Parents[0].Stride)
	superset := make(map[uint32]bool)
	for i := range Parents {
		for j := range Parents[i].Position {
			superset[j] = true
		}
	}
	for i := range superset {
		r.Position[i] = make([]float64, r.Stride)
		count := 0
		for j := range Parents {
			if Parents[j].Position[i] != nil {
				for k := 0; k < r.Stride; k++ {
					r.Position[i][k] += Parents[j].Position[i][k]
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
	tau := (1 / math.Sqrt(2*dim)) * (1.0 - float64(s.Generation)/float64(s.config.Generations))
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
			} else if p.Velocity[i][j] > p.VMax {
				p.Velocity[i][j] = p.VMax
			}
			p.Position[i][j] += p.Velocity[i][j]
			if p.Position[i][j] < p.Min {
				p.Position[i][j] = p.Min
			} else if p.Position[i][j] > p.Max {
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
		if config.ESswarm {
			s.ESStep()
		} else if config.Pswarm {
			s.PSStep()
		}
		s.Generation++
		filename := fmt.Sprintf(s.config.Prefix+".%d.gob", s.Generation)
		s.SaveSwarm(filename)
		log.Printf("generation %d/%d, best %.0f wins, took %d seconds",
			s.Generation, s.config.Generations, s.Best().Fitness,
			(time.Nanoseconds()-start)/1e9)
	}
}

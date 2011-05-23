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
	"container/vector"
	"log"
)

type Swarm struct {
	Mu, P, Lambda uint
	Samples       uint
	Generation    uint
	Generations   uint
	Particles     Particles
	Arch          []int
	Min, Max      float64
	VMax					float64
	GBest					[]float64
	GBestFit			float64
}

func NewSwarm(mu, p, lambda, samples, generations uint, min, max, vMax float64, dim int, arch []int) *Swarm {
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
	s.Min = min
	s.Max = max
	s.VMax = vMax
	s.Particles = make(Particles, s.Mu)
	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = NewParticle(dim, s)
	}
	s.Arch = arch
	s.GBest = make([]float64, dim)
	copy(s.GBest, s.Particles[0].Position)
	s.GBestFit = 0
	return s
}

type Particle struct {
	Dim      int
	Strategy float64
	Position []float64
	Velocity []float64
	PBest		 []float64
	PBestFit float64
	Fitness  float64
}

func NewParticle(dim int, s *Swarm) *Particle {
	p := new(Particle)
	p.Dim = dim
	p.Strategy = rand.Float64() * 0.05
	p.Position = make([]float64, dim)
	p.Velocity = make([]float64, dim)
	for i := 0; i < p.Dim; i++ {
		p.Position[i] = s.Min + (s.Max - s.Min) * rand.Float64()
		p.Velocity[i] = -s.VMax + s.VMax * rand.Float64()
	}
	p.PBest = make([]float64, dim)
	copy(p.PBest, p.Position)
	p.PBestFit = 0
	return p
}

func (p *Particle) Copy() *Particle {
	cp := new(Particle)
	cp.Dim = p.Dim
	cp.Strategy = p.Strategy
	cp.Position = make([]float64, len(p.Position))
	copy(cp.Position, p.Position)
	cp.Velocity = make([]float64, len(p.Velocity))
	copy(cp.Velocity, p.Velocity)
	cp.Fitness = 0
	cp.PBest = make([]float64, len(p.PBest))
	copy(cp.PBest, p.PBest)
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

func swap(i, j uint16, b *uint16) {
	x := ((*b >> i) ^ (*b >> j)) & 1 // XOR temporary
	*b ^= ((x << i) | (x << j))
}

func compute_index(board []uint8, adj []int) uint16 {
	index := uint16(0)
	for i := 0; i < len(adj); i++ {
		// set the 2*i, 2*i+1 bits of the offset
		m0 := uint16(0)
		m1 := uint16(0)
		if adj[i] == -1 {
			m0, m1 = 1, 1
		} else if board[adj[i]] == BLACK {
			m0, m1 = 0, 1
		} else if board[adj[i]] == WHITE {
			m0, m1 = 1, 0
		}
		index |= (m0 << (2*uint(len(adj)-1-i) + 1))
		index |= (m1 << (2 * uint(len(adj)-1-i)))
	}
	if *hex {
		sym := index
		swap(12, 6, &sym)
		swap(13, 7, &sym)
		swap(10, 4, &sym)
		swap(11, 5, &sym)
		swap(8, 2, &sym)
		swap(9, 3, &sym)
		if sym < index { index = sym }
	}
	return index
}

func (p *Particle) Get(board []byte, adj []int) []float64 {
	i := compute_index(board, adj)
	if *cgo {
		return p.Position[i*9 : (i+1)*9]
	} else if *hex {
		return p.Position[i*7 : (i+1)*7]
	}
	panic("Learning not supported for current game")
}

func playOneGame(black PatternMatcher, white PatternMatcher) Tracker {
	t := NewTracker(*size)
	passes := 0
	var vertex int
	for {
		br := NewRoot(BLACK, t)
		genmove(br, t, black)
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
		genmove(wr, t, white)
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
	return t
}

func (s *Swarm) evaluate(p *Particle) {
	defer func() { recover() }()
	var m PatternMatcher
	if *tablepat {
		m = p
	} else {
		m = &NeuralNet{s.Arch, p.Position}
	}
	t := playOneGame(m, nil)
	if t.Winner() == BLACK {
		p.Fitness += 1
	} else if t.Winner() == WHITE {
		p.Fitness -= 1
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

	// evaluate either children (,) or children + parents (+) for fitness
	for i := uint(0); i < s.Lambda; i++ {
		for j := uint(0); j < s.Samples; j++ {
			s.evaluate(children[i])
			log.Printf("%d / %d\n", i*s.Samples+j, s.Lambda*s.Samples)
		}
	}

	// select mu parents from either children (,) or children + parents (+)
	sort.Sort(children)

	f, err := os.Create(fmt.Sprintf("children-%d.gob", s.Generation))
	if err != nil {
		log.Println("failed to save swarm")
		return
	}
	defer func() { f.Close() }()
	e := gob.NewEncoder(f)
	e.Encode(children)

	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = children[i]
	}

	s.Generation++

	s.SaveSwarm(fmt.Sprintf("swarm-%d.gob", s.Generation))
}

/**
	Particle swarm update
*/
func (s *Swarm) PSStep() {

	s.GBestFit *= 0.9

	for i := uint(0); i < s.Mu; i++ {
		s.update_particle(i)
	}

	s.Generation++

	s.SaveSwarm(fmt.Sprintf("swarm-%d.gob", s.Generation))
}

func (s *Swarm) update_particle(i uint) {
	s.Particles[i].PBestFit *= 0.9
	ch := make(chan bool)
	for j := uint(0); j < s.Samples; j++ {
		go func() {
			s.evaluate(s.Particles[i])
			log.Printf("%d / %d\n", i*s.Samples+j, s.Mu*s.Samples)
			ch <- true
		}()
	}
	for j := uint(0); j < s.Samples; j++ { <-ch }
	s.update_gbest(i)
	s.update_pbest(i)
}

func (s *Swarm) update_gbest(i uint) {
	if s.Particles[i].Fitness > s.GBestFit {
		s.GBestFit = s.Particles[i].Fitness
		copy(s.GBest, s.Particles[i].Position)
		log.Println("updated gbest")
	}
}

func (s *Swarm) update_pbest(i uint) {
	if s.Particles[i].Fitness > s.Particles[i].PBestFit {
		s.Particles[i].PBestFit = s.Particles[i].Fitness
		copy(s.Particles[i].PBest, s.Particles[i].Position)
		log.Printf("updated pbest of particle %d\n", i)
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
		for i := 0; i < len(e); i++ {
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
func (s *Swarm) recombine(p Particles) (r *Particle) {
	r = p[0].Copy()
	for i := 0; i < r.Dim; i++ {
		r.Strategy = 0
		r.Position[i] = 0
		for j := 0; j < len(p); j++ {
			r.Strategy += p[j].Strategy
			r.Position[i] += p[j].Position[i]
			r.Velocity[i] += p[j].Velocity[i]
			r.PBest[i] += p[j].PBest[i]
		}
		r.Strategy /= float64(len(p))
		r.Position[i] /= float64(len(p))
		r.Velocity[i] /= float64(len(p))
		r.PBest[i] /= float64(len(p))
	}
	return
}

/*
	randomly permute particle's position and strategy using evolution strategies method
*/
func (s *Swarm) mutate(p *Particle) {
	tau := (1 / math.Sqrt(2*float64(p.Dim))) * (1.0 - float64(s.Generation)/float64(s.Generations))
	p.Strategy *= math.Exp(tau * rand.NormFloat64())
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

/*
	update position and velocity of particle using particle swarm method
*/
func (s *Swarm) ps_update(p *Particle) {
	w := 0.4 + 0.5 * (1.0 - float64(s.Generation)/float64(s.Generations))
	for i := 0; i < p.Dim; i++ {
		p.Velocity[i] = w * p.Velocity[i] +
			2 * rand.Float64() * (p.PBest[i] - p.Position[i]) +
			2 * rand.Float64() * (s.GBest[i] - p.Position[i])
		if p.Velocity[i] < -s.VMax {
			p.Velocity[i] = -s.VMax
		} else if (p.Velocity[i] > s.VMax) {
			p.Velocity[i] = s.VMax
		}
		p.Position[i] += p.Velocity[i]
		if p.Position[i] < s.Min {
			p.Position[i] = s.Min
		} else if (p.Position[i] > s.Max) {
			p.Position[i] = s.Max
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
		log.Println("failed to save swarm")
		return
	}
	defer func() { f.Close() }()
	e := gob.NewEncoder(f)
	e.Encode(s)
}

func (s *Swarm) LoadSwarm(filename string) os.Error {
	f, err := os.Open(filename)
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
			s = NewSwarm(*mu, *parents, *lambda, *samples, *generations, 0.01, 100, 20, 114688, nil)
		} else {
			s = NewSwarm(*mu, *parents, *lambda, *samples, *generations, 0.01, 100, 20, 2359296, nil)
		}
	} else {
		net := NewNeuralNet([]int{inputsize, 20, 1})
		s = NewSwarm(*mu, *parents, *lambda, *samples, 1000, -10, 10, 4, len(net.Config), net.Arch)
	}
	f, err := os.Open(".")
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
		if *esswarm {
			s.ESStep()
		} else if *pswarm {
			s.PSStep();
		}
		log.Printf("generation %d/%d, best %.0f wins, took %d seconds",
			s.Generation, s.Generations, s.Best().Fitness,
			(time.Nanoseconds()-start)/1e9)
	}
}

// converts gob file to arff for machine learning
func ShowSwarm(filename string) {
	var children Particles
	f, _ := os.Open(filename)
	defer func() { f.Close() }()
	d := gob.NewDecoder(f)
	d.Decode(&children)

	f, _ = os.Create("swarm.arff")
	defer func() { f.Close() }()

	fmt.Fprintln(f, "@RELATION swarm")
	fmt.Fprintln(f, "@ATTRIBUTE fitness NUMERIC")
	i := 0
	valid := new(vector.IntVector)
	dim := children[0].Dim
	for i < (dim / 7) {
		all_zero := true
		for j := 0; j < 7; j++ {
			all_zero = all_zero && children[0].Position[i*7+j] != 0
		}
		if !all_zero {
			fmt.Fprintf(f, "@ATTRIBUTE pattern-%d NUMERIC\n", i)
			valid.Push(i)
		}
		i++
	}
	log.Println(valid.Len())
	fmt.Fprintln(f, "@DATA")
	for child := range children {
		log.Printf("%d / %d\n", child, len(children))
		fmt.Fprintf(f, "%.0f,", children[child].Fitness)
		for i := 0; i < valid.Len(); i++ {
			sum := 0.0
			for j := 0; j < 7; j++ {
				sum += children[child].Position[valid.At(i)*7+j]
			}

			entropy := 0.0
			for j := 0; j < 7; j++ {
				weight := children[child].Position[valid.At(i)*7+j]
				if weight > 0.0 {
					p := weight / sum
					entropy += p * math.Log2(p)
				}
			}
			entropy = -entropy
			fmt.Fprintf(f, "%.6f", entropy)
			if i != valid.Len()-1 {
				fmt.Fprintf(f, ",")
			}
		}
		fmt.Fprintln(f)
		f.Sync()
	}
}

func Compare(p1 PatternMatcher, p2 PatternMatcher, name1 string, name2 string) {
	p1_black_wins, p1_white_wins, p2_black_wins, p2_white_wins := 0.0, 0.0, 0.0, 0.0
	rounds := 100000
	log.Printf("running %d playouts\n", rounds)
	for i := 0; i < rounds; i++ {
		t := NewTracker(*size)
		t.Playout(BLACK, -1, &ColorDuplexingMatcher{p1, p2})
		if t.Winner() == BLACK {
			p1_black_wins++
		} else {
			p2_white_wins++
		}
		t = NewTracker(*size)
		t.Playout(BLACK, -1, &ColorDuplexingMatcher{p2, p1})
		if t.Winner() == BLACK {
			p2_black_wins++
		} else {
			p1_white_wins++
		}
	}
	log.Printf("stats from playouts:\n")
	log.Printf("%s as black: %.0f%%\n", name1, (p1_black_wins/float64(rounds))*100.0)
	log.Printf("%s as white: %.0f%%\n", name1, (p1_white_wins/float64(rounds))*100.0)
	log.Printf("%s overall: %.0f%%\n", name1, ((p1_black_wins+p1_white_wins)/float64(2*rounds))*100.0)
	log.Printf("%s as black: %.0f%%\n", name2, (p2_black_wins/float64(rounds))*100.0)
	log.Printf("%s as white: %.0f%%\n", name2, (p2_white_wins/float64(rounds))*100.0)
	log.Printf("%s overall: %.0f%%\n", name2, ((p2_black_wins+p2_white_wins)/float64(2*rounds))*100.0)

	p1_black_wins, p1_white_wins, p2_black_wins, p2_white_wins = 0.0, 0.0, 0.0, 0.0
	rounds = 100
	log.Printf("running %d full games, relevant settings: UCT? %t, UCT coefficient: %.2f, RAVE cutoff: %.0f, playouts: %d, expand after: %.0f\n", rounds, *uct, *c, *k, *maxPlayouts, *expandAfter)
	for i := 0; i < rounds; i++ {
		t := playOneGame(p1, p2)
		if t.Winner() == BLACK {
			p1_black_wins++
		} else if t.Winner() == WHITE {
			p2_white_wins++
		}
		t = playOneGame(p2, p1)
		if t.Winner() == BLACK {
			p2_black_wins++
		} else if t.Winner() == WHITE {
			p1_white_wins++
		}
		log.Printf("finished round %d / %d\n", i+1, rounds)
	}
	log.Printf("stats from full game:\n")
	log.Printf("%s as black: %.0f%%\n", name1, (p1_black_wins/float64(rounds))*100.0)
	log.Printf("%s as white: %.0f%%\n", name1, (p1_white_wins/float64(rounds))*100.0)
	log.Printf("%s overall: %.0f%%\n", name1, ((p1_black_wins+p1_white_wins)/float64(2*rounds))*100.0)
	log.Printf("%s as black: %.0f%%\n", name2, (p2_black_wins/float64(rounds))*100.0)
	log.Printf("%s as white: %.0f%%\n", name2, (p2_white_wins/float64(rounds))*100.0)
	log.Printf("%s overall: %.0f%%\n", name2, ((p2_black_wins+p2_white_wins)/float64(2*rounds))*100.0)
}

// runs input against random
func TestSwarm() {
	if strings.Contains(*file, "children") {
		var children Particles
		f, _ := os.Open(*file)
		defer func() { f.Close() }()
		d := gob.NewDecoder(f)
		d.Decode(&children)
		log.Println("Loaded children from swarm")
		rand := &RandomMatcher{}
		for i := range children {
			Compare(children[i], rand, fmt.Sprintf("child-%d(fitness=%.0f)", i, children[i].Fitness), "rand")
		}
	} else {
		log.Println("Comparing given swarm to random pattern matcher")
		swarm := LoadTablePatternMatcher(*file, false)
		rand := &RandomMatcher{}
		Compare(swarm, rand, "swarm", "rand")
	}
}

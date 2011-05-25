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
	//"container/vector"
	"log"
)

type Swarm struct {
	Mu, P, Lambda uint
	Samples       uint
	Generation    uint
	Generations   uint
	Particles     Particles
	Arch          []int
	GBest					*Particle
}

func NewSwarm(mu, p, lambda, samples, generations uint, min, max, vMax float64, arch []int) *Swarm {
	s := new(Swarm)
	s.Lambda = lambda
	s.Mu = mu
	s.P = p
	if mu >= lambda {
		panic("illegal argument to NewSwarm - mu must be less than lambda")
	}
	if p > mu {
		panic("illegal argument to NewSwarm - p must be less than or equal to mu")
	}
	s.Samples = samples
	s.Generations = generations
	s.Particles = make(Particles, s.Mu)
	for i := uint(0); i < s.Mu; i++ {
		s.Particles[i] = NewParticle(min, max, vMax)
	}
	s.Arch = arch
	s.GBest = s.Particles[0].Copy()
	return s
}

type Particle struct {
	Strategy float64
	Position map[int][]float64
	Velocity map[int][]float64
	Min, Max, VMax	float64
	PBest		 *Particle
	Fitness  float64
	Stride	int
}

func NewParticle(min, max, vMax float64) *Particle {
	p := new(Particle)
	p.Strategy = rand.Float64() * 0.05
	p.Position = make(map[int][]float64)
	p.Min = min
	p.Max = max
	p.Velocity = make(map[int][]float64)
	p.VMax = vMax
	p.PBest = p.Copy()
	if *hex {
		p.Stride = 7
	} else if *cgo {
		p.Stride = 9
	} else {
		panic("game not supported")
	}
	return p
}

func (p *Particle) Copy() *Particle {
	cp := new(Particle)
	cp.Strategy = p.Strategy
	cp.Position = make(map[int][]float64)
	for i := range p.Position {
		cp.Position[i] = make([]float64, p.Stride)
		copy(cp.Position[i], p.Position[i])
	}
	if *pswarm {
		cp.Velocity = make(map[int][]float64)
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

func compute_index(board []uint8, adj []int) int {
	index := uint32(0)
	for i := 0; i < len(adj); i++ {
		// set the 2*i, 2*i+1 bits of the index
		m0 := uint32(0)
		m1 := uint32(0)
		if adj[i] == -1 {
			m0, m1 = 1, 1
		} else if board[adj[i]] == BLACK {
			m0, m1 = 0, 1
		} else if board[adj[i]] == WHITE {
			m0, m1 = 1, 0
		}
		set(uint32(2*i), m0, &index)
		set(uint32(2*i+1), m1, &index)
	}
	if *hex {
		/*
		sym := uint32(0)
		set(0, get(6, index), &sym)
		set(1, get(7, index), &sym)
		set(2, get(8, index), &sym)
		set(3, get(9, index), &sym)
		set(4, get(10, index), &sym)
		set(5, get(11, index), &sym)
		set(6, get(0, index), &sym)
		set(7, get(1, index), &sym)
		set(8, get(2, index), &sym)
		set(9, get(3, index), &sym)
		set(10, get(4, index), &sym)
		set(11, get(5, index), &sym)
		set(12, get(12, index), &sym)
		set(13, get(13, index), &sym)
		if sym < index { index = sym }
		*/
	} else if *cgo {
		/*
		sym := uint32(0)
		set(0, get(16, index), &sym)
		set(1, get(17, index), &sym)
		set(2, get(14, index), &sym)
		set(3, get(15, index), &sym)
		set(4, get(12, index), &sym)
		set(5, get(13, index), &sym)
		set(6, get(10, index), &sym)
		set(7, get(11, index), &sym)
		set(8, get(8, index), &sym)
		set(9, get(9, index), &sym)
		set(10, get(6, index), &sym)
		set(11, get(7, index), &sym)
		set(12, get(4, index), &sym)
		set(13, get(5, index), &sym)
		set(14, get(2, index), &sym)
		set(15, get(3, index), &sym)
		set(16, get(0, index), &sym)
		set(17, get(1, index), &sym)
		if sym < index { index = sym }
		sym = uint32(0)
		set(0, get(4, index), &sym)
		set(1, get(5, index), &sym)
		set(2, get(10, index), &sym)
		set(3, get(11, index), &sym)
		set(4, get(16, index), &sym)
		set(5, get(17, index), &sym)
		set(6, get(2, index), &sym)
		set(7, get(3, index), &sym)
		set(8, get(8, index), &sym)
		set(9, get(9, index), &sym)
		set(10, get(14, index), &sym)
		set(11, get(15, index), &sym)
		set(12, get(0, index), &sym)
		set(13, get(1, index), &sym)
		set(14, get(6, index), &sym)
		set(15, get(7, index), &sym)
		set(16, get(12, index), &sym)
		set(17, get(13, index), &sym)
		if sym < index { index = sym }
		sym = uint32(0)
		set(0, get(12, index), &sym)
		set(1, get(13, index), &sym)
		set(2, get(6, index), &sym)
		set(3, get(7, index), &sym)
		set(4, get(0, index), &sym)
		set(5, get(1, index), &sym)
		set(6, get(14, index), &sym)
		set(7, get(15, index), &sym)
		set(8, get(8, index), &sym)
		set(9, get(9, index), &sym)
		set(10, get(2, index), &sym)
		set(11, get(3, index), &sym)
		set(12, get(16, index), &sym)
		set(13, get(17, index), &sym)
		set(14, get(10, index), &sym)
		set(15, get(11, index), &sym)
		set(16, get(4, index), &sym)
		set(17, get(5, index), &sym)
		if sym < index { index = sym }
		*/
	}
	return int(index)
}

func (p *Particle) Get(board []byte, adj []int) []float64 {
	i := compute_index(board, adj)
	if p.Position[i] == nil {
		p.Init(i)
	}
	return p.Position[i]
}

func (p *Particle) Init(i int) {
	p.Position[i] = make([]float64, p.Stride)
	for j := 0; j < p.Stride; j++ {
		p.Position[i][j] = p.Min + (p.Max - p.Min) * rand.Float64()
	}
	if *pswarm {
		p.Velocity[i] = make([]float64, p.Stride)
		for j := 0; j < p.Stride; j++ {
			p.Velocity[i][j] = -p.VMax + 2 * p.VMax * rand.Float64()
		}
	}
}

func playOneGame(black PatternMatcher, white PatternMatcher) Tracker {
	t := NewTracker(*size)
	passes := 0
	move := 0
	maxMoves := 2 * t.Boardsize() * t.Boardsize()
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
		move++
		if (*hex && t.Winner() != EMPTY) || move >= maxMoves || passes == 2 {
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
		move++
		if (*hex && t.Winner() != EMPTY) || move >= maxMoves || passes == 2 {
			break
		}
	}
	return t
}

func (s *Swarm) evaluate(p *Particle) {
	var m PatternMatcher
	if *tablepat {
		m = p
	} else {
		panic("only support table patterns right now") 
		//m = &NeuralNet{s.Arch, p.Position}
	}
	t := playOneGame(m, nil)
	t.SetKomi(7.5)
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
		children[i].Fitness = 0
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

	s.GBest.Fitness *= 0.9

	for i := uint(0); i < s.Mu; i++ {
		s.update_particle(i)
	}

	s.Generation++

	s.SaveSwarm(fmt.Sprintf("swarm-%d.gob", s.Generation))
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
func (s *Swarm) recombine(parents Particles) (r *Particle) {
	r = NewParticle(parents[0].Min, parents[0].Max, parents[0].VMax)
	superset := make(map[int]bool)
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
	superset := make(map[int]bool)
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
			s = NewSwarm(*mu, *parents, *lambda, *samples, *generations, 0.01, 100, 20, nil)
		} else {
			s = NewSwarm(*mu, *parents, *lambda, *samples, *generations, 0.01, 100, 20, nil)
		}
	} else {
		panic("neural nets not supported")
		//net := NewNeuralNet([]int{inputsize, 20, 1})
		//s = NewSwarm(*mu, *parents, *lambda, *samples, 1000, -10, 10, 4, len(net.Config), net.Arch)
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
	/*
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
	*/
}

func Compare(p1 PatternMatcher, p2 PatternMatcher, name1 string, name2 string) {
	p1_black_wins, p1_white_wins, p2_black_wins, p2_white_wins := 0.0, 0.0, 0.0, 0.0
	rounds := 100000
	log.Printf("running %d playouts\n", rounds)
	for i := 0; i < rounds; i++ {
		t := NewTracker(*size)
		t.Playout(BLACK, &ColorDuplexingMatcher{p1, p2})
		if t.Winner() == BLACK {
			p1_black_wins++
		} else {
			p2_white_wins++
		}
		t = NewTracker(*size)
		t.Playout(BLACK, &ColorDuplexingMatcher{p2, p1})
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

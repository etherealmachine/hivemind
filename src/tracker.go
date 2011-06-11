package main

import "container/vector"
import "rand"

const (
	UP         = 0
	DOWN       = 1
	LEFT       = 2
	RIGHT      = 3
	UP_LEFT    = 0
	UP_RIGHT   = 1
	DOWN_LEFT  = 4
	DOWN_RIGHT = 5
)

var SIDE_UP int
var SIDE_DOWN int
var SIDE_LEFT int
var SIDE_RIGHT int

type Tracker interface {
	Copy() Tracker
	Play(color byte, vertex int)
	Playout(color byte, m PatternMatcher)
	WasPlayed(color byte, vertex int) bool
	Legal(color byte, vertex int) bool
	Score(Komi float64) (float64, float64)
	Winner() byte
	SetKomi(Komi float64)
	GetKomi() float64
	Boardsize() int
	Sqsize() int
	Board() []byte
	Territory(color byte) []float64
	Verify()
	Neighbors(vertex int, Size int) []int
	Adj(vertex int) []int
	String() string
	Vtoa(vertex int) string
	Atov(s string) int
}

func NewTracker(config *Config) Tracker {
	if config.Go {
		return NewGoTracker(config)
	} else if config.Hex {
		return NewHexTracker(config)
	}
	return nil
}

// standard union-find Find op, also does path compression
func find(i int, parent []int) int {
	if i == parent[i] {
		return i
	}
	if i == -1 {
		return i
	}
	root := i
	for root != parent[root] {
		root = parent[root]
	}
	j := i
	for j != parent[j] {
		parent[j] = root
		j = parent[j]
	}
	return root
}

// standard union op, uses union-by-rank
// returns the new root of the tree
func union(i int, j int, parent []int, rank []int) int {
	i = find(i, parent)
	j = find(j, parent)
	if rank[i] > rank[j] {
		parent[j] = i
		rank[i] += rank[j]
		return i
	} else if i != j {
		parent[i] = j
		rank[j] += rank[i]
		return j
	}
	return i
}

// union find that assumes i and j are the parents of their respective trees
func fastUnion(i int, j int, parent []int, rank []int) int {
	if i != j {
		if rank[i] > rank[j] {
			parent[j] = i
			rank[i] += rank[j]
			return i
		} else {
			parent[i] = j
			rank[j] += rank[i]
			return j
		}
	}
	return i
}

// Fisher-Yates (Knuth) Shuffle
func shuffle(v *vector.IntVector) {
	for i := v.Len() - 1; i >= 1; i-- {
		v.Swap(i, rand.Intn(i+1))
	}
}

package main

import (
	"rand"
	"log"
)

var queries int
var matches int

var patternLog []int

type PatternMatcher interface {
	Match(color byte, vertex int, t Tracker) int
}

type ColorDuplexingMatcher struct {
	black PatternMatcher
	white PatternMatcher
}

func (m *ColorDuplexingMatcher) Match(color byte, v int, t Tracker) int {
	if color == BLACK {
		return m.black.Match(color, v, t)
	} else if color == WHITE {
		return m.white.Match(color, v, t)
	}
	panic("can't duplex onto empty")
}

func (m *Particle) Match(color byte, v int, t Tracker) int {
	queries++
	s := t.Boardsize()
	b := t.Board()
	neighbors := t.Neighbors(v)
	i, pat := m.Get(b, neighbors)
	patternLog[i]++
	for i := 0; i < len(neighbors); i++ {
		if neighbors[i] == -1 || b[neighbors[i]] != EMPTY || !t.Legal(color, neighbors[i]) {
			pat[i] = 0
		}
	}
	sum := 0.0
	for i := range pat {
		sum += pat[i]
	}
	if sum == 0 {
		return -1
	}
	r := rand.Float64()
	for i := range pat {
		r -= pat[i] / sum
		if r <= 0 {
			if i == len(neighbors) {
				return -1
			}
			matches++
			return neighbors[i]
		}
	}
	log.Println(Vtoa(v, s))
	log.Println(Bwboard(b, s, true))
	log.Println(pat)
	panic("pattern error, not a valid probability distribution")
}

func LoadPatternMatcher(filename string, disabled []int) PatternMatcher {
	if *pat {
		particle := LoadBest(filename)
		for i := range disabled {
			pattern := particle.Position[disabled[i]]
			for j := range pattern {
				pattern[j] = 0.0
			}
		}
		return particle
	}
	return nil
}	

func init() {
	patternLog = make([]int, 16384)
}

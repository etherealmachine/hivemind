package main

import (
	"rand"
	"log"
)

var queries int
var matches int

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

func (p *Particle) Match(color byte, v int, t Tracker) int {
	queries++
	b := t.Board()
	Neighbors := t.Neighbors(v, 2)
	i := HashVertices(b, Neighbors, 10)
	Pat := p.Get(i)
	for i := 0; i < len(Neighbors); i++ {
		if Neighbors[i] == -1 || b[Neighbors[i]] != EMPTY || !t.Legal(color, Neighbors[i]) {
			Pat[i] = 0
		}
	}
	sum := 0.0
	for i := range Pat {
		sum += Pat[i]
	}
	if sum == 0 {
		return -1
	}
	r := rand.Float64()
	for i := range Pat {
		r -= Pat[i] / sum
		if r <= 0 {
			if i == len(Neighbors) {
				return -1
			}
			matches++
			return Neighbors[i]
		}
	}
	log.Println(t.Vtoa(v))
	log.Println(t.String())
	log.Println(Pat)
	panic("pattern error, not a valid probability distribution")
}

func LoadPatternMatcher(config *Config) {
	if config.Pat && config.Pfile != "" {
		particle := LoadBest(config.Pfile, config)
		/*
			for i := range disabled {
				pattern := particle.Position[disabled[i]]
				for j := range pattern {
					pattern[j] = 0.0
				}
			}
		*/
		config.matcher = particle
	}
}

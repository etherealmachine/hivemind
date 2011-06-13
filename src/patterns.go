package main

import (
	"rand"
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
	neighbors := t.Neighbors(v, 2)
	pos := HashVertices(b, neighbors, 10)
	pat := p.Get(pos)
	for i := 0; i < len(neighbors); i++ {
		if neighbors[i] == -1 || b[neighbors[i]] != EMPTY || !t.Legal(color, neighbors[i]) {
			pat[i] = 0
		}
	}
	sum := 0.0
	for i := range pat {
		sum += pat[i]
	}
	if sum > 0 {
		r := rand.Float64()
		for i := range pat {
			r -= pat[i] / sum
			if r <= 0 {
				if i == len(neighbors) {
					if p.log != nil {
						p.log[pos] = true
					}
					return -1
				}
				matches++
				return neighbors[i]
			}
		}
	}
	return -1
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

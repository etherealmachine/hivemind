package main

type BoardEvaluator interface {
	Eval(color byte, t Tracker) float64
}

func (p *Particle) Eval(color byte, t Tracker) float64 {
	b := t.Board()
	sum := 0.0
	for v := range b {
		sum += p.Get(NeighborhoodHash(color, b, t.Neighbors(v, 0), 1))
		sum += p.Get(NeighborhoodHash(color, b, t.Neighbors(v, 1), 3))
		sum += p.Get(NeighborhoodHash(color, b, t.Neighbors(v, 2), 11))
	}
	return sum / (3 * float64(t.Sqsize()))
}

func LoadBoardEvaluator(config *Config) {
	if config.Eval && config.Efile != "" {
		particle := LoadBest(config.Efile, config)
		config.evaluator = particle
	}
}

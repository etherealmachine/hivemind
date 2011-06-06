package main

type BoardEvaluator interface {
	Eval(color byte, t Tracker) float64
}

func (p *Particle) Eval(color byte, t Tracker) float64 {
	b := t.Board()
	sum := 0.0
	for v := range b {
		sum += p.Get(HashVertices(b, t.Neighbors(v, 0), 0))[0]
		sum += p.Get(HashVertices(b, t.Neighbors(v, 1), 2))[0]
		sum += p.Get(HashVertices(b, t.Neighbors(v, 2), 10))[0]
	}
	return sum / (3 * float64(t.Sqsize()))
}

func LoadBoardEvaluator(config *Config) {
	if config.eval && config.evalFile != "" {
		particle := LoadBest(config.evalFile, config)
		config.evaluator = particle
	}
}	

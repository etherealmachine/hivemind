package main

import (
	"flag"
)

type Config struct {
	Help  bool
	Gtp   bool
	Test  bool
	Speed bool

	MaxPlayouts uint
	Timelimit   int
	Cutoff      float64

	Prefix string
	Sfile  string
	Efile  string
	Pfile  string

	Stats bool

	Go       bool
	Hex      bool
	Swapsafe bool

	Size int
	Komi float64

	Train       bool
	Pswarm      bool
	ESswarm     bool
	Generations uint
	Mu          uint
	Parents     uint
	Lambda      uint
	Samples     uint

	Gfx bool

	Eval   bool
	Pat    bool
	Tenuki bool

	Explore     float64
	RAVE        float64
	ExpandAfter float64
	AMAF        bool
	Neighbors   bool
	Ancestor    bool
	Seed        bool

	Verbose bool
	Lfile   string

	SGF string

	matcher   PatternMatcher
	evaluator BoardEvaluator
}

func NewConfig() *Config {
	config := new(Config)

	flag.BoolVar(&config.Help, "h", false, "Print this usage message")
	flag.BoolVar(&config.Gtp, "gtp", false, "Listen on stdin for GTP commands")
	flag.BoolVar(&config.Test, "test", false, "Just generate a single move")
	flag.BoolVar(&config.Speed, "pps", false, "Gather data on the playouts per second")

	flag.UintVar(&config.MaxPlayouts, "p", 10000, "Max number of playouts")
	flag.IntVar(&config.Timelimit, "t", -1, "Max number of seconds")
	flag.Float64Var(&config.Cutoff, "cutoff", -1, "End search if ratio of visits to top 2 moves is greater than cutoff")

	flag.StringVar(&config.Prefix, "prefix", "", "Prefix to use when saving file")
	flag.StringVar(&config.Sfile, "sfile", "", "Load swarm from file")
	flag.StringVar(&config.Efile, "efile", "", "Load evaluator from file")
	flag.StringVar(&config.Pfile, "pfile", "", "Load pattern matcher from file")

	flag.BoolVar(&config.Stats, "stats", false, "Print out tree search statistics")

	flag.BoolVar(&config.Go, "go", false, "Play Go")
	flag.BoolVar(&config.Hex, "hex", false, "Play Hex")
	flag.BoolVar(&config.Swapsafe, "swapsafe", false, "When playing hex, black will choose a swap-safe opening move")

	flag.IntVar(&config.Size, "size", 9, "Boardsize")
	flag.Float64Var(&config.Komi, "komi", 6.5, "Komi")

	flag.BoolVar(&config.Train, "train", false, "Do crazy unsupervised training stuff")
	flag.BoolVar(&config.Pswarm, "pswarm", false, "Train with particle swarm")
	flag.BoolVar(&config.ESswarm, "esswarm", false, "Train with evolution strategies")
	flag.UintVar(&config.Generations, "gens", 100, "Generations to train for")
	flag.UintVar(&config.Mu, "mu", 30, "(Training) Children to keep")
	flag.UintVar(&config.Parents, "parents", 2, "(Training) Parents per child")
	flag.UintVar(&config.Lambda, "lambda", 50, "(Training) Children")
	flag.UintVar(&config.Samples, "samples", 7, "(Training) Evaluations per generation")

	flag.BoolVar(&config.Gfx, "gfx", false, "Emit live graphics for gogui")

	flag.BoolVar(&config.Eval, "eval", false, "Use evaluator")
	flag.BoolVar(&config.Pat, "pat", false, "Use patterns")
	flag.BoolVar(&config.Tenuki, "tenuki", false, "Use tenuki inside patterns")

	flag.Float64Var(&config.Explore, "c", 0.5, "UCT coefficient")
	flag.Float64Var(&config.RAVE, "k", 1000, "RAVE equivalency cutoff")
	flag.Float64Var(&config.ExpandAfter, "e", 50, "Expand after")
	flag.BoolVar(&config.AMAF, "amaf", false, "Use AMAF results in RAVE mean")
	flag.BoolVar(&config.Neighbors, "neighbors", false, "Use neighbors results in RAVE mean")
	flag.BoolVar(&config.Ancestor, "ancestor", false, "Use ancestors results in RAVE mean")
	flag.BoolVar(&config.Seed, "seed", false, "Seed the playouts using ancestor's results")

	flag.BoolVar(&config.Verbose, "v", false, "Verbose logging")
	flag.StringVar(&config.Lfile, "log", "", "Log to filename")

	flag.StringVar(&config.SGF, "sgf", "", "Load sgf file and generate move")

	flag.Parse()

	if !(config.Go || config.Hex) {
		config.Go = true
	}
	if config.Go {
		config.Hex = false
	}
	if config.Hex {
		config.Go = false
	}
	if !(config.Pswarm || config.ESswarm) {
		config.ESswarm = true
	}
	if config.Pswarm {
		config.ESswarm = false
	}
	if config.ESswarm {
		config.Pswarm = false
	}

	return config
}

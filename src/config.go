package main

import (
	"flag"
)

type Config struct {

	help bool
	modeGTP bool

	maxPlayouts uint
	timelimit int
	cutoff float64

	file string
	configFile string

	stats bool

	cgo bool
	hex bool
	swapsafe bool

	test bool
	size int
	komi float64
	testPPS bool

	makeBook bool
	useBook bool

	train bool
	pswarm bool
	esswarm bool
	generations uint
	mu uint
	parents uint
	lambda uint
	samples uint
	prefix string

	gfx bool

	eval bool
	pat bool
	evalFile string
	patFile string
	tenuki bool

	c float64
	k float64
	expandAfter float64
	amaf bool
	neighbors bool
	seedPlayouts bool

	verbose bool
	logFile string

	sgf string

	matcher PatternMatcher
	evaluator BoardEvaluator
}

func NewConfig() *Config {
	config := new(Config)

	flag.BoolVar(&config.help, "h", false, "Print this usage message")
	flag.BoolVar(&config.modeGTP, "gtp", false, "Listen on stdin for GTP commands")

	flag.UintVar(&config.maxPlayouts, "p", 10000, "Max number of playouts")
	flag.IntVar(&config.timelimit, "t", -1, "Max number of seconds")
	flag.Float64Var(&config.cutoff, "cutoff", -1, "End search if ratio of visits to top 2 moves is greater than cutoff")

	flag.StringVar(&config.file, "file", "", "Load data from file")
	flag.StringVar(&config.configFile, "config", "", "Load config from file")

	flag.BoolVar(&config.stats, "stats", false, "Print out tree search statistics")

	flag.BoolVar(&config.cgo, "go", false, "Play Go")
	flag.BoolVar(&config.hex, "hex", false, "Play Hex")
	flag.BoolVar(&config.swapsafe, "swapsafe", false, "When playing hex, black will choose a swap-safe opening move")

	flag.BoolVar(&config.test, "test", false, "Just generate a single move")
	flag.IntVar(&config.size, "size", 9, "Boardsize")
	flag.Float64Var(&config.komi, "komi", 6.5, "Komi")
	flag.BoolVar(&config.testPPS, "pps", false, "Gather data on the playouts per second")

	flag.BoolVar(&config.makeBook, "makebook", false, "Construct opening book")
	flag.BoolVar(&config.useBook, "book", false, "Use opening book")

	flag.BoolVar(&config.train, "train", false, "Do crazy unsupervised training stuff")
	flag.BoolVar(&config.pswarm, "pswarm", false, "Train with particle swarm")
	flag.BoolVar(&config.esswarm, "esswarm", false, "Train with evolution strategies")
	flag.UintVar(&config.generations, "gens", 100, "Generations to train for")
	flag.UintVar(&config.mu, "mu", 30, "(Training) Children to keep")
	flag.UintVar(&config.parents, "parents", 2, "(Training) Parents per child")
	flag.UintVar(&config.lambda, "lambda", 50, "(Training) Children")
	flag.UintVar(&config.samples, "samples", 7, "(Training) Evaluations per generation")
	flag.StringVar(&config.prefix, "prefix", "swarm", "Prefix to use when saving swarm")

	flag.BoolVar(&config.gfx, "gfx", false, "Emit live graphics for gogui")

	flag.BoolVar(&config.eval, "eval", false, "Use evaluator")
	flag.BoolVar(&config.pat, "pat", false, "Use patterns")
	flag.StringVar(&config.evalFile, "efile", "", "Load pattern matcher from file")
	flag.StringVar(&config.patFile, "pfile", "", "Load evaluator from file")
	flag.BoolVar(&config.tenuki, "tenuki", false, "Use tenuki inside patterns")

	flag.Float64Var(&config.c, "c", 0.5, "UCT coefficient")
	flag.Float64Var(&config.k, "k", 1000, "RAVE equivalency cutoff")
	flag.Float64Var(&config.expandAfter, "e", 2, "Expand after")
	flag.BoolVar(&config.amaf, "amaf", false, "Use AMAF results in RAVE mean")
	flag.BoolVar(&config.neighbors, "neighbors", false, "Use neighbors results in RAVE mean")
	flag.BoolVar(&config.seedPlayouts, "seed", false, "Seed the playouts using ancestor's results")

	flag.BoolVar(&config.verbose, "v", false, "Verbose logging")
	flag.StringVar(&config.logFile, "log", "", "Log to filename")

	flag.StringVar(&config.sgf, "sgf", "", "Load sgf file and generate move")

	flag.Parse()
	
	if !(config.cgo || config.hex) { config.cgo = true }
	if config.cgo { config.hex = false }
	if config.hex { config.cgo = false }
	if !(config.pswarm || config.esswarm) { config.esswarm = true }
	if config.pswarm { config.esswarm = false }
	if config.esswarm { config.pswarm = false }
	
	return config
}

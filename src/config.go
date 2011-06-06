package main

import (
	"flag"
)

type Config struct {
	help *bool
	modeGTP *bool

	maxPlayouts *uint
	timelimit *int

	file *string
	configFile *string

	stats *bool

	cgo *bool
	hex *bool
	swapsafe *bool

	test *bool
	size *int
	komi *float64
	testPPS *bool

	makeBook *bool
	useBook *bool

	train *bool
	pswarm *bool
	esswarm *bool
	generations *uint
	mu *uint
	parents *uint
	lambda *uint
	samples *uint
	prefix *string

	gfx *bool

	eval *bool
	pat *bool
	evalFile *string
	patFile *string
	tenuki *bool
	logpat *bool

	c *float64
	k *float64
	expandAfter *float64
	amaf *bool
	neighbors *bool
	seedPlayouts *bool

	verbose *bool
	logFile *string

	sgf *string
}

func NewConfig() *Config {
	config := new(Config)

	config.help = flag.Bool("h", false, "Print this usage message")
	config.modeGTP = flag.Bool("gtp", false, "Listen on stdin for GTP commands")

	config.maxPlayouts = flag.Uint("p", 10000, "Max number of playouts")
	config.timelimit = flag.Int("t", -1, "Max number of seconds")

	config.file = flag.String("file", "", "Load data from file")
	config.configFile = flag.String("config", "", "Load config from file")

	config.stats = flag.Bool("stats", false, "Print out tree search statistics")

	config.cgo = flag.Bool("go", false, "Play Go")
	config.hex = flag.Bool("hex", false, "Play Hex")
	config.swapsafe = flag.Bool("swapsafe", false, "When playing hex, black will choose a swap-safe opening move")

	config.test = flag.Bool("test", false, "Just generate a single move")
	config.size = flag.Int("size", 9, "Boardsize")
	config.komi = flag.Float64("komi", 6.5, "Komi")
	config.testPPS = flag.Bool("pps", false, "Gather data on the playouts per second")

	config.makeBook = flag.Bool("makebook", false, "Construct opening book")
	config.useBook = flag.Bool("book", false, "Use opening book")

	config.train = flag.Bool("train", false, "Do crazy unsupervised training stuff")
	config.pswarm = flag.Bool("pswarm", false, "Train with particle swarm")
	config.esswarm = flag.Bool("esswarm", false, "Train with evolution strategies")
	config.generations = flag.Uint("gens", 100, "Generations to train for")
	config.mu = flag.Uint("mu", 30, "(Training) Children to keep")
	config.parents = flag.Uint("parents", 2, "(Training) Parents per child")
	config.lambda = flag.Uint("lambda", 50, "(Training) Children")
	config.samples = flag.Uint("samples", 7, "(Training) Evaluations per generation")
	config.prefix = flag.String("prefix", "swarm", "Prefix to use when saving swarm")

	config.gfx = flag.Bool("gfx", false, "Emit live graphics for gogui")

	config.eval = flag.Bool("eval", false, "Use evaluator")
	config.pat = flag.Bool("pat", false, "Use patterns")
	config.evalFile = flag.String("efile", "", "Load pattern matcher from file")
	config.patFile = flag.String("pfile", "", "Load evaluator from file")
	config.tenuki = flag.Bool("tenuki", false, "Use tenuki inside patterns")
	config.logpat = flag.Bool("logpat", false, "Log patterns")

	config.c = flag.Float64("c", 0.5, "UCT coefficient")
	config.k = flag.Float64("k", 1000, "RAVE equivalency cutoff")
	config.expandAfter = flag.Float64("e", 2, "Expand after")
	config.amaf = flag.Bool("amaf", false, "Use AMAF results in RAVE mean")
	config.neighbors = flag.Bool("neighbors", false, "Use neighbors results in RAVE mean")
	config.seedPlayouts = flag.Bool("seed", false, "Seed the playouts using ancestor's results")

	config.verbose = flag.Bool("v", false, "Verbose logging")
	config.logFile = flag.String("log", "", "Log to filename")

	config.sgf = flag.String("sgf", "", "Load sgf file and generate move")

	flag.Parse()
	if !(*config.cgo || *config.hex) { *config.cgo = true }
	if *config.cgo { *config.hex = false }
	if *config.hex { *config.cgo = false }
	if !(*config.pswarm || *config.esswarm) { *config.esswarm = true }
	if *config.pswarm { *config.esswarm = false }
	if *config.esswarm { *config.pswarm = false }
	
	return config
}

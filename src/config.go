package main

import (
	"flag"
	"os"
	"json"
	"log"
)

type Config struct {
	// Modes
	Help  bool
	Gtp   bool
	Book  bool
	Genmove bool
	PlayGame bool
	SGF string
	Cluster bool

	// Time limits
	MaxPlayouts uint
	Timelimit   int
	Cutoff      float64

	// Log search stats
	Stats bool
	// Display search stats live for gogui
	Gfx bool

	// Different games
	Go       bool
	Hex      bool
	HexFast  bool

	// Game-specific variables
	Size int
	Komi float64
	Swapsafe bool

	// Learning
	Train        bool
	Generations  uint
	Mu           uint
	Parents      uint
	Lambda       uint
	Samples      uint
	Propagate    uint
	
	// Load/save different modules
	Prefix string
	Bfile  string
	Efile  string
	Pfile  string
	Sfile  string

	// Tree exploration/expansion
	TreeSearch  bool
	Explore     float64
	RAVE        float64
	ExpandAfter float64
	Var         bool
	AMAF        bool
	Neighbors   bool
	Ancestor    bool
	Seed        bool
	PlayoutProbs bool
	PlayoutSuggest bool
	PlayoutSuggestUniform bool
	PlayoutSuggestUniformTenuki bool

	// Logging
	Verbose bool
	VeryVerbose bool
	Verify  bool
	PrintWeights  bool
	Lfile   string
	
	// Used by cluster to store game history
	Moves []int
	
	// Used by cluster to select message type
	MsgType string
	
	// Used by cluster
	Black_policy_weights   *Particle
	White_policy_weights   *Particle

	// private flag, used to load config from json file
	cfile  string
	
	// private fields, set by Bfile, Pfile and Efile
	book      *Node
	policy_weights   *Particle
	
	// log files
	probLog *os.File
}

func NewConfig() *Config {
	config := new(Config)

	flag.BoolVar(&config.Help, "h", false, "Print this usage message")
	flag.BoolVar(&config.Gtp, "gtp", false, "Listen on stdin for GTP commands")
	flag.StringVar(&config.SGF, "sgf", "", "Load sgf file and generate move")
	flag.BoolVar(&config.Book, "book", false, "Make opening book")
	flag.BoolVar(&config.Genmove, "genmove", false, "Generate one move and quit")
	flag.BoolVar(&config.PlayGame, "playgame", false, "Self-play one game")
	flag.BoolVar(&config.Cluster, "cluster", false, "Start cluster")

	flag.UintVar(&config.MaxPlayouts, "p", 10000, "Max number of playouts")
	flag.IntVar(&config.Timelimit, "t", -1, "Max number of seconds")
	flag.Float64Var(&config.Cutoff, "cutoff", -1, "End search if ratio of visits to top 2 moves is greater than cutoff")

	flag.BoolVar(&config.Stats, "stats", false, "Print out tree search statistics")
	flag.BoolVar(&config.Gfx, "gfx", false, "Emit live graphics for gogui")

	flag.BoolVar(&config.Go, "go", false, "Play Go")
	flag.BoolVar(&config.Hex, "hex", false, "Play Hex")
	flag.BoolVar(&config.HexFast, "hexfast", false, "Play Hex using fast tracker")

	flag.IntVar(&config.Size, "size", 9, "Boardsize")
	flag.Float64Var(&config.Komi, "komi", 6.5, "Komi")
	flag.BoolVar(&config.Swapsafe, "swapsafe", false, "When playing hex, black will choose a swap-safe opening move")

	flag.BoolVar(&config.Train, "train", false, "(Training) Do crazy unsupervised training stuff")
	flag.UintVar(&config.Generations, "gens", 100, "(Training) Generations to train for")
	flag.UintVar(&config.Mu, "mu", 30, "(Training) Children to keep")
	flag.UintVar(&config.Parents, "parents", 2, "(Training) Parents per child")
	flag.UintVar(&config.Lambda, "lambda", 50, "(Training) Children")
	flag.UintVar(&config.Samples, "samples", 7, "(Training) Evaluations per generation")
	flag.UintVar(&config.Propagate, "prop", 2, "(Training) Propagate prop best from last generation")
	
	flag.StringVar(&config.Prefix, "prefix", "", "Prefix to use when saving file")
	flag.StringVar(&config.Sfile, "sfile", "", "Load swarm from file")
	flag.StringVar(&config.Efile, "efile", "", "Load evaluator from file")
	flag.StringVar(&config.Pfile, "pfile", "", "Load policy weights from file")
	flag.StringVar(&config.Bfile, "bfile", "", "Load book from file")
	flag.StringVar(&config.cfile, "cfile", "", "Load config from file")

	flag.BoolVar(&config.TreeSearch, "treesearch", true, "Use UCT (if false, use flat MC playouts)")
	flag.Float64Var(&config.Explore, "c", 0.5, "UCT coefficient")
	flag.Float64Var(&config.RAVE, "k", 1000, "RAVE equivalency cutoff")
	flag.Float64Var(&config.ExpandAfter, "e", 50, "Expand after")
	flag.BoolVar(&config.Var, "var", false, "Use variance in UCB value")
	flag.BoolVar(&config.AMAF, "amaf", false, "Use AMAF results in RAVE mean")
	flag.BoolVar(&config.Neighbors, "neighbors", false, "Use neighbors results in RAVE mean")
	flag.BoolVar(&config.Ancestor, "ancestor", false, "Use ancestors results in RAVE mean")
	flag.BoolVar(&config.Seed, "seed", false, "Seed the playouts using ancestor's results")
	flag.BoolVar(&config.PlayoutProbs, "playout_probs", false, "Use policy weights to bias the playout over time")
	flag.BoolVar(&config.PlayoutSuggest, "playout_suggest", false, "Use policy weights as suggested local response to move")
	flag.BoolVar(&config.PlayoutSuggestUniform, "playout_suggest_uniform", false, "Use uniform random local response")
	flag.BoolVar(&config.PlayoutSuggestUniformTenuki, "playout_suggest_uniform_tenuki", false, "Include probability of tenuki in local response")

	flag.BoolVar(&config.Verbose, "v", false, "Verbose logging")
	flag.BoolVar(&config.VeryVerbose, "vv", false, "Very verbose logging")
	flag.BoolVar(&config.Verify, "verify", false, "Verify correctness of playout")
	flag.BoolVar(&config.PrintWeights, "printweights", false, "Print weights to file")
	flag.StringVar(&config.Lfile, "log", "", "Log to filename")

	flag.Parse()
	
	if config.cfile != "" {
		config.Load()
	}
	
	LoadBook(config)
	if config.Pfile != "" {
		config.policy_weights = LoadBest(config.Pfile, config)
	}

	if !(config.Go || config.Hex) {
		config.Go = true
	}
	if config.Go {
		config.Hex = false
	}
	if config.Hex {
		config.Go = false
	}
	
	var f *os.File
	var err os.Error
	if config.Lfile == "" && config.Gtp && config.Gfx {
		f, err = os.Create("/dev/null")
	} else if config.Lfile == "" {
		f = os.Stderr
	} else {
		f, err = os.Create(config.Lfile)
	}
	if err != nil {
		panic("could not create log file")
	}
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(f)
	
	if config.probLog, err = os.Create("probs.json"); err != nil {
		panic(err)
	}

	return config
}


func (config *Config) Load() {
	f, err := os.Open(config.cfile)
	if err != nil {
		panic(err)
	}
	defer func() { f.Close() }()
	d := json.NewDecoder(f)
	err = d.Decode(config)
	if err != nil {
		panic(err)
	}
}

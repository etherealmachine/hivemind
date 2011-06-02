package main

import (
	"flag"
	"json"
	"io/ioutil"
	"log"
)

var help *bool = flag.Bool("h", false, "Print this usage message")
var modeGTP *bool = flag.Bool("gtp", false, "Listen on stdin for GTP commands")

var maxPlayouts *uint = flag.Uint("p", 10000, "Max number of playouts")
var timelimit *uint = flag.Uint("t", 0, "Max number of seconds")

var file *string = flag.String("file", "", "Load data from file")
var configFile *string = flag.String("config", "", "Load config from file")

var stats *bool = flag.Bool("stats", false, "Print out tree search statistics")

var cgo *bool = flag.Bool("go", false, "Play Go")
var hex *bool = flag.Bool("hex", false, "Play Hex")

var test *bool = flag.Bool("test", false, "Just generate a single move")
var size *int = flag.Int("size", 9, "Boardsize")
var komi *float64 = flag.Float64("komi", 6.5, "Komi")
var testPPS *bool = flag.Bool("pps", false, "Gather data on the playouts per second")

var train *bool = flag.Bool("train", false, "Do crazy unsupervised training stuff")
var pswarm *bool = flag.Bool("pswarm", false, "Train with particle swarm")
var esswarm *bool = flag.Bool("esswarm", false, "Train with evolution strategies")
var generations *uint = flag.Uint("gens", 100, "Generations to train for")
var mu *uint = flag.Uint("mu", 30, "(Training) Children to keep")
var parents *uint = flag.Uint("parents", 2, "(Training) Parents per child")
var lambda *uint = flag.Uint("lambda", 50, "(Training) Children")
var samples *uint = flag.Uint("samples", 5, "(Training) Evaluations per generation")
var vself *bool = flag.Bool("vself", false, "Evaluate versus self")
var vuct *bool = flag.Bool("vuct", false, "Evaluate versus UCT")
var prefix *string = flag.String("prefix", "swarm", "Prefix to use when saving swarm")

var gfx *bool = flag.Bool("gfx", false, "Emit live graphics for gogui")

var pat *bool = flag.Bool("pat", false, "Use Pattern Matcher for 1-ply search")
var hand *bool = flag.Bool("hand", false, "Use hand-crafted patterns")
var randpat *bool = flag.Bool("randpat", false, "Use random patterns")
var tablepat *bool = flag.Bool("tablepat", false, "Use table patterns")
var tenuki *bool = flag.Bool("tenuki", false, "Use tenuki inside table patterns")
var nnpat *bool = flag.Bool("nnpat", false, "Use nn patterns")
var nullpat *bool = flag.Bool("nullpat", false, "Just remember seen patterns")
var logpat *bool = flag.Bool("logpat", false, "Log patterns")

var uct *bool = flag.Bool("uct", false, "Use UCT")
var c *float64 = flag.Float64("c", 0.5, "UCT coefficient")
var k *float64 = flag.Float64("k", 0, "AMAF equivalency cutoff")
var expandAfter *float64 = flag.Float64("e", 50, "Expand after")

var verbose *bool = flag.Bool("v", false, "Verbose logging")
var logFile *string = flag.String("log", "", "Log to filename")

var showSwarm *bool = flag.Bool("showswarm", false, "show info on swarm")
var testSwarm *bool = flag.Bool("testswarm", false, "run swarm tests")
var showPatterns *bool = flag.Bool("showpat", false, "show patterns")
var sgf *string = flag.String("sgf", "", "Load sgf file and generate move")
var compare *bool = flag.Bool("compare", false, "Compare pattern matchers")
var disable *bool = flag.Bool("disable", false, "Disable selected patterns")
var disabled []int // disabled pattern indices

type Config struct {
	ModeGTP bool
	MaxPlayouts uint
	Train, ShowSwarm, TestSwarm, ShowPatterns bool
	Size *int
	Mu, Parents, Lambda, Samples uint
	Gfx, Pat, Hand, Randpat, Tablepat, NNpat, Nullpat, Logpat, Test, TestPPS, Stats, Hex, TTT bool
	Uct bool
	File, Sgf, LogFile string
	Verbose bool
	C, K, ExpandAfter float64
	Disabled []float64
}

func config() {
	flag.Parse()
	if !(*cgo || *hex) { *cgo = true }
	if *cgo { *hex = false }
	if *hex { *cgo = false }
	if !(*pswarm || *esswarm) { *esswarm = true }
	if *pswarm { *esswarm = false }
	if *esswarm { *pswarm = false }
	if !(*vself || *vuct) { *vself = true }
	if *vself { *vuct = false }
	if *vuct { *vself = false }
	if *disable {
		buf, err := ioutil.ReadFile(*configFile)
		if err != nil { log.Println(err); return }
		var conf Config
		err = json.Unmarshal(buf, &conf)
		if err != nil { log.Println(err); return }
		log.Println("config file loaded")
		disabled = make([]int, len(conf.Disabled))
		for i := range conf.Disabled {
			disabled[i] = int(conf.Disabled[i])
		}
	}
}

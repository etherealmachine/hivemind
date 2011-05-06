package main

import (
	"os"
	"flag"
	"fmt"
	"runtime"
	logging "log"
)

var help *bool = flag.Bool("h", false, "Print this usage message")
var modeGTP *bool = flag.Bool("gtp", false, "Listen on stdin for GTP commands")
var server *string = flag.String("server", "", "Connect to redis server")
var maxPlayouts *uint = flag.Uint("p", 10000, "Max number of playouts")
var file *string = flag.String("file", "", "Load data from file")
var showSwarm *bool = flag.Bool("showswarm", false, "show info on swarm")
var testSwarm *bool = flag.Bool("testswarm", false, "run swarm tests")
var showPatterns *bool = flag.Bool("showpat", false, "show patterns")
var sgf *string = flag.String("sgf", "", "Load sgf file and generate move")
var stats *bool = flag.Bool("stats", false, "Print out tree search statistics")
var hex *bool = flag.Bool("hex", false, "Play Hex (default=go)")
var ttt *bool = flag.Bool("ttt", false, "Play Tic-Tac-Toe (default=go)")
var book *bool = flag.Bool("book", false, "Use stored positions table")
var test *bool = flag.Bool("test", false, "Just generate a single move")
var size *int = flag.Int("size", 9, "Boardsize")
var testPPS *bool = flag.Bool("pps", false, "Gather data on the playouts per second")
var train *bool = flag.Bool("train", false, "Do crazy neural network training stuff")
var mu *uint = flag.Uint("mu", 30, "(Training) Children to keep")
var parents *uint = flag.Uint("parents", 2, "(Training) Parents per child")
var lambda *uint = flag.Uint("lambda", 50, "(Training) Children")
var samples *uint = flag.Uint("samples", 5, "(Training) Evaluations per generation")
var gfx *bool = flag.Bool("gfx", false, "Emit live graphics for gogui")
var pat *bool = flag.Bool("pat", false, "Use Pattern Matcher for 1-ply search")
var hand *bool = flag.Bool("hand", false, "Use hand-crafted patterns")
var randpat *bool = flag.Bool("randpat", false, "Use random patterns")
var tablepat *bool = flag.Bool("tablepat", false, "Use table patterns")
var nullpat *bool = flag.Bool("nullpat", false, "Just remember seen patterns")
var logpat *bool = flag.Bool("logpat", false, "Log patterns")
var uct *bool = flag.Bool("uct", false, "Use UCT")
var logFile *string = flag.String("log", "", "Log to filename")
var verbose *bool = flag.Bool("v", false, "Verbose logging")
var c *float64 = flag.Float64("c", 0.5, "UCT coefficient")
var k *float64 = flag.Float64("k", 0, "AMAF equivalency cutoff")
var expandAfter *float64 = flag.Float64("e", 50, "Expand after")
var threads *uint = flag.Uint("t", uint(runtime.GOMAXPROCS(0)-1), "threads")

var matcher PatternMatcher
var log *logging.Logger

func main() {
	flag.Parse()
	var f *os.File
	var err os.Error
	if *logFile == "" && *modeGTP && *gfx {
		f, err = os.Open("/dev/null", os.O_RDWR|os.O_TRUNC|os.O_CREAT, 0666)
	} else if *logFile == "" {
		f = os.Stderr
	} else {
		f, err = os.Open(*logFile, os.O_RDWR|os.O_TRUNC|os.O_CREAT, 0666)
	}
	if err != nil {
		panic("could not create log file")
	}
	log = logging.New(f, "", 0)
	if *hand && *file != "" {
		matcher = LoadHandPatternMatcher(*file)
		log.Println("loaded hand crafted pattern matcher")
	} else if *tablepat && *file != "" {
		matcher = LoadTablePatternMatcher(*file)
		log.Println("loaded table pattern matcher")
	} else if *file != "" {
		matcher = LoadNNPatternMatcher(*file)
		log.Println("loaded neural network pattern matcher")
	} else if *randpat {
		matcher = &RandomMatcher{}
		log.Println("loaded random pattern matcher")
	} else if *nullpat {
		matcher = &NullMatcher{}
		log.Println("loaded null pattern matcher")
	}
	//SetupCluster()
	if *help {
		flag.Usage()
		os.Exit(0)
	} else if *modeGTP {
		GTP()
	} else if *sgf != "" {
		t, color := Load(*sgf)
		root := NewRoot(color, t)
		genmove(root, t, matcher)
		vertex := root.Best().vertex
		t.Play(color, vertex)
		fmt.Println(Ctoa(color), Vtoa(vertex, t.Boardsize()))
		fmt.Println(Bwboard(t.Board(), t.Boardsize(), true))
	} else if *train {
		Train()
	} else if *test {
		t := NewTracker(*size)
		root := NewRoot(BLACK, t)
		genmove(root, t, matcher)
		fmt.Println(Vtoa(root.Best().vertex, t.Boardsize()))
	} else if *testPPS {
		TestPPS()
	} else if *file != "" && *showSwarm {
		ShowSwarm(*file)
	}else if *file != "" && *testSwarm {
		TestSwarm(*file)
	} else if matcher != nil && *showPatterns {
		ShowPatterns()
	}
}

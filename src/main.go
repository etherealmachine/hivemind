package main

import (
	"os"
	"flag"
	"fmt"
	"log"
	"rand"
	"time"
)

var matcher PatternMatcher
var evaluator BoardEvaluator

func main() {
	rand.Seed(time.Nanoseconds())
	config := NewConfig()
	var f *os.File
	var err os.Error
	if config.logFile == "" && config.modeGTP && config.gfx {
		f, err = os.Create("/dev/null")
	} else if config.logFile == "" {
		f = os.Stderr
	} else {
		f, err = os.Create(config.logFile)
	}
	if err != nil {
		panic("could not create log file")
	}
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(f)
	LoadPatternMatcher(config)
	LoadBoardEvaluator(config)
	if config.help {
		flag.Usage()
		os.Exit(0)
	} else if config.modeGTP {
		GTP(config)
	} else if config.sgf != "" {
		t, color := Load(config.sgf)
		root := NewRoot(color, t, config)
		genmove(root, t)
		vertex := root.Best().vertex
		t.Play(color, vertex)
		fmt.Println(Ctoa(color), t.Vtoa(vertex))
		fmt.Println(t.String())
	} else if config.train {
		Train(config)
	} else if config.test {
		t := NewTracker(config)
		root := NewRoot(BLACK, t, config)
		genmove(root, t)
	} else if config.testPPS {
		TestPPS(config)
	}
}

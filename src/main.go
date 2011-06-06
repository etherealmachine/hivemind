package main

import (
	"os"
	"flag"
	"fmt"
	"log"
)

var matcher PatternMatcher
var evaluator BoardEvaluator

func main() {
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
	matcher = LoadPatternMatcher(config)
	evaluator = LoadBoardEvaluator(config)
	if config.help {
		flag.Usage()
		os.Exit(0)
	} else if config.modeGTP {
		GTP(config)
	} else if config.sgf != "" {
		t, color := Load(config.sgf)
		root := NewRoot(color, t, config)
		genmove(root, t, matcher, evaluator)
		vertex := root.Best().vertex
		t.Play(color, vertex)
		fmt.Println(Ctoa(color), t.Vtoa(vertex))
		fmt.Println(t.String())
	} else if config.train {
		Train(config)
	} else if config.test {
		t := NewTracker(config)
		root := NewRoot(BLACK, t, config)
		genmove(root, t, matcher, evaluator)
		fmt.Println(t.Vtoa(root.Best().vertex))
	} else if config.testPPS {
		log.Println("Go:")
		config.cgo = true
		config.hex = false
		config.size = 9
		TestPPS(config)
		log.Println("Hex:")
		config.cgo = false
		config.hex = true
		config.size = 11
		TestPPS(config)
	} else if config.makeBook {
		MakeBook(config)
	}
}

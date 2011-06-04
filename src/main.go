package main

import (
	"os"
	"flag"
	"fmt"
	"log"
)

var matcher PatternMatcher

func main() {
	config()
	var f *os.File
	var err os.Error
	if *logFile == "" && *modeGTP && *gfx {
		f, err = os.Create("/dev/null")
	} else if *logFile == "" {
		f = os.Stderr
	} else {
		f, err = os.Create(*logFile)
	}
	if err != nil {
		panic("could not create log file")
	}
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(f)
	matcher := LoadPatternMatcher(*file, nil)
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
		log.Println("Go:")
		TestPPS(NewGoTracker(9))
		log.Println("Hex:")
		TestPPS(NewHexTracker(11))
	} else if *showSwarm {
		ShowSwarm(*file)
	} else if *compare {
		p1 := LoadPatternMatcher(*file, nil)
		p2 := LoadPatternMatcher(*file, disabled)
		Compare(p1, p2, "default", "disabled")
	} else if *makeBook {
		MakeBook(BLACK, NewTracker(*size))
	}
}

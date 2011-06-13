package main

import (
	"os"
	"flag"
	"fmt"
	"log"
	"rand"
	"time"
)

func main() {
	rand.Seed(time.Nanoseconds())
	config := NewConfig()
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
	if config.Help {
		flag.Usage()
		os.Exit(0)
	} else if config.Gtp {
		GTP(config)
	} else if config.SGF != "" {
		t, color := Load(config.SGF)
		root := NewRoot(color, t, config)
		genmove(root, t)
		vertex := root.Best().Vertex
		t.Play(color, vertex)
		fmt.Println(Ctoa(color), t.Vtoa(vertex))
		fmt.Println(t.String())
	} else if config.Train {
		Train(config)
	} else if config.Test {
		t := NewTracker(config)
		root := NewRoot(BLACK, t, config)
		genmove(root, t)
	} else if config.Speed {
		SpeedTest(config)
	} else if config.Book {
		t := NewTracker(config)
		genmove(config.book, t)
		config.book.SaveBook()
		if config.Verbose {
			log.Println(config.book.String(0, 2, t))
		}
	}
}

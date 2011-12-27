package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"rand"
	"time"
)

func main() {
	rand.Seed(time.Nanoseconds())
	config := NewConfig()

	shutdown := make(chan bool, 1)
	if config.Cluster {
		InitCluster(shutdown)
	} else {
		shutdown <- true
	}

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
	} else if config.Book {
		t := NewTracker(config)
		genmove(config.book, t)
		config.book.SaveBook()
		if config.Verbose {
			log.Println(config.book.String(0, 2, t))
		}
	} else if config.PrintWeights {
		PrintBestWeights(config)
		shutdown <- true
	} else if config.Genmove {
		t := NewTracker(config)
		root := NewRoot(BLACK, t, config)
		genmove(root, t)
	} else if config.PlayGame {
		for i := uint(0); i < config.Samples; i++ {
			t := NewTracker(config)
			color := BLACK
			move := 0
			for {
				var vertex int
				if move == 0 && config.Swapsafe {
					vertex = (3 * t.Boardsize()) + 2
				} else {
					root := NewRoot(color, t, config)
					genmove(root, t)
					vertex = root.Best().Vertex
				}
				t.Play(color, vertex)
				log.Println(t.String())
				if t.Winner() != EMPTY {
					break
				}
				move++
				color = Reverse(color)
			}
		}
	}
	<-shutdown
}

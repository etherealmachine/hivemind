package main

import (
	"github.com/hoisie/redis.go"
	"log"
	"json"
)

var shutdown chan bool

func init() {
	log.Println("setting up cluster...")
	shutdown = make(chan bool, 1)
	subscriptions := make(chan string, 1)
	subscriptions <- "commands"
	commands := make(chan redis.Message)
	var client redis.Client
	client.Addr = "127.0.0.1:6379"
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("Error trying to subscribe to redis channel")
				log.Println(err)
				shutdown <- true
			}
		}()
		client.Subscribe(subscriptions, nil, nil, nil, commands)
	}()
	evals := make(chan Config)
	go listen(commands, evals)
	go process(evals)
}

func listen(commands chan redis.Message, evals chan Config) {
	log.Println("cluster up, listening for commands...")
	for {
		command := <-commands
		if string(command.Message) == "shutdown" {
			shutdown <- true
			return
		} else {
			var config Config
			if err := json.Unmarshal(command.Message, &config); err != nil {
				log.Println(err)
			}
			evals <- config
		}
	}
}

func process(evals chan Config) {
	sem := make(chan int, 4)
	for config := range evals {
		go eval(config, sem)
	}
}

func eval(config Config, sem chan int) {
	sem <- 1
	t := NewTracker(&config)
	color := BLACK
	for i := range config.Moves {
		t.Play(color, config.Moves[i])
		color = Reverse(color)
	}
	root := NewRoot(color, t, &config)
	genmove(root, t)
	eval := &Eval{config.Moves, root.Best().Vertex, root.Best().Wins / root.Best().Visits}
	bytes, err := json.Marshal(eval)
	if err != nil {
		log.Println(err)
	} else {
		var client redis.Client
		client.Addr = "127.0.0.1:6379"
		err := client.Publish("evals", bytes)
		if err != nil {
			log.Println(err)
		}
	}
	<-sem
}

type Eval struct {
	Moves []int
	Next int
	Mean float64
}

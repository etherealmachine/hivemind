package main

import (
	"github.com/etherealmachine/redis.go"
	"log"
	"json"
)

func InitCluster(shutdown chan bool) {
	log.Println("setting up cluster...")
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
	configs := make(chan Config)
	go listen(commands, configs, shutdown)
	go process(configs)
}

func listen(commands chan redis.Message, configs chan Config, shutdown chan bool) {
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
			configs <- config
		}
	}
}

func process(configs chan Config) {
	sem := make(chan int, 4)
	for config := range configs {
		switch config.MsgType {
			case "eval":
				go eval(config, sem)
			case "play":
				go play(config, sem)
		}
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

func play(config Config, sem chan int) byte {
	sem <- 1
	t := NewTracker(&config)
	var vertex int
	for move := 0;; {
		config.policy_weights = config.Black_policy_weights
		br := NewRoot(BLACK, t, &config)
		genmove(br, t)
		vertex = br.Best().Vertex
		t.Play(BLACK, vertex)
		move++
		if t.Winner() != EMPTY || move >= 2 * t.Sqsize() {
			break
		}
		config.policy_weights = config.White_policy_weights
		wr := NewRoot(WHITE, t, &config)
		genmove(wr, t)
		vertex = wr.Best().Vertex
		t.Play(WHITE, vertex)
		move++
		if t.Winner() != EMPTY || move >= 2 * t.Sqsize() {
			break
		}
	}
	if config.VeryVerbose {
		log.Println(Ctoa(t.Winner()))
		log.Println(t.String())
	}
	<-sem
	return t.Winner()
}

type Eval struct {
	Moves []int
	Next int
	Mean float64
}

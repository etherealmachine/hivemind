package main

import (
	"testing"
	"fmt"
	"log"
	"os"
)

var config *Config

func init() {
	setupConfig()
}

func setupConfig() {
	if config == nil {
		config = NewConfig()
		config.Stats = true
		f, err := os.Create("test.log")
		if err != nil {
			panic(err)
		}
		log.SetFlags(0)
		log.SetPrefix("")
		log.SetOutput(f)
	}
}

func TestGoGenmove(t *testing.T) {
	log.Println("Go Genmove")
	config.Go = true
	config.Hex = false
	config.MaxPlayouts = 10000
	tracker := NewTracker(config)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

func TestHexGenmove(t *testing.T) {
	log.Println("Hex Genmove")
	config.Go = false
	config.Hex = true
	config.MaxPlayouts = 10000
	tracker := NewTracker(config)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

func TestGoSwarm(t *testing.T) {
	log.Println("Go Swarm")
	config.Go = true
	config.Hex = false
	config.MaxPlayouts = 10000
        config.policy_weights = LoadBest("./swarm.1.gob", config)
	tracker := NewTracker(config)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

func TestHexSwarm(t *testing.T) {
	log.Println("Hex Swarm")
	config.Go = false
	config.Hex = true
	config.MaxPlayouts = 10000
        config.policy_weights = LoadBest("./swarm.1.gob", config)
	tracker := NewTracker(config)
	fmt.Println(tracker)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

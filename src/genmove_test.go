package main

import "testing"
import "fmt"

var config *Config

func setupConfig() {
	if config == nil {
		config = NewConfig()
		config.Stats = true
	}
}

func TestGoGenmove(t *testing.T) {
	fmt.Println("Go Genmove")
	setupConfig()
	config.Go = true
	config.MaxPlayouts = 10000
	tracker := NewTracker(config)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

func TestHexGenmove(t *testing.T) {
	fmt.Println("Hex Genmove")
	setupConfig()
	config.Hex = true
	config.MaxPlayouts = 10000
	tracker := NewTracker(config)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

func TestGoSwarm(t *testing.T) {
	fmt.Println("Go Swarm")
	setupConfig()
	config.Go = true
	config.MaxPlayouts = 10000
        config.policy_weights = LoadBest("./swarm.1.gob", config)
	tracker := NewTracker(config)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

func TestHexSwarm(t *testing.T) {
	fmt.Println("Hex Swarm")
	setupConfig()
	config.Hex = true
	config.MaxPlayouts = 10000
        config.policy_weights = LoadBest("./swarm.1.gob", config)
	tracker := NewTracker(config)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

func BenchmarkGo(b *testing.B) {
	setupConfig()
	config.Go = true
	config.MaxPlayouts = uint(b.N)
	runPlayouts(config)
}

func BenchmarkHex(b *testing.B) {
	setupConfig()
	config.Hex = true
	config.MaxPlayouts = uint(b.N)
	runPlayouts(config)
}

func runPlayouts(config *Config) {
	t := NewTracker(config)
	for i := uint(0); i < config.MaxPlayouts; i++ {
		cp := t.Copy()
		cp.Playout(BLACK)
	}
}

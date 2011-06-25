package main

import "testing"

var config *Config

func setupConfig() {
	if config == nil {
		config = NewConfig()
	}
}

func TestGoGenmove(t *testing.T) {
	setupConfig()
	config.Go = true
	config.MaxPlayouts = 1000
	tracker := NewTracker(config)
	root := NewRoot(BLACK, tracker, config)
	genmove(root, tracker)
}

func TestHexGenmove(t *testing.T) {
	setupConfig()
	config.Hex = true
	config.MaxPlayouts = 1000
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

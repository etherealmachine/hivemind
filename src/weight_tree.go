package main

import (
	"math"
	"rand"
)

type WeightTree struct {
	size int
	nodes int
	interior int
	leaves int
	weights []int
}

func NewWeightTree(size int) *WeightTree {
	t := new(WeightTree)
	t.size = size
	n := uint(math.Ceil(math.Log2(float64(size))))
	t.leaves = 1 << n
	t.nodes = 0
	for i := uint(0); i <= n; i++ {
		t.nodes += 1 << i
	}
	t.interior = t.nodes - t.leaves
	t.weights = make([]int, 2 * t.nodes)
	return t
}

func (t *WeightTree) Set(color byte, vertex int, weight int) {
	if color == BLACK {
		t.update(t.interior + vertex, 0, weight, t.weights[t.interior + vertex])
	} else {
		t.update(t.interior + vertex, t.nodes, weight, t.weights[t.nodes + t.interior + vertex])
	}
}

func (t *WeightTree) Get(color byte, vertex int) int {
	if color == BLACK {
		return t.weights[t.interior + vertex]
	}
	return t.weights[t.nodes + t.interior + vertex]
}

func (t *WeightTree) Prob(color byte, vertex int) float64 {
	if color == BLACK {
		return float64(t.Get(color, vertex)) / float64(t.weights[0])
	}
	return float64(t.Get(color, vertex)) / float64(t.weights[t.nodes])
}

func (t *WeightTree) Rand(color byte) int {
	if color == BLACK {
		if t.weights[0] == 0 {
			return -1
		}
		return t.rand(0, 0)
	}
	if t.weights[t.nodes] == 0 {
		return -1
	}
	return t.rand(0, t.nodes)
}

func (t *WeightTree) Copy() *WeightTree {
	cp := new(WeightTree)
	cp.nodes = t.nodes
	cp.interior = t.interior
	cp.leaves = t.leaves
	cp.weights = make([]int, len(t.weights))
	copy(cp.weights, t.weights)
	return cp
}

func (t *WeightTree) rand(node int, root int) int {
	if node >= t.interior {
		return node - t.interior
	}
	left := 2 * node + 1
	right := 2 * node + 2
	if rand.Intn(t.weights[root + node]) < t.weights[root + left] {
		return t.rand(left, root)
	}
	return t.rand(right, root)
}

func (t *WeightTree) update(node int, root int, weight int, old_weight int) {
	t.weights[root + node] -= old_weight
	t.weights[root + node] += weight
	if node == 0 {
		return
	}
	parent := int(math.Floor((float64(node) - 1) / 2))
	t.update(parent, root, weight, old_weight)
}

func (t *WeightTree) Weights(color byte) []float64 {
	weights := make([]float64, t.size)
	max := 0.0
	for i := range weights {
		weights[i] = float64(t.Get(color, i))
		if weights[i] > max {
			max = weights[i]
		}
	}
	for i := range weights {
		weights[i] /= max
	}
	return weights
}

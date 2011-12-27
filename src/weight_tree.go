package main

import (
	"math"
	"rand"
)

type WeightTree struct {
	size          int
	nodes         int
	interior      int
	leaves        int
	black_weights []float64
	white_weights []float64
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
	t.black_weights = make([]float64, t.nodes)
	t.white_weights = make([]float64, t.nodes)
	return t
}

func (t *WeightTree) Set(color byte, vertex int, weight float64) {
	if color == BLACK {
		t.black_weights[t.interior+vertex] = weight
		for parent := t.parent(t.interior + vertex); parent >= 0; parent = t.parent(parent) {
			t.black_weights[parent] = t.black_weights[2*parent+1] + t.black_weights[2*parent+2]
		}
	} else {
		t.white_weights[t.interior+vertex] = weight
		for parent := t.parent(t.interior + vertex); parent >= 0; parent = t.parent(parent) {
			t.white_weights[parent] = t.white_weights[2*parent+1] + t.white_weights[2*parent+2]
		}
	}
}

func (t *WeightTree) Get(color byte, vertex int) float64 {
	if color == BLACK {
		return t.black_weights[t.interior+vertex]
	}
	return t.white_weights[t.interior+vertex]
}

func (t *WeightTree) Prob(color byte, vertex int) float64 {
	if color == BLACK {
		return t.Get(color, vertex) / t.black_weights[0]
	}
	return t.Get(color, vertex) / t.white_weights[0]
}

func (t *WeightTree) Rand(color byte) int {
	var r int
	if color == BLACK {
		if t.black_weights[0] == 0 {
			r = -1
		} else {
			r = t.rand(BLACK, 0, 0)
		}
	} else {
		if t.white_weights[0] == 0 {
			r = -1
		} else {
			r = t.rand(WHITE, 0, 0)
		}
	}
	if r >= t.size {
		panic("random vertex out-of-bounds")
	}
	return r
}

func (t *WeightTree) Copy() *WeightTree {
	cp := new(WeightTree)
	cp.size = t.size
	cp.nodes = t.nodes
	cp.interior = t.interior
	cp.leaves = t.leaves
	cp.black_weights = make([]float64, len(t.black_weights))
	copy(cp.black_weights, t.black_weights)
	cp.white_weights = make([]float64, len(t.white_weights))
	copy(cp.white_weights, t.white_weights)
	return cp
}

func (t *WeightTree) rand(color byte, node int, root int) int {
	if node >= t.interior {
		return node - t.interior
	}
	left := 2*node + 1
	right := 2*node + 2
	var weights []float64
	if color == BLACK {
		weights = t.black_weights
	} else {
		weights = t.white_weights
	}
	if rand.Float64()*weights[root+node] < weights[root+left] {
		return t.rand(color, left, root)
	}
	return t.rand(color, right, root)
}

func (t *WeightTree) parent(node int) int {
	return int(math.Floor((float64(node) - 1) / 2))
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

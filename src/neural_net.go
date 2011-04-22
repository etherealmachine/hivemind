package main

import "math"

type NeuralNet struct {
	Arch []int
	Config []float64
}

func NewNeuralNet(arch []int) *NeuralNet {
	net := new(NeuralNet)
	net.Arch = make([]int, len(arch))
	copy(net.Arch, arch)
	dim := 0
	for i := 1; i < len(net.Arch); i++ {
		dim += net.Arch[i-1] * net.Arch[i] + net.Arch[i]
	}
	net.Config = make([]float64, dim)
	return net
}

func (net *NeuralNet) E(input []float64) []float64 {
	in := make([]float64, len(input)+1)
	copy(in, input)
	in[len(in)-1] = 1
	for layer := 1; layer < len(net.Arch); layer++ {
		out := make([]float64, net.Arch[layer])
		for node := 0; node < net.Arch[layer]; node++ {
			out[node] = dot(in, net.weights(layer, node)) + net.bias(layer, node)
			out[node] = transfer(out[node])
		}
		in = out
	}
	return in
}

func (net *NeuralNet) weights(layer int, node int) []float64 {
	start := 0
	for i := 1; i < layer; i++ {
		start += net.Arch[i-1] * net.Arch[i] + net.Arch[i]
	}
	start += net.Arch[layer - 1] * node
	end := start + net.Arch[layer - 1]
	return net.Config[start:end]
}

func (net *NeuralNet) bias(layer int, node int) float64 {
	curr := 0
	for i := 1; i <= layer; i++ {
		curr += net.Arch[i-1] * net.Arch[i] + net.Arch[i]
	}
	return net.Config[curr + node - net.Arch[layer]]
}

func dot(v1 []float64, v2 []float64) (sum float64) {
	for i := 0; i < len(v1) && i < len(v2); i++ {
		sum += v1[i]*v2[i]
	}
	return sum
}

func transfer(out float64) float64 {
	o := math.Tanh(float64(out))
	if !math.IsNaN(o){
		return float64(o)
	}
	if out < -1 { return -1 }
	if out > 1 { return 1 }
	return 0
}

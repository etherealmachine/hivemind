package main

import (
	"math"
	"bufio"
	"os"
	"container/vector"
	"strings"
	"rand"
	"github.com/ajstarks/svgo"
	"fmt"
	"log"
)

var inputsize int

var queries int
var matches int

var patternLog []int

type PatternMatcher interface {
	Match(color byte, vertex int, t Tracker) int
}

type HandCraftedMatcher struct {
	patterns [][]byte
}

type RandomMatcher struct{}

type NullMatcher struct{}

type ColorDuplexingMatcher struct {
	black PatternMatcher
	white PatternMatcher
}

func detectCut(i int, j int, c byte, board []byte, s2 int) bool {
	return i >= 0 && i < s2 && j >= 0 && j < s2 && board[i] == c && board[j] == c
}

func (m *ColorDuplexingMatcher) Match(color byte, v int, t Tracker) int {
	if color == BLACK {
		return m.black.Match(color, v, t)
	} else if color == WHITE {
		return m.white.Match(color, v, t)
	}
	panic("can't duplex onto empty")
}

func (m *NullMatcher) Match(color byte, v int, t Tracker) int {
	queries++
	index := compute_index(t.Board(), hexSliceMap[t.Boardsize()][v])
	if *logpat {
		patternLog[index]++
	}
	return -1
}

func (m *Particle) Match(color byte, v int, t Tracker) int {
	queries++
	s := t.Boardsize()
	b := t.Board()
	var adj []int
	if *hex {
		adj = hexSliceMap[s][v]
	} else {
		adj = goSliceMap[s][v]
	}
	// todo: patternLog?
	pat := m.Get(b, adj)
	for i := 0; i < len(adj); i++ {
		if adj[i] == -1 || b[adj[i]] != EMPTY || !t.Legal(color, adj[i]) {
			pat[i] = 0
		}
	}
	sum := 0.0
	for i := range pat {
		sum += pat[i]
	}
	if sum == 0 {
		return -1
	}
	r := rand.Float64()
	for i := range pat {
		r -= pat[i] / sum
		if r <= 0 {
			matches++
			return adj[i]
		}
	}
	log.Println(Vtoa(v, s))
	log.Println(Bwboard(b, s, true))
	log.Println(pat)
	panic("pattern error, not a valid probability distribution")
}

func (m *HandCraftedMatcher) Match(color byte, v int, t Tracker) int {
	s := t.Boardsize()
	board := t.Board()
	var sliceMap [][]int
	if *hex {
		inputsize = 7
		sliceMap = hexSliceMap[s]
	} else {
		inputsize = 9
		sliceMap = goSliceMap[s]
	}
	queries++

	input := make([]byte, inputsize)

	for i := 0; i < inputsize; i++ {
		j := sliceMap[v][i]
		if j == -1 {
			return -1
		}
		input[i] = board[j]
	}

	for p := 0; p < len(m.patterns); p++ {
		pat := m.patterns[p]
		match := true
		for i := 0; i < inputsize; i++ {
			match = match && (pat[i] == input[i] || pat[i] == BOTH)
		}
		if match {
			matches++
			return sliceMap[v][pat[inputsize]]
		}
	}

	return -1
}

func (net *NeuralNet) Match(color byte, v int, t Tracker) int {
	s := t.Boardsize()
	board := t.Board()
	var sliceMap [][]int
	if *hex {
		inputsize = 7
		sliceMap = hexSliceMap[s]
	} else {
		inputsize = 9
		sliceMap = goSliceMap[s]
	}
	queries++

	input := make([]float64, inputsize)

	opp := Reverse(color)

	for i := 0; i < inputsize; i++ {
		j := sliceMap[v][i]
		if j == -1 {
			return -1
		}
		if board[j] == EMPTY {
			input[i] = 0
		} else if board[j] == color {
			input[i] = 1
		} else if board[j] == opp {
			input[i] = -1
		}
	}

	max := float64(math.Inf(-1))
	suggestion := -1
	for i := 0; i < inputsize; i++ {
		if t.Legal(color, sliceMap[v][i]) {
			input[i] = 1
			output := net.E(input)
			if output[0] > max {
				max = output[0]
				suggestion = i
			}
			input[i] = 0
		}
	}

	output := net.E(input)

	if max >= output[0] {
		matches++
		return sliceMap[v][suggestion]
	}
	return -1
}

func (m *RandomMatcher) Match(color byte, v int, t Tracker) int {
	queries++

	b := t.Board()
	s := t.Boardsize()
	adj := hexSliceMap[s][v]

	index := compute_index(b, adj)
	if *logpat {
		patternLog[index]++
	}
	pat := make([]float64, 7)
	for i := 0; i < 7; i++ {
		pat[i] = 1
		if adj[i] == -1 || b[adj[i]] != EMPTY {
			pat[i] = 0
		}
	}
	sum := 0.0
	for i := range pat {
		sum += pat[i]
	}
	if sum == 0 {
		return -1
	}
	r := rand.Float64()
	for i := range pat {
		r -= pat[i] / sum
		if r <= 0 {
			matches++
			return adj[i]
		}
	}
	log.Println(Vtoa(v, s))
	log.Println(Bwboard(b, s, true))
	log.Println(pat)
	panic("pattern error, not a valid probability distribution")
}

func LoadNNPatternMatcher(filename string) PatternMatcher {
	particle := LoadBest(filename)
	if particle == nil {
		return nil
	}
	net := new(NeuralNet)
	//net.Config = particle.Position
	panic("not supported")
	return net
}

func LoadTablePatternMatcher(filename string, disable bool) PatternMatcher {
	particle := LoadBest(filename)
	if disable {
		/*
		for i := range disabled {
			for j := 0; j < 7; j++ {
				particle.Position[i*7+j] = 0
			}
			log.Printf("disabled pattern %d\n", i)
		}
		*/
		panic("not supported")
	}
	return particle
}

func LoadHandPatternMatcher(filename string) PatternMatcher {
	f, err := os.Open(filename)
	if err != nil {
		log.Println("failed to load", filename)
		return nil
	}
	r := bufio.NewReader(f)
	matcher := new(HandCraftedMatcher)
	patterns := new(vector.Vector)
	if *hex {
		inputsize = 7
	} else {
		inputsize = 9
	}
	pattern := make([]byte, inputsize+1)
	patterns.Push(pattern)
	i := 0
	for s, err := r.ReadString('\n'); err == nil; s, err = r.ReadString('\n') {
		v := strings.Fields(s)
		for j := 0; j < len(v); j++ {
			switch v[j] {
			case "b":
				pattern[i] = BLACK
			case "w":
				pattern[i] = WHITE
			case "e":
				pattern[i] = EMPTY
			case "*":
				pattern[i] = BOTH
			case "s":
				pattern[i] = EMPTY
				pattern[len(pattern)-1] = byte(i)
			}
			i++
			if i == inputsize {
				pattern = make([]byte, inputsize+1)
				patterns.Push(pattern)
				i = 0
			}
		}
	}
	matcher.patterns = make([][]byte, patterns.Len())
	for i := 0; i < patterns.Len(); i++ {
		matcher.patterns[i] = patterns.At(i).([]byte)
	}
	return matcher
}

var goSliceMap [][][]int
var hexSliceMap [][][]int

func init() {
	goSliceMap = make([][][]int, 20)
	hexSliceMap = make([][][]int, 20)
	for i := 3; i <= 19; i++ {
		setupGoSliceMap(i)
		setupHexSliceMap(i)
	}
	patternLog = make([]int, 16384)
}

func setupGoSliceMap(size int) {
	inputsize = 9
	sliceMap := make([][]int, size*size)
	for vertex := 0; vertex < size*size; vertex++ {
		sliceMap[vertex] = make([]int, inputsize)
		sliceMap[vertex][0] = vertex - size - 1
		sliceMap[vertex][1] = vertex - size
		sliceMap[vertex][2] = vertex - size + 1
		sliceMap[vertex][3] = vertex - 1
		sliceMap[vertex][4] = vertex
		sliceMap[vertex][5] = vertex + 1
		sliceMap[vertex][6] = vertex + size - 1
		sliceMap[vertex][7] = vertex + size
		sliceMap[vertex][8] = vertex + size + 1
		if vertex%size == 0 {
			// left
			sliceMap[vertex][0] = -1
			sliceMap[vertex][3] = -1
			sliceMap[vertex][6] = -1
		}
		if (vertex+1)%size == 0 {
			// right
			sliceMap[vertex][2] = -1
			sliceMap[vertex][5] = -1
			sliceMap[vertex][8] = -1
		}
		if vertex < size {
			// top
			sliceMap[vertex][0] = -1
			sliceMap[vertex][1] = -1
			sliceMap[vertex][2] = -1
		}
		if vertex >= (size*size)-size {
			// bottom
			sliceMap[vertex][6] = -1
			sliceMap[vertex][7] = -1
			sliceMap[vertex][8] = -1
		}
	}
	goSliceMap[size] = sliceMap
}

func setupHexSliceMap(size int) {
	inputsize = 7
	sliceMap := make([][]int, size*size)
	for vertex := 0; vertex < size*size; vertex++ {
		sliceMap[vertex] = make([]int, inputsize)
		sliceMap[vertex][0] = vertex - size
		sliceMap[vertex][1] = vertex - size + 1
		sliceMap[vertex][2] = vertex + 1
		sliceMap[vertex][3] = vertex + size
		sliceMap[vertex][4] = vertex + size - 1
		sliceMap[vertex][5] = vertex - 1
		sliceMap[vertex][6] = vertex
		if vertex%size == 0 {
			// left
			sliceMap[vertex][4] = -1
			sliceMap[vertex][5] = -1
		}
		if (vertex+1)%size == 0 {
			// right
			sliceMap[vertex][1] = -1
			sliceMap[vertex][2] = -1
		}
		if vertex < size {
			// top
			sliceMap[vertex][0] = -1
			sliceMap[vertex][1] = -1
		}
		if vertex >= (size*size)-size {
			// bottom
			sliceMap[vertex][3] = -1
			sliceMap[vertex][4] = -1
		}
	}
	hexSliceMap[size] = sliceMap
}

func drawHex(xoff, yoff, width float64, pos int, style1, style2 string, s *svg.SVG) {
	if style1 == "" {
		return
	}
	C := width
	A := 0.5 * C
	B := math.Sin(1.04719755) * C
	switch pos {
	case 0:
	case 1:
		xoff += 1.5 * C
		yoff -= B
	case 2:
		xoff += 3 * C
	case 3:
		xoff += 3 * C
		yoff += 2 * B
	case 4:
		xoff += 1.5 * C
		yoff += 3 * B
	case 5:
		yoff += 2 * B
	case 6:
		xoff += 1.5 * C
		yoff += B
	}
	x := []int{int(xoff), int(A + xoff), int(A + C + xoff), int(2*C + xoff), int(A + C + xoff), int(A + xoff)}
	y := []int{int(B + yoff), int(yoff), int(yoff), int(B + yoff), int(2*B + yoff), int(2*B + yoff)}
	s.Polygon(x, y, style1)
	if style2 != "" {
		s.Circle(int(xoff+C), int(yoff+B), int(width/2), style2)
	}
}

func drawPat(bits string, xoff, yoff, width float64, pat []uint8, weights []float64, index int, s *svg.SVG) {
	sum := 0.0
	for i := range pat {
		sum += weights[i]
	}
	for i := range pat {
		w := weights[i]
		color := "white"
		if w > 0.1 {
			c := 255 - int(255*(w/sum))
			color = fmt.Sprintf("#%.2X%.2X%.2X", c, c, c)
		}
		var style1, style2 string
		switch pat[i] {
		case EMPTY:
			style1 = "fill:" + color + ";stroke:#464646;stroke-width:2"
			style2 = ""
		case BLACK:
			style1 = "fill:" + color + ";stroke:#464646;stroke-width:2"
			style2 = "fill:black;stroke:white;stroke-width:2"
		case WHITE:
			style1 = "fill:" + color + ";stroke:#464646;stroke-width:2"
			style2 = "fill:white;stroke:black;stroke-width:2"
		case 3:
			style1 = ""
			style2 = ""
		}
		drawHex(xoff, yoff, width, i, style1, style2, s)
	}
	s.Text(int(xoff+5*width+10), int(yoff), bits)
	ws := "["
	for i := 0; i < len(weights); i++ {
		ws += fmt.Sprintf("%.2f", weights[i])
		if i == len(weights)-1 {
			ws += "]"
		} else {
			ws += " "
		}
	}
	s.Text(int(xoff+5*width+10), int(yoff+20), ws)
	s.Text(int(xoff+5*width+10), int(yoff+40), fmt.Sprintf("%d", index))
}

func ShowPatterns() {
	if *hex {
		panic("not supported")
		/*
		f, err := os.Create("pats.svg")
		if err != nil {
			panic(err)
		}
		defer func() { f.Close() }()
		s := svg.New(f)

		pats := []int{
			16033,
		}
		s.Start(1000, 40*6*len(pats)+40)
		p := matcher.(*Particle)
		y := 0
		for i := range pats {
			pat := make([]uint8, 7)
			bits := fmt.Sprintf("%.14b", pats[i])
			legal := true
			off := 0
			for j := 0; j < 7; j++ {
				bits2 := bits[2*j : 2*j+2]
				if bits2 == "00" {
					pat[j] = EMPTY
				}
				if bits2 == "01" {
					pat[j] = BLACK
				}
				if bits2 == "10" {
					pat[j] = WHITE
				}
				if bits2 == "11" {
					pat[j] = 3
					off++
				}
				if bits2 == "11" && j == 6 {
					legal = false
				}
				if bits2 == "00" && j == 6 {
					legal = false
				}
			}
			if off%2 != 0 {
				legal = false
			}
			if legal {
				drawPat(bits, 10, 40*6*float64(y)+40, 40, pat, p.Position[pats[i]*7:pats[i]*7+7], i, s)
				y++
			}
		}
		s.End()
		*/
	}
}

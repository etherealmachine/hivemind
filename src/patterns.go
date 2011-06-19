package main

import (
	"os"
	"bufio"
	"strconv"
	"strings"
)

const (
	INIT_WEIGHT = int(5)
)

type PatternMatcher interface {
	Apply(color byte, vertex int, t Tracker) bool
}

type ComboMatcher struct {
	expert map[uint32]float64
	learned *Particle
}

func (m *ComboMatcher) Apply(color byte, vertex int, t Tracker) bool {
	board := t.Board()
	neighbors := t.Neighbors(vertex, 2)
	hash := NeighborhoodHash(color, board, neighbors, 11)
	weight, exists := m.expert[hash]
	if !exists && m.learned != nil {
		weight = m.learned.Get(hash)
	} else if !exists {
		return false
	}
	t.(*GoTracker).weights.Set(color, vertex, int(weight))
	return true
}

func LoadPatternMatcher(config *Config) {
	if config.Pat && config.Xfile != "" {
	
		patterns := make(map[uint32]float64)
	
		f, err := os.Open(config.Xfile)
		if err != nil {
			panic(err)
		}
		r := bufio.NewReader(f)
		for {
			line1, err := r.ReadString('\n')
			if err != nil {
				break
			}
			line2, err := r.ReadString('\n')
			line3, err := r.ReadString('\n')
			line4, err := r.ReadString('\n')
			r.ReadString('\n')
			pat, weight := loadTextPattern(line1, line2, line3, line4)
			patterns[pat] = float64(weight)
		}
		config.expert_patterns = patterns
		var p *Particle
		if config.Pfile != "" {
			p = LoadBest(config.Pfile, config)
		}
		config.matcher = &ComboMatcher{patterns, p}
	}
}

func loadTextPattern(line1, line2, line3, line4 string) (uint32, int) {
	row1 := strings.Split(strings.Trim(line1, "\n"), " ", -1)
	row2 := strings.Split(strings.Trim(line2, "\n"), " ", -1)
	row3 := strings.Split(strings.Trim(line3, "\n"), " ", -1)
	color := Atoc(strings.Trim(line4, "\n"))
	board := make([]byte, 9)
	board[0] = Atoc(row1[0])
	board[1] = Atoc(row1[1])
	board[2] = Atoc(row1[2])
	board[3] = Atoc(row2[0])
	board[4] = EMPTY
	board[5] = Atoc(row2[2])
	board[6] = Atoc(row3[0])
	board[7] = Atoc(row3[1])
	board[8] = Atoc(row3[2])
	neighbors := make([]int, 9)
	for i := 0; i < 9; i++ {
		if board[i] == ILLEGAL {
			neighbors[i] = -1
		} else {
			neighbors[i] = i
		}
	}
	weight, _ := strconv.Atoi(strings.Trim(row2[1], "\n"))
	return NeighborhoodHash(color, board, neighbors, 11), weight
}

func swap(i, j uint32, b *uint32) {
	x := ((*b >> i) ^ (*b >> j)) & 1 // XOR temporary
	*b ^= ((x << i) | (x << j))
}

// set the ith bit of b to j
func set(i, j uint32, b *uint32) {
	*b ^= j << i
}

// get the ith bit of b
func get(i, b uint32) uint32 {
	return b >> i & 0x00000001
}

func update_hash(hash *uint32, i int, offset int, color byte) {
	// set the 2*i, 2*i+1 bits of the index
	m0 := uint32(0)
	m1 := uint32(0)
	if color == ILLEGAL {
		m0, m1 = 1, 1
	} else if color == BLACK {
		m0, m1 = 0, 1
	} else if color == WHITE {
		m0, m1 = 1, 0
	}
	set(uint32(2*i+offset), m0, hash)
	set(uint32(2*i+1+offset), m1, hash)
}

func get_hash(color byte, offset int, board []uint8, neighbors []int, order []int, reverse bool) uint32 {
	hash := uint32(0)
	if color == WHITE {
		set(0, 1, &hash)
	}
	for i := range order {
		var color byte
		if neighbors[order[i]] == -1 {
			color = ILLEGAL
		} else {
			color = board[neighbors[order[i]]]
		}
		if reverse {
			color = Reverse(color)
		}
		update_hash(&hash, i, offset, color)
	}
	return hash
}

func NeighborhoodHash(color byte, board []uint8, neighbors []int, offset int) uint32 {
	reverse := color == WHITE
	h1 := get_hash(color, offset, board, neighbors, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, reverse)
	if hash, exists := hash_cache[h1]; exists {
		return hash
	}
	h2 := get_hash(color, offset, board, neighbors, []int{0, 3, 6, 1, 4, 5, 2, 5, 8}, reverse)
	if hash, exists := hash_cache[h2]; exists {
		return hash
	}
	h3 := get_hash(color, offset, board, neighbors, []int{2, 1, 0, 5, 4, 3, 8, 7, 6}, reverse)
	if hash, exists := hash_cache[h3]; exists {
		return hash
	}
	h4 := get_hash(color, offset, board, neighbors, []int{6, 3, 0, 7, 4, 1, 8, 5, 2}, reverse)
	if hash, exists := hash_cache[h4]; exists {
		return hash
	}
	h5 := get_hash(color, offset, board, neighbors, []int{8, 7, 6, 5, 4, 3, 2, 1, 0}, reverse)
	if hash, exists := hash_cache[h5]; exists {
		return hash
	}
	h6 := get_hash(color, offset, board, neighbors, []int{8, 5, 2, 7, 4, 1, 6, 3, 0}, reverse)
	if hash, exists := hash_cache[h6]; exists {
		return hash
	}
	h7 := get_hash(color, offset, board, neighbors, []int{6, 7, 8, 3, 4, 5, 0, 1, 2}, reverse)
	if hash, exists := hash_cache[h7]; exists {
		return hash
	}
	h8 := get_hash(color, offset, board, neighbors, []int{2, 5, 8, 1, 4, 7, 0, 3, 6}, reverse)
		if hash, exists := hash_cache[h8]; exists {
		return hash
	}
	hash := h1
	if h2 < hash {
		hash = h2
	}
	if h3 < hash {
		hash = h3
	}
	if h4 < hash {
		hash = h4
	}
	if h5 < hash {
		hash = h5
	}
	if h6 < hash {
		hash = h6
	}
	if h7 < hash {
		hash = h7
	}
	if h8 < hash {
		hash = h8
	}
	hash_cache[h1] = hash
	hash_cache[h2] = hash
	hash_cache[h3] = hash
	hash_cache[h4] = hash
	hash_cache[h5] = hash
	hash_cache[h6] = hash
	hash_cache[h7] = hash
	hash_cache[h8] = hash
	return hash
}

func ptoa(board []byte, neighbors []int) string {
	cp := make([]byte, len(neighbors))
	for i := range cp {
		if neighbors[i] == -1 {
			cp[i] = ILLEGAL
		} else {
			cp[i] = board[neighbors[i]]
		}
	}
	s := Ctoa(cp[0]) + " " + Ctoa(cp[1]) + " " + Ctoa(cp[2]) + "\n"
	s += Ctoa(cp[3]) + " " + Ctoa(cp[4]) + " " + Ctoa(cp[5]) + "\n"
	s += Ctoa(cp[6]) + " " + Ctoa(cp[7]) + " " + Ctoa(cp[8])
	return s
}

var hash_cache map[uint32]uint32

func init() {
	hash_cache = make(map[uint32]uint32)
}

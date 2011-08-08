package main

import (
	"strings"
	"strconv"
	"fmt"
	"log"
	"json"
	"os"
)

type HexTracker struct {
	boardsize                                 int
	sqsize                                    int
	parent                                    []int
	rank                                      []int
	board                                     []byte
	weights                                   *WeightTree
	winner                                    byte
	played                                    []byte
	adj                                       []int
	neighbors                                 [][][]int
	config                                    *Config
	SIDE_UP, SIDE_DOWN, SIDE_LEFT, SIDE_RIGHT int
}

func NewHexTracker(config *Config) *HexTracker {
	t := new(HexTracker)

	t.boardsize = config.Size
	t.sqsize = t.boardsize * t.boardsize
	t.adj = hex_adj[t.boardsize]
	t.neighbors = hex_neighbors[t.boardsize]
	t.SIDE_UP = t.sqsize
	t.SIDE_DOWN = t.sqsize + 1
	t.SIDE_LEFT = t.sqsize + 2
	t.SIDE_RIGHT = t.sqsize + 3
	t.board = make([]byte, t.sqsize)
	t.parent = make([]int, t.sqsize+4)
	t.rank = make([]int, t.sqsize+4)
	t.weights = NewWeightTree(t.sqsize)
	// initialize union-find data structure
	for i := 0; i < t.sqsize + 4; i++ {
		t.parent[i] = i
		if i < t.sqsize {
			t.rank[i] = 1
		} else {
			t.rank[i] = t.sqsize
		}
		if i < t.sqsize {
			t.weights.Set(BLACK, i, INIT_WEIGHT)
			t.weights.Set(WHITE, i, INIT_WEIGHT)
		}
	}

	t.winner = EMPTY

	t.played = make([]byte, t.sqsize)
	
	t.config = config

	return t
}

func (t *HexTracker) Copy() Tracker {
	cp := new(HexTracker)

	cp.boardsize = t.boardsize
	cp.adj = t.adj
	cp.neighbors = t.neighbors
	cp.SIDE_UP = t.SIDE_UP
	cp.SIDE_DOWN = t.SIDE_DOWN
	cp.SIDE_LEFT = t.SIDE_LEFT
	cp.SIDE_RIGHT = t.SIDE_RIGHT
	cp.sqsize = t.sqsize
	cp.board = make([]byte, cp.sqsize)
	cp.parent = make([]int, cp.sqsize+4)
	cp.rank = make([]int, cp.sqsize+4)
	copy(cp.parent, t.parent)
	copy(cp.rank, t.rank)
	copy(cp.board, t.board)
	cp.weights = t.weights.Copy()

	cp.winner = t.winner

	cp.played = make([]byte, cp.sqsize)
	
	cp.config = t.config

	return cp
}

func (t *HexTracker) Play(color byte, vertex int) {
	if vertex != -1 {
		root := find(vertex, t.parent)
		for i := 0; i < 6; i++ {
			adj := find(t.adj[vertex*6+i], t.parent)
			if adj == -1 {
				continue
			}
			if color == BLACK &&
				((root == t.SIDE_UP && adj == t.SIDE_DOWN) || (root == t.SIDE_DOWN && adj == t.SIDE_UP)) {
				t.winner = BLACK
				break
			} else if color == WHITE &&
				((root == t.SIDE_LEFT && adj == t.SIDE_RIGHT) || (root == t.SIDE_RIGHT && adj == t.SIDE_LEFT)) {
				t.winner = WHITE
				break
			}
			if (adj < t.sqsize && t.board[adj] == color) ||
				(color == BLACK && (adj == t.SIDE_UP || adj == t.SIDE_DOWN)) ||
				(color == WHITE && (adj == t.SIDE_LEFT || adj == t.SIDE_RIGHT)) {
				root = fastUnion(root, adj, t.parent, t.rank)
			}
		}
		t.board[vertex] = color
		// cannot play on occupied vertex
		t.weights.Set(BLACK, vertex, 0)
		t.weights.Set(WHITE, vertex, 0)
		if t.config.policy_weights != nil {
			t.updateWeights(vertex)
		}
		
		if t.played[vertex] == EMPTY {
			t.played[vertex] = color
		}
	}
}

func (t *HexTracker) updateWeights(vertex int) {
	if t.board[vertex] != EMPTY {
		for i := range t.neighbors[1][vertex] {
			neighbor := t.neighbors[1][vertex][i]
			if neighbor != -1 && t.board[neighbor] == EMPTY {
				t.updateWeights(neighbor)
			}
		}
	} else {
		black_weight := t.weights.Get(BLACK, vertex)
		if black_weight != 0 {
			weight := t.get_pattern_weight(BLACK, vertex)
			black_weight = int(float64(black_weight) * weight)
			if black_weight == 0 {
				black_weight = 1
			}
		}
		t.weights.Set(BLACK, vertex, black_weight)
		white_weight := t.weights.Get(WHITE, vertex)
		if white_weight != 0 {
			weight := t.get_pattern_weight(WHITE, vertex)
			white_weight = int(float64(white_weight) * weight)
			if white_weight == 0 {
				white_weight = 1
			}
		}
		t.weights.Set(WHITE, vertex, white_weight)
	}
}

func (t *HexTracker) get_pattern_weight(color byte, vertex int) float64 {
	hash := hex_min_hash[hex_hash(color, t.board, t.neighbors[1][vertex])]
	return t.config.policy_weights.Get(hash)
}

func (t *HexTracker) Playout(color byte) {
	fmt.Fprintln(jsonLog, "START")
	for {
		vertex := t.weights.Rand(color)
		if t.config.VeryVerbose {
			log.Println(Ctoa(color) + t.Vtoa(vertex))
		}
		t.Play(color, vertex)
		
		if t.config.VeryVerbose {
			m := make(map[string]interface{})
			for i := range t.board {
				m[t.Vtoa(i)] = map[string]interface{} {
					"occ" : Ctoa(t.board[i]),
					"black" : t.weights.Prob(BLACK, i),
					"white" : t.weights.Prob(WHITE, i),
				}
			}
			if bytes, err := json.Marshal(m); err != nil {
				fmt.Fprintln(jsonLog, err)
			} else {
				fmt.Fprintln(jsonLog, string(bytes))
			}
		}
		
		if t.config.VeryVerbose {
			log.Println(t.String())
		}
		if t.winner != EMPTY {
			if t.config.VeryVerbose {
				log.Println("FINAL: " + Ctoa(t.winner))
			}
			fmt.Fprintln(jsonLog, "END")
			return
		}
		color = Reverse(color)
	}
	panic("should never happen")
}

func (t *HexTracker) WasPlayed(color byte, vertex int) bool {
	return t.played[vertex] == color
}

func (t *HexTracker) Legal(color byte, vertex int) bool {
	return vertex != -1 && t.board[vertex] == EMPTY
}

func (t *HexTracker) Score(Komi float64) (float64, float64) {
	if t.winner == BLACK {
		return 1, 0
	} else if t.winner == WHITE {
		return 0, 1
	}
	return 0, 0
}

func (t *HexTracker) Winner() byte {
	return t.winner
}

func (t *HexTracker) SetKomi(Komi float64) {

}

func (t *HexTracker) GetKomi() float64 {
	return 0
}

func (t *HexTracker) Boardsize() int {
	return t.boardsize
}

func (t *HexTracker) Sqsize() int {
	return t.sqsize
}

func (t *HexTracker) Board() []byte {
	return t.board
}

func (t *HexTracker) Territory(color byte) []float64 {
	territory := make([]float64, t.sqsize)
	for i := range t.board {
		if t.board[i] == color {
			territory[i] = 1
		}
	}
	return territory
}

func (t *HexTracker) Verify() {
}

func (t *HexTracker) Adj(vertex int) []int {
	return t.adj[vertex*6 : (vertex+1)*6]
}

func (t *HexTracker) Vtoa(v int) string {
	if v == -1 {
		return "PASS"
	}
	alpha, num := v%t.boardsize, v/t.boardsize
	num++
	alpha = alpha + 'A'
	if alpha >= 'I' {
		alpha++
	}
	return fmt.Sprintf("%s%d", string(alpha), num)
}

func (t *HexTracker) Atov(s string) int {
	if s == "PASS" || s == "pass" {
		return -1
	}
	// pull apart into alpha and int pair
	col := byte(strings.ToUpper(s)[0])
	row, err := strconv.Atoi(s[1:len(s)])
	row--
	if col >= 'I' {
		col--
	}
	if err != nil {
		panic("Failed to convert string to vertex")
	}
	return row*t.boardsize + int(col-'A')
}

func (t *HexTracker) String() (s string) {
	s += "   "
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha++
		}
		s += string(alpha)
		if col != t.boardsize-1 {
			s += " "
		}
	}
	s += "\n"
	for row := 0; row < t.boardsize; row++ {
		for i := 0; i < row; i++ {
			s += " "
		}
		s += fmt.Sprintf("%2.d ", row+1)
		for col := 0; col < t.boardsize; col++ {
			v := row*t.boardsize + col
			s += Ctoa(t.board[v])
			if col != t.boardsize-1 {
				s += " "
			}
		}
		s += fmt.Sprintf(" %2.d", row+1)
		if row != t.boardsize-1 {
			s += "\n"
		}
	}
	s += "\n  "

	for i := 0; i < t.boardsize; i++ {
		s += " "
	}
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha++
		}
		s += string(alpha)
		if col != t.boardsize-1 {
			s += " "
		}
	}
	return
}

func (t *HexTracker) WeightString() string {
	s := ""
	for hash, weight := range t.config.policy_weights.Position {
		color := BLACK
		if hash & 0x80000000 != 0 {
			color = WHITE
		}
		board := []byte{EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY}
		for i := 0; i < 7; i++ {
			mask := hash & (1<<uint32(2*i) | 1<<uint32((2*i)+1))
			if mask == hex_hash_mask[i][BLACK] {
				board[i] = BLACK
			} else if mask == hex_hash_mask[i][WHITE] {
				board[i] = WHITE
			} else if mask == hex_hash_mask[i][ILLEGAL] {
				board[i] = ILLEGAL
			}
		}
		s += fmt.Sprintf("%.3f\n", weight)
		s += fmt.Sprintf(" %s %s\n", Ctoa(board[0]), Ctoa(board[1]))
		s += fmt.Sprintf("%s %s %s\n", Ctoa(board[5]), Ctoa(color), Ctoa(board[2]))
		s += fmt.Sprintf(" %s %s\n", Ctoa(board[4]), Ctoa(board[3]))
	}
	return s
}

var hex_adj map[int][]int
var hex_neighbors map[int][][][]int
var hex_min_hash map[uint32]uint32
var hex_hash_mask [7][4]uint32
var jsonLog *os.File

func init() {
	hex_adj = make(map[int][]int)
	hex_neighbors = make(map[int][][][]int)
	for boardsize := 3; boardsize <= 19; boardsize++ {
		setup_hex_adj(boardsize)
		setup_hex_neighbors(boardsize)
	}
	hex_hash_mask = [7][4]uint32{
		[4]uint32{
			0x00000000,
			0x00000001,
			0x00000002,
			0x00000003,
		},
		[4]uint32{
			0x00000000,
			0x00000004,
			0x00000008,
			0x0000000C,
		},
		[4]uint32{
			0x00000000,
			0x00000010,
			0x00000020,
			0x00000030,
		},
		[4]uint32{
			0x00000000,
			0x00000040,
			0x00000080,
			0x000000C0,
		},
		[4]uint32{
			0x00000000,
			0x00000100,
			0x00000200,
			0x00000300,
		},
		[4]uint32{
			0x00000000,
			0x00000400,
			0x00000800,
			0x00000C00,
		},
		[4]uint32{
			0x00000000,
			0x00001000,
			0x00002000,
			0x00003000,
		},
	}
	setup_hex_min_hash()
	jsonLog, _ = os.Create("json.log")
}

func setup_hex_adj(boardsize int) {
	s := boardsize * boardsize
	hex_adj[boardsize] = make([]int, s*6)

	SIDE_UP = s
	SIDE_DOWN = s + 1
	SIDE_LEFT = s + 2
	SIDE_RIGHT = s + 3

	for i := 0; i < s; i++ {
		hex_adj[boardsize][i*6+UP] = -1
		hex_adj[boardsize][i*6+DOWN] = -1
		hex_adj[boardsize][i*6+UP_RIGHT] = -1
		hex_adj[boardsize][i*6+DOWN_RIGHT] = -1
		hex_adj[boardsize][i*6+UP_LEFT] = -1
		hex_adj[boardsize][i*6+DOWN_LEFT] = -1
	}

	for i := 0; i < s; i++ {
		hex_adj[boardsize][i*6+UP_LEFT] = i - boardsize
		hex_adj[boardsize][i*6+UP_RIGHT] = hex_adj[boardsize][i*6+UP_LEFT] + 1
		hex_adj[boardsize][i*6+DOWN_RIGHT] = i + boardsize
		hex_adj[boardsize][i*6+DOWN_LEFT] = hex_adj[boardsize][i*6+DOWN_RIGHT] - 1
		hex_adj[boardsize][i*6+LEFT] = i - 1
		hex_adj[boardsize][i*6+RIGHT] = i + 1
		if i < boardsize {
			hex_adj[boardsize][i*6+UP_LEFT] = SIDE_UP
			hex_adj[boardsize][i*6+UP_RIGHT] = SIDE_UP
		}
		if i > s-boardsize-1 {
			hex_adj[boardsize][i*6+DOWN_LEFT] = SIDE_DOWN
			hex_adj[boardsize][i*6+DOWN_RIGHT] = SIDE_DOWN
		}
		if i%boardsize == 0 {
			hex_adj[boardsize][i*6+LEFT] = SIDE_LEFT
			hex_adj[boardsize][i*6+DOWN_LEFT] = SIDE_LEFT
		}
		if (i+1)%boardsize == 0 {
			hex_adj[boardsize][i*6+RIGHT] = SIDE_RIGHT
			hex_adj[boardsize][i*6+UP_RIGHT] = SIDE_RIGHT
		}
	}
}

func setup_hex_neighbors(size int) {
	hex_neighbors[size] = make([][][]int, 2)
	hex_neighbors[size][0] = make([][]int, size*size)
	hex_neighbors[size][1] = make([][]int, size*size)

	neighbors := make([][]int, size*size)
	for vertex := 0; vertex < size*size; vertex++ {
		v2 := vertex + 1
		v3 := vertex + size
		v4 := vertex + size + 1
		if (vertex+1) % size == 0 {
			v2 = -1
			v4 = -1
		}
		if vertex >= (size * size) - size {
			v3 = -1
			v4 = -1
		}
		hex_neighbors[size][0][vertex] = []int{vertex, v2, v3, v4}

		neighbors[vertex] = make([]int, 7)
		neighbors[vertex][0] = vertex - size
		neighbors[vertex][1] = vertex - size + 1
		neighbors[vertex][2] = vertex + 1
		neighbors[vertex][3] = vertex + size
		neighbors[vertex][4] = vertex + size - 1
		neighbors[vertex][5] = vertex - 1
		neighbors[vertex][6] = vertex
		if vertex % size == 0 {
			// left
			neighbors[vertex][4] = -1
			neighbors[vertex][5] = -1
		}
		if (vertex+1) % size == 0 {
			// right
			neighbors[vertex][1] = -1
			neighbors[vertex][2] = -1
		}
		if vertex < size {
			// top
			neighbors[vertex][0] = -1
			neighbors[vertex][1] = -1
		}
		if vertex >= (size * size) - size {
			// bottom
			neighbors[vertex][3] = -1
			neighbors[vertex][4] = -1
		}
	}
	hex_neighbors[size][1] = neighbors
}

func setup_hex_min_hash() {
	hex_min_hash = make(map[uint32]uint32)
	symmetries := [][]int{
		[]int{0, 1, 2, 3, 4, 5, 6},
		[]int{1, 0, 5, 4, 3, 2, 6},
		[]int{3, 4, 5, 0, 1, 2, 6},
		[]int{4, 3, 2, 1, 0, 5, 6},
	}
	for board := []byte{EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY}; odometer(board, len(board)-1); {
		black_min_hash := ^uint32(0)
		white_min_hash := ^uint32(0)
		for x := range symmetries {
			black_hash := hex_hash(BLACK, board, symmetries[x])
			if black_hash < black_min_hash {
				black_min_hash = black_hash
			}
			white_hash := hex_hash(WHITE, board, symmetries[x])
			if white_hash < white_min_hash {
				white_min_hash = white_hash
			}
		}
		for x := range symmetries {
			black_hash := hex_hash(BLACK, board, symmetries[x])
			white_hash := hex_hash(WHITE, board, symmetries[x])
			hex_min_hash[black_hash] = black_min_hash
			hex_min_hash[white_hash] = white_min_hash
		}
	}
}

func hex_hash(color byte, board []byte, neighbors []int) uint32 {
	var hash uint32
	if color == WHITE {
		hash = 1<<31
	} else {
		hash = 0
	}
	illegal := 0
	for j := range neighbors {
		neighbor := neighbors[j]
		if neighbor == -1 {
			hash |= hex_hash_mask[j][ILLEGAL]
			illegal++
		} else {
			hash |= hex_hash_mask[j][board[neighbor]]
		}
	}
	return hash
}

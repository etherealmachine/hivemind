package main

import (
	"container/vector"
	"strings"
	"strconv"
	"fmt"
)

type HexTracker struct {
	boardsize                                 int
	sqsize                                    int
	parent                                    []int
	rank                                      []int
	board                                     []byte
	empty                                     *vector.IntVector
	winner                                    byte
	played                                    []byte
	adj                                       []int
	SIDE_UP, SIDE_DOWN, SIDE_LEFT, SIDE_RIGHT int
}

func NewHexTracker(config *Config) *HexTracker {
	t := new(HexTracker)

	t.boardsize = config.Size
	t.sqsize = t.boardsize * t.boardsize
	t.adj = hex_adj[t.boardsize]
	t.SIDE_UP = t.sqsize
	t.SIDE_DOWN = t.sqsize + 1
	t.SIDE_LEFT = t.sqsize + 2
	t.SIDE_RIGHT = t.sqsize + 3
	t.board = make([]byte, t.sqsize)
	t.parent = make([]int, t.sqsize+4)
	t.rank = make([]int, t.sqsize+4)
	t.empty = new(vector.IntVector)
	// initialize union-find data structure
	for i := 0; i < t.sqsize+4; i++ {
		t.parent[i] = i
		if i < t.sqsize {
			t.rank[i] = 1
			t.empty.Push(i)
		} else {
			t.rank[i] = t.sqsize
		}
	}

	t.winner = EMPTY

	t.played = make([]byte, t.sqsize)

	return t
}

func (t *HexTracker) Copy() Tracker {
	cp := new(HexTracker)

	cp.boardsize = t.boardsize
	cp.adj = t.adj
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
	cp.empty = new(vector.IntVector)
	*cp.empty = t.empty.Copy()

	cp.winner = t.winner

	cp.played = make([]byte, cp.sqsize)

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
		// remove vertex from empty
		if t.empty.Len() > 0 && t.empty.Last() == vertex {
			t.empty.Pop()
		}

		if t.played[vertex] == EMPTY {
			t.played[vertex] = color
		}
	}
}

func (t *HexTracker) Playout(color byte, m PatternMatcher) {
	vertex := -1
	shuffle(t.empty)
	for {
		if vertex == -1 {
			vertex = t.randLegal(color)
		}
		t.Play(color, vertex)
		if t.winner != EMPTY {
			return
		}
		color = Reverse(color)
		if m != nil && vertex != -1 {
			suggestion := m.Match(color, vertex, t)
			vertex = suggestion
			if suggestion != -1 && t.board[suggestion] != EMPTY {
				panic("dammit")
			}
		} else {
			vertex = -1
		}
	}
	panic("should never happen")
}

func (t *HexTracker) randLegal(color byte) int {
	for i := t.empty.Len() - 1; i >= 0; i-- {
		v := t.empty.At(i)
		if t.Legal(color, v) {
			return v
		}
		t.empty.Delete(i)
	}
	return -1
}

func (t *HexTracker) WasPlayed(color byte, vertex int) bool {
	return t.played[vertex] == color
}

func (t *HexTracker) Legal(color byte, vertex int) bool {
	return t.board[vertex] == EMPTY
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

func (t *HexTracker) Neighbors(vertex int, Size int) []int {
	return hex_neighbors[t.boardsize][Size][vertex]
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
	s += " "
	if t.boardsize > 9 {
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
	if t.boardsize > 9 {
		s += " "
	}
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

var hex_adj map[int][]int
var hex_neighbors map[int][][][]int

func init() {
	hex_adj = make(map[int][]int)
	hex_neighbors = make(map[int][][][]int)
	for boardsize := 3; boardsize <= 19; boardsize++ {
		setup_hex_adj(boardsize)
		setup_hex_neighbors(boardsize)
	}
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

func setup_hex_neighbors(Size int) {
	hex_neighbors[Size] = make([][][]int, 3)
	hex_neighbors[Size][0] = make([][]int, Size*Size)
	hex_neighbors[Size][1] = make([][]int, Size*Size)
	hex_neighbors[Size][2] = make([][]int, Size*Size)

	Neighbors := make([][]int, Size*Size)
	for vertex := 0; vertex < Size*Size; vertex++ {
		hex_neighbors[Size][0][vertex] = []int{vertex}
		v2 := vertex + 1
		v3 := vertex + Size
		v4 := vertex + Size + 1
		if (vertex+1)%Size == 0 {
			v2 = -1
			v4 = -1
		}
		if vertex >= (Size*Size)-Size {
			v3 = -1
			v4 = -1
		}
		hex_neighbors[Size][1][vertex] = []int{vertex, v2, v3, v4}

		Neighbors[vertex] = make([]int, 7)
		Neighbors[vertex][0] = vertex - Size
		Neighbors[vertex][1] = vertex - Size + 1
		Neighbors[vertex][2] = vertex + 1
		Neighbors[vertex][3] = vertex + Size
		Neighbors[vertex][4] = vertex + Size - 1
		Neighbors[vertex][5] = vertex - 1
		Neighbors[vertex][6] = vertex
		if vertex%Size == 0 {
			// left
			Neighbors[vertex][4] = -1
			Neighbors[vertex][5] = -1
		}
		if (vertex+1)%Size == 0 {
			// right
			Neighbors[vertex][1] = -1
			Neighbors[vertex][2] = -1
		}
		if vertex < Size {
			// top
			Neighbors[vertex][0] = -1
			Neighbors[vertex][1] = -1
		}
		if vertex >= (Size*Size)-Size {
			// bottom
			Neighbors[vertex][3] = -1
			Neighbors[vertex][4] = -1
		}
	}
	hex_neighbors[Size][2] = Neighbors
}

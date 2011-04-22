package main

import "container/vector"

type HexTracker struct {
	boardsize int
	sqsize int
	parent []int
	rank []int
	board []byte
	empty *vector.IntVector
	winner byte
	played []byte
	record []int
	moveCount int
	adj []int
	SIDE_UP, SIDE_DOWN, SIDE_LEFT, SIDE_RIGHT int
}

func NewHexTracker(boardsize int) *HexTracker {
	t := new(HexTracker)
	
	t.boardsize = boardsize
	t.sqsize = boardsize * boardsize
	t.adj = hex_adj[boardsize]
	t.SIDE_UP = t.sqsize
	t.SIDE_DOWN = t.sqsize + 1
	t.SIDE_LEFT = t.sqsize + 2
	t.SIDE_RIGHT = t.sqsize + 3
	t.board = make([]byte, t.sqsize)
	t.parent = make([]int, t.sqsize + 4)
	t.rank = make([]int, t.sqsize + 4)
	t.empty = new(vector.IntVector)
	// initialize union-find data structure
	for i := 0; i < t.sqsize + 4; i++ {
		t.parent[i] = i
		if i < t.sqsize {
			t.rank[i] = 1
			t.empty.Push(i)
		} else {
			t.rank[i] = t.sqsize
		}
	}
	shuffle(t.empty)
	
	t.winner = EMPTY
	
	t.played = make([]byte, t.sqsize)
	
	t.moveCount = 0
	t.record = make([]int, t.sqsize)
	
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
	cp.parent = make([]int, cp.sqsize + 4)
	cp.rank = make([]int, cp.sqsize + 4)
	copy(cp.parent, t.parent)
	copy(cp.rank, t.rank)
	copy(cp.board, t.board)
	cp.empty = new(vector.IntVector)
	*cp.empty = t.empty.Copy()
	shuffle(cp.empty)
	
	cp.winner = t.winner
	
	cp.played = make([]byte, cp.sqsize)
	
	return cp
}

func (t *HexTracker) Play(color byte, vertex int) {
	if vertex != -1 {
		root := find(vertex, t.parent)
		for i := 0; i < 6; i++ {
			adj := find(t.adj[vertex*6+i], t.parent)
			if adj == -1 { continue }
			if (color == BLACK &&
					((root == t.SIDE_UP && adj == t.SIDE_DOWN) || (root == t.SIDE_DOWN && adj == t.SIDE_UP))) {
				t.winner = BLACK
				break
			} else if (color == WHITE &&
					((root == t.SIDE_LEFT && adj == t.SIDE_RIGHT) || (root == t.SIDE_RIGHT && adj == t.SIDE_LEFT))) {
				t.winner = WHITE
				break
			}
			if 	(adj < t.sqsize && t.board[adj] == color) ||
					(color == BLACK && (adj == t.SIDE_UP || adj == t.SIDE_DOWN)) ||
					(color == WHITE && (adj == t.SIDE_LEFT || adj == t.SIDE_RIGHT)) {
				root = fastUnion(root, adj, t.parent, t.rank)
			}
		}
		t.board[vertex] = color
		// remove vertex from empty
		if t.empty.Len() > 0 && t.empty.Last() == vertex { t.empty.Pop() }
	
		if t.played[vertex] == EMPTY {
			t.played[vertex] = color
		} else {
			t.played[vertex] = BOTH
		}
		if t.record != nil {
			if t.moveCount < 0 || t.moveCount > len(t.record) { panic(t.moveCount) }
			t.record[t.moveCount] = vertex
			t.moveCount++
		}
	}
	if *verbose {
		log.Println(Bwboard(t.board, t.boardsize, true))
	}
}

func (t *HexTracker) Playout(color byte, max int, m PatternMatcher) {
	depth := 0
	vertex := -1
	for {
		if vertex == -1 { vertex = t.nextLegal(color) }
		t.Play(color, vertex)
		if t.winner != EMPTY {
			return
		}
		color = Reverse(color)
		depth++
		if max != -1 && depth >= max {
			return
		}
		if m != nil {
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

func (t *HexTracker) WasPlayed(color byte, vertex int) bool {
	return t.played[vertex] == color || t.played[vertex] == BOTH
}

func (t *HexTracker) Legal(color byte, vertex int) bool {
	return t.board[vertex] == EMPTY
}

func (t *HexTracker) nextLegal(color byte) int {
	for i := t.empty.Len()-1; i >= 0; i-- {
		v := t.empty.At(i)
		if t.Legal(color, v) { return v }
		t.empty.Delete(i)
	}
	return -1
}

func (t *HexTracker) Score(komi float64) (float64, float64) {
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

func (t *HexTracker) SetKomi(komi float64) {

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

func (t *HexTracker) Territory() []byte {
	return t.board
}

func (t *HexTracker) Record() []int {
	return t.record
}

func (t *HexTracker) MoveCount() int {
	return 0
}

func (t *HexTracker) SetMaxMoves(max int) {
}

func (t *HexTracker) Finish() {
}

func (t *HexTracker) Verify() {
}

var hex_adj [][]int
func init() {
	hex_adj = make([][]int, 20)
	for boardsize := 3; boardsize <= 19; boardsize++ {
		setup_hex(boardsize)
	}
}

func setup_hex(boardsize int) {
	s := boardsize * boardsize
	hex_adj[boardsize] = make([]int, s*6)
	
	SIDE_UP = s
	SIDE_DOWN = s + 1
	SIDE_LEFT = s + 2
	SIDE_RIGHT = s + 3
	
	for i := 0; i < s; i++ {
		//hex_adj[boardsize][i] = make([]int, 6)
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
		if i > s - boardsize - 1 {
			hex_adj[boardsize][i*6+DOWN_LEFT] = SIDE_DOWN
			hex_adj[boardsize][i*6+DOWN_RIGHT] = SIDE_DOWN
		}
		if i % boardsize == 0 {
			hex_adj[boardsize][i*6+LEFT] = SIDE_LEFT
			hex_adj[boardsize][i*6+DOWN_LEFT] = SIDE_LEFT
		}
		if (i + 1) % boardsize == 0 {
			hex_adj[boardsize][i*6+RIGHT] = SIDE_RIGHT
			hex_adj[boardsize][i*6+UP_RIGHT] = SIDE_RIGHT
		}
	}
}

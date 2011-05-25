package main

import "rand"

type TTTTracker struct {
	boardsize int
	sqsize int
	parent []int
	rank []int
	board []byte
	winner byte
	played []byte
	moveCount int
	record []int
	prob []float64
	probSum float64
	adj [][]int
}

func NewTTTTracker(boardsize int) *TTTTracker {
	t := new(TTTTracker)
	t.boardsize = boardsize
	t.adj = ttt_adj[boardsize]
	t.sqsize = boardsize*boardsize
	t.board = make([]byte, t.sqsize)
	t.parent = make([]int, t.sqsize)
	t.rank = make([]int, t.sqsize)
	t.winner = EMPTY
	t.played = make([]byte, t.sqsize)
	t.moveCount = 0
	t.record = make([]int, t.sqsize)
	t.prob = make([]float64, t.sqsize)
	t.probSum = 0
	
	// initialize union-find data structure
	for i := 0; i < t.sqsize; i++ {
		t.parent[i] = i
		t.rank[i] = 1
		t.prob[i] = 1
		t.probSum += 1
	}
	return t
}

func (t *TTTTracker) Copy() Tracker {
	cp := new(TTTTracker)
	cp.boardsize = t.boardsize
	cp.adj = t.adj
	cp.sqsize = t.sqsize
	cp.board = make([]byte, cp.sqsize)
	cp.parent = make([]int, cp.sqsize)
	cp.rank = make([]int, cp.sqsize)
  cp.prob = make([]float64, t.sqsize) 
	copy(cp.board, t.board)
	copy(cp.parent, t.parent)
	copy(cp.rank, t.rank)
	copy(cp.prob, t.prob)
	cp.probSum = t.probSum
	
	cp.played = make([]byte, t.sqsize)
	
	cp.winner = t.winner
	
	return cp
}

func (t *TTTTracker) Play(color byte, vertex int) {
	if vertex != -1 {
		root := find(vertex, t.parent)
		for i := 0; i < 4; i++ {
			adj := find(t.adj[vertex][i], t.parent)
			if adj == -1 { continue }	
			root = fastUnion(root, adj, t.parent, t.rank)
		}
		
		t.board[vertex] = color
		
		// check for win
		for i := 0; i < t.boardsize; i++ {
			if t.count(i, DOWN, -1) == t.boardsize {
				t.winner = t.board[i]
			}
		}
		for i := 0; i < t.sqsize; i += t.boardsize {
			if t.count(i, RIGHT, -1) == t.boardsize {
				t.winner = t.board[i]
			}
		}
		if t.count(0, DOWN, RIGHT) == t.boardsize { t.winner = t.board[0] }
		if t.count(t.sqsize-t.boardsize, UP, RIGHT) == t.boardsize { t.winner = t.board[t.sqsize-t.boardsize] }
		
		t.probSum -= t.prob[vertex]
		t.prob[vertex] = 0
	
		t.played[vertex] = color
		if t.record != nil {
			if t.moveCount < 0 || t.moveCount > len(t.record) { panic(t.moveCount) }
			t.record[t.moveCount] = vertex
			t.moveCount++
		}
	}
}

func (t *TTTTracker) count(vertex int, d1 int, d2 int) int {
	if vertex == -1 || t.board[vertex] == EMPTY { return 0 }
	var adj int
	if d2 == -1 {
		adj = t.adj[vertex][d1]
		if adj == -1 { return 1 }
	} else {
		adj = t.adj[vertex][d1]
		if adj == -1 { return 1 }
		adj = t.adj[adj][d2]
	}
	if t.board[adj] == t.board[vertex] {
		return 1 + t.count(adj, d1, d2)
	}
	return 1
}

func (t *TTTTracker) Playout(color byte, m PatternMatcher) {
	vertex := -1
	for {
		if vertex == -1 { vertex = t.nextLegal(color) }
		if vertex == -1 { return }
		t.Play(color, vertex)
		if t.winner != EMPTY {
			return
		}
		color = Reverse(color)
		if m != nil {
			suggestion := m.Match(color, vertex, t)
			vertex = -1
			if suggestion != -1 {
				vertex = suggestion
			}
		} else {
			vertex = -1
		}
	}
	panic("should never happen")
}

func (t *TTTTracker) nextLegal(color byte) int {
	vertex := -1
	r := rand.Float64() * t.probSum
	for i := 0; i < t.sqsize; i++ {
		if t.prob[i] != 0 {
			r -= t.prob[i]
			if r <= 0 {
				vertex = i
				break
			}
		}
	}
	return vertex
}

func (t *TTTTracker) Legal(color byte, vertex int) bool {
	return t.board[vertex] == EMPTY
}

func (t *TTTTracker) WasPlayed(color byte, vertex int) bool {
	return t.played[vertex] == color
}

func (t *TTTTracker) Score(komi float64) (float64, float64) {
	if t.winner == BLACK {
		return 1, 0
	} else if t.winner == WHITE {
		return 0, 1
	}
	return 0, 0
}
func (t *TTTTracker) Winner() byte { return t.winner }
func (t *TTTTracker) SetKomi(komi float64) { }
func (t *TTTTracker) GetKomi() float64 { return 0 }
func (t *TTTTracker) Boardsize() int { return t.boardsize }
func (t *TTTTracker) Sqsize() int { return t.sqsize }
func (t *TTTTracker) Board() []byte { return t.board }
func (t *TTTTracker) Territory() []byte { return nil }
func (t *TTTTracker) Record() []int { return t.record }
func (t *TTTTracker) MoveCount() int { return t.moveCount }
func (t *TTTTracker) Finish() { }
func (t *TTTTracker) Verify() { }

var ttt_adj [][][]int
func init() {
	ttt_adj = make([][][]int, 20)
	for boardsize := 4; boardsize <= 19; boardsize++ {
		setup_ttt(boardsize)
	}
}
func setup_ttt(boardsize int) {
	s := boardsize * boardsize
	ttt_adj[boardsize] = make([][]int, s)
	
	for i := 0; i < s; i++ {
		ttt_adj[boardsize][i] = make([]int, 4)
		
		ttt_adj[boardsize][i][UP] = i - boardsize
		ttt_adj[boardsize][i][DOWN] = i + boardsize
		ttt_adj[boardsize][i][LEFT] = i - 1
		ttt_adj[boardsize][i][RIGHT] = i + 1
		
		if i < boardsize {
			ttt_adj[boardsize][i][UP] = -1
		}
		if i >= s - boardsize {
			ttt_adj[boardsize][i][DOWN] = -1
		}
		if i % boardsize == 0 {
			ttt_adj[boardsize][i][LEFT] = -1
		}
		if (i + 1) % boardsize == 0 {
			ttt_adj[boardsize][i][RIGHT] = -1
		}
	}
}

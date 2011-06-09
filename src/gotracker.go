package main

import (
	"fmt"
	"os"
	"container/vector"
	"rand"
	"strings"
	"strconv"
)

// Tracks a game of Go
// parent and rank comprise a union-find dataset to track chains
// vertices in the same set are part of the same chain.
// liberties returns the number of liberties for the chain
// it is only correct for the root of the set
type GoTracker struct {
	boardsize int
	sqsize int
	parent []int
	rank []int
	captured []bool
	liberties [][2]uint64
	board []byte
	komi float64
	koVertex int
	koColor byte
	played []byte
	empty *vector.IntVector
	adj [][]int
	mask [][4]uint64
	passes int
	winner byte
}

// parent must be initialized so each element is a pointer to itself
// rank are initialized to zero
// board will be modified during use, should be a copy of the real board
func NewGoTracker(config *Config) (t *GoTracker) {
	t = new(GoTracker)

	t.boardsize = config.size
	t.adj = go_adj[config.size]
	t.mask = masks[config.size]
	t.sqsize = config.size * config.size

	t.parent = make([]int, t.sqsize)
	t.rank = make([]int, t.sqsize)
	t.captured = make([]bool, t.sqsize)
	t.liberties = make([][2]uint64, t.sqsize)
	t.board = make([]byte, t.sqsize)
	t.played = make([]byte, t.sqsize)
	t.empty = new(vector.IntVector)
	// initialize union-find data structure and move probabilities
	for i := 0; i < t.sqsize; i++ {
		t.parent[i] = i
		t.rank[i] = 1
		t.empty.Push(i)
	}
	shuffle(t.empty)
	t.komi = config.komi
	t.koVertex = -1
	t.koColor = EMPTY
	t.winner = EMPTY
	return
}

func (t *GoTracker) Copy() Tracker {
	cp := new(GoTracker)

	cp.boardsize = t.boardsize
	cp.adj = t.adj
	cp.mask = t.mask
	cp.sqsize = t.sqsize
	cp.parent = make([]int, cp.sqsize)
	cp.rank = make([]int, cp.sqsize)
	cp.captured = make([]bool, cp.sqsize)
	cp.liberties = make([][2]uint64, cp.sqsize)
	cp.board = make([]byte, cp.sqsize)
	copy(cp.parent, t.parent)
	copy(cp.rank, t.rank)
	copy(cp.liberties, t.liberties)
	copy(cp.board, t.board)
	cp.empty = new(vector.IntVector)
	*cp.empty = t.empty.Copy()
	shuffle(cp.empty)

	cp.komi = t.komi
	cp.koVertex = t.koVertex
	cp.koColor = t.koColor

	cp.played = make([]byte, t.sqsize)

	return cp
}

// apply color to vertex, modifying board and updating liberties of any go_adj strings
// return true if the play was legal and resulted in modifying the GoTracker's state
func (t *GoTracker) Play(color byte, vertex int) {
	if vertex != -1 {
		t.passes = 0
		t.koVertex = -1
		t.koColor = EMPTY

		// mask out adjacent liberties
		l0 := uint64(0)
		l1 := uint64(0)
		for i := 0; i < 4; i++ {
			n := t.adj[vertex][i]
			if n != -1 && t.board[n] == EMPTY {
				l0 &= t.mask[n][0]
				l1 &= t.mask[n][1]
			}
		}
		t.liberties[vertex][0] = l0
		t.liberties[vertex][1] = l1

		// remove liberty from adjacent hostiles
		for i := 0; i < 4; i++ {
			n := t.adj[vertex][i]
			root := find(vertex, t.parent)
			if n != -1 && t.board[n] == Reverse(color) {
				t.remove(color, root, find(n, t.parent))
			}
		}

		// merge with adjacent friendlies
		for i := 0; i < 4; i++ {
			n := t.adj[vertex][i]
			if n != -1 && t.board[n] == color {
				t.merge(color, vertex, n)
			}
		}

		// or in liberties?
		root := find(vertex, t.parent)
		l0 = t.liberties[root][0]
		l1 = t.liberties[root][1]
		for i := 0; i < 4; i++ {
			n := t.adj[vertex][i]
			if n != -1 && t.board[n] == EMPTY {
				l0 |= t.mask[n][0]
				l1 |= t.mask[n][1]
			}
		}
		l0 &= t.mask[vertex][2]
		l1 &= t.mask[vertex][3]

		t.liberties[root][0] = l0
		t.liberties[root][1] = l1

		// modify the board
		t.board[vertex] = color
		// remove vertex from empty
		if t.empty.Len() > 0 && t.empty.Last() == vertex { t.empty.Pop() }
		// mark vertex as played for AMAF
		if t.played[vertex] == EMPTY {
			t.played[vertex] = color
		}
	} else {
		t.passes++
	}
}

// playout simulated game, call Winner() to retrive winner based on final territory
func (t *GoTracker) Playout(color byte, m PatternMatcher) {
	vertex := -1
	last := -1
	move := 0
	for {
		vertex = t.playHeuristicMove(color)
		if vertex == -1 && last != -1 {
			vertex = t.playPatternMove(color, last, m)
		}
		if vertex == -1 {
			vertex = t.RandLegal(color)
		}
		t.Play(color, vertex)
		move++
		if move > 3 * t.sqsize { break }
		if t.Winner() != EMPTY { break }
		color = Reverse(color)
		last = vertex
		vertex = -1
	}
}

func (t *GoTracker) playHeuristicMove(color byte) int {
	atari := make(map[int]bool)
	for i := 0; i < t.sqsize; i++ {
		if t.board[i] == Reverse(color) {
			root := find(i, t.parent)
			if bitcount(t.liberties[root][0], t.liberties[root][1]) == 1 {
				var v int
				if t.liberties[root][0] != 0 {
					v = 64 - int(firstbitset(t.liberties[root][0])) - 1
				} else if t.liberties[root][1] != 0 {
					v = 64 - int(firstbitset(t.liberties[root][1])) + 64 - 1
				}
				if v != t.koVertex || color != t.koColor { atari[v] = true }
			}
		}
	}
	if len(atari) > 0 {
		i := rand.Intn(len(atari))
		for j, _ := range atari {
			if i == 0 { return j }
			i--
		}
	}
	return -1
}

func (t *GoTracker) playPatternMove(color byte, last int, m PatternMatcher) int {
	if m != nil {
		suggestion := m.Match(color, last, t)
		if suggestion != -1 && !t.Legal(color, suggestion) {
			panic("dammit")
		}
		return suggestion
	}
	return -1
}

func (t *GoTracker) RandLegal(color byte) int {
	for i := t.empty.Len()-1; i >= 0; i-- {
		v := t.empty.At(i)
		if t.Legal(color, v) { return v }
	}
	return -1
}

func (t *GoTracker) WasPlayed(color byte, vertex int) bool {
	return t.played[vertex] == color
}

// return true iff move is legal, without modifying any state
func (t *GoTracker) Legal(color byte, vertex int) bool {

	// make sure vertex is empty
	if t.board[vertex] != EMPTY { return false }

	if vertex == t.koVertex && color == t.koColor { return false }

	opp := Reverse(color)
	off := 0
	friendly := 0
	suicide := true
	for i := 0; i < 4; i++ {
		n := t.adj[vertex][i]
		if n == -1 { off++; continue }
		// check if an adj vertex is empty
		if t.board[n] == EMPTY { return true }
		root := find(n, t.parent)
		if t.board[root] == opp {
			// a capture would definitely result in a legal position
			if t.wouldCapture(vertex, root) { return true }
		} else {
			friendly++
			// refute suicide by connecting to self without removing last liberty
			if bitcount(t.liberties[root][0], t.liberties[root][1]) > 1 { suicide = false }
		}
	}
	if friendly + off == 4 {
		corners := 0
		u := t.adj[vertex][UP]
		if u != -1 {
			if ul := t.adj[u][LEFT]; ul != -1 && t.board[ul] == color { corners++ }
			if ur := t.adj[u][RIGHT]; ur != -1 && t.board[ur] == color { corners++ }
		}
		d := t.adj[vertex][DOWN]
		if d != -1 {
			if dl := t.adj[d][LEFT]; dl != -1 && t.board[dl] == color { corners++ }
			if dr := t.adj[d][RIGHT]; dr != -1 && t.board[dr] == color { corners++ }
		}
		if off >= 1 && corners >= 1 { return false }
		if off == 0 && corners >= 3 { return false }
	}
	return !suicide
}

func (t *GoTracker) Score(komi float64) (float64, float64) {
	bc, wc := 0.0, 0.0
	for i := 0; i < t.sqsize; i++ {
		if t.board[i] == BLACK {
			bc++
		} else if t.board[i] == WHITE {
			wc++
		} else if t.board[i] == EMPTY {
			checked := make([]bool, t.sqsize)
			reachesBlack, reachesWhite := t.reaches(i, checked)
			if reachesBlack && !reachesWhite {
				bc++
			} else if reachesWhite && !reachesBlack {
				wc++
			}
		}
	}
	wc += komi
	return bc, wc
}

func (t *GoTracker) Winner() byte {
	if t.passes < 2 { return EMPTY }
	if t.winner != EMPTY { return t.winner }
	bc, wc := t.Score(t.komi)
	if bc > wc {
		t.winner = BLACK
	} else {
		t.winner = WHITE
	}
	return t.winner
}

func (t *GoTracker) SetKomi(komi float64) {
	t.komi = komi
}

func (t *GoTracker) GetKomi() float64 {
	return t.komi
}

func (t *GoTracker) Boardsize() int {
	return t.boardsize
}

func (t *GoTracker) Sqsize() int {
	return t.sqsize
}

func (t *GoTracker) Board() []byte {
	return t.board
}

func (t *GoTracker) Adj(vertex int) []int {
	return go_adj[t.boardsize][vertex]
}

func (t *GoTracker) Neighbors(vertex int, size int) []int {
	return go_neighbors[t.boardsize][size][vertex]
}

func (t *GoTracker) Territory(color byte) []float64 {
	territory := make([]float64, t.sqsize)
	for i := range t.board {
		if t.board[i] == EMPTY {
			checked := make([]bool, t.sqsize)
			reachesBlack, reachesWhite := t.reaches(i, checked)
			if reachesBlack && !reachesWhite && color == BLACK {
				territory[i] = 1
			} else if reachesWhite && !reachesBlack && color == WHITE {
				territory[i] = 1
			}
		} else if t.board[i] == color {
				territory[i] = 1
		}
	}
	return territory
}

func (t *GoTracker) Verify() {
	for i := 0; i < len(t.parent); i++ {
		find(i, t.parent)
	}
	for i := 0; i < len(t.parent); i++ {
		if t.board[i] == EMPTY {
			continue
		}
		parent := find(i, t.parent)
		connected := make(map [int] bool)
		c := make(chan int)
		go DFS(parent, t.board, t.adj, c)
		for {
			n := <-c
			if n == -1 {
				break
			}
			connected[n] = true
		}
		found := false
		empty := 0
		for k, _ := range connected {
			if t.board[k] == EMPTY {
				empty++
			} else {
				found = found || k == i
			}
		}
		if !found {
			fmt.Fprintln(os.Stderr, t.Vtoa(i), t.Vtoa(parent))
			fmt.Fprintln(os.Stderr, t.String())
			panic("could not verify connected points")
		}
		liberties := bitcount(t.liberties[parent][0], t.liberties[parent][1])
		if uint(empty) != liberties {
			fmt.Fprintln(os.Stderr, t.Vtoa(parent))
			fmt.Fprintln(os.Stderr, t.String())
			fmt.Fprintln(os.Stderr, empty, liberties)
			fmt.Fprintln(os.Stderr, bitboard(t.liberties[parent][0], t.liberties[parent][1], t.boardsize))
			panic("liberties don't match up")
		}
	}
}

func (t *GoTracker) Vtoa(v int) string {
	if v == -1 { return "PASS" }
	alpha, num := v % t.boardsize, v / t.boardsize
	num = t.boardsize - num
	alpha = alpha + 'A'
	if alpha >= 'I' { alpha++ }
	return fmt.Sprintf("%s%d", string(alpha), num)
}

func (t *GoTracker) Atov(s string) int {
	if s == "PASS" || s == "pass" {
		return -1
	}
	// pull apart into alpha and int pair
	col := byte(strings.ToUpper(s)[0])
	row, err := strconv.Atoi(s[1:len(s)])
	row = t.boardsize - row
	if col >= 'I' { col-- }
	if err != nil {
		panic("Failed to convert string to vertex")
	}
	return row * t.boardsize + int(col - 'A')
}

func (t *GoTracker) String() (s string) {
	s += "  "
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' { alpha++ }
		s += string(alpha)
		if col != t.boardsize - 1 {
				s += " "
		}
	}
	s += "\n"
	for row := 0; row < t.boardsize; row++ {
		s += fmt.Sprintf("%d ", t.boardsize - row)
		for col := 0; col < t.boardsize; col++ {
			v := row * t.boardsize + col
			s += Ctoa(t.board[v])
			if col != t.boardsize - 1 {
				s += " "
			}
		}
		s += fmt.Sprintf(" %d", t.boardsize - row)
		if row != t.boardsize - 1 {
			s += "\n"
		}
	}
	s += "\n  "
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' { alpha++ }
		s += string(alpha)
		if col != t.boardsize - 1 {
				s += " "
		}
	}
	return
}

func (t *GoTracker) dead() []int {
	dead := new(vector.IntVector)
	cp := t.Copy().(*GoTracker)
	color := BLACK
	for {
		vertex := cp.RandLegal(color)
		cp.Play(color, vertex)
		if cp.Winner() != EMPTY { break }
		color = Reverse(color)
	}
	for i := 0; i < t.sqsize; i++ {
		if t.board[i] != EMPTY && cp.board[i] != t.board[i] {
			dead.Push(i)
		}
	}
	stones := make([]int, dead.Len())
	for i := 0; i < dead.Len(); i++ { stones[i] = dead.At(i) }
	return stones
}

func (t *GoTracker) reaches(vertex int, checked []bool) (reachesBlack bool, reachesWhite bool) {
	checked[vertex] = true
	if t.board[vertex] == BLACK {
		return true, false
	} else if t.board[vertex] == WHITE {
		return false, true
	}
	up := t.adj[vertex][UP]
	down := t.adj[vertex][DOWN]
	left := t.adj[vertex][LEFT]
	right := t.adj[vertex][RIGHT]
	if up != -1 && !checked[up] {
		rb, rw := t.reaches(up, checked)
		reachesBlack = reachesBlack || rb
		reachesWhite = reachesWhite || rw
	}
	if down != -1 && !checked[down] {
		rb, rw := t.reaches(down, checked)
		reachesBlack = reachesBlack || rb
		reachesWhite = reachesWhite || rw
	}
	if left != -1 && !checked[left] {
		rb, rw := t.reaches(left, checked)
		reachesBlack = reachesBlack || rb
		reachesWhite = reachesWhite || rw
	}
	if right != -1 && !checked[right] {
		rb, rw := t.reaches(right, checked)
		reachesBlack = reachesBlack || rb
		reachesWhite = reachesWhite || rw
	}
	return
}

// if go_adj chain is hostile, remove vertex from liberties
// capture if liberties are reduced to zero
func (t *GoTracker) remove(color byte, vertex int, n int) {
	// remove this liberty from vertex
	t.liberties[vertex][0] &= t.mask[n][2]
	t.liberties[vertex][1] &= t.mask[n][3]
	// remove this liberty from adj
	t.liberties[n][0] &= t.mask[vertex][2]
	t.liberties[n][1] &= t.mask[vertex][3]
	// check if this reduces adj liberties to zero and capture if so
	if t.liberties[n][0] == 0 && t.liberties[n][1] == 0 {
		// capture vertices
		captured := 0
		for i := 0; i < t.sqsize; i++ {
			if find(i, t.parent) == n {
				t.captured[i] = true
				captured++
			}
		}
		for i := 0; i < t.sqsize; i++ {
			if t.captured[i] { t.capture(i) }
			t.captured[i] = false
		}
		if captured != 1 {
			t.koVertex = -1
			t.koColor = EMPTY
		}
	}
}

func (t *GoTracker) capture(v int) {
	t.koVertex = v
	t.koColor = t.board[v]
	// update board
	t.board[v] = EMPTY
	// re-initialize GoTracker structures to empty
	t.parent[v] = v
	t.rank[v] = 1
	t.liberties[v][0] = 0
	t.liberties[v][1] = 0
	// add new liberty to go_adj occupied vertices
	for i := 0; i < 4; i++ {
		n := t.adj[v][i]
		if n == -1 { continue }
		if t.board[n] != EMPTY {
			root := find(n, t.parent)
			// xor in the liberty bit for vertex at adj
			t.liberties[root][0] |= t.mask[v][0]
			t.liberties[root][1] |= t.mask[v][1]
		}
	}
	// add captured vertex to empty list
	t.empty.Insert(rand.Intn(t.empty.Len()), v)
}

func (t *GoTracker) merge(color byte, vertex int, n int) {
	vertex = find(vertex, t.parent)
	n = find(n, t.parent)
	// xor in vertex liberties to adj chain
	l0 := t.liberties[vertex][0] | t.liberties[n][0]
	l1 := t.liberties[vertex][1] | t.liberties[n][1]
	// remove vertex from adj chain's liberties
	l0 &= t.mask[vertex][2]
	l1 &= t.mask[vertex][3]
	l0 &= t.mask[n][2]
	l1 &= t.mask[n][3]
	// merge vertices into the same chain
	root := union(vertex, n, t.parent, t.rank)
	// use new mask for root of chain
	t.liberties[root][0] = l0
	t.liberties[root][1] = l1
}

// check if move would capture go_adj hostile chain, without modifying state
func (t *GoTracker) wouldCapture(vertex int, n int) bool {
	// remove this liberty from adj
	ar0 := t.liberties[n][0] & t.mask[vertex][2]
	ar1 := t.liberties[n][1] & t.mask[vertex][3]
	// check if this reduces adj liberties to zero and return true if so
	if bitcount(ar0, ar1) == 0 {
		return true
	}
	return false
}

func bitcount(u uint64, v uint64) (c uint) {
	// c accumulates the total bits set in v
	for c = 0; u != 0; c++ {
		u &= u - 1; // clear the least significant bit set
	}
	for ; v != 0; c++ {
		v &= v - 1; // clear the least significant bit set
	}
	return
}

// return the index of the first bit set in u
func firstbitset(u uint64) uint64 {
	for i := uint64(0); i < 64; i++ {
		if ((1 << i) & u) != 0 { return i }
	}
	return 64
}

func bitboard(v0 uint64, v1 uint64, boardsize int) (s string) {
	s += "	"
	for col := 0; col < boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha = alpha + 1
		}
		s += string(alpha)
		if col != boardsize - 1 {
				s += " "
		}
	}
	s += "\n"
	for row := 0; row < boardsize; row++ {
		s += fmt.Sprintf("%d ", boardsize - row)
		for col := 0; col < boardsize; col++ {
			vertex := row * boardsize + col
			var v uint64
			var bit uint64
			if vertex < 64 {
				bit = uint64(64 - vertex - 1)
				v = v0
			} else {
				bit = uint64(64 - (vertex - 64)	- 1)
				v = v1
			}
			var mask uint64 = 1 << bit
			if (mask & v) == 0 {
				s += "0 "
			} else {
				s += "1 "
			}
		}
		s += fmt.Sprintf(" %d", boardsize - row)
		if row != boardsize - 1 {
			s += "\n"
		}
	}
	s += "\n	"
	for col := 0; col < boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha = alpha + 1
		}
		s += string(alpha)
		if col != boardsize - 1 {
				s += " "
		}
	}
	s += "\n"
	return
}

func (t *GoTracker) parentboard() (s string) {
	for row := 0; row < t.boardsize; row++ {
		for col := 0; col < t.boardsize; col++ {
			vertex := row * (t.boardsize) + col
			if t.parent[vertex] == vertex {
				s += ". "
			} else {
				s += t.Vtoa(t.parent[vertex])
			}
			s += " "
		}
		s += "\n"
	}
	return
}

/*
	mask for packing liberties into 2 64-bit integers
	given a vertex int, must return the int64 with 1 in the spot for that vertex
	only works for 9*9 = 81 bits needed
	64 bits of first int are vertices 0-63
	17 bits of second int are vertices 64-81
	last 47 bits of second int are all zero
*/
var masks map[int][][4]uint64
var go_adj map[int][][]int
var go_neighbors map[int][][][]int
func init() {
	go_adj = make(map[int][][]int)
	go_neighbors = make(map[int][][][]int)
	masks = make(map[int][][4]uint64)
	for boardsize := 4; boardsize <= 19; boardsize++ {
		setup_go(boardsize)
	}
}
func setup_go(boardsize int) {
	masks[boardsize] = make([][4]uint64, boardsize * boardsize)
	for i := 0; i < len(masks[boardsize]); i++ {
		var m uint64 = 1
		if i < 64 {
			masks[boardsize][i][0] = m << uint64(64 - i - 1)
		} else {
			masks[boardsize][i][1] = m << uint64(64 - (i - 64)	- 1)
		}
		masks[boardsize][i][2] = masks[boardsize][i][0] ^ 0xFFFFFFFFFFFFFFFF
		masks[boardsize][i][3] = masks[boardsize][i][1] ^ 0xFFFFFFFFFFFFFFFF
	}
	setup_go_adj(boardsize)
	setup_go_neighbors(boardsize)
}

func setup_go_adj(boardsize int) {
	go_adj[boardsize] = make([][]int, boardsize * boardsize)
	for vertex, _ := range go_adj[boardsize] {
		go_adj[boardsize][vertex] = make([]int, 4)
		set_go_adj(vertex, boardsize)
	}
}

func set_go_adj(vertex int, boardsize int) {
	row := vertex / boardsize
	col := vertex % boardsize
	up_row := row - 1
	down_row := row + 1
	left_col := col - 1
	right_col := col + 1
	up := up_row * boardsize + col
	down := down_row * boardsize + col
	left := row * boardsize + left_col
	right := row * boardsize + right_col
	go_adj[boardsize][vertex][UP] = -1
	go_adj[boardsize][vertex][DOWN] = -1
	go_adj[boardsize][vertex][LEFT] = -1
	go_adj[boardsize][vertex][RIGHT] = -1
	if up_row >= 0 && up_row < boardsize {
		go_adj[boardsize][vertex][UP] = up
	}
	if down_row >= 0 && down_row < boardsize {
		go_adj[boardsize][vertex][DOWN] = down
	}
	if left_col >= 0 && left_col < boardsize {
		go_adj[boardsize][vertex][LEFT] = left
	}
	if right_col >= 0 && right_col < boardsize {
		go_adj[boardsize][vertex][RIGHT] = right
	}
}

func setup_go_neighbors(size int) {
	go_neighbors[size] = make([][][]int, 3)
	go_neighbors[size][0] = make([][]int, size*size)
	go_neighbors[size][1] = make([][]int, size*size)
	go_neighbors[size][2] = make([][]int, size*size)
	for vertex := 0; vertex < size*size; vertex++ {
		go_neighbors[size][0][vertex] = []int{vertex}
		v2 := vertex + 1
		v3 := vertex + size
		v4 := vertex + size + 1
		if (vertex+1)%size == 0 { v2 = -1; v4 = -1 }
		if vertex >= (size*size)-size { v3 = -1; v4 = -1 }
		go_neighbors[size][1][vertex] = []int{vertex, v2, v3, v4}
		set_go_neighbors(size, vertex)
	}
}

func set_go_neighbors(size int, vertex int) {
	neighbors := go_neighbors[size][2]
	neighbors[vertex] = make([]int, 9)
	neighbors[vertex][0] = vertex - size - 1
	neighbors[vertex][1] = vertex - size
	neighbors[vertex][2] = vertex - size + 1
	neighbors[vertex][3] = vertex - 1
	neighbors[vertex][4] = vertex
	neighbors[vertex][5] = vertex + 1
	neighbors[vertex][6] = vertex + size - 1
	neighbors[vertex][7] = vertex + size
	neighbors[vertex][8] = vertex + size + 1
	if vertex%size == 0 {
		// left
		neighbors[vertex][0] = -1
		neighbors[vertex][3] = -1
		neighbors[vertex][6] = -1
	}
	if (vertex+1)%size == 0 {
		// right
		neighbors[vertex][2] = -1
		neighbors[vertex][5] = -1
		neighbors[vertex][8] = -1
	}
	if vertex < size {
		// top
		neighbors[vertex][0] = -1
		neighbors[vertex][1] = -1
		neighbors[vertex][2] = -1
	}
	if vertex >= (size*size)-size {
		// bottom
		neighbors[vertex][6] = -1
		neighbors[vertex][7] = -1
		neighbors[vertex][8] = -1
	}
}

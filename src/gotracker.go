package main

import (
	"fmt"
	"container/vector"
	"rand"
	"strings"
	"strconv"
	"log"
	"math"
)

// Tracks a game of Go
// parent and rank comprise a union-find dataset to track chains
// vertices in the same set are part of the same chain.
// liberties returns the number of liberties for the chain
// it is only correct for the root of the set
type GoTracker struct {
	boardsize      int
	sqsize         int
	parent         []int
	rank           []int
	liberties      [][2]uint64
	board          []byte
	weights        *WeightTree
	atari          []map[int]int
	komi           float64
	koVertex       int
	koColor        byte
	played         []byte
	adj            [][]int
	mask           [][4]uint64
	neighbors      [][][]int
	passes         int
	winner         byte
	superko        bool
	moves          *vector.IntVector
	history        *vector.Vector
	config         *Config
}

// parent must be initialized so each element is a pointer to itself
// rank are initialized to zero
// board will be modified during use, should be a copy of the real board
func NewGoTracker(config *Config) (t *GoTracker) {
	t = new(GoTracker)

	t.boardsize = config.Size
	t.adj = go_adj[config.Size]
	t.mask = masks[config.Size]
	t.neighbors = go_neighbors[config.Size]
	t.sqsize = config.Size * config.Size

	t.parent = make([]int, t.sqsize)
	t.rank = make([]int, t.sqsize)
	t.liberties = make([][2]uint64, t.sqsize)
	t.board = make([]byte, t.sqsize)
	t.weights = NewWeightTree(t.sqsize)
	t.atari = make([]map[int]int, 3)
	t.atari[BLACK] = make(map[int]int)
	t.atari[WHITE] = make(map[int]int)
	t.played = make([]byte, t.sqsize)
	// initialize union-find data structure and move probabilities
	for i := 0; i < t.sqsize; i++ {
		t.parent[i] = i
		t.rank[i] = 1
		t.weights.Set(BLACK, i, INIT_WEIGHT)
		t.weights.Set(WHITE, i, INIT_WEIGHT)
		for j := 0; j < 4; j++ {
			adj := t.adj[i][j]
			if adj != -1 {
				t.liberties[i][0] |= t.mask[adj][0]
				t.liberties[i][1] |= t.mask[adj][1]
			}
		}
	}
	t.komi = config.Komi
	t.koVertex = -1
	t.koColor = EMPTY
	t.winner = EMPTY
	t.superko = true
	t.moves = new(vector.IntVector)
	t.history = new(vector.Vector)
	t.config = config
	return
}

func (t *GoTracker) Copy() Tracker {
	cp := new(GoTracker)

	cp.boardsize = t.boardsize
	cp.adj = t.adj
	cp.mask = t.mask
	cp.neighbors = t.neighbors
	cp.sqsize = t.sqsize
	cp.parent = make([]int, cp.sqsize)
	cp.rank = make([]int, cp.sqsize)
	cp.liberties = make([][2]uint64, cp.sqsize)
	cp.board = make([]byte, cp.sqsize)
	cp.weights = t.weights.Copy()
	cp.atari = make([]map[int]int, 3)
	cp.atari[BLACK] = make(map[int]int)
	cp.atari[WHITE] = make(map[int]int)
	copy(cp.parent, t.parent)
	copy(cp.rank, t.rank)
	copy(cp.liberties, t.liberties)
	copy(cp.board, t.board)
	for v, _ := range t.atari[BLACK] {
		cp.atari[BLACK][v] = t.atari[BLACK][v]
	}
	for v, _ := range t.atari[WHITE] {
		cp.atari[WHITE][v] = t.atari[WHITE][v]
	}

	cp.komi = t.komi
	cp.koVertex = t.koVertex
	cp.koColor = t.koColor
	cp.winner = t.winner

	cp.played = make([]byte, t.sqsize)

	cp.superko = true
	cp.moves = new(vector.IntVector)
	*cp.moves = t.moves.Copy()
	cp.history = new(vector.Vector)
	*cp.history = t.history.Copy()
	cp.config = t.config

	return cp
}

// apply color to vertex, modifying board and updating liberties of any go_adj strings
func (t *GoTracker) Play(color byte, vertex int) {
	if vertex != -1 {
		t.passes = 0
		
		if t.koVertex != -1 {
			t.weights.Set(t.koColor, t.koVertex, INIT_WEIGHT)
			t.koVertex = -1
			t.koColor = EMPTY
		}

		if t.superko {
			if t.history.Len() == 0 {
				t.history.Push(*NewHash(t.boardsize))
			}
			cp := t.Copy()
			cp.(*GoTracker).superko = false
			cp.Play(color, vertex)
			t.history.Push(*MakeHash(cp))
		}
		
		// modify the board
		t.board[vertex] = color
		
		// update parents and liberties of adjacent stones
		
		opp := Reverse(color)
		root := vertex
		for i := 0; i < 4; i++ {
			adj := t.adj[vertex][i]
			if adj != -1 && t.board[adj] == color {
				adj := find(adj, t.parent)
				// take adjacent chain out of atari (may be added back later)
				t.atari[color][adj] = 0, false
				// or in liberties to friendly chains
				new_root, old_root := union(root, adj, t.parent, t.rank)
				t.liberties[new_root][0] |= t.liberties[old_root][0]
				t.liberties[new_root][1] |= t.liberties[old_root][1]
				// xor out liberty from self
				t.liberties[new_root][0] &= ^t.mask[adj][0]
				t.liberties[new_root][1] &= ^t.mask[adj][1]
				root = new_root
			} else if adj != -1 && t.board[adj] == EMPTY {
				// xor out liberty from empty vertices
				t.liberties[adj][0] &= ^t.mask[vertex][0]
				t.liberties[adj][1] &= ^t.mask[vertex][1]
			} else if adj != -1 {
				// xor out liberties from enemy chains
				enemy := find(adj, t.parent)
				t.liberties[enemy][0] &= ^t.mask[vertex][0]
				t.liberties[enemy][1] &= ^t.mask[vertex][1]
			}
		}
		// xor out liberty from self
		t.liberties[root][0] &= ^t.mask[vertex][0]
		t.liberties[root][1] &= ^t.mask[vertex][1]
		
		// capture any adjacent enemies reduced to zero liberties
		var captured *vector.IntVector
		for i := 0; i < 4; i++ {
			adj := t.adj[vertex][i]
			if adj != -1  && t.board[adj] == opp {
				enemy := find(adj, t.parent)
				libs := t.libs(enemy)
				if libs == 0 {
					// take chain out of atari
					t.atari[opp][enemy] = 0, false
					if captured == nil {
						captured = t.capture(enemy)
					} else {
						captured.AppendVector(t.capture(enemy))
					}
				}
			}
		}
		
		// check for suicide of affected empty points
		for i := 0; i < 4; i++ {
			adj := t.adj[vertex][i]
			if adj != -1 && t.board[adj] == EMPTY && t.libs(adj) == 0 {
				t.check_suicide(adj)
			} else if adj != -1 && (t.board[adj] == BLACK || t.board[adj] == WHITE) {
				adj = find(adj, t.parent)
				if t.libs(adj) == 1 {
					last_liberty := t.lastliberty(adj)
					if t.libs(last_liberty) == 0 {
						t.check_suicide(last_liberty)
					}
				}
			}
		}
		
		// ko check
		if captured != nil && captured.Len() == 1 {
			capture := captured.At(0)
			t.check_suicide(capture)
			if t.libs(root) == 1 {
				t.koColor = opp
				t.koVertex = capture
			}
		}
		
		// check if capture took adjacent chains out of atari
		// if so, check suicide status of their previous last liberty
		for i := 0; captured != nil && i < captured.Len(); i++ {
			capture := captured.At(i)
			for j := 0; j < 4; j++ {
				adj := t.adj[capture][j]
				if adj != -1 && t.board[adj] == color {
					adj = find(adj, t.parent)
					if last_liberty, exists := t.atari[color][adj]; exists {
						t.check_suicide(last_liberty)
						t.atari[color][adj] = 0, false
					}
				}
			}
		}
		
		// cannot play on occupied vertex
		t.weights.Set(BLACK, vertex, 0)
		t.weights.Set(WHITE, vertex, 0)
		
		// update atari status of adjacent chains
		for i := 0; i < 4; i++ {
			adj := t.adj[vertex][i]
			if adj != -1  && (t.board[adj] == BLACK || t.board[adj] == WHITE) {
				adj = find(adj, t.parent)
				if t.libs(adj) == 1 {
					t.atari[t.board[adj]][adj] = t.lastliberty(adj)
				}
			}
		}
		
		// update atari status of current chain
		if t.libs(root) == 1 {
			t.atari[color][root] = t.lastliberty(root)
		}
		
		// apply patterns
		neighbors := t.neighbors[1][vertex]
		for i := range neighbors {
			if neighbors[i] != -1  && t.board[neighbors[i]] == EMPTY {
				t.updateWeights(neighbors[i])
			}
		}
		for i := 0; captured != nil && i < captured.Len(); i++ {
			t.updateWeights(captured.At(i))
		}

		// mark vertex as played for AMAF
		if t.played[vertex] == EMPTY {
			t.played[vertex] = color
		}
		
	} else {
		t.passes++
	}
	t.moves.Push(vertex)
}

// check if empty point is suicide
// vertex is assumed to be empty and have 0 liberties
func (t *GoTracker) check_suicide(vertex int) {
	// vertex is suicide for both colors unless they can either:
	//  connect to an adjacent friendly chain with > 1 liberty
	//  kill an adjacent enemy chain with 1 liberty
	suicide_black, suicide_white := true, true
	for i := 0; i < 4; i++ {
		adj := t.adj[vertex][i]
		if adj != -1 {
			adj = find(adj, t.parent)
			if t.libs(adj) > 1 {
				// can connect to adjacent friendly not in atari
				if t.board[adj] == BLACK {
					suicide_black = false
				} else {
					suicide_white = false
				}
			} else {
				// can kill adjacent enemy in atari
				if t.board[adj] == BLACK {
					suicide_white = false
				} else {
					suicide_black = false
				}
			}
		}
	}
	if suicide_black {
		t.weights.Set(BLACK, vertex, 0)
	} else if t.weights.Get(BLACK, vertex) == 0 {
		t.weights.Set(BLACK, vertex, INIT_WEIGHT)
		t.updateWeights(vertex)
	}
	if suicide_white {
		t.weights.Set(WHITE, vertex, 0)
	} else if t.weights.Get(WHITE, vertex) == 0 {
		t.weights.Set(WHITE, vertex, INIT_WEIGHT)
		t.updateWeights(vertex)
	}
}

// capture any points connected to vertex, resetting their parent, rank and weight, and liberties
// check if chains adjacent to captured are now out of atari
func (t *GoTracker) capture(vertex int) *vector.IntVector {
	// do a linear search for connected points
	captured := new(vector.IntVector)
	for i := 0; i < t.sqsize; i++ {
		if find(i, t.parent) == vertex {
			captured.Push(i)
		}
	}
	// reset
	for i := 0; i < captured.Len(); i++ {
		capture := captured.At(i)
		t.parent[capture] = capture
		t.rank[capture] = 1
		t.liberties[capture][0] = 0
		t.liberties[capture][1] = 0
		t.board[capture] = EMPTY
		t.weights.Set(BLACK, capture, INIT_WEIGHT)
		t.weights.Set(WHITE, capture, INIT_WEIGHT)
	}
	// update liberties
	for i := 0; i < captured.Len(); i++ {
		capture := captured.At(i)
		for j := 0; j < 4; j++ {
			adj := t.adj[capture][j]
			if adj != -1 {
				root := find(adj, t.parent)
				t.liberties[root][0] |= t.mask[capture][0]
				t.liberties[root][1] |= t.mask[capture][1]
			}
		}
	}
	return captured
}

// playout simulated game, call Winner() to retrive winner based on final territory
func (t *GoTracker) Playout(color byte) {
	move := 0
	t.superko = false
	for {
		vertex := t.playHeuristicMove(color)
		if vertex == -1 {
			vertex = t.weights.Rand(color)
		}
		if t.config.VeryVerbose {
			log.Println(Ctoa(color)+t.Vtoa(vertex))
		}
		t.Play(color, vertex)
		if t.config.VeryVerbose {
			log.Println(t.String())
			log.Println(t.weightboard(BLACK))
			log.Println(t.weightboard(WHITE))
		}
		if t.config.Verify {
			t.Verify()
		}
		move++
		if move > 2*t.sqsize || t.Winner() != EMPTY {
			break
		}
		color = Reverse(color)
	}
	if t.config.VeryVerbose {
		log.Println("FINAL")
		log.Println(t.String())
		log.Println("winner: ", Ctoa(t.Winner()))
	}
	if t.config.Verify {
		t.checkNoMoreLegal()
	}
	t.superko = true
}

func (t *GoTracker) updateWeights(vertex int) {
	if t.board[vertex] != EMPTY {
		for i := 0; i < 4; i++ {
			adj := t.adj[vertex][i]
			if adj != -1 && t.board[adj] == EMPTY {
				t.updateWeights(adj)
			}
		}
	} else {
		black_weight := t.weights.Get(BLACK, vertex)
		if black_weight != 0 {
			weight := t.get_weight(BLACK, vertex)
			if math.IsNaN(weight) {
				black_weight = 0
			} else if black_weight + weight > 0 {
				black_weight += weight
			}
		}
		t.weights.Set(BLACK, vertex, black_weight)
		white_weight := t.weights.Get(WHITE, vertex)
		if white_weight != 0 {
			weight := t.get_weight(WHITE, vertex)
			if math.IsNaN(weight) {
				white_weight = 0
			} else if white_weight + weight > 0 {
				white_weight += weight
			}
		}
		t.weights.Set(WHITE, vertex, white_weight)
	}
}

func (t *GoTracker) playHeuristicMove(color byte) int {
	if len(t.atari[color]) > 0 {
		// play a random saving move
		saves := new(vector.IntVector)
		for _, last_liberty := range t.atari[color] {
			if t.weights.Get(color, last_liberty) > 0 {
				saves.Push(last_liberty)
			}
		}
		if saves.Len() > 0 {
			return saves.At(rand.Intn(saves.Len()))
		}
	}
	if len(t.atari[Reverse(color)]) > 0 {
		// play a random capture move
		captures := new(vector.IntVector)
		for _, last_liberty := range t.atari[Reverse(color)] {
			captures.Push(last_liberty)
		}
		return captures.At(rand.Intn(captures.Len()))
	}
	return -1
}

func (t *GoTracker) WasPlayed(color byte, vertex int) bool {
	if vertex == -1 {
		return false
	}
	return t.played[vertex] == color
}

// return true iff move is legal, without modifying any state
func (t *GoTracker) Legal(color byte, vertex int) bool {
	if t.superko {
		cp := t.Copy()
		cp.(*GoTracker).superko = false
		cp.Play(color, vertex)
		pos := *MakeHash(cp)
		for i := 0; i < t.history.Len(); i++ {
			if pos == t.history.At(i).(Hash) {
				return false
			}
		}
	}
	return vertex == -1 || t.weights.Get(color, vertex) != 0
}

func (t *GoTracker) Score(komi float64) (float64, float64) {
	bc, wc := 0.0, 0.0
	for i := 0; i < t.sqsize; i++ {
		if t.board[i] == BLACK {
			bc++
		} else if t.board[i] == WHITE {
			wc++
		}
	}
	wc += komi
	return bc, wc
}

func (t *GoTracker) Winner() byte {
	if t.passes < 2 {
		return EMPTY
	}
	if t.winner != EMPTY {
		return t.winner
	}
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

func (t *GoTracker) Territory(color byte) []float64 {
	territory := make([]float64, t.sqsize)
	for i := range t.board {
		if t.board[i] == color {
			territory[i] = 1
		}
	}
	return territory
}

func (t *GoTracker) Verify() {
	for i := 0; i < t.sqsize; i++ {
		if t.board[i] == EMPTY {
			libs := t.libs(i)
			if libs == 0 {
				suicide := make(map[byte]bool)
				suicide[BLACK], suicide[WHITE] = true, true
				for j := 0; j < 4; j++ {
					adj := t.adj[i][j]
					if adj != -1 {
						adj = find(adj, t.parent)
						if t.libs(adj) > 1 {
							// can connect to adjacent friendly not in atari
							suicide[t.board[adj]] = false
						} else {
							// can kill adjacent enemy in atari
							suicide[Reverse(t.board[adj])] = false
						}
					}
				}
				if t.Legal(BLACK, i) && suicide[BLACK] {
					log.Println(t.Vtoa(i), "black: legal", t.Legal(BLACK, i), "suicide", suicide[BLACK])
				}
				if  t.Legal(WHITE, i) && suicide[WHITE] {
					log.Println(t.Vtoa(i), "white: legal", t.Legal(WHITE, i), "suicide", suicide[WHITE])
				}
				if (t.Legal(BLACK, i) && suicide[BLACK]) || (t.Legal(WHITE, i) && suicide[WHITE]) {
					log.Println(t.String())
					panic("suicide incorrect")
				}
			}
		} else {
			root := find(i, t.parent)
			libs := t.libs(root)
			if libs == 0 {
				panic("zero liberties")
			} else if libs == 1 {
				if _, exists := t.atari[t.board[root]][root]; !exists {
					log.Println(t.Vtoa(root), "should be in atari")
					panic("not marked as atari")
				}
			}
		}
	}
}

func (t *GoTracker) checkNoMoreLegal() {
	if t.winner == EMPTY {
		return
	}
	for i := 0; i < t.sqsize; i++ {
		if t.board[i] == EMPTY {
			legal_black := t.weights.Get(BLACK, i) != 0
			legal_white := t.weights.Get(WHITE, i) != 0
			if legal_black || legal_white {
				panic(t.Vtoa(i))
			}
			for j := 0; j < 4; j++ {
				if t.adj[i][j] != -1 {
					adj := find(t.adj[i][j], t.parent)
					if t.libs(adj) == 1 {
						panic(t.Vtoa(i))
					}
				}
			}
			if (!legal_black && t.weights.Get(BLACK, i) != 0) || (!legal_white && t.weights.Get(WHITE, i) != 0) {
				panic(t.Vtoa(i))
			}
		}
	}
}

func (t *GoTracker) Moves() *vector.IntVector {
	return t.moves
}

func (t *GoTracker) Vtoa(v int) string {
	if v == -1 {
		return "PASS"
	}
	alpha, num := v%t.boardsize, v/t.boardsize
	num = t.boardsize - num
	alpha = alpha + 'A'
	if alpha >= 'I' {
		alpha++
	}
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
	if col >= 'I' {
		col--
	}
	if err != nil {
		panic("Failed to convert string to vertex")
	}
	return row*t.boardsize + int(col-'A')
}

func (t *GoTracker) String() (s string) {
	s += "  "
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
		s += fmt.Sprintf("%d ", t.boardsize-row)
		for col := 0; col < t.boardsize; col++ {
			v := row*t.boardsize + col
			s += Ctoa(t.board[v])
			if col != t.boardsize-1 {
				s += " "
			}
		}
		s += fmt.Sprintf(" %d", t.boardsize-row)
		if row != t.boardsize-1 {
			s += "\n"
		}
	}
	s += "\n  "
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

func (t *GoTracker) dead() []int {
	dead := new(vector.IntVector)
	cp := t.Copy().(*GoTracker)
	color := BLACK
	move := 0
	for {
		vertex := cp.weights.Rand(color)
		cp.Play(color, vertex)
		move++
		if move > 3*t.sqsize || cp.Winner() != EMPTY {
			break
		}
		color = Reverse(color)
	}
	for i := 0; i < t.sqsize; i++ {
		if t.board[i] != EMPTY && cp.board[i] != t.board[i] {
			dead.Push(i)
		}
	}
	stones := make([]int, dead.Len())
	for i := 0; i < dead.Len(); i++ {
		stones[i] = dead.At(i)
	}
	return stones
}

// assuming vertex is empty, return new weight for black to play at vertex
// this weight will be added to the old weight (or subtracted, for negative weights),
// and floored at 1
// returning NaN will immediately set the probability of playing at vertex to zero
func (t *GoTracker) get_weight(color byte, vertex int) float64 {
	hash := go_min_hash[go_hash(color, t.board, t.neighbors[1][vertex])]
	weight, exists := go_expert_policy_weights[hash]
	if !exists && t.config.policy_weights != nil {
		return t.config.policy_weights.Get(hash)
	}
	return weight
}

func (t *GoTracker) libs(vertex int) uint {
	return bitcount(t.liberties[vertex][0], t.liberties[vertex][1])
}

func bitcount(u uint64, v uint64) (c uint) {
	// c accumulates the total bits set in v
	for c = 0; u != 0; c++ {
		u &= u - 1 // clear the least significant bit set
	}
	for ; v != 0; c++ {
		v &= v - 1 // clear the least significant bit set
	}
	return
}

// return the index of the first liberty
func (t *GoTracker) lastliberty(root int) int {
	v0, v1 := t.liberties[root][0], t.liberties[root][1]
	for row := 0; row < t.boardsize; row++ {
		for col := 0; col < t.boardsize; col++ {
			vertex := row * t.boardsize + col
			var v uint64
			var bit uint64
			if vertex < 64 {
				bit = uint64(64 - vertex - 1)
				v = v0
			} else {
				bit = uint64(64 - (vertex - 64) - 1)
				v = v1
			}
			var mask uint64 = 1 << bit
			if (mask & v) != 0 {
				return vertex
			}
		}
	}
	return -1
}

func (t *GoTracker) libertyboard(root int) (s string) {
	v0, v1 := t.liberties[root][0], t.liberties[root][1]
	s += "  "
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha = alpha + 1
		}
		s += string(alpha)
		if col != t.boardsize-1 {
			s += " "
		}
	}
	s += "\n"
	for row := 0; row < t.boardsize; row++ {
		s += fmt.Sprintf("%d ", t.boardsize-row)
		for col := 0; col < t.boardsize; col++ {
			vertex := row * t.boardsize + col
			var v uint64
			var bit uint64
			if vertex < 64 {
				bit = uint64(64 - vertex - 1)
				v = v0
			} else {
				bit = uint64(64 - (vertex - 64) - 1)
				v = v1
			}
			var mask uint64 = 1 << bit
			if (mask & v) == 0 {
				s += "0 "
			} else {
				s += "1 "
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
		if alpha >= 'I' {
			alpha = alpha + 1
		}
		s += string(alpha)
		if col != t.boardsize - 1 {
			s += " "
		}
	}
	s += "\n"
	return
}

func (t *GoTracker) maskboard(root int) (s string) {
	v0, v1 := t.mask[root][0], t.mask[root][1]
	s += "  "
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha = alpha + 1
		}
		s += string(alpha)
		if col != t.boardsize-1 {
			s += " "
		}
	}
	s += "\n"
	for row := 0; row < t.boardsize; row++ {
		s += fmt.Sprintf("%d ", t.boardsize-row)
		for col := 0; col < t.boardsize; col++ {
			vertex := row * t.boardsize + col
			var v uint64
			var bit uint64
			if vertex < 64 {
				bit = uint64(64 - vertex - 1)
				v = v0
			} else {
				bit = uint64(64 - (vertex - 64) - 1)
				v = v1
			}
			var mask uint64 = 1 << bit
			if (mask & v) == 0 {
				s += "0 "
			} else {
				s += "1 "
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
		if alpha >= 'I' {
			alpha = alpha + 1
		}
		s += string(alpha)
		if col != t.boardsize - 1 {
			s += " "
		}
	}
	s += "\n"
	return
}

func (t *GoTracker) libertycountboard() (s string) {
	s += "  "
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha = alpha + 1
		}
		s += string(alpha)
		if col != t.boardsize-1 {
			s += " "
		}
	}
	s += "\n"
	for row := 0; row < t.boardsize; row++ {
		s += fmt.Sprintf("%d ", t.boardsize-row)
		for col := 0; col < t.boardsize; col++ {
			vertex := row * t.boardsize + col
			s += fmt.Sprintf("%d ", t.libs(find(vertex, t.parent)))
		}
		s += fmt.Sprintf(" %d", t.boardsize - row)
		if row != t.boardsize - 1 {
			s += "\n"
		}
	}
	s += "\n  "
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha = alpha + 1
		}
		s += string(alpha)
		if col != t.boardsize - 1 {
			s += " "
		}
	}
	s += "\n"
	return
}

func (t *GoTracker) parentboard() (s string) {
	for row := 0; row < t.boardsize; row++ {
		for col := 0; col < t.boardsize; col++ {
			vertex := row*(t.boardsize) + col
			if t.board[vertex] == EMPTY {
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

func (t *GoTracker) weightboard(color byte) (s string) {
	s += "  "
	for col := 0; col < t.boardsize; col++ {
		alpha := col + 'A'
		if alpha >= 'I' {
			alpha++
		}
		s += " " + string(alpha) + " "
		if col != t.boardsize-1 {
			s += " "
		}
	}
	s += "\n"
	for row := 0; row < t.boardsize; row++ {
		s += fmt.Sprintf("%d ", t.boardsize-row)
		for col := 0; col < t.boardsize; col++ {
			v := row*t.boardsize + col
			s += fmt.Sprintf("%3.d", t.weights.Get(color, v))
			if col != t.boardsize-1 {
				s += " "
			}
		}
		s += fmt.Sprintf(" %d", t.boardsize-row)
		if row != t.boardsize-1 {
			s += "\n"
		}
	}
	s += "\n  "
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
var go_expert_policy_weights map[uint32]float64
var go_min_hash map[uint32]uint32
var go_hash_mask [9][4]uint32

func init() {
	go_adj = make(map[int][][]int)
	go_neighbors = make(map[int][][][]int)
	masks = make(map[int][][4]uint64)
	for boardsize := 4; boardsize <= 19; boardsize++ {
		setup_go(boardsize)
	}
	go_hash_mask = [9][4]uint32{
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
			0x0000000D,
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
			0x000000D0,
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
			0x00000D00,
		},
		[4]uint32{
			0x00000000,
			0x00001000,
			0x00002000,
			0x00003000,
		},
		[4]uint32{
			0x00000000,
			0x00004000,
			0x00008000,
			0x0000D000,
		},
		[4]uint32{
			0x00000000,
			0x00010000,
			0x00020000,
			0x00030000,
		},
	}
	setup_go_min_hash()
	setup_go_expert_policy_weights()
}

func setup_go(size int) {
	masks[size] = make([][4]uint64, size*size)
	for i := 0; i < len(masks[size]); i++ {
		var m uint64 = 1
		if i < 64 {
			masks[size][i][0] = m << uint64(64-i-1)
		} else {
			masks[size][i][1] = m << uint64(64-(i-64)-1)
		}
		masks[size][i][2] = masks[size][i][0] ^ 0xFFFFFFFFFFFFFFFF
		masks[size][i][3] = masks[size][i][1] ^ 0xFFFFFFFFFFFFFFFF
	}
	setup_go_adj(size)
	setup_go_neighbors(size)
}

func setup_go_adj(size int) {
	go_adj[size] = make([][]int, size*size)
	for vertex, _ := range go_adj[size] {
		go_adj[size][vertex] = make([]int, 4)
		set_go_adj(vertex, size)
	}
}

func set_go_adj(vertex int, size int) {
	row := vertex / size
	col := vertex % size
	up_row := row - 1
	down_row := row + 1
	left_col := col - 1
	right_col := col + 1
	up := up_row*size + col
	down := down_row*size + col
	left := row*size + left_col
	right := row*size + right_col
	go_adj[size][vertex][UP] = -1
	go_adj[size][vertex][DOWN] = -1
	go_adj[size][vertex][LEFT] = -1
	go_adj[size][vertex][RIGHT] = -1
	if up_row >= 0 && up_row < size {
		go_adj[size][vertex][UP] = up
	}
	if down_row >= 0 && down_row < size {
		go_adj[size][vertex][DOWN] = down
	}
	if left_col >= 0 && left_col < size {
		go_adj[size][vertex][LEFT] = left
	}
	if right_col >= 0 && right_col < size {
		go_adj[size][vertex][RIGHT] = right
	}
}

func setup_go_neighbors(size int) {
	go_neighbors[size] = make([][][]int, 3)
	go_neighbors[size][0] = make([][]int, size*size)
	go_neighbors[size][1] = make([][]int, size*size)
	for vertex := 0; vertex < size*size; vertex++ {
		v2 := vertex + 1
		v3 := vertex + size
		v4 := vertex + size + 1
		if (vertex+1) % size == 0 {
			v2 = -1
			v4 = -1
		}
		if vertex >= (size*size)-size {
			v3 = -1
			v4 = -1
		}
		go_neighbors[size][0][vertex] = []int{vertex, v2, v3, v4}
		set_go_neighbors(size, vertex)
	}
}

func set_go_neighbors(size int, vertex int) {
	neighbors := go_neighbors[size][1]
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
	if vertex % size == 0 {
		// left
		neighbors[vertex][0] = -1
		neighbors[vertex][3] = -1
		neighbors[vertex][6] = -1
	}
	if (vertex+1) % size == 0 {
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
	if vertex >= (size * size) - size {
		// bottom
		neighbors[vertex][6] = -1
		neighbors[vertex][7] = -1
		neighbors[vertex][8] = -1
	}
}

func setup_go_min_hash() {
	go_min_hash = make(map[uint32]uint32)
	symmetries := [][]int{
		[]int{0, 1, 2, 3, 4, 5, 6, 7, 8},
		[]int{0, 3, 6, 1, 4, 7, 2, 5, 8},
		[]int{2, 1, 0, 5, 4, 3, 8, 7, 6},
		[]int{6, 3, 0, 7, 4, 1, 8, 5, 2},
		[]int{8, 7, 6, 5, 4, 3, 2, 1, 0},
		[]int{8, 5, 2, 7, 4, 1, 6, 3, 0},
		[]int{6, 7, 8, 3, 4, 5, 0, 1, 2},
		[]int{2, 5, 8, 1, 4, 7, 0, 3, 6},
	}
	board := []byte{EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY}
	var a, b, c, d, e, f, g, h, i byte
	for a = 0; a <= 3; a++ {
		board[0] = a
		for b = 0; b <= 3; b++ {
			board[1] = b
			for c = 0; c <= 3; c++ {
				board[2] = c
				for d = 0; d <= 3; d++ {
					board[3] = d
					for e = 0; e <= 3; e++ {
						board[4] = e
						for f = 0; f <= 3; f++ {
							board[5] = f
							for g = 0; g <= 3; g++ {
								board[6] = g
								for h = 0; h <= 3; h++ {
									board[7] = h
									for i = 0; i <= 3; i++ {
										board[8] = i
										black_min_hash := ^uint32(0)
										white_min_hash := ^uint32(0)
										for x := range symmetries {
											black_hash := go_hash(BLACK, board, symmetries[x])
											if black_hash < black_min_hash {
												black_min_hash = black_hash
											}
											white_hash := go_hash(WHITE, board, symmetries[x])
											if white_hash < white_min_hash {
												white_min_hash = white_hash
											}
										}
										for x := range symmetries {
											black_hash := go_hash(BLACK, board, symmetries[x])
											white_hash := go_hash(WHITE, board, symmetries[x])
											go_min_hash[black_hash] = black_min_hash
											go_min_hash[white_hash] = white_min_hash
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func setup_go_expert_policy_weights() {
	go_expert_policy_weights = make(map[uint32]float64)
	neighbors := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	board := []byte{
		EMPTY, BLACK, BLACK,
		BLACK, EMPTY, BLACK,
		BLACK, BLACK, BLACK,
	}
	go_expert_policy_weights[go_min_hash[go_hash(BLACK, board, neighbors)]] = math.NaN()
	board = []byte{
		WHITE, BLACK, BLACK,
		BLACK, EMPTY, BLACK,
		BLACK, BLACK, BLACK,
	}
	go_expert_policy_weights[go_min_hash[go_hash(BLACK, board, neighbors)]] = math.NaN()
	board = []byte{
		BLACK, BLACK, BLACK,
		BLACK, EMPTY, BLACK,
		BLACK, BLACK, BLACK,
	}
	go_expert_policy_weights[go_min_hash[go_hash(BLACK, board, neighbors)]] = math.NaN()
	board = []byte{
		ILLEGAL, BLACK, BLACK,
		ILLEGAL, EMPTY, BLACK,
		ILLEGAL, BLACK, BLACK,
	}
	go_expert_policy_weights[go_min_hash[go_hash(BLACK, board, neighbors)]] = math.NaN()
	board = []byte{
		ILLEGAL, ILLEGAL, ILLEGAL,
		ILLEGAL, EMPTY, BLACK,
		ILLEGAL, BLACK, BLACK,
	}
	go_expert_policy_weights[go_min_hash[go_hash(BLACK, board, neighbors)]] = math.NaN()
	board = []byte{
		EMPTY, WHITE, WHITE,
		WHITE, EMPTY, WHITE,
		WHITE, WHITE, WHITE,
	}
	go_expert_policy_weights[go_min_hash[go_hash(WHITE, board, neighbors)]] = math.NaN()
	board = []byte{
		BLACK, WHITE, WHITE,
		WHITE, EMPTY, WHITE,
		WHITE, WHITE, WHITE,
	}
	go_expert_policy_weights[go_min_hash[go_hash(WHITE, board, neighbors)]] = math.NaN()
	board = []byte{
		WHITE, WHITE, WHITE,
		WHITE, EMPTY, WHITE,
		WHITE, WHITE, WHITE,
	}
	go_expert_policy_weights[go_min_hash[go_hash(WHITE, board, neighbors)]] = math.NaN()
	board = []byte{
		ILLEGAL, WHITE, WHITE,
		ILLEGAL, EMPTY, WHITE,
		ILLEGAL, WHITE, WHITE,
	}
	go_expert_policy_weights[go_min_hash[go_hash(WHITE, board, neighbors)]] = math.NaN()
	board = []byte{
		ILLEGAL, ILLEGAL, ILLEGAL,
		ILLEGAL, EMPTY, WHITE,
		ILLEGAL, WHITE, WHITE,
	}
	go_expert_policy_weights[go_min_hash[go_hash(WHITE, board, neighbors)]] = math.NaN()
}

func go_hash(color byte, board []byte, neighbors []int) uint32 {
	var hash uint32
	if color == WHITE {
		hash = 1<<31
	} else {
		hash = 0
	}
	for j := range neighbors {
		neighbor := neighbors[j]
		if neighbor == -1 {
			hash |= go_hash_mask[j][ILLEGAL]
		} else {
			hash |= go_hash_mask[j][board[neighbor]]
		}
	}
	return hash
}

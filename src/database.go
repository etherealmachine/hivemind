package main

import (
	"os"
	"gob"
	"fmt"
)

type Position struct {
	wins   float64
	visits float64
}

var hashMap map[Hash]*Position

func init() {
	hashMap = make(map[Hash]*Position)
	f, err := os.Open("database.gob")
	if err == nil {
		dec := gob.NewDecoder(f)
		dec.Decode(&emptyBoard)
		dec.Decode(&blackBoard)
		dec.Decode(&whiteBoard)
		dec.Decode(&hashMap)
		f.Close()
		fmt.Fprintln(os.Stderr, len(hashMap), "positions loaded into hashmap")
	}
}

func save(t Tracker, root *Node) {
	if !*book {
		return
	}

	var oldBest *Node
	if root.Best() != nil {
		oldBest = root.Best()
	}

	doSave(t, root)

	if root.Best() != nil {
		if root.Best() != oldBest {
			fmt.Fprintf(os.Stderr,
				"%s\nswitched from %s to %s\n",
				Bwboard(t.Board(), t.Boardsize(), true),
				Vtoa(oldBest.vertex, t.Boardsize()),
				Vtoa(root.Best().vertex, t.Boardsize()))
		}
	}
}

func doSave(t Tracker, node *Node) {
	for child := node.child; child != nil; child = child.sibling {
		cp := t.Copy()
		cp.Play(child.color, child.vertex)
		hash := hash(cp)
		newWins := child.wins
		newVisits := child.visits
		if pos, ok := hashMap[*hash]; ok {
			newWins += pos.wins
			newVisits += pos.visits
		} else {
			hashMap[*hash] = new(Position)
		}
		pos := hashMap[*hash]
		pos.wins = newWins
		pos.visits = newVisits
		child.wins = newWins
		child.visits = newVisits
		doSave(cp, child)
	}
}

func hash(t Tracker) *Hash {
	var hashes []*Hash
	size := t.Boardsize()
	if *hex {
		hashes = make([]*Hash, 2)
		hashes[0] = NewHash(size, false)
		hashes[1] = NewHash(size, false)
	} else {
		hashes = make([]*Hash, 4)
		hashes[0] = NewHash(size, false)
		hashes[1] = NewHash(size, false)
		hashes[2] = NewHash(size, false)
		hashes[3] = NewHash(size, false)
	}
	board := t.Board()
	for i := 0; i < len(board); i++ {
		for j := 0; j < len(hashes); j++ {
			switch board[i] {
			case BLACK:
				*hashes[j] ^= blackBoard[symmetricBoard[j][i]]
			case WHITE:
				*hashes[j] ^= whiteBoard[symmetricBoard[j][i]]
			case EMPTY:
				*hashes[j] ^= emptyBoard[symmetricBoard[j][i]]
			}
		}
	}
	min := hashes[0]
	for i := 0; i < len(hashes); i++ {
		if *hashes[i] < *min {
			min = hashes[i]
		}
	}
	return min
}

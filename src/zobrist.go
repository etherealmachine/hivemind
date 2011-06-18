package main

import (
	"rand"
)

// Zobrist hashing
var emptyBoard map[int][]Hash
var blackBoard map[int][]Hash
var whiteBoard map[int][]Hash

func init() {
	emptyBoard = make(map[int][]Hash)
	blackBoard = make(map[int][]Hash)
	whiteBoard = make(map[int][]Hash)
	for size := 2; size <= 19; size++ {
		setupHash(size)
	}
}

func setupHash(size int) {
	emptyBoard[size] = make([]Hash, size*size)
	blackBoard[size] = make([]Hash, size*size)
	whiteBoard[size] = make([]Hash, size*size)
	rand.Seed(int64(size))
	for i := 0; i < size*size; i++ {
		emptyBoard[size][i] = Hash(rand.Uint32())
		blackBoard[size][i] = Hash(rand.Uint32())
		whiteBoard[size][i] = Hash(rand.Uint32())
	}
}

type Hash uint32

func NewHash(size int) (hash *Hash) {
	hash = new(Hash)
	for i := 0; i < len(emptyBoard[size]); i++ {
		*hash ^= emptyBoard[size][i]
	}
	return
}

func MakeHash(t Tracker) *Hash {
	hash := NewHash(t.Boardsize())
	board := t.Board()
	for i := 0; i < t.Sqsize(); i++ {
		hash.Update(t.Boardsize(), EMPTY, board[i], i)
	}
	return hash
}

func (hash *Hash) Update(size int, oldColor byte, newColor byte, vertex int) {
	switch oldColor {
	case BLACK:
		*hash ^= blackBoard[size][vertex]
	case WHITE:
		*hash ^= whiteBoard[size][vertex]
	case EMPTY:
		*hash ^= emptyBoard[size][vertex]
	}
	switch newColor {
	case BLACK:
		*hash ^= blackBoard[size][vertex]
	case WHITE:
		*hash ^= whiteBoard[size][vertex]
	case EMPTY:
		*hash ^= emptyBoard[size][vertex]
	}
}

func (hash *Hash) Copy() (cp *Hash) {
	cp = new(Hash)
	*cp = *hash
	return
}

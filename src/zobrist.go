package main

import (
	"rand"
)

// Zobrist hashing
var emptyBoard map[int][]Hash
var blackBoard map[int][]Hash
var whiteBoard map[int][]Hash
//var symmetricBoard [4][]int

func init() {
	emptyBoard = make(map[int][]Hash)
	blackBoard = make(map[int][]Hash)
	whiteBoard = make(map[int][]Hash)
	for size := 3; size < 19; size++ {
		setupHash(size)
	}
}

func setupHash(size int) {
	emptyBoard[size] = make([]Hash, size * size)
	blackBoard[size] = make([]Hash, size * size)
	whiteBoard[size] = make([]Hash, size * size)
	rand.Seed(int64(size))
	for i := 0; i < size * size; i++ {
		emptyBoard[size][i] = Hash(rand.Uint32())
		blackBoard[size][i] = Hash(rand.Uint32())
		whiteBoard[size][i] = Hash(rand.Uint32())
	}
	
	/*
	if *hex {
		symmetricBoard[0] = make([]int, size * size)
		symmetricBoard[1] = make([]int, size * size)
		for i := 0; i < size * size; i++ {
			symmetricBoard[0][i] = i
			symmetricBoard[1][i] = ((size * size) - 1) - i
		}
	} else {
		symmetricBoard[0] = make([]int, size * size)
		symmetricBoard[1] = make([]int, size * size)
		symmetricBoard[2] = make([]int, size * size)
		symmetricBoard[3] = make([]int, size * size)
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				i := y * size + x
				symmetricBoard[0][i] = i;
				symmetricBoard[1][i] = (size - 1 - x) * size + y
				symmetricBoard[2][i] = (size - 1 - y) * size + (size - 1 - x)
				symmetricBoard[3][i] = x * size + (size - 1 - y)
			}
		}
	}
	*/
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
	for i := 0; i < len(board); i++ {
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

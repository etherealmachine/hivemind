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
	for Size := 3; Size <= 19; Size++ {
		setupHash(Size)
	}
}

func setupHash(Size int) {
	emptyBoard[Size] = make([]Hash, Size*Size)
	blackBoard[Size] = make([]Hash, Size*Size)
	whiteBoard[Size] = make([]Hash, Size*Size)
	rand.Seed(int64(Size))
	for i := 0; i < Size*Size; i++ {
		emptyBoard[Size][i] = Hash(rand.Uint32())
		blackBoard[Size][i] = Hash(rand.Uint32())
		whiteBoard[Size][i] = Hash(rand.Uint32())
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

func NewHash(Size int) (hash *Hash) {
	hash = new(Hash)
	for i := 0; i < len(emptyBoard[Size]); i++ {
		*hash ^= emptyBoard[Size][i]
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

func (hash *Hash) Update(Size int, oldColor byte, newColor byte, vertex int) {
	switch oldColor {
	case BLACK:
		*hash ^= blackBoard[Size][vertex]
	case WHITE:
		*hash ^= whiteBoard[Size][vertex]
	case EMPTY:
		*hash ^= emptyBoard[Size][vertex]
	}
	switch newColor {
	case BLACK:
		*hash ^= blackBoard[Size][vertex]
	case WHITE:
		*hash ^= whiteBoard[Size][vertex]
	case EMPTY:
		*hash ^= emptyBoard[Size][vertex]
	}
}

func (hash *Hash) Copy() (cp *Hash) {
	cp = new(Hash)
	*cp = *hash
	return
}

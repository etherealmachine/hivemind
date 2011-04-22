package main

import (
	"rand"
	"time"
)

// Zobrist hashing
var hashSize int
var emptyBoard []Hash
var blackBoard []Hash
var whiteBoard []Hash
var symmetricBoard [4][]int

func setupHash(size int) {
	hashSize = size
	emptyBoard = make([]Hash, size * size)
	blackBoard = make([]Hash, size * size)
	whiteBoard = make([]Hash, size * size)
	rand.Seed(time.Nanoseconds())
	for i := 0; i < size * size; i++ {
		emptyBoard[i] = Hash(rand.Uint32())
		blackBoard[i] = Hash(rand.Uint32())
		whiteBoard[i] = Hash(rand.Uint32())
	}
	
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
}

type Hash uint32

func NewHash(size int, init bool) (hash *Hash) {
	if size != hashSize {
		setupHash(size)
	}
	hash = new(Hash)
	if init {
		for i := 0; i < len(emptyBoard); i++ {
			*hash ^= emptyBoard[i]
		}
	}
	return
}

func (hash *Hash) Update(oldColor byte, newColor byte, vertex int) {
	switch oldColor {
		case BLACK:
			*hash ^= blackBoard[vertex]
		case WHITE:
			*hash ^= whiteBoard[vertex]
		case EMPTY:
			*hash ^= emptyBoard[vertex]
	}
	switch newColor {
		case BLACK:
			*hash ^= blackBoard[vertex]
		case WHITE:
			*hash ^= whiteBoard[vertex]
		case EMPTY:
			*hash ^= emptyBoard[vertex]
	}
}

func (hash *Hash) Copy() (cp *Hash) {
	cp = new(Hash)
	*cp = *hash
	return
}

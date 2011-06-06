package main

import (
	"math"
	"gob"
	"os"
	"log"
) 

type Book struct {
	Wins map[Hash] float64
	Visits map[Hash] float64
}

func NewBook() *Book {
	book := new(Book)
	book.Wins = make(map[Hash] float64)
	book.Visits = make(map[Hash] float64)
	f, err := os.Open("book.gob")
	if err != nil {
		log.Println(err)
	} else {
		defer func() { f.Close() }()
		d := gob.NewDecoder(f)
		err = d.Decode(book)
		if err != nil {
			panic(err)
		}
	}
	return book
}

func (book *Book) Load(color byte, t Tracker) int {
	maxValue := math.Inf(-1)
	maxVertex := -1
	for i := 0; i < t.Sqsize(); i++ {
		cp := t.Copy()
		if cp.Legal(color, i) {
			cp.Play(color, i)
			position := *MakeHash(cp)
			wins := book.Wins[position]
			visits := book.Visits[position]
			mean := wins / visits
			if visits > 0 && mean > maxValue {
				maxValue = mean
				maxVertex = i
			}
		}
	}
	return maxVertex
}

func (book *Book) Save(root *Node, t Tracker) {
	for child := root.child; child != nil; child = child.sibling {
		cp := t.Copy()
		cp.Play(child.color, child.vertex)
		position := *MakeHash(cp)
		book.Wins[position] += child.wins
		book.Visits[position] += child.visits
	}
	f, err := os.Create("book.gob")
	if err != nil {
		panic(err)
	}
	defer func() { f.Close() }()
	e := gob.NewEncoder(f)
	err = e.Encode(book)
	if err != nil {
		panic(err)
	}
}

func MakeBook(config *Config) {
	color := BLACK
	t := NewTracker(config)
	book := NewBook()
	root := NewRoot(color, t, config)
	cp := t.Copy()
	genmove(root, cp, nil, nil)
	book.Save(root, t)
	for i := 0; i < t.Sqsize(); i++ {
		cp := t.Copy()
		cp.Play(color, i)
		root = NewRoot(color, t, config)
		genmove(root, cp, nil, nil)
		book.Save(root, t)
	}
}

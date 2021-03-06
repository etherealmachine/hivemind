package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	EMPTY       = byte(0)
	BLACK       = byte(1)
	WHITE       = byte(2)
	ILLEGAL     = byte(3)
	BOTH        = byte(4)
	LEGAL_BLACK = byte(5)
	LEGAL_WHITE = byte(6)
	LEGAL_BOTH  = byte(7)
)

// convert a string to a color
func Atoc(s string) (c byte) {
	switch strings.ToUpper(s) {
	case ".":
		c = EMPTY
	case "X":
		c = ILLEGAL
	case "B":
		c = BLACK
	case "W":
		c = WHITE
	case "BLACK":
		c = BLACK
	case "WHITE":
		c = WHITE
	}
	return
}

// convert a color to a string
func Ctoa(c byte) (s string) {
	switch c {
	case BLACK:
		s = "B"
	case WHITE:
		s = "W"
	case EMPTY:
		s = "."
	case ILLEGAL:
		s = "X"
	case LEGAL_BLACK:
		s = "+"
	case LEGAL_WHITE:
		s = "-"
	case LEGAL_BOTH:
		s = "@"
	}
	return
}

func Reverse(c byte) byte {
	switch c {
	case BLACK:
		return WHITE
	case WHITE:
		return BLACK
	case EMPTY:
		return EMPTY
	case ILLEGAL:
		return ILLEGAL
	}
	panic("err")
}

func VisitsBoard(root *Node, t Tracker) (s string) {
	boardsize := t.Boardsize()
	board := make([]float64, boardsize*boardsize)
	max := 0.0
	for child := root.Child; child != nil; child = child.Sibling {
		if child.Visits > max {
			max = child.Visits
		}
	}
	for child := root.Child; child != nil; child = child.Sibling {
		board[child.Vertex] = child.Visits / max
		if root.Color == Reverse(WHITE) {
			board[child.Vertex] = -board[child.Vertex]
		}
	}
	for row := 0; row < boardsize; row++ {
		for col := 0; col < boardsize; col++ {
			v := row*boardsize + col
			s += fmt.Sprintf("%.3f", board[v])
			if col != boardsize-1 {
				s += " "
			}
		}
		if row != boardsize-1 {
			s += "\n"
		}
	}
	return
}

func TerritoryBoard(territory []float64, samples float64, t Tracker) (s string) {
	boardsize := t.Boardsize()
	for row := 0; row < boardsize; row++ {
		for col := 0; col < boardsize; col++ {
			v := row*boardsize + col
			r := territory[v] / samples
			red := uint32(0)
			green := uint32(r * 255)
			blue := uint32((1 - r) * 255)
			s += fmt.Sprintf("0x%02.x%02.x%02.x", red, green, blue)
			if col != boardsize-1 {
				s += " "
			}
		}
		if row != boardsize-1 {
			s += "\n"
		}
	}
	return
}

func StatsBoard(root *Node, t Tracker) (s string) {
	board := make([]string, t.Sqsize())
	for child := root.Child; child != nil; child = child.Sibling {
		board[child.Vertex] = fmt.Sprintf("%.0f/%.0f", child.Wins, child.Visits)
	}
	for row := 0; row < t.Boardsize(); row++ {
		for col := 0; col < t.Boardsize(); col++ {
			v := row*t.Boardsize() + col
			if board[v] == "" {
				s += "\"\""
			} else {
				s += board[v]
			}
			if col != t.Boardsize()-1 {
				s += " "
			}
		}
		if row != t.Boardsize()-1 {
			s += "\n"
		}
	}
	return
}

func FormatScore(t Tracker) string {
	bc, wc := t.Score(t.GetKomi())
	ex := bc - wc
	if ex > 0 {
		return fmt.Sprintf("B+%.1f", ex)
	}
	return fmt.Sprintf("W+%.1f", -ex)
}

func LegalBoard(t Tracker, label map[byte]string) (s string) {
	boardsize := t.Boardsize()
	for row := 0; row < boardsize; row++ {
		for col := 0; col < boardsize; col++ {
			v := row*boardsize + col
			if t.Legal(BLACK, v) && t.Legal(WHITE, v) {
				s += label[BOTH]
			} else if t.Legal(BLACK, v) && !t.Legal(WHITE, v) {
				s += label[BLACK]
			} else if !t.Legal(BLACK, v) && t.Legal(WHITE, v) {
				s += label[WHITE]
			} else if !t.Legal(BLACK, v) && !t.Legal(WHITE, v) {
				s += label[EMPTY]
			}
			if col != boardsize-1 {
				s += " "
			}
		}
		if row != boardsize-1 {
			s += "\n"
		}
	}
	return
}

var lastEmitTime int64

func EmitGFX(root *Node, t Tracker) {
	if time.Nanoseconds()-lastEmitTime < 400000000 {
		return
	}
	fmt.Fprintln(os.Stderr, "gogui-gfx:")

	for v := 0; v < t.Sqsize(); v++ {
		r := root.territory[v] / root.Visits
		red := uint32(0)
		green := uint32(r * 255)
		blue := uint32((1 - r) * 255)
		fmt.Fprintf(os.Stderr, "COLOR 0x%02.x%02.x%02.x %s\n", red, green, blue, t.Vtoa(v))
	}

	maxValue := 0.0
	for v := 0; v < t.Sqsize(); v++ {
		value := float64(0)
		for child := root.Child; child != nil; child = child.Sibling {
			if child.Vertex == v {
				value = child.Visits
			}
		}
		if value > maxValue {
			maxValue = value
		}
	}
	influenceString := ""
	for v := 0; v < t.Sqsize(); v++ {
		value := 0.0
		for child := root.Child; child != nil; child = child.Sibling {
			if child.Vertex == v {
				value = child.Visits
			}
		}
		influenceString += t.Vtoa(v)
		influenceString += fmt.Sprintf(" %.2f ", -value/maxValue)
	}
	fmtString := "INFLUENCE %s\n"
	fmt.Fprintf(os.Stderr, fmtString, influenceString)

	fmt.Fprintln(os.Stderr)
	lastEmitTime = time.Nanoseconds()
}

func Version(config *Config) string {
	version := "0.2"
	version += fmt.Sprintf("_c=%.2f", config.Explore)
	version += fmt.Sprintf("_k=%.2f", config.RAVE)
	version += fmt.Sprintf("_e=%.0f", config.ExpandAfter)
	version += fmt.Sprintf("_p=%d", config.MaxPlayouts)
	version += fmt.Sprintf("_t=%d", config.Timelimit)
	version += fmt.Sprintf("_o=%.2f", config.Cutoff)
	if config.Var {
		version += "_var"
	}
	if config.AMAF {
		version += "_amaf"
	}
	if config.Neighbors {
		version += "_neighbors"
	}
	if config.Ancestor {
		version += "_ancestor"
	}
	if config.Seed {
		version += "_seed"
	}
	return version
}

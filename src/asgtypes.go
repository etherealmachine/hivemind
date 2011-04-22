package main

import (
	"strings"
	"fmt"
	"strconv"
	"os"
	"time"
	"math"
)

const (
	EMPTY = byte(0)
	BLACK = byte(1)
	WHITE = byte(2)
	BOTH = byte(3)
	ILLEGAL = byte(4)
	LEGAL_BLACK = byte(5)
	LEGAL_WHITE = byte(6)
	LEGAL_BOTH = byte(7)
)

// convert a string to a vertex
func Atov(s string, boardsize int) int {
	if s == "PASS" || s == "pass" {
		return -1
	}
	// pull apart into alpha and int pair
	col := byte(strings.ToUpper(s)[0])
	row, err := strconv.Atoi(s[1:len(s)])
	if err != nil {
		panic("Failed to convert string to vertex")
	}
	row = boardsize - row
	switch {
		case col < 'I':
			col = col - 'A'
		case col >= 'I':
			col = col - 'A' - 1
	}
	return row * boardsize + int(col)
}

// convert a vertex to a string
func Vtoa(v int, boardsize int) string {
	if v == -1 { return "PASS" }
	alpha, num := v % boardsize, v / boardsize
	num = boardsize - num
	alpha = alpha + 'A'
	if alpha >= 'I' {
		alpha = alpha + 1
	}
	return fmt.Sprintf("%s%d", string(alpha), num)
}

// convert a string to a color
func Atoc(s string) (c byte) {
	switch strings.ToUpper(s) {
		case "B":
			c = BLACK
		case "W":
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
			s = "@"
		case LEGAL_BLACK:
			s = "+"
		case LEGAL_WHITE:
			s = "-"
		case LEGAL_BOTH:
			s = "."
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
	}
	return EMPTY
}

func Bwboard(board []byte, boardsize int, label bool) string {
	if *hex {
		return Bwhexboard(board, boardsize, label)
	} else {
		return Bwgoboard(board, boardsize, label)
	}
	return ""
}

// puts the internal board representation into a string using "b", "w", and "."
// newlines seperate rows
func Bwgoboard(board []byte, boardsize int, label bool) (s string) {
	if label {
		s += "  "
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
	}
	for row := 0; row < boardsize; row++ {
		if label {
			s += fmt.Sprintf("%d ", boardsize - row)
		}
		for col := 0; col < boardsize; col++ {
			v := row * boardsize + col
			s += Ctoa(board[v])
			if col != boardsize - 1 {
				s += " "
			}
		}
		if label {
		s += fmt.Sprintf(" %d", boardsize - row)
		}
		if row != boardsize - 1 {
			s += "\n"
		}
	}
	if label {
		s += "\n  "
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
	}
	return
}

// puts the internal board representation into a string using "b", "w", and "."
// newlines seperate rows
func Bwhexboard(board []byte, boardsize int, label bool) (s string) {
	if label {
		s += " "
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
	}
	for row := 0; row < boardsize; row++ {
		if label {
			for i := 0; i < row; i++ {
				s += " "
			}
			s += fmt.Sprintf("%d ", boardsize - row)
		}
		for col := 0; col < boardsize; col++ {
			v := row * boardsize + col
			s += Ctoa(board[v])
			if col != boardsize - 1 {
				s += " "
			}
		}
		if label {
			s += fmt.Sprintf(" %d", boardsize - row)
		}
		if row != boardsize - 1 {
			s += "\n"
		}
	}
	if label {
		s += "\n  "
		for i := 0; i < boardsize; i++ {
			s += " "
		}
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
	}
	return
}

func Fboard(board []float64, boardsize int, label bool) (s string) {
	if label {
		s += "  "
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
	}
	for row := 0; row < boardsize; row++ {
		if label {
			s += fmt.Sprintf("%d ", boardsize - row)
		}
		for col := 0; col < boardsize; col++ {
			v := row * boardsize + col
			s += fmt.Sprintf("%f", board[v])
			if col != boardsize - 1 {
				s += " "
			}
		}
		if label {
		s += fmt.Sprintf(" %d", boardsize - row)
		}
		if row != boardsize - 1 {
			s += "\n"
		}
	}
	if label {
		s += "\n  "
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
	}
	return
}

func VisitsBoard(root *Node, t Tracker) (s string) {
	boardsize := t.Boardsize()
	board := make([]float64, boardsize * boardsize)
	for child := root.child; child != nil; child = child.sibling {
		board[child.vertex] = math.Log(child.visits) / math.Log(root.visits)
		if child.color == WHITE {
			board[child.vertex] = -board[child.vertex]
		}
	}
	for row := 0; row < boardsize; row++ {
		for col := 0; col < boardsize; col++ {
			v := row * boardsize + col
			s += fmt.Sprintf("%.3f", board[v])
			if col != boardsize - 1 {
				s += " "
			}
		}
		if row != boardsize - 1 {
			s += "\n"
		}
	}
	return
}

func TerritoryBoard(root *Node, t Tracker) (s string) {
	boardsize := t.Boardsize()
	for row := 0; row < boardsize; row++ {
		for col := 0; col < boardsize; col++ {
			v := row * boardsize + col
			r := root.territory[v] / root.visits
			red := uint32(0)
			green := uint32(r * 255)
			blue := uint32((1 - r) * 255)
			s += fmt.Sprintf("0x%02.x%02.x%02.x", red, green, blue)
			if col != boardsize - 1 {
				s += " "
			}
		}
		if row != boardsize - 1 {
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

func LegalBoard(t Tracker) (s string) {
	boardsize := t.Boardsize()
	for row := 0; row < boardsize; row++ {
		for col := 0; col < boardsize; col++ {
			v := row * boardsize + col
			if t.Legal(BLACK, v) && t.Legal(WHITE, v) {
				s += "green"
			} else if t.Legal(BLACK, v) && !t.Legal(WHITE, v) {
				s += "black"
			} else if !t.Legal(BLACK, v) && t.Legal(WHITE, v) {
				s += "white"
			} else if !t.Legal(BLACK, v) && !t.Legal(WHITE, v) {
				s += "none"
			}
			if col != boardsize - 1 {
				s += " "
			}
		}
		if row != boardsize - 1 {
			s += "\n"
		}
	}
	return
}

func TreeToString(depth int, node *Node, t Tracker) (s string) {
	if node.visits == 0 && node.amafVisits == 0 { return "" }
	for i := 0; i < depth; i++ {
		s += "  "
	}
	s += fmt.Sprintf("%s%s UCT: (%5.2f %5.2f %6.0f) AMAF: (%5.2f %6.0f %6.0f)\n",
						Ctoa(node.color), Vtoa(node.vertex, t.Boardsize()),
						node.mean, node.mean + node.UCB, node.visits,
						node.amafMean, node.amafVisits, node.amafMean + node.amafUCB)
	if node.child != nil {
		for child := node.child; child != nil; child = child.sibling {
			s += TreeToString(depth + 1, child, t)
		}
	}
	return
}

var lastEmitTime int64;

func EmitGFX(root *Node, t Tracker) {
	if !*modeGTP { return }
	if time.Nanoseconds() - lastEmitTime < 400000000 { return }
	fmt.Fprintln(os.Stderr, "gogui-gfx:")
	
	for v := 0; v < t.Sqsize(); v++ {
		r := root.territory[v] / root.visits
		red := uint32(0)
		green := uint32(r * 255)
		blue := uint32((1 - r) * 255)
		fmt.Fprintf(os.Stderr, "COLOR 0x%02.x%02.x%02.x %s\n", red, green, blue, Vtoa(v, t.Boardsize()))
	}
	
	influenceString := ""
	for v := 0; v < t.Sqsize(); v++ {
		visits := float64(0)
		for child := root.child; child != nil; child = child.sibling {
			if child.vertex == v { visits = child.visits }
		}
		s := math.Log(visits) / math.Log(root.visits)
		influenceString += Vtoa(v, t.Boardsize())
		influenceString += fmt.Sprintf(" %.2f ", -s)
	}
	fmtString := "INFLUENCE %s\n"
	fmt.Fprintf(os.Stderr, fmtString, influenceString)
	
	fmt.Fprintln(os.Stderr)
	lastEmitTime = time.Nanoseconds()
}

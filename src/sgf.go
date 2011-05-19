package main

import (
	"os"
	"bufio"
	"fmt"
)

const (
	KM = iota // komi
	PB        // name of Black player
	PW        // name of White player
	DT        // date and time
	B         // Black move
	W         // White move
)

type Property int
type Value string

func (p *Property) String() string {
	switch *p {
	case KM:
		return "KM"
	case PB:
		return "PB"
	case PW:
		return "PW"
	case DT:
		return "DT"
	case B:
		return "B"
	case W:
		return "W"
	}
	panic("property not supported")
}

func Load(filename string) (Tracker, byte) {
	t := NewTracker(*size)
	file, _ := os.Open(filename)
	defer func() { file.Close() }()
	reader := bufio.NewReader(file)
	// consume opening '('
	reader.ReadString('(')
	// consume opening ';'
	reader.ReadString(';')
	// throw out first node
	reader.ReadString(';')
	more := true
	var color byte
	for more {
		var s string
		// pull out move nodes until no more left
		s, _ = reader.ReadString(';')
		if s == "" {
			s, _ = reader.ReadString(')')
			more = false
		}
		if len(s) == 6 || len(s) == 7 {
			color = Atoc(string(s[0]))
			row := s[2] - 97
			col := s[3] - 97
			vertex := int(col*uint8(*size) + row)
			t.Play(color, vertex)
		}
	}
	return t, Reverse(color)
}

func SGFMove(color byte, vertex int, size int) (s string) {
	s += Ctoa(color)
	s += "["
	if vertex != -1 {
		col := vertex % size
		row := vertex / size
		s += string(col + 97)
		s += string(row + 97)
	}
	s += "]"
	return s
}

func SGF(t Tracker) string {
	sgf := fmt.Sprintf("(;FF[4]CA[UTF-8]SZ[%d]KM[%.1f]!RE[%s]", t.Boardsize(), t.GetKomi(), FormatScore(t))
	color := BLACK
	record := t.Record()
	moves := t.MoveCount()
	for i := 0; i < moves; i++ {
		sgf += fmt.Sprintf(";%s", SGFMove(color, record[i], t.Boardsize()))
		color = Reverse(color)
	}
	sgf += ")"
	return sgf
}

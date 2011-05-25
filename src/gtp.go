package main

import (
	"os"
	"bufio"
	"fmt"
	"strings"
	"strconv"
	"log"
)

var supported_commands = `name
protocol_version
version
known_command
list_commands
quit
boardsize
clear_board
komi
play
genmove
final_score
showboard
time_settings
time_left
gogui-analyze_commands`
var gogui_commands = `bwboard/Occupied Points/occupied
dboard/Visits/visits
cboard/Territory/territory
cboard/Legal/legal`

func known_command(command_name string) string {
	for _, s := range strings.Split(supported_commands, "\n", -1) {
		if strings.TrimSpace(s) == command_name {
			return "true"
		}
	}
	for _, s := range strings.Split(gogui_commands, "\n", -1) {
		if strings.Split(s, "/", -1)[2] == command_name {
			return "true"
		}
	}
	return "false"
}

func set_timelimit(timeleft int) uint {
	if timeleft < 60 {
		return 1
	} else if timeleft < 120 {
		return 3
	}
	return *timelimit
}

func GTP() {
	var boardsize int
	var t Tracker
	var root *Node
	var color byte
	main_time := -1
	time_left_color := EMPTY
	time_left_time := -1
	passcount := 0
	r := bufio.NewReader(os.Stdin)
	for {
		s, err := r.ReadString('\n')
		if err == os.EOF {
			break
		}
		cmds := strings.Split(s[0:len(s)-1], " ", -1)
		var res string
		var fail bool
		switch cmds[0] {
			case "protocol_version":
				res = "2"
			case "name":
				res = "GoGo"
			case "version":
				res = "0.1"
			case "known_command":
				res = known_command(cmds[1])
			case "list_commands":
				res = supported_commands
			case "quit":
				fmt.Fprint(os.Stdout, "=\n\n")
				return
			case "boardsize":
				boardsize, err = strconv.Atoi(cmds[1])
				if err != nil {
					res = fmt.Sprintf("Could not convert %s to integer", cmds[1])
					fail = true
				}
				if *hex {
					t = NewTracker(boardsize)
					color = WHITE
					passcount = 0
				}
			case "clear_board":
				t = NewTracker(boardsize)
				color = WHITE
				passcount = 0
			case "komi":
				new_komi, err := strconv.Atof64(cmds[1])
				if err != nil {
					res = fmt.Sprintf("Could not convert %s to float", cmds[1])
					fail = true
				} else {
					t.SetKomi(new_komi)
				}
			case "play":
				if len(cmds) != 3 {
					res = "missing argument"
				} else {
					color = Atoc(cmds[1])
					vertex := Atov(cmds[2], t.Boardsize())
					t.Play(color, vertex)
					log.Print(Bwboard(t.Board(), t.Boardsize(), true))
					if vertex == -1 { passcount++ }
					if root != nil {
						root = root.Play(color, vertex, t)
					}
				}
			case "genmove":
				if len(cmds) != 2 {
					res = "missing argument"
				} else if *cgo && passcount >= 3 {
					res = Vtoa(-1, t.Boardsize())
				} else {
					saved_timelimit := *timelimit
					color = Atoc(cmds[1])
					if color == time_left_color { *timelimit = set_timelimit(time_left_time) }
					var vertex int
					if !(*hex && t.Winner() != EMPTY) {
						if *pat {
							vertex = matcher.Match(color, t.Sqsize()/2, t)
						} else {
							if root == nil {
								root = NewRoot(color, t)
							}
							genmove(root, t, matcher)
							if root == nil || root.Best() == nil {
								vertex = -1
							} else {
								vertex = root.Best().vertex
							}
						}
					} else {
						vertex = -1
					}
					t.Play(color, vertex)
					if *verbose {
						t.Verify()
					}
					log.Print(Bwboard(t.Board(), t.Boardsize(), true))
					if root != nil {
						root = root.Play(color, vertex, t)
					}
					if vertex == -1 && *hex && t.Winner() == Reverse(color) {
						res = "resign"
					} else {
						res = Vtoa(vertex, t.Boardsize())
					}
					*timelimit = saved_timelimit
				}
			case "final_score":
				res = FormatScore(t)
			case "showboard":
				res = ""
			case "gogui-analyze_commands":
				res = gogui_commands
			case "occupied":
				res = Bwboard(t.Board(), t.Boardsize(), false)
			case "visits":
				if !(*hex && t.Winner() != EMPTY) {
					tmpRoot := NewRoot(Reverse(color), t)
					genmove(tmpRoot, t, matcher)
					res = VisitsBoard(tmpRoot, t)
				} else {
					res = ""
				}
			case "territory":
				if !(*hex && t.Winner() != EMPTY) {
					tmpRoot := NewRoot(Reverse(color), t)
					genmove(tmpRoot, t, matcher)
					res = TerritoryBoard(tmpRoot, t)
				} else {
					res = ""
				}
			case "legal":
				res = LegalBoard(t)
			case "time_settings":
				main_time, _ = strconv.Atoi(cmds[1])
				byo_yomi_time, _ := strconv.Atoi(cmds[2])
				byo_yomi_stones, _ := strconv.Atoi(cmds[3])
				log.Printf("Time settings: m: %d, b: %d, s: %d\n", main_time, byo_yomi_time, byo_yomi_stones)
			case "time_left":
				time_left_color = Atoc(cmds[1])
				time_left_time, _ = strconv.Atoi(cmds[2])
				time_left_stones, _ := strconv.Atoi(cmds[3])
				log.Printf("Time Left: %s, %d, %d\n", Ctoa(time_left_color), time_left_time, time_left_stones)
		}
		if known_command(cmds[0]) == "false" {
			fail = true
			res = "unknown command"
		}
		switch {
			case fail:
				fmt.Fprintf(os.Stdout, "? %s\n\n", res)
			case res == "":
				fmt.Fprint(os.Stdout, "= \n\n")
			default:
				fmt.Fprintf(os.Stdout, "= %s\n\n", res)
		}
	}
}

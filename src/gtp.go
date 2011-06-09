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
final_status_list
gogui-analyze_commands`
var gogui_commands = `dboard/Visits/visits
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

func set_timelimit(timeleft int) int {
	if timeleft < 15 {
		return 0
	}	else if timeleft < 30 {
		return 2
	} else if timeleft < 60 {
		return 3
	} else if timeleft < 120 {
		return 4
	}
	return -1
}

func GTP(config *Config) {
	var boardsize int
	var t Tracker
	var root *Node
	var color byte
	book := NewBook()
	main_time := -1
	time_left_color := EMPTY
	time_left_time := -1
	passcount := 0
	movecount := 0
	game_over := false
	r := bufio.NewReader(os.Stdin)
	for {
		s, err := r.ReadString('\n')
		if err == os.EOF {
			break
		}
		args := strings.Split(s[0:len(s)-1], " ", -1)
		var res string
		var fail bool
		switch args[0] {
			case "protocol_version":
				res = "2"
			case "name":
				res = "GoGo"
			case "version":
				res = "0.1"
			case "known_command":
				res = known_command(args[1])
			case "list_commands":
				res = supported_commands
			case "quit":
				fmt.Fprint(os.Stdout, "=\n\n")
				return
			case "boardsize":
				boardsize, err = strconv.Atoi(args[1])
				if err != nil {
					res = fmt.Sprintf("Could not convert %s to integer", args[1])
					fail = true
				}
				config.size = boardsize
			case "clear_board":
				t = NewTracker(config)
				color = WHITE
				passcount = 0
				movecount = 0
				game_over = false
			case "komi":
				new_komi, err := strconv.Atof64(args[1])
				if err != nil {
					res = fmt.Sprintf("Could not convert %s to float", args[1])
					fail = true
				} else {
					t.SetKomi(new_komi)
				}
			case "play":
				if len(args) != 3 {
					fail = true
					res = "missing argument"
				} else {
					color = Atoc(args[1])
					vertex := t.Atov(args[2])
					t.Play(color, vertex)
					movecount++
					t.String()
					if vertex == -1 { passcount++ }
					if root != nil {
						root = root.Play(color, vertex, t)
					}
				}
			case "genmove":
				if len(args) != 2 {
					fail = true
					res = "missing argument"
				} else if config.cgo && passcount >= 3 {
					res = t.Vtoa(-1)
				} else {
					saved_timelimit := config.timelimit
					color = Atoc(args[1])
					if time_left_time != -1 && color == time_left_color { config.timelimit = set_timelimit(time_left_time) }
					var vertex int
					// HEX, swap-safe: if black and first move of game, play a move that should be safe from swapping
					if config.hex && color == BLACK && config.swapsafe && movecount == 0 {
						vertex = (3 * t.Boardsize()) + 2
					// Pass if: no time left, game definitely won
					} else if config.timelimit != 0 && t.Winner() == EMPTY && !game_over {
						if root == nil {
							root = NewRoot(color, t, config)
						}
						if config.useBook { vertex = book.Load(color, t) }
						genmove(root, t)
						// if genmove predicts win in >95% of playouts, set a flag and pass next time around
						if root.wins / root.visits > 0.95 { game_over = true }
						// if genmove predicts win in <5% of playouts, set a flag and pass next time around
						if root.wins / root.visits < 0.05 { game_over = true }
						if config.useBook { book.Save(root, t) }
						if root == nil || root.Best() == nil {
							vertex = -1
						} else {
							vertex = root.Best().vertex
						}
					} else {
						vertex = -1
					}
					t.Play(color, vertex)
					movecount++
					if config.verbose {
						t.Verify()
					}
					log.Print(t.String())
					if root != nil {
						root = root.Play(color, vertex, t)
					}
					if vertex == -1 && config.hex && t.Winner() == Reverse(color) {
						res = "resign"
					} else {
						res = t.Vtoa(vertex)
					}
					config.timelimit = saved_timelimit
				}
			case "final_score":
				res = FormatScore(t)
			case "showboard":
				res = t.String()
			case "gogui-analyze_commands":
				res = gogui_commands
			case "visits":
				if !(config.hex && t.Winner() != EMPTY) {
					tmpRoot := NewRoot(Reverse(color), t, config)
					genmove(tmpRoot, t)
					res = VisitsBoard(tmpRoot, t)
				} else {
					res = ""
				}
			case "territory":
				if t.Winner() == EMPTY {
					tmpRoot := NewRoot(Reverse(color), t, config)
					genmove(tmpRoot, t)
					res = TerritoryBoard(tmpRoot.territory, tmpRoot.visits, t)
				} else {
					res = TerritoryBoard(t.Territory(color), 1, t)
				}
			case "legal":
				res = LegalBoard(t)
			case "time_settings":
				main_time, _ = strconv.Atoi(args[1])
				byo_yomi_time, _ := strconv.Atoi(args[2])
				byo_yomi_stones, _ := strconv.Atoi(args[3])
				log.Printf("Time settings: m: %d, b: %d, s: %d\n", main_time, byo_yomi_time, byo_yomi_stones)
			case "time_left":
				time_left_color = Atoc(args[1])
				time_left_time, _ = strconv.Atoi(args[2])
				time_left_stones, _ := strconv.Atoi(args[3])
				log.Printf("Time Left: %s, %d, %d\n", Ctoa(time_left_color), time_left_time, time_left_stones)
			case "final_status_list":
				if config.cgo {
					gotracker := t.(*GoTracker)
					stones := gotracker.dead()
					for i := range stones {
						res += t.Vtoa(stones[i])
						if i != len(stones) - 1 { res += "\n" }
					}
				} else {
					fail = true
					res = "cannot determine status for hex"
				}
		}
		if known_command(args[0]) == "false" {
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

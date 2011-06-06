competition_type = 'playoff'

home_dir="/home/jpettit/code"

eval_path = home_dir + "/gogo/hex_esswarm_eval.19.gob"
pat_path = home_dir + "/gogo/hex_esswarm_pat.26.gob"

players = {
		"gogo" : Player(home_dir + "/gogo/gogo -gtp -p=10000 -hex -swapsafe"),
		"gogo_eval" : Player(home_dir + "/gogo/gogo -gtp -hex -p=10000 -e=10 -swapsafe -eval -efile " + eval_path),
		"gogo_pat" : Player(home_dir + "/gogo/gogo -gtp -hex -p=10000 -e=10 -swapsafe -pat -pfile " + pat_path),
		}

board_size = 11
komi = 0

matchups = []
matchups.append(Matchup("gogo", "gogo_eval", alternating=True, number_of_games=100))
#matchups.append(Matchup("gogo", "gogo_pat", alternating=True, number_of_games=100))

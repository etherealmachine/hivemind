competition_type = 'playoff'

board_size = 13
komi = 0

home_dir="/home/jpettit/code"

swarm_path = home_dir + "/gogo/swarm-1.gob"

players = {
		"six" : Player(home_dir + "/six-0.5.3/six/gtp beginner"),
		"gogo_swarm" : Player(home_dir + "/gogo/gogo -gtp -p=50000 -c=0.5 -e=50 -uct -hex -stats -tablepat -file " + swarm_path),
		}

matchups = []
matchups.append(Matchup("six", "gogo_swarm", alternating=True, number_of_games=500))

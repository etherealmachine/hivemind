competition_type = 'playoff'

board_size = 11
komi = 0

home_dir="/home/jpettit/code"

swarm_path = home_dir + "/gogo/analysis/swarm-1-1.gob"

players = {
		"six" : Player(home_dir + "/six-0.5.3/six/gtp beginner"),
		"mohex" : Player("/home/jpettit/Downloads/benzene-0.9.0/src/mohex/mohex"),
		"gogo" : Player(home_dir + "/gogo/gogo -gtp -p=10000 -c=0.5 -e=50 -k=1000 -uct -hex -stats"),
		"gogo_swarm" : Player(home_dir + "/gogo/gogo -gtp -p=50000 -c=0.5 -e=50 -uct -hex -stats -tablepat -file " + swarm_path),
		}

matchups = []
matchups.append(Matchup("mohex", "gogo", alternating=True, number_of_games=1))

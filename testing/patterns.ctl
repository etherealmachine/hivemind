competition_type = "playoff"

board_size = 13
komi = 0

import os
six = os.path.expanduser("~/code/six-0.5.3/six/gtp") + " beginner"
hive = os.path.expanduser("~/code/hivemind/src/hive") + " -gtp -t 5 -var -amaf -k 550 -hex -swapsafe"
pfile = os.path.expanduser("~/code/hivemind/src/swarm.2.gob")
hive_swarm = os.path.expanduser("~/code/hivemind/src/hive") + " -gtp -t 5 -var -amaf -k 550 -hex -swapsafe -pfile " + pfile

players = {
		"six" : Player(six),
		"hive" : Player(hive),
		"hive_swarm" : Player(hive_swarm),
		}

matchups = []
matchups.append(Matchup("hive", "hive_swarm", alternating=True, number_of_games=500))

competition_type = "allplayall"

board_size = 13
komi = 0

import os
six = os.path.expanduser("~/code/six-0.5.3/six/gtp") + " beginner"
hive = os.path.expanduser("~/code/hivemind/src/hive") + " -gtp -p 10000 -var -hex -swapsafe"
pfile = os.path.expanduser("~/code/hivemind/src/swarm.16.gob")
hive_learned_no_suggest = hive + " -pfile " + pfile
hive_learned_suggest = hive + " -pfile " + pfile + " -playout_suggest"
hive_uniform_suggest = hive + " -playout_suggest_uniform"

players = {
#"six" : Player(six),
		"hive" : Player(hive),
		"hive_learned_no_suggest" : Player(hive_learned_no_suggest),
		"hive_learned_suggest" : Player(hive_learned_suggest),
		"hive_uniform_suggest" : Player(hive_uniform_suggest),
		}

rounds = 100
competitors = players.keys()

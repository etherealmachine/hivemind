competition_type = 'playoff'

import os
code_dir = os.path.expanduser("~/code")

def splitter(s):
	return s.split(".")
files = os.listdir(code_dir + "/gogo/")
eval_files = map(splitter, filter(lambda s: "eval" in s and "gob" in s, files))
pat_files = map(splitter, filter(lambda s: "pat" in s and "gob" in s, files))
eval_files.sort(key=lambda (prefix, number, suffix): int(number), reverse=True)
pat_files.sort(key=lambda (prefix, number, suffix): int(number), reverse=True)

eval_path = ""
if eval_files:
	eval_path = code_dir + "/gogo/" + ".".join(eval_files[0])
pat_path = ""
if pat_files:
	pat_path = code_dir + "/gogo/" + ".".join(pat_files[0])
	
print 'eval file:', eval_path
print 'pat file:', pat_path

players = {
		"gogo" : Player(code_dir + "/gogo/gogo -gtp -p=10000 -hex -swapsafe -log gogo.log -stats "),
		"gogo_eval" : Player(code_dir + "/gogo/gogo -gtp -hex -p=10000 -e=10 -swapsafe -log gogo_eval.log -stats -eval -efile " + eval_path),
		"gogo_pat" : Player(code_dir + "/gogo/gogo -gtp -hex -p=10000 -e=10 -swapsafe -pat -pfile " + pat_path),
		}

board_size = 11
komi = 0

matchups = []
matchups.append(Matchup("gogo", "gogo_eval", alternating=True, number_of_games=100))
#matchups.append(Matchup("gogo", "gogo_pat", alternating=True, number_of_games=100))

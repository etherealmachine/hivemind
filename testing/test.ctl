competition_type = "playoff"

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

print "eval file:", eval_path
print "pat file:", pat_path

d = {
	"code" : code_dir,
	"pat" : pat_path,
	"eval" : eval_path,
}

players = {
		"gogo" : Player("{code}/gogo/gogo -gtp -t=5 -hex -stats -amaf".format(**d)),
		"gogo_eval" : Player("{code}/gogo/gogo -gtp -hex -t=5 -e=10 -swapsafe -stats -amaf -eval -efile {eval}".format(**d)),
		"mohex" : Player("{code}/benzene-0.9.0/src/mohex/mohex".format(**d)),
		"gogo_pat" : Player("/gogo/gogo -gtp -hex -p=10000 -e=10 -swapsafe -pat -pfile {pat}".format(**d)),
		"gogo_eval_pat" : Player("{code}/gogo/gogo -gtp -hex -p=50000 -e=10 -c 0.01 -stats -amaf -pat -pfile {pat} -eval -efile {eval}".format(**d)),
		}

board_size = 11
komi = 0

matchups = []
matchups.append(Matchup("gogo_eval_pat", "mohex", alternating=True, number_of_games=100))

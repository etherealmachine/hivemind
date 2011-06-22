package main

import (
	"fmt"
	"math"
	"time"
	"log"
	"rand"
	"container/vector"
	"os"
	"gob"
)

type Node struct {
	parent                                                                    *Node
	Child                                                                     *Node
	Last                                                                      *Node
	Sibling                                                                   *Node
	Color                                                                     byte
	Vertex                                                                    int
	Wins, Visits, Mean                                                        float64
	amafWins, amafVisits, amafMean                                            float64
	ancestorWins, ancestorVisits, ancestorMean                                float64
	evalWins, evalVisits, evalMean                                            float64
	blendedMean                                                               float64
	value                                                                     float64
	territory                                                                 []float64
	seeds, totalseeds                                                         int
	playout_time, update_time, win_calc_time, next_time, play_time, copy_time int64
	next_count, play_count                                                    int64
	config                                                                    *Config
}

func NewRoot(color byte, t Tracker, config *Config) *Node {
	node := new(Node)
	node.Color = Reverse(color)
	node.Vertex = -1
	node.config = config
	return node
}

func NewNode(parent *Node, color byte, vertex int) *Node {
	node := new(Node)
	node.parent = parent
	node.Color = color
	node.Vertex = vertex
	node.config = parent.config
	return node
}

// step through the tree for some number of playouts
func genmove(root *Node, t Tracker) {
	if root.config.Stats {
		log.Printf("kept %.0f visits\n", root.Visits)
	}
	if t.Winner() != EMPTY {
		return
	}
	start := time.Nanoseconds()
	root.territory = make([]float64, t.Sqsize())
	root.playout_time = 0
	root.update_time = 0
	root.win_calc_time = 0
	root.next_time = 0
	root.play_time = 0
	root.copy_time = 0
	root.next_count = 0
	root.play_count = 0
	playouts := float64(treeSearch(root, t))
	elapsed := time.Nanoseconds() - start
	elapsed_seconds := float64(elapsed) / 1e9
	if root.config.Stats {
		pps := float64(playouts) / elapsed_seconds
		log.Printf("%.0f playouts in %.2f s, %.2f pps\n", playouts, elapsed_seconds, pps)
		if root.config.Verbose {
			avg_playout_time := float64(root.playout_time) / playouts
			avg_update_time := float64(root.update_time) / playouts
			avg_win_calc_time := float64(root.win_calc_time) / playouts
			avg_next_time := float64(root.next_time) / float64(root.next_count)
			avg_play_time := float64(root.play_time) / float64(root.play_count)
			avg_copy_time := float64(root.copy_time) / root.Visits
			pps = 1e9 / avg_playout_time
			log.Printf("%.0f nanoseconds per playout, %.2f pps\n", avg_playout_time, pps)
			log.Printf("%.0f nanoseconds per update\n", avg_update_time)
			log.Printf("%.0f nanoseconds per win_calc\n", avg_win_calc_time)
			log.Printf("%.0f nanoseconds per next\n", avg_next_time)
			log.Printf("%.0f nanoseconds per play\n", avg_play_time)
			log.Printf("%.0f nanoseconds per copy\n", avg_copy_time)
			unaccounted := elapsed -
				root.playout_time - root.update_time - root.win_calc_time - root.next_time - root.play_time - root.copy_time
			log.Printf("%.2f seconds unaccounted\n", float64(unaccounted)/1e9)
		}
		if root.config.Timelimit > 0 {
			if elapsed_seconds > float64(root.config.Timelimit) {
				log.Printf("%.2f seconds overtime\n", elapsed_seconds-float64(root.config.Timelimit))
			} else {
				log.Printf("%.2f seconds left\n", float64(root.config.Timelimit)-elapsed_seconds)
			}
		}
		log.Printf("winrate: %.2f\n", root.Wins/root.Visits)
		territory_mean := 0.0
		for v := 0; v < t.Sqsize(); v++ {
			territory_mean += root.territory[v] / root.Visits
		}
		territory_mean /= float64(t.Sqsize())
		territory_var := 0.0
		for v := 0; v < t.Sqsize(); v++ {
			territory_var += math.Pow((root.territory[v]/root.Visits)-territory_mean, 2)
		}
		territory_var /= float64(t.Sqsize())
		log.Printf("territory: mean: %.2f, var: %.2f\n", territory_mean, territory_var)
		log.Printf("max depth: %d\n", root.maxdepth())
		log.Printf("nodes: %d\n", root.nodes())
		log.Printf("visits: %.0f\n", root.Visits)
		if root.config.Seed {
			seeds, totalseeds := root.seedstats()
			log.Printf("seeds: %.2f\n", float64(seeds)/float64(totalseeds))
		}
	}
}

func treeSearch(root *Node, t Tracker) uint {
	playouts := uint(0)
	start := time.Nanoseconds()
	s := time.Nanoseconds()
	trackers := make([]Tracker, 10000)
	for i := 0; i < len(trackers); i++ {
		trackers[i] = t.Copy()
	}
	root.copy_time += (time.Nanoseconds() - s)
	tracker := 0
	for {
		cp := trackers[tracker]
		tracker++
		if tracker >= len(trackers) {
			s := time.Nanoseconds()
			for i := 0; i < len(trackers); i++ {
				trackers[i] = t.Copy()
			}
			root.copy_time += (time.Nanoseconds() - s)
			tracker = 0
		}
		root.step(cp)
		playouts++
		territory := cp.Territory(Reverse(root.Color))
		for i := range territory {
			root.territory[i] += territory[i]
		}
		if root.config.Gfx {
			EmitGFX(root, cp)
		}
		if root.Visits > 1000 && root.config.Cutoff != -1 {
			var bests [2]float64
			for child := root.Child; child != nil; child = child.Sibling {
				if child.Visits > bests[0] {
					bests[0] = child.Visits
				} else if child.Visits > bests[1] {
					bests[1] = child.Visits
				}
			}
			if (bests[0]-bests[1])/root.Visits > root.config.Cutoff {
				break
			}
		}
		if root.Child == nil {
			break
		} else if root.config.Timelimit > 0 {
			elapsed := time.Nanoseconds() - start
			if uint64(elapsed) > uint64(root.config.Timelimit)*uint64(1e9) {
				break
			}
		} else if playouts >= root.config.MaxPlayouts {
			break
		}
	}
	return playouts
}

// navigate through the tree until a leaf node is found to playout
func (root *Node) step(t Tracker) {
	var start int64
	path := new(vector.Vector)
	start = time.Nanoseconds()
	curr := root.Next(root, t)
	root.next_time += time.Nanoseconds() - start
	root.next_count++
	if curr == nil {
		root.Visits = math.Inf(1)
		return
	}
	for {
		path.Push(curr)
		// apply node's position to the board
		start = time.Nanoseconds()
		t.Play(curr.Color, curr.Vertex)
		root.play_time += time.Nanoseconds() - start
		root.play_count++
		if curr.Visits <= root.config.ExpandAfter {
			var color byte
			if root.config.Seed {
				color = curr.seedPlayout(t)
			} else {
				color = Reverse(curr.Color)
			}
			start = time.Nanoseconds()
			t.Playout(color)
			root.playout_time += time.Nanoseconds() - start
			break
		}
		time.Nanoseconds()
		next := curr.Next(root, t)
		root.next_time += time.Nanoseconds() - start
		root.next_count++
		curr = next
		if curr == nil {
			break
		}
	}
	start = time.Nanoseconds()
	winner := t.Winner()
	root.win_calc_time += time.Nanoseconds() - start
	start = time.Nanoseconds()
	for j := 0; j < path.Len(); j++ {
		path.At(j).(*Node).update(t)
	}
	root.update_time += time.Nanoseconds() - start
	if winner == Reverse(root.Color) {
		root.Wins++
	}
	root.Visits++
	root.Mean = root.Wins / root.Visits
}

// add all legal children to node
func (node *Node) expand(t Tracker) {
	color := Reverse(node.Color)
	for i := -1; i < t.Sqsize(); i++ {
		if t.Legal(color, i) {
			child := NewNode(node, color, i)
			if node.Child == nil {
				node.Child = child
			} else {
				node.Last.Sibling = child
			}
			node.Last = child
			cp := t.Copy()
			cp.Play(child.Color, child.Vertex)
			child.Wins = 5
			child.Visits = 10
			if node.config.Ancestor {
				granduncle := child.granduncle()
				if granduncle != nil {
					child.ancestorVisits += granduncle.Visits + granduncle.amafVisits
					child.ancestorWins += granduncle.Wins + granduncle.amafWins
				}
			}
			if node.config.Eval {
				child.evalVisits += node.config.RAVE
				child.evalWins += node.config.RAVE * node.config.evaluator.Eval(Reverse(child.Color), cp)
			}
			child.recalc()
		}
	}
}

// select the next node in the tree to navigate to from this node's children
func (node *Node) Next(root *Node, t Tracker) *Node {
	if node.Child == nil {
		node.expand(t)
		if node.Child == nil {
			cp := t.Copy()
			cp.Play(node.Color, -1)
			cp.Play(Reverse(node.Color), -1)
			if node.Color == cp.Winner() {
				node.Wins = math.Inf(1)
			} else {
				node.Wins = 0
			}
			node.Visits = math.Inf(1)
		}
	}
	var best *Node
	for child := node.Child; child != nil; child = child.Sibling {
		if (best == nil || child.value > best.value) && !math.IsInf(child.Visits, 1) {
			best = child
		}
	}
	return best
}

func (node *Node) Best() *Node {
	var best *Node
	for child := node.Child; child != nil; child = child.Sibling {
		if best == nil || child.Visits > best.Visits {
			best = child
		}
	}
	if best == nil {
		node.Vertex = -1
		return node
	}
	return best
}

func (node *Node) update(t Tracker) {
	if t.Winner() == node.Color {
		node.Wins++
	}
	node.Visits++
	node.recalc()
	if node.config.AMAF {
		for sibling := node.parent.Child; sibling != nil; sibling = sibling.Sibling {
			if t.WasPlayed(sibling.Color, sibling.Vertex) {
				if t.Winner() == sibling.Color {
					sibling.amafWins++
				}
				sibling.amafVisits++
			}
			sibling.recalc()
		}
	}
}

func (node *Node) recalc() {
	if node.Visits == 10 {
		node.value = 1 + 0.1 * rand.Float64()
		return
	}
	node.Mean = node.Wins / node.Visits
	node.blendedMean = node.Mean
	rave := node.config.AMAF || node.config.Neighbors || node.config.Ancestor || node.config.Eval
	if rave {
		beta := math.Sqrt(node.config.RAVE / (3*node.Visits + node.config.RAVE))
		if beta > 0 {
			node.amafMean = node.amafWins / node.amafVisits
			node.ancestorMean = node.ancestorWins / node.ancestorVisits
			node.evalMean = node.evalWins / node.evalVisits
			if math.IsNaN(node.Mean) {
				node.Mean = 0
			}
			if math.IsNaN(node.amafMean) {
				node.amafMean = 0
			}
			if math.IsNaN(node.ancestorMean) {
				node.ancestorMean = 0
			}
			if math.IsNaN(node.evalMean) {
				node.evalMean = 0
			}
			estimatedMean := 0.0
			Samples := 0.0
			if node.config.AMAF {
				estimatedMean += node.amafMean
				Samples++
			}
			if node.config.Neighbors {
				neighborWins := 0.0
				neighborVisits := 0.0
				for sibling := node.parent.Child; sibling != nil; sibling = sibling.Sibling {
					if sibling.Vertex != node.Vertex {
						neighborWins += sibling.Wins
						neighborVisits += sibling.Visits
					}
				}
				estimatedMean += neighborWins / neighborVisits
			}
			if node.config.Ancestor {
				estimatedMean += node.ancestorMean
				Samples++
			}
			if node.config.Eval {
				estimatedMean += node.evalMean
				Samples++
			}
			estimatedMean /= Samples
			if math.IsNaN(estimatedMean) {
				estimatedMean = 0
			}
			node.blendedMean = beta*estimatedMean + (1-beta)*node.Mean
		}
	}
	r := math.Log1p(node.parent.Visits) / node.Visits
	v := 1.0
	if node.config.Var {
		v = math.Fmin(0.25, node.blendedMean - (node.blendedMean * node.blendedMean) + math.Sqrt(2*r))
	}
	node.value = node.blendedMean + node.config.Explore * math.Sqrt(r*v)
}

// return node's grandparent's sibling corresponding to node's move
func (node *Node) granduncle() *Node {
	if node.parent == nil || node.parent.parent == nil || node.parent.parent.parent == nil {
		return nil
	}
	great_grandparent := node.parent.parent.parent
	for granduncle := great_grandparent.Child; granduncle != nil; granduncle = granduncle.Sibling {
		if granduncle.Vertex == node.Vertex {
			return granduncle
		}
	}
	return nil
}

// use node's parent, grandparent, and great-grandparent (gpp) distribution as the initial
// distribution for the playout from node
// returns the new color to play
// NOTE: reverse of node.color is first to play
// this means gpp is the correct color, if gpp doesn't exist, just use parent
func (node *Node) seedPlayout(t Tracker) byte {
	color := Reverse(node.Color)
	var parent, grandparent, great_grandparent *Node
	if node.parent != nil {
		parent = node.parent
	}
	if parent != nil {
		grandparent = parent.parent
	}
	if grandparent != nil {
		great_grandparent = grandparent.parent
	}
	// if it exists, try seeding using great_grandparent
	if great_grandparent != nil {
		// if that succeeds, try seeding with grandparent
		if great_grandparent.seed(t, []int{great_grandparent.Vertex, grandparent.Vertex, parent.Vertex}) {
			color = Reverse(color)
			// if that succeeds, try seeding with parent
			if grandparent.seed(t, []int{great_grandparent.Vertex, grandparent.Vertex, parent.Vertex}) {
				color = Reverse(color)
				if parent.seed(t, []int{great_grandparent.Vertex, grandparent.Vertex, parent.Vertex}) {
					color = Reverse(color)
				}
			}
		}
	} else { // else try seeding using parent
		if parent.seed(t, []int{parent.Vertex}) {
			color = Reverse(color)
		}
	}
	return color
}

// use win-rate distribution of node to play a legal move in tracker
func (node *Node) seed(t Tracker, path []int) bool {
	if node.parent == nil {
		return false
	}
	dist := new(vector.Vector)
	sum := 0.0
	for sibling := node.parent.Child; sibling != nil; sibling = sibling.Sibling {
		for i := 0; i < len(path); i++ {
			if sibling.Vertex == path[i] {
				continue
			}
		}
		dist.Push(sibling.blendedMean)
		sum += sibling.blendedMean
	}
	node.totalseeds++
	r := rand.Float64() * sum
	for i := 0; i < dist.Len(); i++ {
		r -= dist.At(i).(float64)
		if r <= 0 {
			if t.Legal(node.Color, i) {
				t.Play(node.Color, i)
				node.seeds++
				return true
			}
			return false
		}
	}
	return false
}

func (root *Node) merge(node *Node) {
	for child1, child2 := root.Child, node.Child; child1 != nil && child2 != nil; child1, child2 = child1.Sibling, child2.Sibling {
		child1.Wins += child2.Wins
		child1.Visits += child2.Visits
	}
	root.Wins += node.Wins
	root.Visits += node.Visits
}

func (node *Node) maxdepth() int {
	max := 0
	if node.Child != nil {
		for child := node.Child; child != nil; child = child.Sibling {
			d := child.maxdepth()
			if d > max {
				max = d
			}
		}
	}
	return max + 1
}

func (node *Node) nodes() int {
	children := 0
	if node.Child != nil {
		for child := node.Child; child != nil; child = child.Sibling {
			children += child.nodes()
		}
	}
	return children + 1
}

func (node *Node) seedstats() (int, int) {
	count, total := node.seeds, node.totalseeds
	if node.Child != nil {
		for child := node.Child; child != nil; child = child.Sibling {
			c, t := child.seedstats()
			count += c
			total += t
		}
	}
	return count, total
}

func (node *Node) Play(color byte, vertex int, t Tracker) *Node {
	var best *Node
	for child := node.Child; child != nil; child = child.Sibling {
		if best == nil || child.Visits > best.Visits {
			best = child
		}
	}
	for child := node.Child; child != nil; child = child.Sibling {
		if child.Color == color && child.Vertex == vertex {
			log.Print(fmt.Sprintf("predicted: %s%s(%.0f)",
				Ctoa(best.Color), t.Vtoa(best.Vertex), best.Visits))
			log.Print(fmt.Sprintf("actual:    %s%s(%.0f)",
				Ctoa(child.Color), t.Vtoa(child.Vertex), child.Visits))
			child.parent = nil
			return child
		}
	}
	return nil
}

func (root *Node) SaveBook() {
	var filename string
	if root.config.Prefix != "" {
		filename = fmt.Sprintf(root.config.Prefix + ".book.gob")
	} else {
		filename = fmt.Sprintf("book.gob")
	}
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer func() { f.Close() }()
	e := gob.NewEncoder(f)
	err = e.Encode(root)
	if err != nil {
		panic(err)
	}
}

func (node *Node) fix_refs(config *Config) {
	node.config = config
	for child := node.Child; child != nil; child = child.Sibling {
		child.parent = node
		child.fix_refs(config)
	}
	if node.parent != nil {
		node.recalc()
	}
}

func LoadBook(config *Config) {
	t := NewTracker(config)
	config.book = NewRoot(BLACK, t, config)
	if config.Bfile != "" {
		f, err := os.Open(config.Bfile)
		if err != nil {
			panic(err)
		}
		defer func() { f.Close() }()
		d := gob.NewDecoder(f)
		err = d.Decode(config.book)
		if err != nil {
			panic(err)
		}
		config.book.fix_refs(config)
	}
}

func (node *Node) String(depth, maxdepth int, t Tracker) (s string) {
	if node.Visits == 0 {
		return ""
	}
	for i := 0; i < depth; i++ {
		s += "  "
	}
	AMAF := ""
	if node.config.AMAF {
		AMAF = fmt.Sprintf("(%5.2f %6.0f)", node.amafMean, node.amafVisits)
	}
	s += fmt.Sprintf("%s%s (%5.2f %5.2f %6.2f %6.2f) %s\n",
		Ctoa(node.Color), t.Vtoa(node.Vertex),
		node.Mean, node.value, node.Wins, node.Visits, AMAF)
	if depth < maxdepth && node.Child != nil {
		for child := node.Child; child != nil; child = child.Sibling {
			if child.Visits > 0 {
				s += child.String(depth + 1, maxdepth, t)
			}
		}
	}
	return
}

func SpeedTest(config *Config) {
	t := NewTracker(config)
	playoutTime := int64(0)
	start := time.Nanoseconds()
	var elapsed int64
	i := 0
	for {
		cp := t.Copy()
		start1 := time.Nanoseconds()
		cp.Playout(BLACK)
		i++
		end1 := time.Nanoseconds()
		playoutTime += end1 - start1
		elapsed = time.Nanoseconds() - start
		if config.Timelimit == -1 && i >= int(config.MaxPlayouts) {
			break
		}
		if config.Timelimit != -1 && float64(elapsed)/1e9 >= float64(config.Timelimit) {
			break
		}
	}
	pps := float64(i) / (float64(elapsed) / 1e9)
	fmt.Printf("%d playouts in %.2f s, %.2f pps\n", i, float64(elapsed)/1e9, pps)
	fmt.Printf("percent spent in playout: %.2f\n", float64(playoutTime)/float64(elapsed))
}

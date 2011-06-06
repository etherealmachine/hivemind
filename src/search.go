package main

import (
	"fmt"
	"math"
	"time"
	"log"
	"rand"
	"container/vector"
)

type Node struct {
	parent *Node
	child *Node
	last *Node
	sibling *Node
	wins, visits, mean float64
	amafWins, amafVisits, amafMean float64
	neighborWins, neighborVisits, neighborMean float64
	evalWins, evalVisits, evalMean float64
	blendedMean float64
	value float64
	color byte
	vertex int
	territory []float64
	seeds, totalseeds int
	config *Config
}

func NewRoot(color byte, t Tracker, config *Config) *Node {
	node := new(Node)
	node.color = Reverse(color)
	node.vertex = -1
	node.config = config
	return node
}

func NewNode(parent *Node, color byte, vertex int) *Node {
	node := new(Node)
	node.parent = parent
	node.color = color
	node.vertex = vertex
	node.config = parent.config
	return node
}

// step through the tree for some number of playouts
func genmove(root *Node, t Tracker, m PatternMatcher, e BoardEvaluator) {
	if *root.config.stats { log.Printf("kept %.0f visits\n", root.visits) }
	root.wins = 0
	root.visits = 0
	if t.Winner() != EMPTY {
		return
	}
	start := time.Nanoseconds()
	root.territory = make([]float64, t.Sqsize())
	treeSearch(root, t, m, e)
	elapsed := float64(time.Nanoseconds() - start) / 1e9
	if *root.config.stats {
		pps := float64(root.visits) / elapsed
		log.Printf("%.0f playouts in %.2f s, %.2f pps\n", root.visits, elapsed, pps)
		if *root.config.timelimit > 0 {
			if elapsed > float64(*root.config.timelimit) {
				log.Printf("%.2f seconds overtime\n", elapsed - float64(*root.config.timelimit))
			} else {
				log.Printf("%.2f seconds left\n", float64(*root.config.timelimit) - elapsed)
			}
		}
		log.Printf("winrate: %.2f\n", root.wins / root.visits)
		territory_mean := 0.0
		for v := 0; v < t.Sqsize(); v++ {
			territory_mean += root.territory[v] / root.visits
		}
		territory_mean /= float64(t.Sqsize())
		territory_var := 0.0
		for v := 0; v < t.Sqsize(); v++ {
			territory_var += math.Pow((root.territory[v] / root.visits) - territory_mean, 2)
		}
		territory_var /= float64(t.Sqsize())
		log.Printf("territory: mean: %.2f, var: %.2f\n", territory_mean, territory_var)
		log.Printf("max depth: %d\n", root.maxdepth())
		seeds, totalseeds := root.seedstats()
		log.Printf("seeds: %.2f\n", float64(seeds) / float64(totalseeds))
		if m != nil {
			log.Printf("patterns stats: %.2f\n", float64(matches) / float64(queries))
			if *root.config.logpat {
				log.Println("patterns:", patternLog)
				for i := range patternLog {
					patternLog[i] = 0
				}
			}
			matches, queries = 0, 0
		}
	}
}

func treeSearch(root *Node, t Tracker, m PatternMatcher, e BoardEvaluator) {
	start := time.Nanoseconds()
	for {
		cp := t.Copy()
		root.step(cp, m, e)
		if *root.config.gfx {
			board := cp.Territory()
			for v := 0; v < cp.Sqsize(); v++ {
				if board[v] == Reverse(root.color) {
					root.territory[v]++
				}
			}
			EmitGFX(root, cp)
		}
		if *root.config.timelimit != -1 {
			elapsed := time.Nanoseconds() - start
			if uint64(elapsed) > uint64(*root.config.timelimit) * uint64(1e9) { break }
		} else if root.visits >= float64(*root.config.maxPlayouts) {
			break
		}
	}
}

// navigate through the tree until a leaf node is found to playout
func (root *Node) step(t Tracker, m PatternMatcher, e BoardEvaluator) {
	path := new(vector.Vector)
	curr := root.Next(root, t, e)
	if curr == nil { root.visits = math.Inf(1); return }
	for {
		path.Push(curr)
		// apply node's position to the board
		t.Play(curr.color, curr.vertex)
		if curr.visits <= *root.config.expandAfter {
			var color byte
			if *root.config.seedPlayouts {
				color = curr.seedPlayout(t)
			} else {
				color = Reverse(curr.color)
			}
			t.Playout(color, m)
			break
		}
		next := curr.Next(root, t, e)
		curr = next
		if curr == nil { break }
	}
	winner := t.Winner()
	var result float64
	if winner == root.color {
		result = 0.0
	} else {
		result = 1.0
	}
	for j := 0; j < path.Len(); j++ {
		path.At(j).(*Node).update(result, t, e)
		result = 1 - result
	}
	if winner == Reverse(root.color) {
		root.wins++
	}
	root.visits++
	root.mean = root.wins / root.visits
}

// add all legal children to node
func (node *Node) expand(t Tracker, e BoardEvaluator) {
	color := Reverse(node.color)
	for i := 0; i < t.Sqsize(); i++ {
		if t.Legal(color, i) {
			child := NewNode(node, color, i)
			if node.child == nil {
				node.child = child
			} else {
				node.last.sibling = child
			}
			node.last = child
			cp := t.Copy()
			cp.Play(child.color, child.vertex)
			if cp.Winner() != EMPTY {
				child.visits = math.Inf(1)
				child.wins = 0
				if cp.Winner() == child.color { child.wins = math.Inf(1) }
			} else {
				child.wins = 1
				child.visits = 1 + 0.01 * rand.Float64()
				if *node.config.neighbors {
					granduncle := child.granduncle()
					if granduncle != nil {
						child.neighborVisits += granduncle.visits + granduncle.neighborVisits
						child.neighborWins += granduncle.wins + granduncle.neighborWins
					}
				}
				if *node.config.eval {
					child.evalVisits += *node.config.k
					child.evalWins += *node.config.k * e.Eval(Reverse(child.color), cp)
				}
			}
			child.recalc()
		}
	}
}

// select the next node in the tree to navigate to from this node's children
func (node *Node) Next(root *Node, t Tracker, e BoardEvaluator) *Node {
	if node.child == nil {
		node.expand(t, e)
	}
	var best *Node
	for child := node.child; child != nil; child = child.sibling {
		if best == nil || child.value > best.value {
			best = child
		}
	}
	return best
}

func (node *Node) Best() *Node {
	var best *Node
	for child := node.child; child != nil; child = child.sibling {
		if best == nil || child.visits > best.visits {
			best = child
		}
	}
	if best == nil { node.vertex = -1; return node }
	return best
}

func (node *Node) update(result float64, t Tracker, e BoardEvaluator) {
	node.wins += result
	node.visits++
	node.recalc()
	for sibling := node.parent.child; sibling != nil; sibling = sibling.sibling {
		sibling.neighborWins += result
		sibling.neighborVisits++
		if t.WasPlayed(sibling.color, sibling.vertex) {
			sibling.amafWins += result
			sibling.amafVisits++
		}
		sibling.recalc()
	}
}

func (node *Node) recalc() {
	node.mean = node.wins / node.visits
	node.amafMean = node.amafWins / node.amafVisits
	node.neighborMean = node.neighborWins / node.neighborVisits
	node.evalMean = node.evalWins / node.evalVisits
	if math.IsNaN(node.mean) { node.mean = 0 }
	if math.IsNaN(node.amafMean) { node.amafMean = 0 }
	if math.IsNaN(node.neighborMean) { node.neighborMean = 0 }
	if math.IsNaN(node.evalMean) { node.evalMean = 0 }
	beta := math.Sqrt(*node.config.k / (3*node.visits + *node.config.k))
	if !(*node.config.amaf || *node.config.neighbors || *node.config.eval) || *node.config.k == 0 || beta < 0 { beta = 0 }
	estimatedMean := 0.0
	samples := 0.0
	if *node.config.amaf {
		estimatedMean += node.amafMean
		samples++
	}
	if *node.config.neighbors {
		estimatedMean += node.neighborMean
		samples++
	}
	if *node.config.eval {
		estimatedMean += node.evalMean
		samples++
	}
	estimatedMean /= samples
	if math.IsNaN(estimatedMean) { estimatedMean = 0 }
	node.blendedMean = beta * estimatedMean + (1 - beta) * node.mean
	r := math.Log(node.parent.visits) / node.visits
	v := node.blendedMean - (node.blendedMean*node.blendedMean) + math.Sqrt(2*r)
	node.value = node.blendedMean + *node.config.c * math.Sqrt(r * math.Fmin(0.25, v))
}

// return node's grandparent's sibling corrosponding to node's move
func (node *Node) granduncle() *Node {
	if node.parent == nil || node.parent.parent == nil || node.parent.parent.parent == nil {
		return nil
	}
	great_grandparent := node.parent.parent.parent
	for granduncle := great_grandparent.child; granduncle != nil; granduncle = granduncle.sibling {
		if granduncle.vertex == node.vertex {
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
	color := Reverse(node.color)
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
		if great_grandparent.seed(t, []int{great_grandparent.vertex, grandparent.vertex, parent.vertex}) {
			color = Reverse(color)
			// if that succeeds, try seeding with parent
			if grandparent.seed(t, []int{great_grandparent.vertex, grandparent.vertex, parent.vertex}) {
				color = Reverse(color)
				if parent.seed(t, []int{great_grandparent.vertex, grandparent.vertex, parent.vertex}) {
					color = Reverse(color)
				}
			}
		}
	} else { // else try seeding using parent
		if parent.seed(t, []int{parent.vertex}) {
			color = Reverse(color)
		}
	}
	return color
}

// use win-rate distribution of node to play a legal move in tracker
func (node *Node) seed(t Tracker, path []int) bool {
	if node.parent == nil { return false }
	dist := new(vector.Vector)
	sum := 0.0
	for sibling := node.parent.child; sibling != nil; sibling = sibling.sibling {
		for i := 0; i < len(path); i++ {
			if sibling.vertex == path[i] {
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
			if t.Legal(node.color, i) {
				t.Play(node.color, i)
				node.seeds++
				return true
			}
			return false
		}
	}
	return false
}

func (root *Node) merge(node *Node) {
	for child1, child2 := root.child, node.child;
			child1 != nil && child2 != nil;
			child1, child2 = child1.sibling, child2.sibling {
			child1.wins += child2.wins
			child1.visits += child2.visits
	}
	root.wins += node.wins
	root.visits += node.visits
}

func (node *Node) maxdepth() int {
	max := 0
	if node.child != nil {
		for child := node.child; child != nil; child = child.sibling {
			d := child.maxdepth()
			if d > max { max = d }
		}
	}
	return max + 1
}

func (node *Node) seedstats() (int, int) {
	count, total := node.seeds, node.totalseeds
	if node.child != nil {
		for child := node.child; child != nil; child = child.sibling {
			c, t := child.seedstats()
			count += c
			total += t
		}
	}
	return count, total
}

func (node *Node) Play(color byte, vertex int, t Tracker) *Node {
	var best *Node
	for child := node.child; child != nil; child = child.sibling {
		if best == nil || child.visits > best.visits {
			best = child
		}
	}
	for child := node.child; child != nil; child = child.sibling {
		if child.color == color && child.vertex == vertex {
			log.Print(fmt.Sprintf("predicted: %s%s(%.0f)",
				Ctoa(best.color), t.Vtoa(best.vertex), best.visits))
			log.Print(fmt.Sprintf("actual:    %s%s(%.0f)",
				Ctoa(child.color), t.Vtoa(child.vertex), child.visits))
			child.parent = nil
			return child
		}
	}
	return nil
}

func TestPPS(config *Config) {
	t := NewTracker(config)
	playoutTime := int64(0)
	start := time.Nanoseconds()
	for i := 0; i < int(*config.maxPlayouts); i++ {
		cp := t.Copy()
		start1 := time.Nanoseconds()
		cp.Playout(BLACK, matcher)
		end1 := time.Nanoseconds()
		playoutTime += end1 - start1
	}
	end := time.Nanoseconds()
	elapsed := float64(end - start) / 1000000000.0
	pps := float64(*config.maxPlayouts) / elapsed
	fmt.Printf("%d playouts in %.2f s, %.2f pps\n", *config.maxPlayouts, elapsed, pps)
	fmt.Printf("percent spent in playout: %.2f\n", float64(playoutTime) / float64(end - start))
}

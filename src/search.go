package main

import (
	"fmt"
	"math"
	"time"
	"log"
	"rand"
)

type Node struct {
	parent *Node
	child *Node
	last *Node
	sibling *Node
	value, visits, mean, UCB float64
	amafValue, amafVisits, amafMean float64
	neighborValue, neighborVisits, neighborMean float64
	blendedMean float64
	color byte
	vertex int
	territory []float64
}

func NewRoot(color byte, t Tracker) *Node {
	node := new(Node)
	node.color = Reverse(color)
	node.vertex = -1
	return node
}

func NewNode(parent *Node, color byte, vertex int) *Node {
	node := new(Node)
	node.parent = parent
	node.color = color
	node.vertex = vertex
	return node
}

// step through the tree for some number of playouts
func genmove(root *Node, t Tracker, m PatternMatcher) {
	if *stats { log.Printf("kept %.0f visits\n", root.visits) }
	root.value = 0
	root.visits = 0
	if *hex && t.Winner() != EMPTY {
		root.child = nil
	}
	start := time.Nanoseconds()
	root.territory = make([]float64, t.Sqsize())
	if *uct {
		treeSearch(root, t, m)
	} else {
		noTreeSearch(root, t, m)
	}
	elapsed := float64(time.Nanoseconds() - start) / 1e9
	if *stats {
		pps := float64(root.visits) / elapsed
		log.Printf("%.0f playouts in %.2f s, %.2f pps\n", root.visits, elapsed, pps)
		if *timelimit > 0 {
			if elapsed > float64(*timelimit) {
				log.Printf("%.2f seconds overtime\n", elapsed - float64(*timelimit))
			} else {
				log.Printf("%.2f seconds left\n", float64(*timelimit) - elapsed)
			}
		}
		log.Printf("winrate: %.2f\n", root.value / root.visits)
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
		if m != nil {
			log.Printf("patterns stats: %.2f\n", float64(matches) / float64(queries))
			if *logpat {
				if *tablepat {
					log.Println("tablepat hist:", patternLog)
				} else if *nullpat {
					log.Println("nullpat hist:", patternLog)
				} else if *randpat {
					log.Println("randpat hist:", patternLog)
				}
				for i := 0; i < len(patternLog); i++ {
					patternLog[i] = 0
				}
			}
			matches, queries = 0, 0
		}
	}
}

func treeSearch(root *Node, t Tracker, m PatternMatcher) {
	start := time.Nanoseconds()
	for {
		cp := t.Copy()
		root.step(cp, m)
		if *gfx {
			board := cp.Territory()
			for v := 0; v < cp.Sqsize(); v++ {
				if board[v] == Reverse(root.color) {
					root.territory[v]++
				}
			}
			EmitGFX(root, cp)
		}
		if *timelimit != 0 {
			elapsed := time.Nanoseconds() - start
			if uint64(elapsed) > uint64(*timelimit) * uint64(1e9) { break }
		} else if root.visits >= float64(*maxPlayouts) {
			break
		}
	}
}

func noTreeSearch(root *Node, t Tracker, m PatternMatcher) {
	if root.child == nil {
		root.expand(t)
		if root.child == nil { root.visits = math.Inf(1); return; }
	}
	start := time.Nanoseconds()
	for root.visits < float64(*maxPlayouts) {
		for child := root.child; child != nil; child = child.sibling {
			cp := t.Copy()
			cp.Play(child.color, child.vertex)
			cp.Playout(Reverse(child.color), m)
			if cp.Winner() == child.color {
				child.visits++
			}
			root.visits++
			if *timelimit != 0 {
				elapsed := time.Nanoseconds() - start
				if uint64(elapsed) > uint64(*timelimit) * uint64(1e9) { break }
			}
			if root.visits >= float64(*maxPlayouts) { break }
			if *gfx {
				board := cp.Territory()
				for v := 0; v < t.Sqsize(); v++ {
					if board[v] == Reverse(root.color) {
						root.territory[v]++
					}
				}
				EmitGFX(root, t)
			}
		}
	}
}

// navigate through the tree until a leaf node is found to playout
func (root *Node) step(t Tracker, m PatternMatcher) {
	path := make([]*Node, 2 * t.Boardsize() * t.Boardsize())
	i := 0
	curr := root.Next(root, t)
	if curr == nil { root.visits = math.Inf(1); return }
	for {
		path[i] = curr
		i++
		// apply node's position to the board
		t.Play(curr.color, curr.vertex)
		if curr.visits <= *expandAfter {
			t.Playout(Reverse(curr.color), m)
			break
		}
		next := curr.Next(root, t)
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
	for j := 0; j < i; j++ {
		path[j].update(result, t)
		result = 1 - result
	}
	if winner == Reverse(root.color) {
		root.value++
	}
	root.visits++
	root.mean = root.value / root.visits
}

// add all legal children to node
func (node *Node) expand(t Tracker) {
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
		}
	}
}

// select the next node in the tree to navigate to from this node's children
func (node *Node) Next(root *Node, t Tracker) *Node {
	if node.child == nil {
		node.expand(t)
	}
	bestValue := math.Inf(-1)
	var best *Node
	for child := node.child; child != nil; child = child.sibling {
		var value float64
		if child.visits > 0 {
			value = child.UCB
		} else {
			greatuncle := child.greatuncle()
			if greatuncle != nil {
				value = greatuncle.mean
			} else {
				value = 1
			}
			value += 0.01 * rand.Float64()
		}
		if value > bestValue {
			best = child
			bestValue = value
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
	return best
}

func (node *Node) update(result float64, t Tracker) {
	node.value += result
	node.visits++
	node.mean = node.value / node.visits
	for sibling := node.parent.child; sibling != nil; sibling = sibling.sibling {
		sibling.neighborValue += result
		sibling.neighborVisits++
		sibling.neighborMean = sibling.neighborValue / sibling.neighborVisits
		if t.WasPlayed(sibling.color, sibling.vertex) {
			sibling.amafValue += result
			sibling.amafVisits++
			sibling.amafMean = sibling.amafValue / sibling.amafVisits
		}
	}
	beta := (*k - node.visits) / *k
	if *k == 0 || beta < 0 { beta = 0 }
	node.blendedMean = (beta * 0.5 * (node.amafMean + node.neighborMean) + (1 - beta) * node.mean)
	r := math.Log(node.parent.visits) / node.visits
	v := node.blendedMean - (node.blendedMean*node.blendedMean) + math.Sqrt(2*r)
	node.UCB = node.blendedMean + *c * math.Sqrt(r * math.Fmin(0.25, v))
}

// return node's grandparent's sibling corrosponding to node's move
func (node *Node) greatuncle() *Node {
	if node.parent == nil || node.parent.parent == nil || node.parent.parent.parent == nil {
		return nil
	}
	great_grandparent := node.parent.parent.parent
	for greatuncle := great_grandparent.child; greatuncle != nil; greatuncle = greatuncle.sibling {
		if greatuncle.vertex == node.vertex {
			return greatuncle
		}
	}
	return nil
}

func (root *Node) merge(node *Node) {
	for child1, child2 := root.child, node.child;
			child1 != nil && child2 != nil;
			child1, child2 = child1.sibling, child2.sibling {
			child1.value += child2.value
			child1.visits += child2.visits
	}
	root.value += node.value
	root.visits += node.visits
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
				Ctoa(best.color), Vtoa(best.vertex, t.Boardsize()), best.visits))
			log.Print(fmt.Sprintf("actual:    %s%s(%.0f)",
				Ctoa(child.color), Vtoa(child.vertex, t.Boardsize()), child.visits))
			child.parent = nil
			return child
		}
	}
	return nil
}

func TestPPS() {
	t := NewTracker(*size)
	playoutTime := int64(0)
	start := time.Nanoseconds()
	for i := 0; i < int(*maxPlayouts); i++ {
		cp := t.Copy()
		start1 := time.Nanoseconds()
		cp.Playout(BLACK, matcher)
		end1 := time.Nanoseconds()
		playoutTime += end1 - start1
	}
	end := time.Nanoseconds()
	elapsed := float64(end - start) / 1000000000.0
	pps := float64(*maxPlayouts) / elapsed
	fmt.Printf("%d playouts in %.2f s, %.2f pps\n", *maxPlayouts, elapsed, pps)
	fmt.Printf("percent spent in playout: %.2f\n", float64(playoutTime) / float64(end - start))
}

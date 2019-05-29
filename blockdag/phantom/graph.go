// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"container/list"
	"sort"
	"strings"
	"sync"
)

var keyExists = struct{}{}

// GRAPH
type Graph struct {
	tips *orderedNodeSet
	nodes map[string]*Node
	sync.RWMutex
}

func NewGraph() *Graph {
	return &Graph {
		tips: newOrderedNodeSet(),
		nodes: make(map[string]*Node),
	}
}

func (g *Graph) addNodeById(id string) bool {
	if _, ok := g.nodes[id]; ok {
		return false
	}

	node := newNode(id)
	g.nodes[id] = node
	g.tips.add(node)

	return true
}

func (g *Graph) AddNodeById(id string) bool {
	g.Lock()
	response := g.addNodeById(id)
	g.Unlock()

	return response
}

func (g *Graph) addNode(n *Node) bool {
	if n == nil {
		return false
	}

	if _, ok := g.nodes[n.id]; ok {
		return false
	}

	g.nodes[n.id] = n
	g.tips.add(n)

	return true
}

func (g *Graph) AddNode(n *Node) bool {
	g.Lock()
	response := g.addNode(n)
	g.Unlock()

	return response
}

// add edge n1 -> n2
func (g *Graph) addEdgeById(n1 string, n2 string) bool {

	var node1 *Node
	var node2 *Node

	node1, ok := g.nodes[n1]
	if !ok {
		return false
	}

	node2, ok = g.nodes[n2]
	if !ok {
		return false
	}

	if node1.parents.contains(node2) {
		return false
	}

	// edge points from parent to child
	node1.parents.add(node2)
	node2.children.add(node1)
	g.tips.remove(node2)

	return true
}

func (g *Graph) AddEdgeById(n1 string, n2 string) bool {
	g.Lock()
	response := g.addEdgeById(n1, n2)
	g.Unlock()

	return response
}

func (g *Graph) addEdge(n1 *Node, n2 *Node) bool {

	if _, ok := g.nodes[n1.id]; !ok {
		return false
	}
	if _, ok := g.nodes[n2.id]; !ok {
		return false
	}

	if n1.parents.contains(n2) {
		return false
	}

	n1.parents.add(n2)
	n2.children.add(n1)
	g.tips.remove(n2)

	return true
}

func (g *Graph) removeEdge(n1 *Node, n2 *Node) bool {
	if _, ok := g.nodes[n1.id]; !ok {
		return false
	}
	if _, ok := g.nodes[n2.id]; !ok {
		return false
	}

	if !n1.parents.contains(n2) {
		return false
	}

	n1.parents.remove(n2)
	n2.children.remove(n1)

	return true
}

func (g *Graph) AddEdge(n1 *Node, n2 *Node) bool {
	g.Lock()
	response := g.addEdge(n1, n2)
	g.Unlock()

	return response
}

func (g *Graph) AddEdgesById(n1 string, parents []string) []bool{
	g.Lock()
	added := make([]bool, len(parents))
	for i, parent := range parents {
		added[i] = g.addEdgeById(n1, parent)
	}
	g.Unlock()
	return added
}

func (g *Graph) PrintGraph() string {
	var sb strings.Builder
	expanded := make(map[*Node]bool)
	todo := list.New()

	g.RLock()
	tips := g.tips.elements()
	for _, tip := range tips {
		todo.PushBack(tip)
	}

	for todo.Len() > 0 {
		node2 := todo.Front().Value.(*Node)

		if !expanded[node2] {
			//fmt.Printf("node2 ID not expanded: %s\n", node2.id)
			if node2.parents.size() > 0 {
				// sort parents so order is always the same
				var parents = make([]*Node, node2.parents.size())
				x := 0
				for _, k := range node2.parents.elements() {
					parents[x] = k
					x++
				}

				sort.Slice(parents, func(i, j int) bool {
					if parents[i].id < parents[j].id {
						return true
					} else {
						return false
					}
				})


				for _, p := range parents {
					todo.PushBack(p)
					sb.WriteString(node2.id + "->" + p.id + "\n")
				}
			} else {
				sb.WriteString(node2.id + "\n")
			}
		}

		todo.Remove(todo.Front())
		expanded[node2] = true
	}
	g.RUnlock()

	return sb.String()
}

func (g *Graph) getNodeById(nodeId string) *Node {
	node, ok := g.nodes[nodeId]

	if ok {
		return node
	}

	return nil
}

func (g *Graph) GetNodeById(nodeId string) *Node {
	g.RLock()
	response := g.getNodeById(nodeId)
	g.RUnlock()

	return response
}

func (g *Graph) getSize() int {
	return len(g.nodes)
}

func (g *Graph) GetSize() int {
	g.RLock()
	response := g.getSize()
	g.RUnlock()

	return response
}

func (g *Graph) getTips() []*Node {
	return g.tips.elements()
}

func (g *Graph) GetTips() []*Node {
	g.RLock()
	response := g.getTips()
	g.RUnlock()

	return response
}

// sub graph with node's parents as tips
func (g *Graph) getPast(node2 *Node) *Graph {
	if node2 == nil {
		return nil
	}

	var subgraph = NewGraph()
	todo := list.New()
	expanded := make(map[*Node]bool)

	for _, p := range node2.parents.elements() {
		todo.PushBack(p)
		subgraph.tips.add(p)
	}

	for todo.Len() > 0 {
		node := todo.Front().Value.(*Node)

		if node.parents.size() > 0 && !expanded[node] {
			for _, p := range node.parents.elements() {
				todo.PushBack(p)

			}
		}
		todo.Remove(todo.Front())
		expanded[node] = true
		subgraph.nodes[node.id] = node
	}
	
	return subgraph
}

// GetPast returns a sub graph with node's parents as tips
func (g *Graph) GetPast(node2 *Node) *Graph {
	g.RLock()
	defer g.RUnlock()

	return g.getPast(node2)
}

func (g *Graph) getFuture(node2 *Node) *nodeSet {
	if node2 == nil {
		return nil
	}

	var futureNodes = newNodeSet()

	todo := list.New()
	expanded := make(map[*Node]bool)

	for _, c := range node2.children.elements() {
		todo.PushBack(c)

	}

	for todo.Len() > 0 {
		node := todo.Front().Value.(*Node)

		if node.children.size() > 0 && !expanded[node] {
			for _, c := range node.children.elements() {
				todo.PushBack(c)

			}
		}
		todo.Remove(todo.Front())
		expanded[node] = true
		futureNodes.add(node)
	}

	return futureNodes
}

// anticone of node on g: set of all nodes of g - past(node) - future(node) - node
func (g *Graph) getAnticone(node *Node) *nodeSet {
	if node == nil {
		return nil
	}

	anticone := newNodeSet()

	var pastOfNode  = g.getPast(node)

	futureNodes := g.getFuture(node)
	for k := range g.nodes {
		candidate := g.getNodeById(k)
		_, past := pastOfNode.nodes[k];
		future := futureNodes.contains(candidate)
		if !past && !future {
			anticone.add(candidate)
		}
	}

	anticone.remove(node)

	return anticone
}

// GetAnticone returns anticone of node on g: set of all nodes of g - past(node) - future(node) - node
func (g *Graph) GetAnticone(node *Node) *nodeSet {
	g.RLock()
	defer g.RUnlock()

	return g.getAnticone(node)
}

func (g *Graph) GetMissingNodes(subtips []string) []string {
	subtipMap := make(map[string]struct{})

	subtipPasts := make(map[string]*Graph)
	missing := make([]string, 0)

	// get pasts of subtips
	for _, subtip := range subtips {
		node := g.getNodeById(subtip)
		if node != nil {
			subtipPast := g.getPast(node)
			subtipPasts[subtip] = subtipPast
			subtipMap[subtip] = struct{}{}
		}
	}

	// loop through all nodes,
	// include nodes not in any of the subtips' pasts
	for id := range g.nodes {
		if _, ok := subtipMap[id]; ok {
			continue
		}
		includeNode := true
		for _, subgraph := range subtipPasts {
			node := subgraph.GetNodeById(id)
			if node != nil {
				includeNode = false
			}
		}

		if includeNode {
			missing = append(missing, id)
		}
	}

	return missing
}

// returns a copy of the graph with a virtual node at the end, whose parents are the tips of the graph
func (g *Graph) getVirtual() *Graph {
	vg := NewGraph()

	for k,v := range g.nodes {
		vg.nodes[k] = v
	}

	vnode := newNode("VIRTUAL")
	vg.AddNode(vnode)
	for _, tip := range g.GetTips() {
		vg.AddEdge(vnode, tip)
	}

	return vg
}

// GetVirtual returns a copy of the graph with a virtual node at the end, whose parents are the tips of the graph
func (g *Graph) GetVirtual() *Graph {
	g.RLock()
	vg := g.getVirtual()
	g.RUnlock()

	return vg
}

// Remove node from graph
// Can only remove tips
// Undefined what would happen if remove a node from the middle of the DAG

func (g *Graph) removeTip(n1 *Node) {
	if g.tips.contains(n1) {
		// remove from tip set
		g.tips.remove(n1)
		for _, parent := range n1.parents.elements() {
			// remove from children
			parent.children.remove(n1)
			// add to tips set if parent has no children
			if parent.children.size() == 0 {
				g.tips.add(parent)
			}
		}
		// remove from graph
		delete(g.nodes, n1.GetId())
	}
}

func (g *Graph) RemoveTip(n1 *Node) {
	g.Lock()
	defer g.Unlock()

	g.removeTip(n1)
}

func (g *Graph) RemoveTipById(nodeId string) {
	g.Lock()
	defer g.Unlock()

	node, exists := g.nodes[nodeId]
	if exists {
		g.removeTip(node)
	}
}
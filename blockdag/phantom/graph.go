// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"container/list"
	"strings"
	"sync"
)

var keyExists = struct{}{}

// GRAPH
type Graph struct {
	tips *orderedNodeSet
	nodes map[string]*Node
	pastCache *NodeGraphCache
	sync.RWMutex
}

func NewGraph() *Graph {
	return &Graph {
		tips: newOrderedNodeSet(),
		nodes: make(map[string]*Node),
		pastCache: NewNodeGraphCache("past"),
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

	if _, ok := node1.parents[node2]; ok {
		return false
	}

	// edge points from parent to child
	node1.parents[node2] = keyExists
	node2.children[node1] = keyExists
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

	if _, ok := n1.parents[n2]; ok {
		return false
	}

	n1.parents[n2] = keyExists
	n2.children[n1] = keyExists
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

	if _, ok := n1.parents[n2]; !ok {
		return false
	}

	delete(n1.parents, n2)
	delete(n2.children, n1)

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
	todo := newNodeList(g.GetSize())

	g.RLock()
	tips := g.tips.elements()
	for _, tip := range tips {
		todo.push(tip)
	}

	for todo.size() > 0 {
		node2 := todo.shift()

		if !expanded[node2] {
			//fmt.Printf("node2 ID not expanded: %s\n", node2.id)
			size := len(node2.parents)
			if size > 0 {
				// sort parents so order is always the same
				var parents = make([]*Node, size)
				x := 0
				for k := range node2.parents {
					parents[x] = k
					x++
				}

				for _, p := range SortNodes(parents) {
					todo.push(p)
					sb.WriteString(node2.id + "->" + p.id + "\n")
				}
			} else {
				sb.WriteString(node2.id + "\n")
			}
		}

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

	if g.pastCache != nil {
		e, ok := g.pastCache.Get(node2)
		if ok {
			return e.Graph
		}
	}

	var subgraph = NewGraph()
	todo := newNodeList(g.getSize())
	expanded := make(map[*Node]bool)

	// We add nodes to the subgraph in a deterministic way, because the execution flow is:
	// Node.parents (as a map) -> getPast -> getAnticone -> calculateBlueSet -> OrderDAG
	// * blueSet is calculated in order of tips, and
	// * dag order is calculated based off of blueSet, so
	// if the tips order here changes between calls, the result produced by OrderDAG can change between calls too.
	// We want OrderDAG results to be consistent, so we make sure the parents are added in a consistent way here.
	parents := make([]*Node, len(node2.parents))
	index := 0
	for p := range node2.parents {
		parents[index] = p
		index += 1
	}

	for _, p := range SortNodes(parents) {
		todo.push(p)
		subgraph.tips.add(p)
	}

	for todo.size() > 0 {
		node := todo.shift()

		if len(node.parents) > 0 && !expanded[node] {
			for p := range node.parents {
				todo.push(p)
			}
		}

		expanded[node] = true
		subgraph.nodes[node.id] = node
	}

	// Cache the resulting graph
	g.pastCache.Add(node2, subgraph)
	
	return subgraph
}

func (g *Graph) getPastWithHorizon(node2 *Node, horizon int) *Graph {
	if node2 == nil {
		return nil
	}

	// The past is stored in the subGraph
	var subGraph = NewGraph()

	// Track which nodes past has already been traversed, to reduce duplicate work
	var expanded = make(map[*Node]bool)

	var parents = make([]nodeWithDistance, len(node2.parents))
	index := 0
	for p := range node2.parents {
		parents[index] = nodeWithDistance{
			node:     p,
			distance: 1,
		}
		index += 1
	}

	// We add nodes to the subgraph in a deterministic way, because the execution flow is:
	// Node.parents (as a map) -> getPast -> getAnticone -> calculateBlueSet -> OrderDAG
	// * blueSet is calculated in order of tips, and
	// * dag order is calculated based off of blueSet, so
	// if the tips order here changes between calls, the result produced by OrderDAG can change between calls too.
	// We want OrderDAG results to be consistent, so we make sure the parents are added in a consistent way here.
	var todo = list.New()
	for _, p := range sortNodeWithDistance(parents) {
		todo.PushBack(p)
		subGraph.tips.add(p.node)
	}

	for todo.Len() > 0 {
		e := todo.Front()
		nd := e.Value.(nodeWithDistance)
		todo.Remove(e)

		if len(nd.node.parents) > 0 && !expanded[nd.node] && nd.distance + 1 <= horizon {
			for p := range nd.node.parents {
				todo.PushBack(nodeWithDistance{
					node:     p,
					distance: nd.distance + 1,
				})
			}
		}

		expanded[nd.node] = true
		subGraph.nodes[nd.node.id] = nd.node
	}

	return subGraph
}

// GetPast returns a sub graph with node's parents as tips
func (g *Graph) GetPast(node2 *Node) *Graph {
	g.RLock()
	defer g.RUnlock()

	return g.getPast(node2)
}

func (g *Graph) GetPastWithHorizon(node2 *Node, horizon int) *Graph {
	g.RLock()
	defer g.RUnlock()

	return g.getPastWithHorizon(node2, horizon)
}

func (g *Graph) getFuture(node2 *Node) *nodeSet {
	if node2 == nil {
		return nil
	}

	var futureNodes = newNodeSet()

	todo := newNodeList(g.getSize())
	expanded := make(map[*Node]bool)

	for c := range node2.children {
		todo.push(c)
	}

	for todo.size() > 0 {
		node := todo.shift()

		if len(node.children) > 0 && !expanded[node] {
			for c := range node.children {
				todo.push(c)
			}
		}

		expanded[node] = true
		futureNodes.add(node)
	}

	return futureNodes
}

func (g *Graph) getFutureWithHorizon(node2 *Node, horizon int) *nodeSet {
	if node2 == nil {
		return nil
	}

	var futureNodes = newNodeSet()

	// Track which nodes past has already been traversed, to reduce duplicate work
	var expanded = make(map[*Node]bool)

	var todo = list.New()
	for c := range node2.children {
		todo.PushBack(nodeWithDistance{
			node:     c,
			distance: 1,
		})
	}

	for todo.Len() > 0 {
		e := todo.Front()
		nd := e.Value.(nodeWithDistance)
		todo.Remove(e)

		if len(nd.node.children) > 0 && !expanded[nd.node] && nd.distance + 1 <= horizon {
			for c := range nd.node.children {
				todo.PushBack(nodeWithDistance{
					node:     c,
					distance: nd.distance + 1,
				})
			}
		}

		expanded[nd.node] = true
		futureNodes.add(nd.node)
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

// getAnticoneWithHorizon returns the anticone of node on the graph, limited in the past to the depth of
// ancestor nodes given by horizon.
//
// anticone of node on g: set of all nodes of g - past(node) - future(node) - node
func (g *Graph) getAnticoneWithHorizon(node *Node, horizon int) *nodeSet {
	if node == nil {
		return nil
	}

	anticone := newNodeSet()

	var pastOfNode  = g.getPastWithHorizon(node, horizon)

	futureNodes := g.getFutureWithHorizon(node, horizon)
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

// GetAnticoneWithHorizon returns the anticone of node on the graph, limited in the past to the depth of
// ancestor nodes given by horizon.
//
// anticone of node on g: set of all nodes of g - past(node) - future(node) - node
func (g *Graph) GetAnticoneWithHorizon(node *Node, horizon int) *nodeSet {
	g.RLock()
	defer g.RUnlock()

	return g.getAnticoneWithHorizon(node, horizon)
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
	// Share the node past cache with the copy of the graph
	vg.pastCache = g.pastCache

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
		for parent := range n1.parents {
			// remove from children
			delete(parent.children, n1)
			// add to tips set if parent has no children
			if len(parent.children) == 0 {
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
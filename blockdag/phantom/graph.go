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

// NODE
// in the BlockDAG case:
// parent of the node has an edge from the node to it: node -> parent
// child of the node has an edge from it to the node: child -> node
type node struct {
	id string
	parents map[*node]struct{}
	children map[*node]struct{}
}

func newNode(id string) *node {
	return &node{
		id: id,
		parents: make(map[*node]struct{}),
		children: make(map[*node]struct{}),
	}
}

func (n *node) GetId() string {
	return n.id
}

func (n *node) getChildren() *nodeSet {
	set := newNodeSet()

	for k := range n.children {
		set.add(k)
	}

	return set
}

// NODESET
type nodeSet struct {
	nodes map[*node]struct{}
}

func newNodeSet() *nodeSet {
	return &nodeSet{
		nodes: make(map[*node]struct{}),
	}
}

func (nset *nodeSet) size() int {
	return len(nset.nodes)
}

func (nset *nodeSet) add(node *node) {
	if node != nil {
		nset.nodes[node] = keyExists
	}
	//fmt.Printf("Added node %s\n", node.id)
}

func (nset *nodeSet) remove(node *node) {
	delete(nset.nodes, node)
	//fmt.Printf("Removed node %s\n", node.id)
}

// returns elements of set, sorted by id
func (nset *nodeSet) elements() []*node {
	nodes := make([]*node, 0, len(nset.nodes))

	for k := range nset.nodes {
		nodes = append(nodes, k)
		//fmt.Print(k.id)
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].id < nodes[j].id {
			return true
		} else {
			return false
		}
	})

	return nodes
}

func (nset *nodeSet) contains(node *node) bool {
	if _, ok := nset.nodes[node]; ok {
		return true
	} else {
		return false
	}
}

// returns nset - nset2
func (nset *nodeSet) difference(nset2 *nodeSet) *nodeSet {
	diff := newNodeSet()
	if nset2 == nil {
		return diff
	}

	for k := range nset.nodes {
		if _, ok := nset2.nodes[k]; !ok {
			diff.add(k)
		}
	}
	return diff
}

// returns nset intersection nset2
func (nset *nodeSet) intersection(nset2 *nodeSet) *nodeSet {
	intersection := newNodeSet()
	if nset2 == nil {
		return intersection
	}

	for k := range nset.nodes {
		if _, ok := nset2.nodes[k]; ok {
			intersection.add(k)
		}
	}
	return intersection

}

func (nset *nodeSet) clone() *nodeSet {
	clone := newNodeSet()
	for k := range nset.nodes {
		clone.add(k)
	}

	return clone
}

// GRAPH
type Graph struct {
	tips *nodeSet
	nodes map[string]*node
	orderCache *orderCache
	sync.RWMutex
}

func NewGraph() *Graph {
	return &Graph {
		tips: newNodeSet(),
		nodes: make(map[string]*node),
		orderCache: newOrderCache(),
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

func (g *Graph) addNode(n *node) bool {
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

func (g *Graph) AddNode(n *node) bool {
	g.Lock()
	response := g.addNode(n)
	g.Unlock()

	return response
}

// add edge n1 -> n2
func (g *Graph) addEdgeById(n1 string, n2 string) bool {

	var node1 *node
	var node2 *node

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

func (g *Graph) addEdge(n1 *node, n2 *node) bool {

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

func (g *Graph) AddEdge(n1 *node, n2 *node) bool {
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
	expanded := make(map[*node]bool)
	todo := list.New()

	g.RLock()
	tips := g.tips.elements()
	for _, tip := range tips {
		todo.PushBack(tip)
	}

	for todo.Len() > 0 {
		node2 := todo.Front().Value.(*node)

		if !expanded[node2] {
			//fmt.Printf("node2 ID not expanded: %s\n", node2.id)
			if len(node2.parents) > 0 {
				// sort parents so order is always the same
				var parents = make([]*node, len(node2.parents))
				x := 0
				for k := range node2.parents {
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

func (g *Graph) getNodeById(nodeId string) *node {
	node, ok := g.nodes[nodeId]

	if ok {
		return node
	}

	return nil
}

func (g *Graph) GetNodeById(nodeId string) *node {
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

func (g *Graph) getTips() []*node {
	return g.tips.elements()
}

func (g *Graph) GetTips() []*node {
	g.RLock()
	response := g.getTips()
	g.RUnlock()

	return response
}

// sub graph with node's parents as tips
func (g *Graph) getPast(node2 *node) *Graph {
	if node2 == nil {
		return nil
	}

	var subgraph = NewGraph()
	todo := list.New()
	expanded := make(map[*node]bool)

	for p := range node2.parents {
		todo.PushBack(p)
		subgraph.tips.add(p)
	}

	for todo.Len() > 0 {
		node := todo.Front().Value.(*node)

		if len(node.parents) > 0 && !expanded[node] {
			for p := range node.parents {
				todo.PushBack(p)

			}
		}
		todo.Remove(todo.Front())
		expanded[node] = true
		subgraph.nodes[node.id] = node
		//fmt.Printf("Past added %s\n", node2.id)
	}
	
	return subgraph
}

func (g *Graph) getFuture(node2 *node) *nodeSet {
	if node2 == nil {
		return nil
	}

	var futureNodes = newNodeSet()

	todo := list.New()
	expanded := make(map[*node]bool)

	for c := range node2.children {
		todo.PushBack(c)

	}

	for todo.Len() > 0 {
		node := todo.Front().Value.(*node)

		if len(node.children) > 0 && !expanded[node] {
			for c := range node.children {
				todo.PushBack(c)

			}
		}
		todo.Remove(todo.Front())
		expanded[node] = true
		futureNodes.add(node)
		//fmt.Printf("Future added %s\n", node2.id)
	}

	return futureNodes
}

// anticone of node on g: set of all nodes of g - past(node) - future(node) - node
func (g *Graph) getAnticone(node *node) *nodeSet {
	if node == nil {
		return nil
	}

	anticone := newNodeSet()

	var pastOfNode  = g.getPast(node)

	futureNodes := g.getFuture(node)
	for k := range g.nodes {
		candidate := g.getNodeById(k)
		//fmt.Printf("Testing node %s\n", k)
		_, past := pastOfNode.nodes[k];
		future := futureNodes.contains(candidate)
		if !past && !future {
			//fmt.Printf("node %s\n", k)
			anticone.add(candidate)
		}
	}

	anticone.remove(node)

	return anticone
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

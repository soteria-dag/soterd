// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func getIds(nodes []*node) []string {
	var ids = make([]string, len(nodes))
	for i, node := range nodes {
		ids[i] = node.GetId()
	}
	return ids
}

// Figure 3 in PHANTOM paper
func createGraph() *Graph {
	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("B")
	g.AddNodeById("C")
	g.AddNodeById("D")
	g.AddNodeById("E")
	g.AddNodeById("F")
	g.AddNodeById("H")
	g.AddNodeById("I")
	g.AddNodeById("J")
	g.AddNodeById("K")
	g.AddNodeById("L")
	g.AddNodeById("M")

	g.AddEdgeById("B", "GENESIS")
	g.AddEdgeById("C", "GENESIS")
	g.AddEdgeById("D", "GENESIS")
	g.AddEdgeById("E", "GENESIS")

	g.AddEdgesById("F", []string{"B", "C"})
	g.AddEdgesById("H", []string{"C", "D", "E"})
	g.AddEdgeById("I", "E")

	g.AddEdgesById("J", []string{"F", "H"})
	g.AddEdgesById("K", []string{"B", "H", "I"})
	g.AddEdgesById("L", []string{"D", "I"})
	g.AddEdgesById("M", []string{"F", "K"})

	return g
}

func createFigure4DAG() *Graph {
	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("B")
	g.AddNodeById("C")
	g.AddNodeById("D")
	g.AddNodeById("E")
	g.AddNodeById("F")
	g.AddNodeById("H")
	g.AddNodeById("I")
	g.AddNodeById("J")
	g.AddNodeById("K")
	g.AddNodeById("L")
	g.AddNodeById("M")
	g.AddNodeById("N")
	g.AddNodeById("O")
	g.AddNodeById("P")
	g.AddNodeById("Q")
	g.AddNodeById("R")
	g.AddNodeById("S")
	g.AddNodeById("T")
	g.AddNodeById("U")

	g.AddEdgeById("B", "GENESIS")
	g.AddEdgeById("C", "GENESIS")
	g.AddEdgeById("D", "GENESIS")
	g.AddEdgeById("E", "GENESIS")

	g.AddEdgesById("F", []string{"B", "C"})
	g.AddEdgesById("H", []string{"E"})
	g.AddEdgesById("I", []string{"C", "D"})

	g.AddEdgesById("J", []string{"D", "F"})

	g.AddEdgesById("K", []string{"E", "I", "J"})
	g.AddEdgesById("L", []string{"F"})
	g.AddEdgesById("M", []string{"K", "L"})
	g.AddEdgesById("N", []string{"D", "H"})

	g.AddEdgesById("O", []string{"K"})
	g.AddEdgesById("P", []string{"K"})
	g.AddEdgesById("Q", []string{"N"})
	g.AddEdgesById("R", []string{"N","O", "P"})

	g.AddEdgesById("S", []string{"Q"})
	g.AddEdgesById("T", []string{"S"})
	g.AddEdgesById("U", []string{"T"})

	return g
}

// genNodes generates a random number of nodes, and connects them to a random number of tips as parents
func genNodes(g *Graph, maxNodes int) error {
	if g.GetSize() == 0 {
		g.AddNodeById("GENESIS")
		return nil
	}

	tips := g.GetTips()

	pick, err := RandInt(maxNodes)
	if err != nil {
		return err
	}
	// We add 1 to the chosen number because RandInt picks number from 0 to max exclusive.
	nodeCount := pick + 1

	for i := 0; i < nodeCount; i++ {
		pick, err := RandInt(len(tips))
		if err != nil {
			return err
		}
		parentCount := pick + 1

		parents := PickNodes(tips, parentCount)
		parentIds := getIds(parents)

		id, err := RandString(7)
		if err != nil {
			return err
		}

		g.AddNodeById(id)
		g.AddEdgesById(id, parentIds)
	}

	return nil
}

// genRepeatedDAG returns a graph with a pattern of connectivity between blocks repeated the given number of times.
func genRepeatedDAG(count int) *Graph {
	g := NewGraph()
	g.AddNodeById("GENESIS")
	tips := g.tips
	for i := 0; i < count; i++ {
		n := fmt.Sprintf("%d", i)
		g.AddNodeById("B" + n)
		g.AddNodeById("C" + n)
		g.AddNodeById("D" + n)
		g.AddNodeById("E" + n)
		g.AddNodeById("F" + n)
		g.AddNodeById("H" + n)
		g.AddNodeById("I" + n)
		g.AddNodeById("J" + n)
		g.AddNodeById("K" + n)
		g.AddNodeById("L" + n)
		g.AddNodeById("M" + n)

		tipIds := getIds(tips.elements())
		g.AddEdgesById("B" + n, tipIds)
		g.AddEdgesById("C" + n, tipIds)
		g.AddEdgesById("D" + n, tipIds)
		g.AddEdgesById("E" + n, tipIds)

		g.AddEdgesById("F" + n, []string{"B" + n, "C" + n})
		g.AddEdgesById("H" + n, []string{"C" + n, "D" + n, "E" + n})
		g.AddEdgeById("I" + n, "E" + n)

		g.AddEdgesById("J" + n, []string{"F" + n, "H" + n})
		g.AddEdgesById("K" + n, []string{"B" + n, "H" + n, "I" + n})
		g.AddEdgesById("L" + n, []string{"D" + n, "I" + n})
		g.AddEdgesById("M" + n, []string{"F" + n, "K" + n})

		tips = g.tips
	}

	return g
}

// PickNodes returns a randomly-chosen amount of nodes to return
func PickNodes(nodes []*node, amount int) []*node {
	available := make([]*node, len(nodes))
	copy(available, nodes)

	if amount <= 0 {
		return []*node{}
	} else if amount >= len(nodes) {
		return available
	} else if len(nodes) == 0 {
		return available
	}

	picks := make([]*node, 0)
	for len(picks) < amount {
		i, err := RandInt(len(available))
		if err != nil {
			continue
		}

		picks = append(picks, available[i])
		available = append(available[:i], available[i+1:]...)
	}

	return picks
}

// RandInt returns a random number between 0 and max value.
// The zero is inclusive, and the max value is exclusive; randInt(3) returns values from 0 to 2.
func RandInt(max int) (int, error) {
	if max == 0 {
		return 0, nil
	}

	val, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return -1, err
	}
	return int(val.Int64()), nil
}

// RandString returns a randomly-generated string of the given length
func RandString(length int) (string, error) {
	runes := []rune(
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"abcdefghijklmnopqrstuvwxyz" +
			"0123456789")

	var s strings.Builder
	for i := 0; i < length; i++ {
		choice, err := RandInt(len(runes))
		if err != nil {
			return "", err
		}
		s.WriteRune(runes[choice])
	}

	return s.String(), nil
}

func TestCalculateBlueSet(t *testing.T) {
	var graph = createGraph()
	var genesis = graph.GetNodeById("GENESIS")
	var blueSet = calculateBlueSet(graph, genesis, 0, nil)

	var expected = []string{"C", "GENESIS", "H", "K", "M"}

	if !reflect.DeepEqual(expected, getIds(blueSet.elements())) {
		t.Errorf("Incorrect blue set for k = 0. Expecting %v, got %v",
			expected, getIds(blueSet.elements()))
	}

	blueSet = calculateBlueSet(graph, genesis, 3, nil)
	expected = []string{"B", "C", "D", "F", "GENESIS", "H", "J", "K", "M"}

	if !reflect.DeepEqual(expected, getIds(blueSet.elements())) {
		t.Errorf("Incorrect blue set for k = 3. Expecting %v, got %v",
			expected, getIds(blueSet.elements()))
	}
}

func TestOrderDAG(t *testing.T) {
	var graph = createGraph()
	var genesis = graph.GetNodeById("GENESIS")

	var orderedNodes = OrderDAG(graph, genesis, 0, nil)
	var expected = []string{"GENESIS", "C","D","E", "H", "B", "I", "K", "F", "M", "J", "L"}

	if !reflect.DeepEqual(expected, getIds(orderedNodes)) {
		t.Errorf("Incorrect ordering for k = 0. Expecting %v, got %v", expected, getIds(orderedNodes))
	}

	var blueSetCache = NewBlueSetCache()

	orderedNodes = OrderDAG(graph, genesis, 3, blueSetCache)
	expected = []string{"GENESIS", "B", "C", "D", "F", "E", "H", "I", "K", "J", "M", "L"}

	if !reflect.DeepEqual(expected, getIds(orderedNodes)) {
		t.Errorf("Incorrect ordering for k = 3. Expecting %v, got %v", expected, getIds(orderedNodes))
	}

	// Check use of orderCache in OrderDAG, by generating a larger dag with repeated patterns of connectivity
	largeGraph := genRepeatedDAG(maxOrderCacheSize * 2)
	largeGraphGenesis := largeGraph.GetNodeById("GENESIS")
	largeGraphBlueSet := NewBlueSetCache()

	orderedNodesUsingCache := OrderDAG(largeGraph, largeGraphGenesis, 3, largeGraphBlueSet)
	orderedNodesNoCache := OrderDAGNoCache(largeGraph, largeGraphGenesis, 3, largeGraphBlueSet)
	if !reflect.DeepEqual(orderedNodesUsingCache, orderedNodesNoCache) {
		t.Error("OrderDAG result is not the same between using cache and not using cache!")
	}
}

func TestFigure4BlueSet(t *testing.T) {
	var graph = createFigure4DAG()
	var genesis = graph.GetNodeById("GENESIS")

	var blueSetCache = NewBlueSetCache()
	var blueSet = calculateBlueSet(graph, genesis, 3, blueSetCache)
	var expected = []string{"B", "C", "D", "E", "F", "GENESIS", "I", "J", "K", "M", "O", "P", "R"}

	if !reflect.DeepEqual(expected, getIds(blueSet.elements())) {
		t.Errorf("Incorrect blue set for figure 4,  k = 3. Expecting %v, got %v",
			expected, getIds(blueSet.elements()))
	}

	var orderedNodes = OrderDAG(graph, genesis, 3, blueSetCache)
	expected = []string{"GENESIS", "B", "C", "D", "E", "F", "I", "J", "K", "L", "M",
		"O", "P", "H", "N", "Q", "R", "S", "T", "U"}

	if !reflect.DeepEqual(expected, getIds(orderedNodes)) {
		t.Errorf("Incorrect ordering for figure 4,  k = 3. Expecting %v, got %v",
			expected, getIds(orderedNodes))
	}
}

// BenchmarkOrderDAG helps to compare the ordering speed when using OrderDAG with and without orderCache.
// You can target the benchmark like this:
// go test -v -timeout 3h -bench=BenchmarkOrderDAG -benchtime 1x -run Benchmark github.com/soteria-dag/soterd/blockdag/phantom
func BenchmarkOrderDAG(b *testing.B) {
	maxNodesPerGeneration := 4
	generations := maxOrderCacheSize * 2

	g := NewGraph()
	g.AddNodeById("GENESIS")
	genesis := g.GetNodeById("GENESIS")
	blueSetCache := NewBlueSetCache()

	for i := 0; i < generations; i++ {
		_ = genNodes(g, maxNodesPerGeneration)
		_ = OrderDAG(g, genesis, 3, blueSetCache)
	}
}

func BenchmarkOrderDAGNoCache(b *testing.B) {
	maxNodesPerGeneration := 4
	generations := maxOrderCacheSize * 2

	g := NewGraph()
	g.AddNodeById("GENESIS")
	genesis := g.GetNodeById("GENESIS")
	blueSetCache := NewBlueSetCache()

	for i := 0; i < generations; i++ {
		_ = genNodes(g, maxNodesPerGeneration)
		_ = OrderDAGNoCache(g, genesis, 3, blueSetCache)
	}
}
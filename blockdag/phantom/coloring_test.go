package phantom

import (
	"reflect"
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

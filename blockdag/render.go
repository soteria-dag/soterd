// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"bytes"
	"fmt"

	"github.com/soteria-dag/soterd/soterutil"
)

// RenderDot returns a representation of the dag in graphviz DOT file format.
// Graphviz: http://graphviz.org/
//
// DOT file format:
// https://graphviz.gitlab.io/_pages/doc/info/lang.html
//
// The DOT bytes could then be written to a file or piped to the graphviz `dot` command to generate a graphical
// representation of the dag.
func (b *BlockDAG) RenderDot() ([]byte, error) {
	var dot bytes.Buffer

	// How many characters of a hash string to use for the 'label' of a block in the graph
	smallHashLen := 7

	// graphIndex tracks block hash -> graph node number, which is used to connect parent-child blocks together.
	graphIndex := make(map[string]int)
	// n keeps track of the 'node' number in graph file language
	n := 0

	// Specify that this graph is directed, and set the ID to 'dag'
	_, err := fmt.Fprintln(&dot, "digraph dag {")
	if err != nil {
		return dot.Bytes(), err
	}

	// Set graph-level attribute to help keep a tighter left-aligned layout of graph in large renderings.
	_, err = fmt.Fprintln(&dot, "ordering=out;")
	if err != nil {
		return dot.Bytes(), err
	}

	// Create a node in the graph for each block in dag
	dagState := b.DAGSnapshot()
	for height := int32(0); height <= dagState.MaxHeight; height++ {
		blocks, err := b.BlocksByHeight(height)
		if err != nil {
			return dot.Bytes(), err
		}

		for _, block := range blocks {
			hash := block.Hash().String()
			smallHashIndex := len(hash) - smallHashLen
			graphIndex[hash] = n

			// A node entry with label, tooltip (mouse-over) attributes
			_, err = fmt.Fprintf(&dot, "n%d [label=\"%s\", tooltip=\"height %d hash %s\"];\n",
				n, hash[smallHashIndex:], height, hash)
			if err != nil {
				return dot.Bytes(), err
			}

			n++
		}
	}

	// Connect the nodes in the graph together
	for height := int32(0); height <= dagState.MaxHeight; height++ {
		blocks, err := b.BlocksByHeight(height)
		if err != nil {
			return dot.Bytes(), err
		}

		for _, block := range blocks {
			hash := block.Hash().String()

			blockN, inGraph := graphIndex[hash]
			if !inGraph {
				// Skip connecting blocks that have been added to the dag since we started rendering it
				continue
			}

			for _, parent := range block.MsgBlock().Parents.Parents {
				parentN, inGraph := graphIndex[parent.Hash.String()]
				if !inGraph {
					// Blocks were added to graph from lowest-to-highest height, so all parents should already be
					// defined in the graph.
					return dot.Bytes(), fmt.Errorf("RenderDot block %s parent %s not in dot graph",
						block.Hash(), parent.Hash)
				}

				// A connection between a block node and its parent node
				_, err := fmt.Fprintf(&dot, "n%d -> n%d;\n", blockN, parentN)
				if err != nil {
					return dot.Bytes(), err
				}
			}
		}
	}

	// Close the graph statement list
	dot.WriteString("}")

	return dot.Bytes(), nil
}

// RenderSvg returns a representation of the dag in SVG format.
//
// This method makes use of the graphviz `dot` command, so graphviz needs to be installed.
// Graphviz: http://graphviz.org/
//
// NOTE(cedric): If you're embedding the svg file contents in an HTML document, you'll want to use the StripSvgXmlDecl
// function to strip the xml declaration tag from the svg contents before embedding the svg as a <figure>.
func (b *BlockDAG) RenderSvg() ([]byte, error) {
	// Render the dag in graphviz dot format
	dot, err := b.RenderDot()
	if err != nil {
		return nil, err
	}

	// Render the graphviz DOT file as an SVG image
	svg, err := soterutil.DotToSvg(dot)
	if err != nil {
		return nil, err
	}

	return svg, nil
}
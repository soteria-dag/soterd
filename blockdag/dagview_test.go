// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/wire"
	"math/rand"
	"testing"
)

var testNoncePrng = rand.New(rand.NewSource(0))

func createBlock(parents []*blockNode) *blockNode {
	header := wire.BlockHeader{Nonce: testNoncePrng.Uint32()}
	var parentHeader wire.ParentSubHeader
	if parents != nil {
		parentHeader := wire.ParentSubHeader{
			Version: 1,
			Parents: make([]*wire.Parent, len(parents)),
			Size:    int32(len(parents)),
		}
		for i, parent := range parents {
			parentHeader.Parents[i] =  &wire.Parent{Hash: parent.hash}
		}
	}
	return newBlockNode(&header, &parentHeader, parents)
}

func maxHeight(blocks []*blockNode) int32 {
	var maxHeight = int32(0)

	for _, block := range blocks {
		if block.height > maxHeight {
			maxHeight = block.height
		}
	}

	return maxHeight
}

func createSet(blocks []*blockNode) map[*blockNode]struct{} {
	hasBlock := make(map[*blockNode]struct{})
	for _, block := range blocks {
		hasBlock[block] = struct{}{}
	}

	return hasBlock
}

func TestDagView(t *testing.T) {

	// 0 <- 1 <- 3 <--
	//  \<- 2 <-/     \
	//       \<- 4 <- 5
	block0 := createBlock(nil)
	block1 := createBlock([]*blockNode{ block0 })
	block2 := createBlock([]*blockNode{ block0 })
	block3 := createBlock([]*blockNode{ block1, block2 })
	block4 := createBlock([]*blockNode{ block1, block2 })
	block5 := createBlock([]*blockNode{ block3, block4 })

	dagView1 := newDAGView([]*blockNode{block3})
	dagView2 := newDAGView([]*blockNode{block3, block4})

	tests := []struct {
		name       string
		view       *dagView   // active view
		genesis    *blockNode   // expected genesis block of active view
		tips       []*blockNode   // expected tip of active view
		contains   []*blockNode // expected nodes in active view
		noContains []*blockNode // expected nodes NOT in active view
		equal      *dagView   // view expected equal to active view
		unequal    *dagView   // view expected NOT equal to active
	}{
		{
			name:       "dag1",
			view:       dagView1,
			genesis:    block0,
			tips:       []*blockNode{block3},
			contains:   []*blockNode{block0, block1, block2, block3},
			noContains: []*blockNode{block4, block5},
			equal:      newDAGView([]*blockNode{block3}),
			unequal:    newDAGView([]*blockNode{block4}),
		},
		{
			name:       "dag2",
			view:       dagView2,
			genesis:    block0,
			tips:       []*blockNode{block3, block4},
			contains:   []*blockNode{block0, block1, block2, block3, block4},
			noContains: []*blockNode{block5},
			equal:      newDAGView([]*blockNode{block3, block4}),
			unequal:    newDAGView([]*blockNode{block3}),
		},
	}
testLoop:
	for _, test := range tests {
		// Ensure height is expected value: max height of tips
		if test.view.Height() != maxHeight(test.tips) {
			t.Errorf("%s: unexpected active view height -- got "+
				"%d, want %d", test.name, test.view.Height(),
				maxHeight(test.tips))
			continue
		}

		// Check genesis block
		if test.view.Genesis() != test.genesis {
			t.Errorf("%s: unexpected active view genesis -- got "+
				"%v, want %v", test.name, test.view.Genesis(),
				test.genesis)
			continue
		}


		// Check tip set
		testTips := make(map[chainhash.Hash]struct{})
		for _, tip := range test.tips {
			testTips[tip.hash] = struct{}{}
		}
		viewTips := test.view.Tips()
		if len(viewTips) != len(test.tips) {
			t.Errorf("%s: wrong number of tips -- got %d, "+
				"want %d", test.name, len(viewTips), len(test.tips))
			continue
		}

		for _, tip := range viewTips {
			if _, ok := testTips[tip.hash]; !ok {
				t.Errorf("%s: unexpected tip -- got %v", test.name, tip.hash)
				continue testLoop
			}
		}

		// Ensure all expected nodes are contained in the active view.
		for _, node := range test.contains {
			if !test.view.Contains(node) {
				t.Errorf("%s: expected %v in active view",
					test.name, node)
				continue testLoop
			}
		}

		// Ensure all nodes from side chain view are NOT contained in
		// the active view.
		for _, node := range test.noContains {
			if test.view.Contains(node) {
				t.Errorf("%s: unexpected %v in active view",
					test.name, node)
				continue testLoop
			}
		}

		// Ensure equality of different views into the same chain works
		// as intended.
		if !test.view.Equals(test.equal) {
			t.Errorf("%s: unexpected unequal views", test.name)
			continue
		}
		if test.view.Equals(test.unequal) {
			t.Errorf("%s: unexpected equal views", test.name)
			continue
		}

		// Ensure all nodes contained in the view return the expected
		// next node.
		//for i, node := range test.contains {
		//	// Final node expects nil for the next node.
		//	var expected *blockNode
		//	if i < len(test.contains)-1 {
		//		expected = test.contains[i+1]
		//	}
		//	if next := test.view.Next(node); next != expected {
		//		t.Errorf("%s: unexpected next node -- got %v, "+
		//			"want %v", test.name, next, expected)
		//		continue testLoop
		//	}
		//}

		// Ensure nodes that are not contained in the view do not
		// produce a successor node.
		for _, node := range test.noContains {
			if next := test.view.Next(node); next != nil {
				t.Errorf("%s: unexpected next node -- got %v, "+
					"want nil", test.name, next)
				continue testLoop
			}
		}

		// Ensure all nodes contained in the view can be retrieved by
		// height.
		//for _, wantNode := range test.contains {
		//	node := test.view.NodesByHeight(wantNode.height)
		//	if node != wantNode {
		//		t.Errorf("%s: unexpected node for height %d -- "+
		//			"got %v, want %v", test.name,
		//			wantNode.height, node, wantNode)
		//		continue testLoop
		//	}
		//}
	}
}

func TestDagViewNodesByHeight(t *testing.T) {
	block0 := createBlock(nil)
	block1 := createBlock([]*blockNode{ block0 })
	block2 := createBlock([]*blockNode{ block0 })

	dagView := newDAGView([]*blockNode{block1, block2})

	height0 := dagView.NodesByHeight(0)
	if height0 == nil || len(height0) != 1 {
		t.Errorf("Expecting one block at height 0")
	}
	if height0[0] != block0 {
		t.Errorf("Blocks not matching at height 0")
	}

	height1 := dagView.NodesByHeight(1)
	if height1 == nil || len(height1) != 2 {
		t.Errorf("Expecting two blocks at height 1")
	}
	height1Set := createSet([]*blockNode{block1, block2})
	for _, block := range height1 {
		if _, ok := height1Set[block]; !ok {
			t.Errorf("Expecting %v in height 1", block.hash)
		}
	}
}

// AddTip only adds tip to the tip set
// adding a block through connectBlock will add the tip AND remove parents from the tip set
func TestDagViewAddTip(t *testing.T) {
	block0 := createBlock(nil)
	block1 := createBlock([]*blockNode{ block0 })
	block2 := createBlock([]*blockNode{ block0 })

	dagView1 := newDAGView(nil)
	dagView1.AddTip(block1)

	tips := dagView1.Tips()

	if tips == nil || len(tips) != 1 {
		t.Errorf("Expecting one tip, got %d", len(tips))
	}

	if tips[0] != block1 {
		t.Errorf("Expecting tip to be block1: %v, got %v", block1.hash, tips[0].hash)
	}

	dagView2 := newDAGView(nil)
	dagView2.AddTip(block1)
	dagView2.AddTip(block2)

	tips = dagView2.Tips()

	if tips == nil || len(tips) != 2 {
		t.Errorf("Expecting two tips, got %d", len(tips))
	}

	tipSet := createSet(tips)

	if _, ok := tipSet[block1]; !ok {
		t.Errorf("Expecting block1 to be in tip set: %v", block1.hash)
	}

	if _, ok := tipSet[block2]; !ok {
		t.Errorf("Expecting block2 to be in tip set: %v", block2.hash)
	}
}

func TestDagViewRemoveTip(t *testing.T) {
	block0 := createBlock(nil)
	block1 := createBlock([]*blockNode{ block0 })
	block2 := createBlock([]*blockNode{ block0 })

	dagView1 := newDAGView(nil)
	dagView1.AddTip(block1)
	dagView1.AddTip(block2)

	tips := dagView1.Tips()
	if tips == nil || len(tips) != 2 {
		t.Errorf("Expecting two tips, got %d", len(tips))
	}

	dagView1.RemoveTip(block1)
	tips = dagView1.Tips()
	if tips == nil || len(tips) != 1 {
		t.Errorf("Expecting one tip, got %d", len(tips))
	}

	dagView1.RemoveTip(block0)
	tips = dagView1.Tips()
	if tips == nil || len(tips) != 1 {
		t.Errorf("Expecting one tip, got %d", len(tips))
	}
}

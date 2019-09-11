// This file is ignored during the regular tests due to the following build tag.
// +build dag confluxattack
// You can run tests from this file in isolation by using the build tags, like so:
// go test -v -count=1 -tags "confluxattack" github.com/soteria-dag/soterd/blockdag/phantom

package phantom

import (
	"fmt"
	"testing"
)

func calcHeight(h int)  int {
	return ((h - 1) * (h - 2) / 2) + 1
}

func calcReleaseHeight(i int, kDelta int) int {
	height := calcHeight(i - 1)
	return height + kDelta
}

// given the height we should generate malicious blocks for
// calculate the heights each malicious block should be released
// return map of release height to block height and the max release height
func planMaliciousBlocks(height int, kDelta int) (map[int][]int, int){
	releaseHeights := make(map[int][]int)
	maxReleaseHeight := kDelta

	if height < 1 {
		return nil, 0
	}

	releaseHeights[kDelta] = []int{1}

	for i := 2; i <= height ; i++ {
		releaseHeight := calcReleaseHeight(i, kDelta)
		maxReleaseHeight = releaseHeight
		releaseHeights[releaseHeight] = append(releaseHeights[releaseHeight], i)
	}

	return releaseHeights, maxReleaseHeight
}

// Simulates attack specified in Conflux paper: https://arxiv.org/pdf/1805.03870.pdf, appendix A
// This is worst case scenario where the malicious miner can always produce the malicious blocks in time.
func TestConfluxAttack(t *testing.T) {
	// for this attack: kDelta * (kDelta - 7) >= 4 * kPrime
	kPrime := 1
	kDelta := 8
	k := kPrime + kDelta

	rh, maxHeight := planMaliciousBlocks(5, kDelta)
	honestBlockByHeight := make(map[int][]string)

	g := NewGraph()
	g.AddNodeById("GENESIS")
	honestBlockByHeight[0] = []string{"GENESIS"}

	for h := 1; h <= maxHeight; h++ {
		// add honest nodes
		tips := g.tips.elements()
		for i := 0 ; i <= kPrime; i++ {
			nodeId := fmt.Sprintf("B%d.%d", h, i)
			g.AddNodeById(nodeId)
			for _, elem := range tips {
				g.AddEdgeById(nodeId, elem.id)
			}
			honestBlockByHeight[h] = append(honestBlockByHeight[h], nodeId)
		}

		// add malicious nodes, if any that should be released at this height
		if malHeights, ok := rh[h]; ok {
			for _, malHeight := range malHeights {
				malNodeId := fmt.Sprintf("A%d", malHeight)
				g.addNodeById(malNodeId)
				// parents are malheight - 1
				if malHeight == 1 {
					g.addEdgeById(malNodeId, "GENESIS")
				} else {
					g.addEdgeById(malNodeId, fmt.Sprintf("A%d", malHeight - 1))
					honestNodeHeight := calcHeight(malHeight)
					for _, id := range honestBlockByHeight[honestNodeHeight] {
						g.addEdgeById(malNodeId, id)
					}
				}
			}
		}

		// sort
		//noCacheBlueSet := NewBlueSetCache()
		_, noCacheOrder, err := OrderDAG(g, g.GetNodeById("GENESIS"), k, nil, 0, nil)
		if err != nil {
			t.Fatalf("OrderDAG failed: %s", err)
		}
		fmt.Printf("Order at %d: %v\n", h, GetIds(noCacheOrder))
		// print graph
		fmt.Println(g.PrintGraph())
	}
}
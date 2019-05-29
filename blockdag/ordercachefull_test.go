// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// This file is ignored during the regular tests due to the following build tag.
// +build full
// You can run tests from this file in isolation by using the build tags, like so:
// go test -v -count=1 -timeout 3h -tags "full" -run TestOrderDAGFull github.com/soteria-dag/soterd/blockdag

package blockdag

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/soteria-dag/soterd/blockdag/phantom"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/wire"
)

var minOrderCacheDistance = int32(200)

type cacheCase struct {
	name string
	graph *phantom.Graph
	genesis *phantom.Node
	coloringK int
	blueSetCache *phantom.BlueSetCache
	orderCache *phantom.OrderCache
}

type caseResult struct {
	c *cacheCase
	tips []*phantom.Node
	order []*phantom.Node
	time time.Duration
}

func (c *cacheCase) check(minHeight int32) (*caseResult, error) {
	start := time.Now()
	tips, order, err := phantom.OrderDAG(c.graph, c.genesis, c.coloringK, c.blueSetCache, minHeight, c.orderCache)
	if err != nil {
		return nil, err
	}
	elapsed := time.Now().Sub(start)

	return &caseResult{c, tips, order, elapsed}, nil
}

// genBlocks generates a random number of blocks (up to maxBlocks),
// and connects them to a random number of tips as parents.
func genBlocks(b *BlockDAG, maxBlocks int) error {
	var tips = b.dView.tips()
	var height = b.dView.Height()

	// Choose how many blocks to generate
	blockCount, err := RandInt(maxBlocks)
	if err != nil {
		return err
	}

	// We add 1 to the chosen number, because RandInt picks a number from 0 to max exclusive,
	// and we want to generate at least 1 block.
	blockCount = blockCount + 1

	for i := 0; i < blockCount; i++ {
		// Choose how many parents to connect the new block to
		parentsCount, err := RandInt(len(tips))
		if err != nil {
			return err
		}

		parentsCount = parentsCount + 1
		parents := PickBlockNodes(tips, parentsCount)

		// Determine what timestamp of new block should be
		maxTime := maxBlockTime(parents)
		seconds, err := RandInt(2000)
		seconds = seconds + 60
		createTime := maxTime + int64(seconds)

		// Convert the parent blockNode types to wire.MsgBlock types, for createMsgBlockForTest
		parentMsgBlocks := make([]*wire.MsgBlock, 0)
		for _, p := range parents {
			msgBlock := wire.MsgBlock{
				Header: p.Header(),
				Parents: p.ParentSubHeader(),
			}
			parentMsgBlocks = append(parentMsgBlocks, &msgBlock)
		}

		block := createMsgBlockForTest(uint32(height + 1), createTime, parentMsgBlocks, nil)
		_, err = addBlockForTest(b, block)
		if err != nil {
			return fmt.Errorf("failed to add block at height %d to dag: %s", i, err)
		}
	}

	return nil
}

// Return the largest timestamp of the blocks
func maxBlockTime(blocks []*blockNode) int64 {
	var max int64
	for _, b := range blocks {
		if b.timestamp > max {
			max = b.timestamp
		}
	}

	return max
}

func modInt32(x, y int32) int32 {
	return int32(math.Mod(float64(x), float64(y)))
}

// PickBlockNodes returns a randomly-chosen amount of blockNodes to return
func PickBlockNodes(nodes []*blockNode, amount int) []*blockNode {
	available := make([]*blockNode, len(nodes))
	copy(available, nodes)

	if amount <= 0 {
		return []*blockNode{}
	} else if amount >= len(nodes) {
		return available
	} else if len(nodes) == 0 {
		return available
	}

	picks := make([]*blockNode, 0)
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

func TestOrderDAGFull(t *testing.T) {
	// In this test we want to check that the dag tips, order are the same,
	// when sorting a dag with and without the use of order and blueSet caches.
	//
	// This test can take a long time to complete, so we won't run it under certain circumstances
	if testing.Short() {
		t.Skip("skipping due to -test.short flag")
	}
	val, ok := os.LookupEnv("CI")
	if ok && val == "true" {
		t.Skip("skipping due to running in CI environment")
	}

	// Create a new database and dag instance to run tests against.
	dag, teardownFunc, err := chainSetup("dagsortfull",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	genesisHash := dag.dView.Genesis().hash.String()
	var genesisNode = dag.graph.GetNodeById(genesisHash)

	var cases = []*cacheCase{
		&cacheCase{
			name: "order and blueSet cache",
			graph: dag.graph,
			genesis: genesisNode,
			coloringK: coloringK,
			blueSetCache: phantom.NewBlueSetCache(),
			orderCache: phantom.NewOrderCache(),
		},
		&cacheCase{
			name: "blueSet cache only",
			graph: dag.graph,
			genesis: genesisNode,
			coloringK: coloringK,
			blueSetCache: phantom.NewBlueSetCache(),
			orderCache: nil,
		},
		// The time it takes to compare results of cases to one another increases the more cases there are,
		// so we'll normally just want to compare the two we're most interested in at a time.
/*		&cacheCase{
			name: "order cache only",
			graph: dag.graph,
			genesis: genesisNode,
			coloringK: coloringK,
			blueSetCache: nil,
			orderCache: phantom.NewOrderCache(),
		},
		&cacheCase{
			name: "no cache",
			graph: dag.graph,
			genesis: genesisNode,
			coloringK: coloringK,
			blueSetCache: nil,
			orderCache: nil,
		},*/
	}

	maxBlocksPerGeneration := 2
	desiredCacheEntries := 3
	endHeight := minOrderCacheDistance * int32(desiredCacheEntries)
	for i := int32(1); i <= endHeight; i++ {
		err := genBlocks(dag, maxBlocksPerGeneration)
		if err != nil {
			t.Fatalf("failed to generate blocks at height %d: %s", i, err.Error())
		}

		// Run all the cases
		minHeight := dag.DAGSnapshot().MinHeight
		results := make([]*caseResult, 0)
		for _, c := range cases {
			r, err := c.check(minHeight)
			if err != nil {
				t.Fatalf("dag height %d\t%s\tfailed check: %s", i, c.name, err)
			}
			results = append(results, r)
		}

		if modInt32(i, 50) == 0 {
			// Show height and sort times periodically
			for _, r := range results {
				fmt.Printf("dag height %d\t%s\tsort time %s\n", i, r.c.name, r.time)
			}
		}

		// Check that order and coloring is the same, between different usage combinations of order and blueSet cache
		for a, aResult := range results {
			for b, bResult := range results {
				if a == b {
					// No need to compare result to itself
					continue
				}

				// Compare tips
				if !reflect.DeepEqual(phantom.GetIds(aResult.tips), phantom.GetIds(bResult.tips)) {
					fmt.Printf("dag height %d\t%s\tsort time %s\n", i, aResult.c.name, aResult.time)
					fmt.Printf("dag height %d\t%s\tsort time %s\n", i, bResult.c.name, bResult.time)
					fmt.Println(aResult.c.name, "tips:", phantom.GetIds(aResult.tips))
					fmt.Println(bResult.c.name, "tips:", phantom.GetIds(bResult.tips))
					t.Fatalf("Height %d OrderDAG tips not same between '%s' and '%s'",
						i, aResult.c.name, bResult.c.name)
				}

				// Compare order
				if !reflect.DeepEqual(phantom.GetIds(aResult.order), phantom.GetIds(bResult.order)) {
					fmt.Printf("dag height %d\t%s\tsort time %s\n", i, aResult.c.name, aResult.time)
					fmt.Printf("dag height %d\t%s\tsort time %s\n", i, bResult.c.name, bResult.time)

					fmt.Println(aResult.c.name, "order size", len(aResult.order),"order:", phantom.GetIds(aResult.order))
					fmt.Println(bResult.c.name, "order size", len(bResult.order),"order:", phantom.GetIds(bResult.order))

					t.Fatalf("Height %d OrderDAG order not same between '%s' and '%s'",
						i, aResult.c.name, bResult.c.name)
				}

				// Compare blueSet cache
/*				if aResult.c.blueSetCache != nil && bResult.c.blueSetCache != nil {
					if !reflect.DeepEqual(aResult.c.blueSetCache, bResult.c.blueSetCache) {
						fmt.Printf("dag height %d\t%s\tsort time %s\n", i, aResult.c.name, aResult.time)
						fmt.Printf("dag height %d\t%s\tsort time %s\n", i, bResult.c.name, bResult.time)
						fmt.Println(aResult.c.name, "blueSetCache:")
						fmt.Print(aResult.c.blueSetCache.String())
						fmt.Println(bResult.c.name, "blueSetCache:")
						fmt.Print(bResult.c.blueSetCache.String())

						t.Fatalf("Height %d OrderDAG blueSetCache not same between '%s' and '%s'",
							i, aResult.c.name, bResult.c.name)
					}
				}*/
			}
		}
	}
}

// BenchmarkOrderDAG helps to compare the ordering speed when using OrderDAG with and without orderCache.
// You can target the benchmark like this:
// go test -v -timeout 3h -tags "full" -bench=BenchmarkOrderDAG -benchtime 1x -run Benchmark github.com/soteria-dag/soterd/blockdag
func BenchmarkOrderDAG(b *testing.B) {
	// This test can take a long time to complete, so we won't run it under certain circumstances
	if testing.Short() {
		b.Skip("skipping due to -test.short flag")
	}
	val, ok := os.LookupEnv("CI")
	if ok && val == "true" {
		b.Skip("skipping due to running in CI environment")
	}

	// Create a new database and dag instance to run tests against.
	dag, teardownFunc, err := chainSetup("dagbenchorder",
		&chaincfg.SimNetParams)
	if err != nil {
		b.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	maxBlocksPerGeneration := 4
	desiredCacheEntries := 3
	endHeight := minOrderCacheDistance * int32(desiredCacheEntries)
	for i := int32(1); i <= endHeight; i++ {
		err := genBlocks(dag, maxBlocksPerGeneration)
		if err != nil {
			b.Fatalf("failed to generate blocks at height %d: %s", i, err.Error())
		}
	}
}

func BenchmarkOrderDAGNoCache(b *testing.B) {
	// This test can take a long time to complete, so we won't run it under certain circumstances
	if testing.Short() {
		b.Skip("skipping due to -test.short flag")
	}
	val, ok := os.LookupEnv("CI")
	if ok && val == "true" {
		b.Skip("skipping due to running in CI environment")
	}

	// Create a new database and dag instance to run tests against.
	dag, teardownFunc, err := chainSetup("dagbenchordernocache",
		&chaincfg.SimNetParams)
	if err != nil {
		b.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	maxBlocksPerGeneration := 4
	desiredCacheEntries := 3
	endHeight := minOrderCacheDistance * int32(desiredCacheEntries)
	for i := int32(1); i <= endHeight; i++ {
		err := genBlocks(dag, maxBlocksPerGeneration)
		if err != nil {
			b.Fatalf("failed to generate blocks at height %d: %s", i, err.Error())
		}
	}
}
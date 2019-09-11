// This file is ignored during the regular tests due to the following build tag.
// +build rpctest p2p addrcache
// You can run tests from this file in isolation by using the build tags, like so:
// go test -v -count=1 -tags "p2p" github.com/soteria-dag/soterd/integration

package integration

import (
	"testing"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/integration/rpctest"
)

func TestConnection(t *testing.T) {
	var miners []*rpctest.Harness

	// Number of miners to spawn
	minerCount := 4

	// Set to debug or trace to produce more logging output from miners
	extraArgs := []string{
		//"--debuglevel=debug",
	}

	for i := 0; i < minerCount; i++ {
		miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, false)
		if err != nil {
			t.Fatalf("unable to create mining node %v: %v", i, err)
		}

		if err := miner.SetUp(false, 0); err != nil {
			t.Fatalf("unable to complete mining node %v setup: %v", i, err)
		}

		miners = append(miners, miner)
	}
	// NOTE(cedric): We'll call defer on a single anonymous function instead of minerCount times in the above loop
	defer func() {
		for _, miner := range miners {
			_ = (*miner).TearDown()
		}
	}()

	// Connect the nodes to one another
	err := rpctest.ConnectNodes(miners)
	if err != nil {
		t.Fatalf("unable to connect nodes: %v", err)
	}

	for i, node := range miners {
		for j, peer := range miners {
			if i == j {
				continue
			}

			connected, err := rpctest.IsConnected(peer, node)
			if err != nil {
				t.Fatalf("Couldn't determine if %v was connected to %v", peer, node)
			}
			if connected {
				continue
			}

			connected, err = rpctest.IsConnected(node, peer)
			if err != nil {
				t.Fatalf("Couldn't determine if %v was connected to %v", node, peer)
			}
			if !connected {
				t.Fatalf("node %v isn't connected to %v", node, peer)
			}
		}
	}
}

func TestRPCGetAddrCache(t *testing.T) {
	var miners []*rpctest.Harness

	// Number of miners to spawn
	minerCount := 4

	// Set to debug or trace to produce more logging output from miners
	extraArgs := []string{
		//"--debuglevel=debug",
	}

	for i := 0; i < minerCount; i++ {
		miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, false)
		if err != nil {
			t.Fatalf("unable to create mining node %v: %v", i, err)
		}

		if err := miner.SetUp(false, 0); err != nil {
			t.Fatalf("unable to complete mining node %v setup: %v", i, err)
		}

		miners = append(miners, miner)
	}
	// NOTE(cedric): We'll call defer on a single anonymous function instead of minerCount times in the above loop
	defer func() {
		for _, miner := range miners {
			_ = (*miner).TearDown()
		}
	}()

	// Connect the nodes to one another
	err := rpctest.ConnectNodes(miners)
	if err != nil {
		t.Fatalf("unable to connect nodes: %v", err)
	}

	// Ask miners to share addresses with one another (getaddrcache RPC call has node ask its peers what their addresses are)
	for i, m := range miners {
		_, err := m.Node.GetAddrCache()
		if err != nil {
			t.Fatalf("failed to call getaddrcache on miner %d: %s", i, err)
		}
	}

	// Give miners some time to collect addresses
	time.Sleep(3)

	// Confirm that each miner has at least one other miner's address in its cache
	for i, m := range miners {
		cache, err := m.Node.GetAddrCache()
		if err != nil {
			t.Fatalf("failed to call getaddrcache on miner %d: %s", i, err)
		}

		count := 0
		for _, a := range cache.Inbound {
			if a == m.P2PAddress() {
				t.Fatalf("miner %d returned its own address (%s) in getaddrcache call. This shouldn't happen!", i, a)
			}
			count++
		}

		for _, a := range cache.Outbound {
			if a == m.P2PAddress() {
				t.Fatalf("miner %d returned its own address (%s) in getaddrcache call. This shouldn't happen!", i, a)
			}
			count++
		}

		if count == 0 {
			t.Fatalf("failed to find other miners' addresses in getaddrcache response for miner %d", i)
		}
	}
}

func TestGetListenAddrs(t *testing.T) {
	// Set to debug or trace to produce more logging output from miners
	extraArgs := []string{
		//"--debuglevel=debug",
	}

	m, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, false)
	if err != nil {
		t.Fatalf("unable to create soterd node: %s", err)
	}
	if err := m.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete soterd node setup: %s", err)
	}

	defer m.TearDown()

	resp, err := m.Node.GetListenAddrs()
	if err != nil {
		t.Fatalf("failed to call getlistenaddrs: %s", err)
	}

	if len(resp.P2P) == 0 {
		t.Fatalf("no listen address returned!")
	}

	var match bool
	for _, a := range resp.P2P {
		if a == m.P2PAddress() {
			match = true
			break
		}
	}
	if !match {
		t.Fatalf("listen address %s not found in getlistenaddrs response %s", m.P2PAddress(), resp.P2P)
	}
}
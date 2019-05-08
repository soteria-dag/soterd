// Copyright (c) 2014-2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcclient

import (
	"encoding/json"

	"github.com/soteria-dag/soterd/soterjson"
)

// AddNodeCommand enumerates the available commands that the AddNode function
// accepts.
type AddNodeCommand string

// Constants used to indicate the command for the AddNode function.
const (
	// ANAdd indicates the specified host should be added as a persistent
	// peer.
	ANAdd AddNodeCommand = "add"

	// ANRemove indicates the specified peer should be removed.
	ANRemove AddNodeCommand = "remove"

	// ANOneTry indicates the specified host should try to connect once,
	// but it should not be made persistent.
	ANOneTry AddNodeCommand = "onetry"
)

// String returns the AddNodeCommand in human-readable form.
func (cmd AddNodeCommand) String() string {
	return string(cmd)
}

// FutureAddNodeResult is a future promise to deliver the result of an
// AddNodeAsync RPC invocation (or an applicable error).
type FutureAddNodeResult chan *response

// Receive waits for the response promised by the future and returns an error if
// any occurred when performing the specified command.
func (r FutureAddNodeResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

// AddNodeAsync returns an instance of a type that can be used to get the result
// of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See AddNode for the blocking version and more details.
func (c *Client) AddNodeAsync(host string, command AddNodeCommand) FutureAddNodeResult {
	cmd := soterjson.NewAddNodeCmd(host, soterjson.AddNodeSubCmd(command))
	return c.sendCmd(cmd)
}

// AddNode attempts to perform the passed command on the passed persistent peer.
// For example, it can be used to add or a remove a persistent peer, or to do
// a one time connection to a peer.
//
// It may not be used to remove non-persistent peers.
func (c *Client) AddNode(host string, command AddNodeCommand) error {
	return c.AddNodeAsync(host, command).Receive()
}

// FutureNodeResult is a future promise to deliver the result of a NodeAsync
// RPC invocation (or an applicable error).
type FutureNodeResult chan *response

// Receive waits for the response promised by the future and returns an error if
// any occurred when performing the specified command.
func (r FutureNodeResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

// NodeAsync returns an instance of a type that can be used to get the result
// of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See Node for the blocking version and more details.
func (c *Client) NodeAsync(command soterjson.NodeSubCmd, host string,
	connectSubCmd *string) FutureNodeResult {
	cmd := soterjson.NewNodeCmd(command, host, connectSubCmd)
	return c.sendCmd(cmd)
}

// Node attempts to perform the passed node command on the host.
// For example, it can be used to add or a remove a persistent peer, or to do
// connect or diconnect a non-persistent one.
//
// The connectSubCmd should be set either "perm" or "temp", depending on
// whether we are targetting a persistent or non-persistent peer. Passing nil
// will cause the default value to be used, which currently is "temp".
func (c *Client) Node(command soterjson.NodeSubCmd, host string,
	connectSubCmd *string) error {
	return c.NodeAsync(command, host, connectSubCmd).Receive()
}

// FutureGetAddedNodeInfoResult is a future promise to deliver the result of a
// GetAddedNodeInfoAsync RPC invocation (or an applicable error).
type FutureGetAddedNodeInfoResult chan *response

// Receive waits for the response promised by the future and returns information
// about manually added (persistent) peers.
func (r FutureGetAddedNodeInfoResult) Receive() ([]soterjson.GetAddedNodeInfoResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal as an array of getaddednodeinfo result objects.
	var nodeInfo []soterjson.GetAddedNodeInfoResult
	err = json.Unmarshal(res, &nodeInfo)
	if err != nil {
		return nil, err
	}

	return nodeInfo, nil
}

// GetAddedNodeInfoAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetAddedNodeInfo for the blocking version and more details.
func (c *Client) GetAddedNodeInfoAsync(peer string) FutureGetAddedNodeInfoResult {
	cmd := soterjson.NewGetAddedNodeInfoCmd(true, &peer)
	return c.sendCmd(cmd)
}

// GetAddedNodeInfo returns information about manually added (persistent) peers.
//
// See GetAddedNodeInfoNoDNS to retrieve only a list of the added (persistent)
// peers.
func (c *Client) GetAddedNodeInfo(peer string) ([]soterjson.GetAddedNodeInfoResult, error) {
	return c.GetAddedNodeInfoAsync(peer).Receive()
}

// FutureGetAddedNodeInfoNoDNSResult is a future promise to deliver the result
// of a GetAddedNodeInfoNoDNSAsync RPC invocation (or an applicable error).
type FutureGetAddedNodeInfoNoDNSResult chan *response

// Receive waits for the response promised by the future and returns a list of
// manually added (persistent) peers.
func (r FutureGetAddedNodeInfoNoDNSResult) Receive() ([]string, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as an array of strings.
	var nodes []string
	err = json.Unmarshal(res, &nodes)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// GetAddedNodeInfoNoDNSAsync returns an instance of a type that can be used to
// get the result of the RPC at some future time by invoking the Receive
// function on the returned instance.
//
// See GetAddedNodeInfoNoDNS for the blocking version and more details.
func (c *Client) GetAddedNodeInfoNoDNSAsync(peer string) FutureGetAddedNodeInfoNoDNSResult {
	cmd := soterjson.NewGetAddedNodeInfoCmd(false, &peer)
	return c.sendCmd(cmd)
}

// GetAddedNodeInfoNoDNS returns a list of manually added (persistent) peers.
// This works by setting the dns flag to false in the underlying RPC.
//
// See GetAddedNodeInfo to obtain more information about each added (persistent)
// peer.
func (c *Client) GetAddedNodeInfoNoDNS(peer string) ([]string, error) {
	return c.GetAddedNodeInfoNoDNSAsync(peer).Receive()
}

// FutureGetConnectionCountResult is a future promise to deliver the result
// of a GetConnectionCountAsync RPC invocation (or an applicable error).
type FutureGetConnectionCountResult chan *response

// Receive waits for the response promised by the future and returns the number
// of active connections to other peers.
func (r FutureGetConnectionCountResult) Receive() (int64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

	// Unmarshal result as an int64.
	var count int64
	err = json.Unmarshal(res, &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// FutureGetAddrCacheResult is a promise to deliver the result of a GetAddrCacheAsync RPC invocation (or error).
type FutureGetAddrCacheResult chan *response

// Receive waits for the response promised by the future and returns known addresses result provided by the RPC server.
func (r FutureGetAddrCacheResult) Receive() (*soterjson.GetAddrCacheResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var addrCache soterjson.GetAddrCacheResult
	if err := json.Unmarshal(res, &addrCache); err != nil {
		return nil, err
	}

	return &addrCache, nil
}

// GetAddrCacheAsync returns a promise that can be used to retrieve the result of the RPC call at some future time by
// invoking the Receive function on the promise.
func (c *Client) GetAddrCacheAsync() FutureGetAddrCacheResult {
	cmd := soterjson.NewGetAddrCacheCmd()
	return c.sendCmd(cmd)
}

// GetAddrCache returns all known addresses for all peers.
func (c *Client) GetAddrCache() (*soterjson.GetAddrCacheResult, error) {
	return c.GetAddrCacheAsync().Receive()
}

// GetConnectionCountAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetConnectionCount for the blocking version and more details.
func (c *Client) GetConnectionCountAsync() FutureGetConnectionCountResult {
	cmd := soterjson.NewGetConnectionCountCmd()
	return c.sendCmd(cmd)
}

// GetConnectionCount returns the number of active connections to other peers.
func (c *Client) GetConnectionCount() (int64, error) {
	return c.GetConnectionCountAsync().Receive()
}

// FuturePingResult is a future promise to deliver the result of a PingAsync RPC
// invocation (or an applicable error).
type FuturePingResult chan *response

// Receive waits for the response promised by the future and returns the result
// of queueing a ping to be sent to each connected peer.
func (r FuturePingResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

// FutureGetListenAddrsResult is a promise to deliver the result of a GetAddrCacheAsync RPC invocation (or error).
type FutureGetListenAddrsResult chan *response

// Receive waits for the response promised by the future and returns known addresses result provided by the RPC server.
func (r FutureGetListenAddrsResult) Receive() (*soterjson.GetListenAddrsResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var listenAddrs soterjson.GetListenAddrsResult
	if err := json.Unmarshal(res, &listenAddrs); err != nil {
		return nil, err
	}

	return &listenAddrs, nil
}

// GetListenAddrsAsync returns a promise that can be used to retrieve the result of the RPC call at some future time by
// invoking the Receive function on the promise.
func (c *Client) GetListenAddrsAsync() FutureGetListenAddrsResult {
	cmd := soterjson.NewGetListenAddrsCmd()
	return c.sendCmd(cmd)
}

// GetListenAddrs returns a list of p2p addresses the server is listening on..
func (c *Client) GetListenAddrs() (*soterjson.GetListenAddrsResult, error) {
	return c.GetListenAddrsAsync().Receive()
}

// PingAsync returns an instance of a type that can be used to get the result of
// the RPC at some future time by invoking the Receive function on the returned
// instance.
//
// See Ping for the blocking version and more details.
func (c *Client) PingAsync() FuturePingResult {
	cmd := soterjson.NewPingCmd()
	return c.sendCmd(cmd)
}

// Ping queues a ping to be sent to each connected peer.
//
// Use the GetPeerInfo function and examine the PingTime and PingWait fields to
// access the ping times.
func (c *Client) Ping() error {
	return c.PingAsync().Receive()
}

// FutureGetPeerInfoResult is a future promise to deliver the result of a
// GetPeerInfoAsync RPC invocation (or an applicable error).
type FutureGetPeerInfoResult chan *response

// Receive waits for the response promised by the future and returns  data about
// each connected network peer.
func (r FutureGetPeerInfoResult) Receive() ([]soterjson.GetPeerInfoResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as an array of getpeerinfo result objects.
	var peerInfo []soterjson.GetPeerInfoResult
	err = json.Unmarshal(res, &peerInfo)
	if err != nil {
		return nil, err
	}

	return peerInfo, nil
}

// GetPeerInfoAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetPeerInfo for the blocking version and more details.
func (c *Client) GetPeerInfoAsync() FutureGetPeerInfoResult {
	cmd := soterjson.NewGetPeerInfoCmd()
	return c.sendCmd(cmd)
}

// GetPeerInfo returns data about each connected network peer.
func (c *Client) GetPeerInfo() ([]soterjson.GetPeerInfoResult, error) {
	return c.GetPeerInfoAsync().Receive()
}

// FutureGetNetTotalsResult is a future promise to deliver the result of a
// GetNetTotalsAsync RPC invocation (or an applicable error).
type FutureGetNetTotalsResult chan *response

// Receive waits for the response promised by the future and returns network
// traffic statistics.
func (r FutureGetNetTotalsResult) Receive() (*soterjson.GetNetTotalsResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a getnettotals result object.
	var totals soterjson.GetNetTotalsResult
	err = json.Unmarshal(res, &totals)
	if err != nil {
		return nil, err
	}

	return &totals, nil
}

// GetNetTotalsAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetNetTotals for the blocking version and more details.
func (c *Client) GetNetTotalsAsync() FutureGetNetTotalsResult {
	cmd := soterjson.NewGetNetTotalsCmd()
	return c.sendCmd(cmd)
}

// GetNetTotals returns network traffic statistics.
func (c *Client) GetNetTotals() (*soterjson.GetNetTotalsResult, error) {
	return c.GetNetTotalsAsync().Receive()
}

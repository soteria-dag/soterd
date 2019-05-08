// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcclient

import (
	"encoding/json"

	"github.com/soteria-dag/soterd/soterjson"
)

// FutureGetDAGTipsResult is a promise to deliver the result of a
// GetDAGTipsAsync RPC invocation (or an applicable error).
type FutureGetDAGTipsResult chan *response

// Receive waits for the response promised by the future and returns chain info
// result provided by the server.
func (r FutureGetDAGTipsResult) Receive() (*soterjson.GetDAGTipsResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var dagSnapshot soterjson.GetDAGTipsResult
	if err := json.Unmarshal(res, &dagSnapshot); err != nil {
		return nil, err
	}
	return &dagSnapshot, nil
}

// GetDAGTipsAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function
// on the returned instance.
//
// See GetDAGTips for the blocking version and more details.
func (c *Client) GetDAGTipsAsync() FutureGetDAGTipsResult {
	cmd := soterjson.NewGetDAGTipsCmd()
	return c.sendCmd(cmd)
}

// GetDAGTips returns information about the tip of the block DAG
func (c *Client) GetDAGTips() (*soterjson.GetDAGTipsResult, error) {
	return c.GetDAGTipsAsync().Receive()
}

// FutureRenderDagResult is a promise to deliver the result of a RenderDagAsync RPC invocation (or error).
type FutureRenderDagResult chan *response

// Receive waits for the response promised by the future and returns rendered dag result provided by the RPC server.
func (r FutureRenderDagResult) Receive() (*soterjson.RenderDagResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var renderedDag soterjson.RenderDagResult
	if err := json.Unmarshal(res, &renderedDag); err != nil {
		return nil, err
	}

	return &renderedDag, nil
}

// RenderDagAsync returns a promise that can be used to retrieve the result of the RPC call at some future time by
// invoking the Receive function on the promise.
func (c *Client) RenderDagAsync() FutureRenderDagResult {
	cmd := soterjson.NewRenderDagCmd()
	return c.sendCmd(cmd)
}

// RenderDag returns a dag rendered in graphviz DOT file format.
func (c *Client) RenderDag() (*soterjson.RenderDagResult, error) {
	return c.RenderDagAsync().Receive()
}

// FutureGetDAGColoringResult is a promise to deliver the result of a GetDAGColoringAsync RPC invocation (or error).
type FutureGetDAGColoringResult chan *response

// Receive waits for the response promised by the future and returns the DAG coloring provided by the RPC server.
func (r FutureGetDAGColoringResult) Receive() ([]*soterjson.GetDAGColoringResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var dagColoring []*soterjson.GetDAGColoringResult
	if err := json.Unmarshal(res, &dagColoring); err != nil {
		return nil, err
	}
	return dagColoring, nil
}

// GetDAGColoringAsync is the async version of GetDAGColoring.
func (c *Client) GetDAGColoringAsync() FutureGetDAGColoringResult {
	cmd := soterjson.NewGetDAGColoringCmd()
	return c.sendCmd(cmd)
}

// GetDAGColoring returns the coloring of the block DAG
func (c *Client) GetDAGColoring() ([]*soterjson.GetDAGColoringResult, error) {
	return c.GetDAGColoringAsync().Receive()
}
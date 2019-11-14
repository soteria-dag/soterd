// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// NOTE: This file is intended to house the RPC commands that are supported by
// a chain server with soterd extensions.

package soterjson

// NodeSubCmd defines the type used in the addnode JSON-RPC command for the
// sub command field.
type NodeSubCmd string

const (
	// NConnect indicates the specified host that should be connected to.
	NConnect NodeSubCmd = "connect"

	// NRemove indicates the specified peer that should be removed as a
	// persistent peer.
	NRemove NodeSubCmd = "remove"

	// NDisconnect indicates the specified peer should be disonnected.
	NDisconnect NodeSubCmd = "disconnect"
)

// NodeCmd defines the dropnode JSON-RPC command.
type NodeCmd struct {
	SubCmd        NodeSubCmd `jsonrpcusage:"\"connect|remove|disconnect\""`
	Target        string
	ConnectSubCmd *string `jsonrpcusage:"\"perm|temp\""`
}

// NewNodeCmd returns a new instance which can be used to issue a `node`
// JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewNodeCmd(subCmd NodeSubCmd, target string, connectSubCmd *string) *NodeCmd {
	return &NodeCmd{
		SubCmd:        subCmd,
		Target:        target,
		ConnectSubCmd: connectSubCmd,
	}
}

// DebugLevelCmd defines the debuglevel JSON-RPC command.  This command is not a
// standard Bitcoin command.  It is an extension for soterd.
type DebugLevelCmd struct {
	LevelSpec string
}

// NewDebugLevelCmd returns a new DebugLevelCmd which can be used to issue a
// debuglevel JSON-RPC command.  This command is not a standard Bitcoin command.
// It is an extension for soterd.
func NewDebugLevelCmd(levelSpec string) *DebugLevelCmd {
	return &DebugLevelCmd{
		LevelSpec: levelSpec,
	}
}

// GenerateCmd defines the generate JSON-RPC command.
type GenerateCmd struct {
	NumBlocks uint32
}

// NewGenerateCmd returns a new instance which can be used to issue a generate
// JSON-RPC command.
func NewGenerateCmd(numBlocks uint32) *GenerateCmd {
	return &GenerateCmd{
		NumBlocks: numBlocks,
	}
}

// GetAddrCacheCmd defines the getaddrcache JSON-RPC command.
type GetAddrCacheCmd struct{}

// NewGetAddrCacheCmd returns a new instance which can be used to issue a getaddrcache JSON-RPC command.
func NewGetAddrCacheCmd() *GetAddrCacheCmd {
	return &GetAddrCacheCmd{}
}

// GetBestBlockCmd defines the getbestblock JSON-RPC command.
type GetBestBlockCmd struct{}

// NewGetBestBlockCmd returns a new instance which can be used to issue a
// getbestblock JSON-RPC command.
func NewGetBestBlockCmd() *GetBestBlockCmd {
	return &GetBestBlockCmd{}
}

// GetBlockMetricsCmd defines the getblockmetrics JSON-RPC command.
type GetBlockMetricsCmd struct {}

// NewGetBlockMetricsCmd returns a new instance, which can be used to issue a getblockmetrics JSON-RPC command.
func NewGetBlockMetricsCmd() *GetBlockMetricsCmd {
	return &GetBlockMetricsCmd{}
}

// GetCurrentNetCmd defines the getcurrentnet JSON-RPC command.
type GetCurrentNetCmd struct{}

// NewGetCurrentNetCmd returns a new instance which can be used to issue a
// getcurrentnet JSON-RPC command.
func NewGetCurrentNetCmd() *GetCurrentNetCmd {
	return &GetCurrentNetCmd{}
}

// GetDAGColoringCmd defines the getdagcoloring JSON-RPC command.
type GetDAGColoringCmd struct{}

// NewGetDAGColoringCmd returns a new instance which can be used to issue a
// getdagcoloring JSON-RPC command.
func NewGetDAGColoringCmd() *GetDAGColoringCmd {
	return &GetDAGColoringCmd{}
}
// GetDAGTipsCmd defines the getdagtips JSON-RPC command.
type GetDAGTipsCmd struct{}

// NewGetDAGTipsCmd returns a new instance which can be used to issue a
// getdagtips JSON-RPC command.
func NewGetDAGTipsCmd() *GetDAGTipsCmd {
	return &GetDAGTipsCmd{}
}

// GetHeadersCmd defines the getheaders JSON-RPC command.
//
// NOTE: This is a soterd extension ported from
// github.com/decred/dcrd/dcrjson.
type GetHeadersCmd struct {
	BlockLocators []string `json:"blocklocators"`
	HashStop      string   `json:"hashstop"`
}

// NewGetHeadersCmd returns a new instance which can be used to issue a
// getheaders JSON-RPC command.
//
// NOTE: This is a soterd extension ported from
// github.com/decred/dcrd/dcrjson.
func NewGetHeadersCmd(blockLocators []string, hashStop string) *GetHeadersCmd {
	return &GetHeadersCmd{
		BlockLocators: blockLocators,
		HashStop:      hashStop,
	}
}

// GetListenAddrsCmd defines the getlistenaddrs JSON-RPC command.
type GetListenAddrsCmd struct{}

// NewGetListenAddrsCmd returns a new instance which can be used to issue a getlistenaddrs JSON-RPC command.
func NewGetListenAddrsCmd() *GetListenAddrsCmd {
	return &GetListenAddrsCmd{}
}

// RenderDagCmd defines the renderdag JSON-RPC command.
type RenderDagCmd struct{}

// NewRenderDagCmd returns a new instance which can be used to issue a renderdag JSON-RPC command.
func NewRenderDagCmd() *RenderDagCmd {
	return &RenderDagCmd{}
}

// VersionCmd defines the version JSON-RPC command.
//
// NOTE: This is a soterd extension ported from
// github.com/decred/dcrd/dcrjson.
type VersionCmd struct{}

// NewVersionCmd returns a new instance which can be used to issue a JSON-RPC
// version command.
//
// NOTE: This is a soterd extension ported from
// github.com/decred/dcrd/dcrjson.
func NewVersionCmd() *VersionCmd { return new(VersionCmd) }

type GetBlockFeeCmd struct {
	Hash string
}

// NewGetBlockFeeCmd returns a new instance which can be used to issue a getblockfee
// JSON-RPC command.
func NewGetBlockFeeCmd(hash string) *GetBlockFeeCmd {
	return &GetBlockFeeCmd{
		Hash:      hash,
	}
}

type GetBlockFeeAncestorsCmd struct {
	Hashes []string `json:"parentHashes"`
	Height int32    `json:"height"`
}

// NewGetBlockFeeCmd returns a new instance which can be used to issue a getblockfee
// JSON-RPC command.
func NewGetBlockFeeAncestorsCmd(height int32, hashes []string) *GetBlockFeeAncestorsCmd {
	return &GetBlockFeeAncestorsCmd{
		Hashes:      hashes,
		Height:      height,
	}
}

func init() {
	// No special flags for commands in this file.
	flags := UsageFlag(0)

	MustRegisterCmd("debuglevel", (*DebugLevelCmd)(nil), flags)
	MustRegisterCmd("node", (*NodeCmd)(nil), flags)
	MustRegisterCmd("generate", (*GenerateCmd)(nil), flags)
	MustRegisterCmd("getaddrcache", (*GetAddrCacheCmd)(nil), flags)
	MustRegisterCmd("getbestblock", (*GetBestBlockCmd)(nil), flags)
	MustRegisterCmd("getblockfee", (*GetBlockFeeCmd)(nil), flags)
	MustRegisterCmd("getblockfeeancestors", (*GetBlockFeeAncestorsCmd)(nil), flags)
	MustRegisterCmd("getblockmetrics", (*GetBlockMetricsCmd)(nil), flags)
	MustRegisterCmd("getcurrentnet", (*GetCurrentNetCmd)(nil), flags)
	MustRegisterCmd("getdagcoloring", (*GetDAGColoringCmd)(nil), flags)
	MustRegisterCmd("getdagtips", (*GetDAGTipsCmd)(nil), flags)
	MustRegisterCmd("getheaders", (*GetHeadersCmd)(nil), flags)
	MustRegisterCmd("getlistenaddrs", (*GetListenAddrsCmd)(nil), flags)
	MustRegisterCmd("renderdag", (*RenderDagCmd)(nil), flags)
	MustRegisterCmd("version", (*VersionCmd)(nil), flags)
}

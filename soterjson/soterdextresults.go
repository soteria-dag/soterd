// Copyright (c) 2016-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterjson

// VersionResult models objects included in the version response.  In the actual
// result, these objects are keyed by the program or API name.
//
// NOTE: This is a soterd extension ported from
// github.com/decred/dcrd/dcrjson.
type VersionResult struct {
	VersionString string `json:"versionstring"`
	Major         uint32 `json:"major"`
	Minor         uint32 `json:"minor"`
	Patch         uint32 `json:"patch"`
	Prerelease    string `json:"prerelease"`
	BuildMetadata string `json:"buildmetadata"`
}

// GetAddrCacheResult models the data returned from the getaddrcache RPC command.
type GetAddrCacheResult struct {
	Addresses []string `json:"addresses"`
}

// GetDAGColoringResult models the data returned from the getdagcoloring command.
type GetDAGColoringResult struct {
	Hash string `json:"hash"`
	IsBlue bool `json:"isblue"`
}

// GetDAGTipsResult models the data returned from the getdagtips command.
type GetDAGTipsResult struct {
	Tips []string `json:"tips"`
	Hash	string	`json:"hash"`
	MinHeight int32 `json:"minheight"`
	MaxHeight int32 `json:"maxheight"`
	BlkCount uint32 `json:"blkcount"`
}

// GetBlockMetricsResult models the data returned from the getblockmetrics RPC command.
type GetBlockMetricsResult struct {
	BlkGenCount int64 	  `json:"blkgencount"`
	BlkHashes   []string  `json:"blkhashes"`
	BlkGenTimes []float64 `json:"blkgentimes"`
}

// GetListenAddrsResult models the data returned from the getlistenaddrs RPC command.
type GetListenAddrsResult struct {
	P2P []string `json:"p2p"`
}

// RenderDagResult models the data returned from the renderdag RPC call.
type RenderDagResult struct {
	Dot string `json:"dot"`
}
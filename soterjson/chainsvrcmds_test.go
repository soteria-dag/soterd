// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterjson_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/soteria-dag/soterd/soterjson"
	"github.com/soteria-dag/soterd/wire"
)

// TestChainSvrCmds tests all of the chain server commands marshal and unmarshal
// into valid results include handling of optional fields being omitted in the
// marshalled command, while optional fields with defaults have the default
// assigned on unmarshalled commands.
func TestChainSvrCmds(t *testing.T) {
	t.Parallel()

	testID := int(1)
	tests := []struct {
		name         string
		newCmd       func() (interface{}, error)
		staticCmd    func() interface{}
		marshalled   string
		unmarshalled interface{}
	}{
		{
			name: "addnode",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("addnode", "127.0.0.1", soterjson.ANRemove)
			},
			staticCmd: func() interface{} {
				return soterjson.NewAddNodeCmd("127.0.0.1", soterjson.ANRemove)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"addnode","params":["127.0.0.1","remove"],"id":1}`,
			unmarshalled: &soterjson.AddNodeCmd{Addr: "127.0.0.1", SubCmd: soterjson.ANRemove},
		},
		{
			name: "createrawtransaction",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("createrawtransaction", `[{"txid":"123","vout":1}]`,
					`{"456":0.0123}`)
			},
			staticCmd: func() interface{} {
				txInputs := []soterjson.TransactionInput{
					{Txid: "123", Vout: 1},
				}
				amounts := map[string]float64{"456": .0123}
				return soterjson.NewCreateRawTransactionCmd(txInputs, amounts, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"createrawtransaction","params":[[{"txid":"123","vout":1}],{"456":0.0123}],"id":1}`,
			unmarshalled: &soterjson.CreateRawTransactionCmd{
				Inputs:  []soterjson.TransactionInput{{Txid: "123", Vout: 1}},
				Amounts: map[string]float64{"456": .0123},
			},
		},
		{
			name: "createrawtransaction optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("createrawtransaction", `[{"txid":"123","vout":1}]`,
					`{"456":0.0123}`, int64(12312333333))
			},
			staticCmd: func() interface{} {
				txInputs := []soterjson.TransactionInput{
					{Txid: "123", Vout: 1},
				}
				amounts := map[string]float64{"456": .0123}
				return soterjson.NewCreateRawTransactionCmd(txInputs, amounts, soterjson.Int64(12312333333))
			},
			marshalled: `{"jsonrpc":"1.0","method":"createrawtransaction","params":[[{"txid":"123","vout":1}],{"456":0.0123},12312333333],"id":1}`,
			unmarshalled: &soterjson.CreateRawTransactionCmd{
				Inputs:   []soterjson.TransactionInput{{Txid: "123", Vout: 1}},
				Amounts:  map[string]float64{"456": .0123},
				LockTime: soterjson.Int64(12312333333),
			},
		},

		{
			name: "decoderawtransaction",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("decoderawtransaction", "123")
			},
			staticCmd: func() interface{} {
				return soterjson.NewDecodeRawTransactionCmd("123")
			},
			marshalled:   `{"jsonrpc":"1.0","method":"decoderawtransaction","params":["123"],"id":1}`,
			unmarshalled: &soterjson.DecodeRawTransactionCmd{HexTx: "123"},
		},
		{
			name: "decodescript",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("decodescript", "00")
			},
			staticCmd: func() interface{} {
				return soterjson.NewDecodeScriptCmd("00")
			},
			marshalled:   `{"jsonrpc":"1.0","method":"decodescript","params":["00"],"id":1}`,
			unmarshalled: &soterjson.DecodeScriptCmd{HexScript: "00"},
		},
		{
			name: "getaddednodeinfo",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getaddednodeinfo", true)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetAddedNodeInfoCmd(true, nil)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getaddednodeinfo","params":[true],"id":1}`,
			unmarshalled: &soterjson.GetAddedNodeInfoCmd{DNS: true, Node: nil},
		},
		{
			name: "getaddednodeinfo optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getaddednodeinfo", true, "127.0.0.1")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetAddedNodeInfoCmd(true, soterjson.String("127.0.0.1"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getaddednodeinfo","params":[true,"127.0.0.1"],"id":1}`,
			unmarshalled: &soterjson.GetAddedNodeInfoCmd{
				DNS:  true,
				Node: soterjson.String("127.0.0.1"),
			},
		},
		{
			name: "getbestblockhash",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getbestblockhash")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBestBlockHashCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getbestblockhash","params":[],"id":1}`,
			unmarshalled: &soterjson.GetBestBlockHashCmd{},
		},
		{
			name: "getblock",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblock", "123")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBlockCmd("123", nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblock","params":["123"],"id":1}`,
			unmarshalled: &soterjson.GetBlockCmd{
				Hash:      "123",
				Verbose:   soterjson.Bool(true),
				VerboseTx: soterjson.Bool(false),
			},
		},
		{
			name: "getblock required optional1",
			newCmd: func() (interface{}, error) {
				// Intentionally use a source param that is
				// more pointers than the destination to
				// exercise that path.
				verbosePtr := soterjson.Bool(true)
				return soterjson.NewCmd("getblock", "123", &verbosePtr)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBlockCmd("123", soterjson.Bool(true), nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblock","params":["123",true],"id":1}`,
			unmarshalled: &soterjson.GetBlockCmd{
				Hash:      "123",
				Verbose:   soterjson.Bool(true),
				VerboseTx: soterjson.Bool(false),
			},
		},
		{
			name: "getblock required optional2",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblock", "123", true, true)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBlockCmd("123", soterjson.Bool(true), soterjson.Bool(true))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblock","params":["123",true,true],"id":1}`,
			unmarshalled: &soterjson.GetBlockCmd{
				Hash:      "123",
				Verbose:   soterjson.Bool(true),
				VerboseTx: soterjson.Bool(true),
			},
		},
		{
			name: "getblockchaininfo",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblockchaininfo")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBlockChainInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getblockchaininfo","params":[],"id":1}`,
			unmarshalled: &soterjson.GetBlockChainInfoCmd{},
		},
		{
			name: "getblockcount",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblockcount")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBlockCountCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getblockcount","params":[],"id":1}`,
			unmarshalled: &soterjson.GetBlockCountCmd{},
		},
		{
			name: "getblockhash",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblockhash", 123)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBlockHashCmd(123)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getblockhash","params":[123],"id":1}`,
			unmarshalled: &soterjson.GetBlockHashCmd{Index: 123},
		},
		{
			name: "getblockheader",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblockheader", "123")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBlockHeaderCmd("123", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblockheader","params":["123"],"id":1}`,
			unmarshalled: &soterjson.GetBlockHeaderCmd{
				Hash:    "123",
				Verbose: soterjson.Bool(true),
			},
		},
		{
			name: "getblocktemplate",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblocktemplate")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetBlockTemplateCmd(nil)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getblocktemplate","params":[],"id":1}`,
			unmarshalled: &soterjson.GetBlockTemplateCmd{Request: nil},
		},
		{
			name: "getblocktemplate optional - template request",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"]}`)
			},
			staticCmd: func() interface{} {
				template := soterjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
				}
				return soterjson.NewGetBlockTemplateCmd(&template)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"]}],"id":1}`,
			unmarshalled: &soterjson.GetBlockTemplateCmd{
				Request: &soterjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
				},
			},
		},
		{
			name: "getblocktemplate optional - template request with tweaks",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":500,"sizelimit":100000000,"maxversion":2}`)
			},
			staticCmd: func() interface{} {
				template := soterjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   500,
					SizeLimit:    100000000,
					MaxVersion:   2,
				}
				return soterjson.NewGetBlockTemplateCmd(&template)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":500,"sizelimit":100000000,"maxversion":2}],"id":1}`,
			unmarshalled: &soterjson.GetBlockTemplateCmd{
				Request: &soterjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   int64(500),
					SizeLimit:    int64(100000000),
					MaxVersion:   2,
				},
			},
		},
		{
			name: "getblocktemplate optional - template request with tweaks 2",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":true,"sizelimit":100000000,"maxversion":2}`)
			},
			staticCmd: func() interface{} {
				template := soterjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   true,
					SizeLimit:    100000000,
					MaxVersion:   2,
				}
				return soterjson.NewGetBlockTemplateCmd(&template)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":true,"sizelimit":100000000,"maxversion":2}],"id":1}`,
			unmarshalled: &soterjson.GetBlockTemplateCmd{
				Request: &soterjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   true,
					SizeLimit:    int64(100000000),
					MaxVersion:   2,
				},
			},
		},
		{
			name: "getcfilter",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getcfilter", "123",
					wire.GCSFilterRegular)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetCFilterCmd("123",
					wire.GCSFilterRegular)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getcfilter","params":["123",0],"id":1}`,
			unmarshalled: &soterjson.GetCFilterCmd{
				Hash:       "123",
				FilterType: wire.GCSFilterRegular,
			},
		},
		{
			name: "getcfilterheader",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getcfilterheader", "123",
					wire.GCSFilterRegular)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetCFilterHeaderCmd("123",
					wire.GCSFilterRegular)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getcfilterheader","params":["123",0],"id":1}`,
			unmarshalled: &soterjson.GetCFilterHeaderCmd{
				Hash:       "123",
				FilterType: wire.GCSFilterRegular,
			},
		},
		{
			name: "getchaintips",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getchaintips")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetChainTipsCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getchaintips","params":[],"id":1}`,
			unmarshalled: &soterjson.GetChainTipsCmd{},
		},
		{
			name: "getconnectioncount",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getconnectioncount")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetConnectionCountCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getconnectioncount","params":[],"id":1}`,
			unmarshalled: &soterjson.GetConnectionCountCmd{},
		},
		{
			name: "getdifficulty",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getdifficulty")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetDifficultyCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getdifficulty","params":[],"id":1}`,
			unmarshalled: &soterjson.GetDifficultyCmd{},
		},
		{
			name: "getgenerate",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getgenerate")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetGenerateCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getgenerate","params":[],"id":1}`,
			unmarshalled: &soterjson.GetGenerateCmd{},
		},
		{
			name: "gethashespersec",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("gethashespersec")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetHashesPerSecCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"gethashespersec","params":[],"id":1}`,
			unmarshalled: &soterjson.GetHashesPerSecCmd{},
		},
		{
			name: "getinfo",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getinfo")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getinfo","params":[],"id":1}`,
			unmarshalled: &soterjson.GetInfoCmd{},
		},
		{
			name: "getmempoolentry",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getmempoolentry", "txhash")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetMempoolEntryCmd("txhash")
			},
			marshalled: `{"jsonrpc":"1.0","method":"getmempoolentry","params":["txhash"],"id":1}`,
			unmarshalled: &soterjson.GetMempoolEntryCmd{
				TxID: "txhash",
			},
		},
		{
			name: "getmempoolinfo",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getmempoolinfo")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetMempoolInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getmempoolinfo","params":[],"id":1}`,
			unmarshalled: &soterjson.GetMempoolInfoCmd{},
		},
		{
			name: "getmininginfo",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getmininginfo")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetMiningInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getmininginfo","params":[],"id":1}`,
			unmarshalled: &soterjson.GetMiningInfoCmd{},
		},
		{
			name: "getnetworkinfo",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getnetworkinfo")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetNetworkInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getnetworkinfo","params":[],"id":1}`,
			unmarshalled: &soterjson.GetNetworkInfoCmd{},
		},
		{
			name: "getnettotals",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getnettotals")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetNetTotalsCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getnettotals","params":[],"id":1}`,
			unmarshalled: &soterjson.GetNetTotalsCmd{},
		},
		{
			name: "getnetworkhashps",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getnetworkhashps")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetNetworkHashPSCmd(nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[],"id":1}`,
			unmarshalled: &soterjson.GetNetworkHashPSCmd{
				Blocks: soterjson.Int(120),
				Height: soterjson.Int(-1),
			},
		},
		{
			name: "getnetworkhashps optional1",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getnetworkhashps", 200)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetNetworkHashPSCmd(soterjson.Int(200), nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[200],"id":1}`,
			unmarshalled: &soterjson.GetNetworkHashPSCmd{
				Blocks: soterjson.Int(200),
				Height: soterjson.Int(-1),
			},
		},
		{
			name: "getnetworkhashps optional2",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getnetworkhashps", 200, 123)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetNetworkHashPSCmd(soterjson.Int(200), soterjson.Int(123))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[200,123],"id":1}`,
			unmarshalled: &soterjson.GetNetworkHashPSCmd{
				Blocks: soterjson.Int(200),
				Height: soterjson.Int(123),
			},
		},
		{
			name: "getpeerinfo",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getpeerinfo")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetPeerInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getpeerinfo","params":[],"id":1}`,
			unmarshalled: &soterjson.GetPeerInfoCmd{},
		},
		{
			name: "getrawmempool",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getrawmempool")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetRawMempoolCmd(nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getrawmempool","params":[],"id":1}`,
			unmarshalled: &soterjson.GetRawMempoolCmd{
				Verbose: soterjson.Bool(false),
			},
		},
		{
			name: "getrawmempool optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getrawmempool", false)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetRawMempoolCmd(soterjson.Bool(false))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getrawmempool","params":[false],"id":1}`,
			unmarshalled: &soterjson.GetRawMempoolCmd{
				Verbose: soterjson.Bool(false),
			},
		},
		{
			name: "getrawtransaction",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getrawtransaction", "123")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetRawTransactionCmd("123", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getrawtransaction","params":["123"],"id":1}`,
			unmarshalled: &soterjson.GetRawTransactionCmd{
				Txid:    "123",
				Verbose: soterjson.Int(0),
			},
		},
		{
			name: "getrawtransaction optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getrawtransaction", "123", 1)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetRawTransactionCmd("123", soterjson.Int(1))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getrawtransaction","params":["123",1],"id":1}`,
			unmarshalled: &soterjson.GetRawTransactionCmd{
				Txid:    "123",
				Verbose: soterjson.Int(1),
			},
		},
		{
			name: "gettxout",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("gettxout", "123", 1)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetTxOutCmd("123", 1, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettxout","params":["123",1],"id":1}`,
			unmarshalled: &soterjson.GetTxOutCmd{
				Txid:           "123",
				Vout:           1,
				IncludeMempool: soterjson.Bool(true),
			},
		},
		{
			name: "gettxout optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("gettxout", "123", 1, true)
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetTxOutCmd("123", 1, soterjson.Bool(true))
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettxout","params":["123",1,true],"id":1}`,
			unmarshalled: &soterjson.GetTxOutCmd{
				Txid:           "123",
				Vout:           1,
				IncludeMempool: soterjson.Bool(true),
			},
		},
		{
			name: "gettxoutproof",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("gettxoutproof", []string{"123", "456"})
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetTxOutProofCmd([]string{"123", "456"}, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettxoutproof","params":[["123","456"]],"id":1}`,
			unmarshalled: &soterjson.GetTxOutProofCmd{
				TxIDs: []string{"123", "456"},
			},
		},
		{
			name: "gettxoutproof optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("gettxoutproof", []string{"123", "456"},
					soterjson.String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"))
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetTxOutProofCmd([]string{"123", "456"},
					soterjson.String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettxoutproof","params":[["123","456"],` +
				`"000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"],"id":1}`,
			unmarshalled: &soterjson.GetTxOutProofCmd{
				TxIDs:     []string{"123", "456"},
				BlockHash: soterjson.String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"),
			},
		},
		{
			name: "gettxoutsetinfo",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("gettxoutsetinfo")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetTxOutSetInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"gettxoutsetinfo","params":[],"id":1}`,
			unmarshalled: &soterjson.GetTxOutSetInfoCmd{},
		},
		{
			name: "getwork",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getwork")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetWorkCmd(nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getwork","params":[],"id":1}`,
			unmarshalled: &soterjson.GetWorkCmd{
				Data: nil,
			},
		},
		{
			name: "getwork optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("getwork", "00112233")
			},
			staticCmd: func() interface{} {
				return soterjson.NewGetWorkCmd(soterjson.String("00112233"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getwork","params":["00112233"],"id":1}`,
			unmarshalled: &soterjson.GetWorkCmd{
				Data: soterjson.String("00112233"),
			},
		},
		{
			name: "help",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("help")
			},
			staticCmd: func() interface{} {
				return soterjson.NewHelpCmd(nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"help","params":[],"id":1}`,
			unmarshalled: &soterjson.HelpCmd{
				Command: nil,
			},
		},
		{
			name: "help optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("help", "getblock")
			},
			staticCmd: func() interface{} {
				return soterjson.NewHelpCmd(soterjson.String("getblock"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"help","params":["getblock"],"id":1}`,
			unmarshalled: &soterjson.HelpCmd{
				Command: soterjson.String("getblock"),
			},
		},
		{
			name: "invalidateblock",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("invalidateblock", "123")
			},
			staticCmd: func() interface{} {
				return soterjson.NewInvalidateBlockCmd("123")
			},
			marshalled: `{"jsonrpc":"1.0","method":"invalidateblock","params":["123"],"id":1}`,
			unmarshalled: &soterjson.InvalidateBlockCmd{
				BlockHash: "123",
			},
		},
		{
			name: "ping",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("ping")
			},
			staticCmd: func() interface{} {
				return soterjson.NewPingCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"ping","params":[],"id":1}`,
			unmarshalled: &soterjson.PingCmd{},
		},
		{
			name: "preciousblock",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("preciousblock", "0123")
			},
			staticCmd: func() interface{} {
				return soterjson.NewPreciousBlockCmd("0123")
			},
			marshalled: `{"jsonrpc":"1.0","method":"preciousblock","params":["0123"],"id":1}`,
			unmarshalled: &soterjson.PreciousBlockCmd{
				BlockHash: "0123",
			},
		},
		{
			name: "reconsiderblock",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("reconsiderblock", "123")
			},
			staticCmd: func() interface{} {
				return soterjson.NewReconsiderBlockCmd("123")
			},
			marshalled: `{"jsonrpc":"1.0","method":"reconsiderblock","params":["123"],"id":1}`,
			unmarshalled: &soterjson.ReconsiderBlockCmd{
				BlockHash: "123",
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("searchrawtransactions", "1Address")
			},
			staticCmd: func() interface{} {
				return soterjson.NewSearchRawTransactionsCmd("1Address", nil, nil, nil, nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address"],"id":1}`,
			unmarshalled: &soterjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     soterjson.Int(1),
				Skip:        soterjson.Int(0),
				Count:       soterjson.Int(100),
				VinExtra:    soterjson.Int(0),
				Reverse:     soterjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("searchrawtransactions", "1Address", 0)
			},
			staticCmd: func() interface{} {
				return soterjson.NewSearchRawTransactionsCmd("1Address",
					soterjson.Int(0), nil, nil, nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0],"id":1}`,
			unmarshalled: &soterjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     soterjson.Int(0),
				Skip:        soterjson.Int(0),
				Count:       soterjson.Int(100),
				VinExtra:    soterjson.Int(0),
				Reverse:     soterjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("searchrawtransactions", "1Address", 0, 5)
			},
			staticCmd: func() interface{} {
				return soterjson.NewSearchRawTransactionsCmd("1Address",
					soterjson.Int(0), soterjson.Int(5), nil, nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5],"id":1}`,
			unmarshalled: &soterjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     soterjson.Int(0),
				Skip:        soterjson.Int(5),
				Count:       soterjson.Int(100),
				VinExtra:    soterjson.Int(0),
				Reverse:     soterjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("searchrawtransactions", "1Address", 0, 5, 10)
			},
			staticCmd: func() interface{} {
				return soterjson.NewSearchRawTransactionsCmd("1Address",
					soterjson.Int(0), soterjson.Int(5), soterjson.Int(10), nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10],"id":1}`,
			unmarshalled: &soterjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     soterjson.Int(0),
				Skip:        soterjson.Int(5),
				Count:       soterjson.Int(10),
				VinExtra:    soterjson.Int(0),
				Reverse:     soterjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1)
			},
			staticCmd: func() interface{} {
				return soterjson.NewSearchRawTransactionsCmd("1Address",
					soterjson.Int(0), soterjson.Int(5), soterjson.Int(10), soterjson.Int(1), nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1],"id":1}`,
			unmarshalled: &soterjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     soterjson.Int(0),
				Skip:        soterjson.Int(5),
				Count:       soterjson.Int(10),
				VinExtra:    soterjson.Int(1),
				Reverse:     soterjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1, true)
			},
			staticCmd: func() interface{} {
				return soterjson.NewSearchRawTransactionsCmd("1Address",
					soterjson.Int(0), soterjson.Int(5), soterjson.Int(10), soterjson.Int(1), soterjson.Bool(true), nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1,true],"id":1}`,
			unmarshalled: &soterjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     soterjson.Int(0),
				Skip:        soterjson.Int(5),
				Count:       soterjson.Int(10),
				VinExtra:    soterjson.Int(1),
				Reverse:     soterjson.Bool(true),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1, true, []string{"1Address"})
			},
			staticCmd: func() interface{} {
				return soterjson.NewSearchRawTransactionsCmd("1Address",
					soterjson.Int(0), soterjson.Int(5), soterjson.Int(10), soterjson.Int(1), soterjson.Bool(true), &[]string{"1Address"})
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1,true,["1Address"]],"id":1}`,
			unmarshalled: &soterjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     soterjson.Int(0),
				Skip:        soterjson.Int(5),
				Count:       soterjson.Int(10),
				VinExtra:    soterjson.Int(1),
				Reverse:     soterjson.Bool(true),
				FilterAddrs: &[]string{"1Address"},
			},
		},
		{
			name: "sendrawtransaction",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("sendrawtransaction", "1122")
			},
			staticCmd: func() interface{} {
				return soterjson.NewSendRawTransactionCmd("1122", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"sendrawtransaction","params":["1122"],"id":1}`,
			unmarshalled: &soterjson.SendRawTransactionCmd{
				HexTx:         "1122",
				AllowHighFees: soterjson.Bool(false),
			},
		},
		{
			name: "sendrawtransaction optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("sendrawtransaction", "1122", false)
			},
			staticCmd: func() interface{} {
				return soterjson.NewSendRawTransactionCmd("1122", soterjson.Bool(false))
			},
			marshalled: `{"jsonrpc":"1.0","method":"sendrawtransaction","params":["1122",false],"id":1}`,
			unmarshalled: &soterjson.SendRawTransactionCmd{
				HexTx:         "1122",
				AllowHighFees: soterjson.Bool(false),
			},
		},
		{
			name: "setgenerate",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("setgenerate", true)
			},
			staticCmd: func() interface{} {
				return soterjson.NewSetGenerateCmd(true, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"setgenerate","params":[true],"id":1}`,
			unmarshalled: &soterjson.SetGenerateCmd{
				Generate:     true,
				GenProcLimit: soterjson.Int(-1),
			},
		},
		{
			name: "setgenerate optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("setgenerate", true, 6)
			},
			staticCmd: func() interface{} {
				return soterjson.NewSetGenerateCmd(true, soterjson.Int(6))
			},
			marshalled: `{"jsonrpc":"1.0","method":"setgenerate","params":[true,6],"id":1}`,
			unmarshalled: &soterjson.SetGenerateCmd{
				Generate:     true,
				GenProcLimit: soterjson.Int(6),
			},
		},
		{
			name: "stop",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("stop")
			},
			staticCmd: func() interface{} {
				return soterjson.NewStopCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"stop","params":[],"id":1}`,
			unmarshalled: &soterjson.StopCmd{},
		},
		{
			name: "submitblock",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("submitblock", "112233")
			},
			staticCmd: func() interface{} {
				return soterjson.NewSubmitBlockCmd("112233", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"submitblock","params":["112233"],"id":1}`,
			unmarshalled: &soterjson.SubmitBlockCmd{
				HexBlock: "112233",
				Options:  nil,
			},
		},
		{
			name: "submitblock optional",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("submitblock", "112233", `{"workid":"12345"}`)
			},
			staticCmd: func() interface{} {
				options := soterjson.SubmitBlockOptions{
					WorkID: "12345",
				}
				return soterjson.NewSubmitBlockCmd("112233", &options)
			},
			marshalled: `{"jsonrpc":"1.0","method":"submitblock","params":["112233",{"workid":"12345"}],"id":1}`,
			unmarshalled: &soterjson.SubmitBlockCmd{
				HexBlock: "112233",
				Options: &soterjson.SubmitBlockOptions{
					WorkID: "12345",
				},
			},
		},
		{
			name: "uptime",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("uptime")
			},
			staticCmd: func() interface{} {
				return soterjson.NewUptimeCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"uptime","params":[],"id":1}`,
			unmarshalled: &soterjson.UptimeCmd{},
		},
		{
			name: "validateaddress",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("validateaddress", "1Address")
			},
			staticCmd: func() interface{} {
				return soterjson.NewValidateAddressCmd("1Address")
			},
			marshalled: `{"jsonrpc":"1.0","method":"validateaddress","params":["1Address"],"id":1}`,
			unmarshalled: &soterjson.ValidateAddressCmd{
				Address: "1Address",
			},
		},
		{
			name: "verifychain",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("verifychain")
			},
			staticCmd: func() interface{} {
				return soterjson.NewVerifyChainCmd(nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifychain","params":[],"id":1}`,
			unmarshalled: &soterjson.VerifyChainCmd{
				CheckLevel: soterjson.Int32(3),
				CheckDepth: soterjson.Int32(288),
			},
		},
		{
			name: "verifychain optional1",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("verifychain", 2)
			},
			staticCmd: func() interface{} {
				return soterjson.NewVerifyChainCmd(soterjson.Int32(2), nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifychain","params":[2],"id":1}`,
			unmarshalled: &soterjson.VerifyChainCmd{
				CheckLevel: soterjson.Int32(2),
				CheckDepth: soterjson.Int32(288),
			},
		},
		{
			name: "verifychain optional2",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("verifychain", 2, 500)
			},
			staticCmd: func() interface{} {
				return soterjson.NewVerifyChainCmd(soterjson.Int32(2), soterjson.Int32(500))
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifychain","params":[2,500],"id":1}`,
			unmarshalled: &soterjson.VerifyChainCmd{
				CheckLevel: soterjson.Int32(2),
				CheckDepth: soterjson.Int32(500),
			},
		},
		{
			name: "verifymessage",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("verifymessage", "1Address", "301234", "test")
			},
			staticCmd: func() interface{} {
				return soterjson.NewVerifyMessageCmd("1Address", "301234", "test")
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifymessage","params":["1Address","301234","test"],"id":1}`,
			unmarshalled: &soterjson.VerifyMessageCmd{
				Address:   "1Address",
				Signature: "301234",
				Message:   "test",
			},
		},
		{
			name: "verifytxoutproof",
			newCmd: func() (interface{}, error) {
				return soterjson.NewCmd("verifytxoutproof", "test")
			},
			staticCmd: func() interface{} {
				return soterjson.NewVerifyTxOutProofCmd("test")
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifytxoutproof","params":["test"],"id":1}`,
			unmarshalled: &soterjson.VerifyTxOutProofCmd{
				Proof: "test",
			},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Marshal the command as created by the new static command
		// creation function.
		marshalled, err := soterjson.MarshalCmd(testID, test.staticCmd())
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshalled, []byte(test.marshalled)) {
			t.Errorf("Test #%d (%s) unexpected marshalled data - "+
				"got %s, want %s", i, test.name, marshalled,
				test.marshalled)
			t.Errorf("\n%s\n%s", marshalled, test.marshalled)
			continue
		}

		// Ensure the command is created without error via the generic
		// new command creation function.
		cmd, err := test.newCmd()
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected NewCmd error: %v ",
				i, test.name, err)
		}

		// Marshal the command as created by the generic new command
		// creation function.
		marshalled, err = soterjson.MarshalCmd(testID, cmd)
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshalled, []byte(test.marshalled)) {
			t.Errorf("Test #%d (%s) unexpected marshalled data - "+
				"got %s, want %s", i, test.name, marshalled,
				test.marshalled)
			continue
		}

		var request soterjson.Request
		if err := json.Unmarshal(marshalled, &request); err != nil {
			t.Errorf("Test #%d (%s) unexpected error while "+
				"unmarshalling JSON-RPC request: %v", i,
				test.name, err)
			continue
		}

		cmd, err = soterjson.UnmarshalCmd(&request)
		if err != nil {
			t.Errorf("UnmarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !reflect.DeepEqual(cmd, test.unmarshalled) {
			t.Errorf("Test #%d (%s) unexpected unmarshalled command "+
				"- got %s, want %s", i, test.name,
				fmt.Sprintf("(%T) %+[1]v", cmd),
				fmt.Sprintf("(%T) %+[1]v\n", test.unmarshalled))
			continue
		}
	}
}

// TestChainSvrCmdErrors ensures any errors that occur in the command during
// custom mashal and unmarshal are as expected.
func TestChainSvrCmdErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		result     interface{}
		marshalled string
		err        error
	}{
		{
			name:       "template request with invalid type",
			result:     &soterjson.TemplateRequest{},
			marshalled: `{"mode":1}`,
			err:        &json.UnmarshalTypeError{},
		},
		{
			name:       "invalid template request sigoplimit field",
			result:     &soterjson.TemplateRequest{},
			marshalled: `{"sigoplimit":"invalid"}`,
			err:        soterjson.Error{ErrorCode: soterjson.ErrInvalidType},
		},
		{
			name:       "invalid template request sizelimit field",
			result:     &soterjson.TemplateRequest{},
			marshalled: `{"sizelimit":"invalid"}`,
			err:        soterjson.Error{ErrorCode: soterjson.ErrInvalidType},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		err := json.Unmarshal([]byte(test.marshalled), &test.result)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("Test #%d (%s) wrong error - got %T (%v), "+
				"want %T", i, test.name, err, err, test.err)
			continue
		}

		if terr, ok := test.err.(soterjson.Error); ok {
			gotErrorCode := err.(soterjson.Error).ErrorCode
			if gotErrorCode != terr.ErrorCode {
				t.Errorf("Test #%d (%s) mismatched error code "+
					"- got %v (%v), want %v", i, test.name,
					gotErrorCode, terr, terr.ErrorCode)
				continue
			}
		}
	}
}

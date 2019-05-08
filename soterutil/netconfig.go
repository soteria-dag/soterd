// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterutil

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/jessevdk/go-flags"

	"github.com/soteria-dag/soterd/chaincfg"
)

const (
	soterNetIniSection = "Soter Net Options"
)

// soterBigInt is a container for a big.Int, so that we can write a MarshalFlag and UnmarshalFlag method that applies
// to its bigInt field (can't add methods to existing standard library types, like big.Int directly)
type soterBigInt struct {
	bigInt *big.Int
}

// MarshalFlag for soterBigInt returns a string representing the bigInt value
func (b soterBigInt) MarshalFlag() (string, error) {
	bytes, err := b.bigInt.MarshalText()
	if err != nil {
		return "", nil
	}

	return fmt.Sprintf("%s", bytes), nil
}

// UnmarshalFlag for soterBigInt returns soterBigInt from the string
func (b *soterBigInt) UnmarshalFlag(v string) error {
	bigInt := new(big.Int)
	err := bigInt.UnmarshalText([]byte(v))
	if err != nil {
		return err
	}

	b.bigInt = bigInt

	return nil
}

// netCfg describes the configuration options for chaincfg.Params that may be modified before a node is started from a
// test harness.
//
// NOTE(cedric): Update ReadNetCfg whenever you update the netCfg struct, so that the params are applied when read.
type netCfg struct {
	Name string `short:"n" long:"name" description:"Name of net params type"`
	TargetTimespan time.Duration `short:"t" long:"targettimespan" description:"Desired amount of time that should elapse before checking if block difficulty requirement should be changed to maintain desired block generation rate"`
	TargetTimePerBlock time.Duration `short:"d" long:"targettimeperblock" description:"Desired amount of time to generate each block"`
}

// paramsToArgs returns netCfg arg values taken from the provided chaincfg.Param.
// This is necessary because the go-flags module doesn't support populating flag.Params types directly from structs.
func paramsToArgs(params *chaincfg.Params) []string {
	return []string{
		"--name", params.Name,
		"--targettimespan", params.TargetTimespan.String(),
		"--targettimeperblock", params.TargetTimePerBlock.String(),
	}
}

// TODO: Have WriteNetCfg compare given params against defaults, and write differences.
//  This will require writing MarshalFlag and UnmarshalFlag for all types without built-in parsing support in go-flags.
//
// WriteNetCfg writes chaincfg.Param values that we may be modifying during the test run to the open file descriptor, in
// ini file format.
func WriteNetCfg(f *os.File, params *chaincfg.Params) error {
	// Create a Parser for the options specified in netCfg
	parser := flags.NewParser(nil, flags.None)
	_, err := parser.AddGroup(soterNetIniSection, soterNetIniSection, &netCfg{})
	if err != nil {
		return err
	}

	// Add the params to the parser
	_, err = parser.ParseArgs(paramsToArgs(params))
	if err != nil {
		return err
	}

	// Write the ini file contents to the file descriptor
	iniParser := flags.NewIniParser(parser)
	iniParser.Write(f, flags.IniDefault)

	return nil
}

// ReadNetCfg reads chaincfg.Param values from the provided file name, and returns a *chaincfg.Params instance
func ReadNetCfg(name string) (*chaincfg.Params, error) {
	var params *chaincfg.Params
	var cfg netCfg

	// Create a parser, which will expect netCfg options and store them in cfg variable
	parser := flags.NewParser(nil, flags.None)
	_, err := parser.AddGroup(soterNetIniSection, soterNetIniSection, &cfg)
	if err != nil {
		return params, err
	}

	// Read ini file contents, and set them in cfg variable
	// (because iniParser was created from parser, which references cfg)
	iniParser := flags.NewIniParser(parser)
	err = iniParser.ParseFile(name)
	if err != nil {
		return params, err
	}

	// Determine which chaincfg.Params to update and return
	switch cfg.Name {
		case "mainnet":
			params = &chaincfg.MainNetParams
		case "regtest":
			params = &chaincfg.RegressionNetParams
		case "testnet1":
			params = &chaincfg.TestNet1Params
		case "simnet":
			params = &chaincfg.SimNetParams
		default:
			return params, fmt.Errorf("ReadNetCfg doesn't know what to do with net name %v", cfg.Name)
	}

	// Update relevant params
	// NOTE(cedric): Updates here should match the fields defined in netCfg type
	params.TargetTimespan = cfg.TargetTimespan
	params.TargetTimePerBlock = cfg.TargetTimePerBlock

	return params, nil
}
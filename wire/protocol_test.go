// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import "testing"

// TestServiceFlagStringer tests the stringized output for service flag types.
func TestServiceFlagStringer(t *testing.T) {
	tests := []struct {
		in   ServiceFlag
		want string
	}{
		{0, "0x0"},
		{SFNodeNetwork, "SFNodeNetwork"},
		{SFNodeGetUTXO, "SFNodeGetUTXO"},
		{SFNodeBloom, "SFNodeBloom"},
		{SFNodeWitness, "SFNodeWitness"},
		{SFNodeXthin, "SFNodeXthin"},
		{SFNodeBit5, "SFNodeBit5"},
		{SFNodeCF, "SFNodeCF"},
		{SFNode2X, "SFNode2X"},
		{0xffffffff, "SFNodeNetwork|SFNodeGetUTXO|SFNodeBloom|SFNodeWitness|SFNodeXthin|SFNodeBit5|SFNodeCF|SFNode2X|0xffffff00"},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

// TestSoterNetStringer tests the stringized output for soter net types.
func TestSoterNetStringer(t *testing.T) {
	tests := []struct {
		in   SoterNet
		want string
	}{
		{MainNet, "MainNet"},
		{TestNet, "TestNet"},
		{TestNet1, "TestNet1"},
		{SimNet, "SimNet"},
		{0xffffffff, "Unknown SoterNet (4294967295)"},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/soteria-dag/soterd/chaincfg/chainhash"
)

// TestParentsSubHeaderWire tests the block sub-header serialization and deserialization.
func TestParentsSubHeaderSerialize(t *testing.T) {

	// NOTE(cedric): This is the same hash defined in github.com/soteria-dag/soterd/chaincfg package.
	// We re-define it here, because the chaincfg package imports the wire package,
	// and golang doesn't allow for circular imports in tests.
	// (wire -> chaincfg -> wire -> ...)
	// 5f2694c9f5d5808adf308c62995154d29596c36c554b15d3d95cdd6bebaf6cb2
	var simNetGenesisHash = chainhash.Hash([chainhash.HashSize]byte{
		0xb2, 0x6c, 0xaf, 0xeb, 0x6b, 0xdd, 0x5c, 0xd9,
		0xd3, 0x15, 0x4b, 0x55, 0x6c, 0xc3, 0x96, 0x95,
		0xd2, 0x54, 0x51, 0x99, 0x62, 0x8c, 0x30, 0xdf,
		0x8a, 0x80, 0xd5, 0xf5, 0xc9, 0x94, 0x26, 0x5f,
	})

	parents := []*Parent{
		&Parent{
			Hash: simNetGenesisHash,
			Data: [32]byte{}},
	}

	psh := ParentSubHeader{
		Version: int32(0),
		Size:    int32(len(parents)),
		Parents: parents}

	encodedHeader := []byte{
		0x00, 0x00, 0x00, 0x00, // Version

		0x01, 0x00, 0x00, 0x00, // Size

		0xb2, 0x6c, 0xaf, 0xeb, 0x6b, 0xdd, 0x5c, 0xd9, // Parents -> Hash
		0xd3, 0x15, 0x4b, 0x55, 0x6c, 0xc3, 0x96, 0x95,
		0xd2, 0x54, 0x51, 0x99, 0x62, 0x8c, 0x30, 0xdf,
		0x8a, 0x80, 0xd5, 0xf5, 0xc9, 0x94, 0x26, 0x5f,

		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Parents -> Data
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	tests := []struct {
		header        *ParentSubHeader // Data to encode
		decodedHeader *ParentSubHeader // Expected decoded type
		encoded       []byte           // Serialized data
	}{
		{
			header:        &psh,
			decodedHeader: &psh,
			encoded:       encodedHeader,
		},
	}

	for i, test := range tests {
		// Serialize header
		var buf bytes.Buffer
		err := test.header.Serialize(&buf)
		if err != nil {
			t.Errorf("Test #%d Serialize error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.encoded) {
			t.Errorf("Test #%d\n got: %s want: %s",
				i,
				spew.Sdump(buf.Bytes()),
				spew.Sdump(test.encoded))
			continue
		}

		// Deserialize header
		var h ParentSubHeader
		rbuf := bytes.NewReader(test.encoded)
		err = h.Deserialize(rbuf)
		if err != nil {
			t.Errorf("Test #%d Deserialize error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&h, test.decodedHeader) {
			t.Errorf("Test #%d\n got: %s want: %s",
				i,
				spew.Sdump(&h),
				spew.Sdump(test.decodedHeader))
			continue
		}
	}

	var buf bytes.Buffer
	err := psh.Serialize(&buf)
	if err != nil {
		t.Errorf("Serialize error %v", err)
	}
}

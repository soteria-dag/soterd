// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

// TestVerificationSubHeader tests the block verification sub-header serialization and deserialization
func TestVerificationSubHeader(t *testing.T) {
	vsh := VerificationSubHeader{
		Version:     3,
		Size:        1, // This is set wrong on purpose, to check behaviour of Serialize method
		CycleNonces: []uint32{33, 29},
	}

	encoded := []byte{
		0x03, 0x00, 0x00, 0x00, // Version
		0x02, 0x00, 0x00, 0x00, // Size
		0x21, 0x00, 0x00, 0x00, // First CycleNonce
		0x1d, 0x00, 0x00, 0x00, // Second CycleNonce
	}

	// Serialize header
	var buf bytes.Buffer
	err := vsh.Serialize(&buf)
	if err != nil {
		t.Errorf("serialize error %v", err)
	}

	if !bytes.Equal(buf.Bytes(), encoded) {
		t.Errorf("wrong serialized data; got %s, want %s",
			spew.Sdump(buf.Bytes()),
			spew.Sdump(encoded))
	}

	// Deserialize header
	var h VerificationSubHeader
	rbuf := bytes.NewReader(encoded)
	err = h.Deserialize(rbuf)
	if err != nil {
		t.Errorf("deserialize error %v", err)
	}

	if h.Version != vsh.Version {
		t.Errorf("wrong version; got %d, want %d", h.Version, vsh.Version)
	}

	// Even though we set the Size to 1, the Serialize method should have corrected the serialized
	// size to be equal to the length of the CycleNonces field.
	if h.Size == vsh.Size || h.Size != int32(len(vsh.CycleNonces)) {
		t.Errorf("wrong decoded size; got %d, want %d", h.Size, len(vsh.CycleNonces))
	}

	if !reflect.DeepEqual(h.CycleNonces, vsh.CycleNonces) {
		t.Errorf("wrong CycleNonces; got %v, want %v",
			h.CycleNonces, vsh.CycleNonces)
	}
}

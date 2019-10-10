// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import "io"

const (
	// How many bytes to represent the Version field
	VerificationVersionSize = 4

	// How many bytes to represent the Size field
	CycleNoncesCountSize = 4

	// How many bytes to represent each cycle nonce
	CycleNonceSize = 4
)

// VerificationSubHeader represents a block verification sub-header
type VerificationSubHeader struct {
	// Version of VerificationSubHeader
	Version int32
	// Size of CycleNonces array
	// (Used for serialization/deserialization of VerificationSubHeader)
	Size int32
	// CycleNonces produced for cuckoo cycle solution for the block
	CycleNonces []uint32
}

// Deserialize decodes a verification sub-header from r into the receiver h using a format
// that is suitable for long-term storage (such as a database)
func (h *VerificationSubHeader) Deserialize(r io.Reader) error {
	// The protocol version 0 is the same between wire and long-term storage
	return readVerificationSubHeader(r, 0, h)
}

// Serialize encodes a verification sub-header from h into the receiver w using a format
// that is suitable for long-term storage (such as a database)
func (h *VerificationSubHeader) Serialize(w io.Writer) error {
	// The protocol version 0 is the same between wire and long-term storage
	return writeVerificationSubHeader(w, 0, h)
}

// readVerificationSubHeader reads a block's verification sub-header from r to h
func readVerificationSubHeader(r io.Reader, pver uint32, vsh *VerificationSubHeader) error {
	// Type fields are read in the same order they were written, in writeVerificationSubHeader

	// Read version
	err := readElement(r, &vsh.Version)
	if err != nil {
		return err
	}

	// Read the size of CycleNonces
	err = readElement(r, &vsh.Size)
	if err != nil {
		return err
	}

	// Attempt to read vsh.Size CycleNonces from r
	cycleNonces := make([]uint32, 0)
	for i := int32(0); i < vsh.Size; i++ {
		var nonce uint32
		err = readElement(r, &nonce)
		if err != nil {
			return err
		}
		cycleNonces = append(cycleNonces, nonce)
	}

	vsh.CycleNonces = cycleNonces

	return nil
}

// writeVerificationSubHeader writes a block's verification sub-header to w
func writeVerificationSubHeader(w io.Writer, pver uint32, vsh *VerificationSubHeader) error {
	// Write version info
	err := writeElement(w, vsh.Version)
	if err != nil {
		return err
	}

	// Write size of CycleNonces
	currentSize := int32(len(vsh.CycleNonces))
	var size int32
	if vsh.Size == currentSize {
		size = vsh.Size
	} else {
		// Verification sub-header Size field wasn't updated when CycleNonces field was updated.
		// The CycleNonces size is the source of truth in this case.
		size = currentSize
	}
	err = writeElement(w, size)
	if err != nil {
		return err
	}

	for _, nonce := range vsh.CycleNonces {
		err = writeElement(w, nonce)
		if err != nil {
			return err
		}
	}

	return nil
}
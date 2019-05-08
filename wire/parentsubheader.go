// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"io"
)

const (
	// maxParents is the maximum number of parents that a ParentSubHeader can contain
	maxParents = 8

	// ParentSize is the size in bytes of a parent
	ParentSize = chainhash.HashSize + 32

	ParentVersionSize = 4

	ParentCountSize = 4

	// MaxParentSubHeaderPayload is the maximum number of bytes a parent header can be.
	// Version 4 bytes + Size 4 bytes + max size of Parents
	MaxParentSubHeaderPayload = ParentVersionSize + ParentCountSize + (ParentSize * maxParents)
)

// ParentSubHeader represent a block parent sub-header
type ParentSubHeader struct {
	// Version of the parents sub header.
	Version int32
	// Size of Parents array (number of Parent associations)
	// (Used for serialization/deserialization of ParentSubHeader)
	Size int32
	// Array of meta data for previous block tips in the block DAG.
	Parents []*Parent
}

// Parent represents a parent of a block
type Parent struct {
	// Hash of parent block in the block dag.
	Hash chainhash.Hash
	// Metadata of this parent block. Currently a place-holder.
	Data [32]byte
}

// ParentHashes returns the hashes of the block's parents
func (h *ParentSubHeader) ParentHashes() []chainhash.Hash {
	hashes := make([]chainhash.Hash, len(h.Parents), len(h.Parents))
	for i := 0; i < len(h.Parents); i++ {
		hashes[i] = h.Parents[i].Hash
	}

	return hashes
}

// IsParent returns true if the given hash is a parent of the this block
func (h *ParentSubHeader) IsParent(hash *chainhash.Hash) bool {
	for _, parent := range h.Parents {
		if hash.IsEqual(&parent.Hash) {
			return true
		}
	}
	return false
}

// Deserialize decodes a parent sub-header from r into the receiver using a format
// that is suitable for long-term storage (such as a database)
func (h *ParentSubHeader) Deserialize(r io.Reader) error {
	// At time of writing the encoding for protocol version 0 is the same between wire and long-term storage.
	return readParentSubHeader(r, 0, h)
}

// Serialize encodes a block header from r into the receiver using a format
// that is suitable for long-term storage (such as a database)
func (h *ParentSubHeader) Serialize(w io.Writer) error {
	// At time of writing the encoding for protocol version 0 is the same between wire and long-term storage.
	return writeParentSubHeader(w, 0, h)
}

// readParentSubHeader reads a block's parent sub-header from r
func readParentSubHeader(r io.Reader, pver uint32, psh *ParentSubHeader) error {
	// Here we list type elements in the same order that they appear in the encoded data

	// Read version info
	err := readElement(r, &psh.Version)
	if err != nil {
		return err
	}

	// Read the size of Parents
	err = readElement(r, &psh.Size)
	if err != nil {
		return err
	}

	// readElement and writeElement deals mostly with primitive types, so
	// we'll build needed complex types for fields that use them, then populate them in psh.
	// At time of writing this is just the Parents field.
	p := []*Parent{}

	// Attempt to read psh.Size Parent data-structures from r
	for i := int32(1); i <= psh.Size; i++ {
		pi := Parent{}
		err = readElements(r, &pi.Hash, &pi.Data)
		if err != nil {
			return err
		}
		p = append(p, &pi)
	}

	psh.Parents = p

	return nil
}

// writeParentSubHeader writes a block's parent sub-header to w
func writeParentSubHeader(w io.Writer, pver uint32, psh *ParentSubHeader) error {
	// Write version info
	err := writeElement(w, psh.Version)
	if err != nil {
		return err
	}

	// Write size of Parents
	currentSize := int32(len(psh.Parents))
	var size int32
	if psh.Size == currentSize {
		size = psh.Size
	} else {
		// Something didn't update psh.Size when number of Parents changed. In this case the size
		// of the Parents slice is used.
		size = currentSize
	}
	err = writeElement(w, size)
	if err != nil {
		return err
	}

	for _, pi := range psh.Parents {
		err = writeElements(w, &pi.Hash, &pi.Data)
		if err != nil {
			return err
		}
	}

	return nil
}

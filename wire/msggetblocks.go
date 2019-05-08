// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
	"io"

	"github.com/soteria-dag/soterd/chaincfg/chainhash"
)

// MaxBlockLocatorsPerMsg is the maximum number of block locator heights allowed
// per message.
const MaxBlockLocatorsPerMsg = 1

// MsgGetBlocks implements the Message interface and represents a soter
// getblocks message.  It is used to request a list of blocks starting after the
// last known hash in the slice of block locator hashes.  The list is returned
// via an inv message (MsgInv) and is limited by a specific hash to stop at or
// the maximum number of blocks per message, which is currently 500.
//
// Set the HashStop field to the hash at which to stop and use
// AddBlockLocatorHeight to add block locator height.
type MsgGetBlocks struct {
	ProtocolVersion    uint32
	BlockLocatorHeight []*int32
	HashStop           chainhash.Hash
}

// AddBlockLocatorHeight adds a new block locator height to the message.
func (msg *MsgGetBlocks) AddBlockLocatorHeight(height *int32) error {
	if len(msg.BlockLocatorHeight)+1 > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator heights for message [max %v]",
			MaxBlockLocatorsPerMsg)
		return messageError("MsgGetBlocks.AddBlockLocatorHeight", str)
	}

	msg.BlockLocatorHeight = append(msg.BlockLocatorHeight, height)
	return nil
}

// SotoDecode decodes r using the soter protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgGetBlocks) SotoDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readElement(r, &msg.ProtocolVersion)
	if err != nil {
		return err
	}

	// Read num block locator hashes and limit to max.
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	if count > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator heights for message "+
			"[count %v, max %v]", count, MaxBlockLocatorsPerMsg)
		return messageError("MsgGetBlocks.SotoDecode", str)
	}

	// Create a contiguous slice of hashes to deserialize into in order to
	// reduce the number of allocations.
	locatorHeight := make([]int32, count)
	msg.BlockLocatorHeight = make([]*int32, 0, count)
	for i := uint64(0); i < count; i++ {
		height := &locatorHeight[i]
		err := readElement(r, height)
		if err != nil {
			return err
		}
		msg.AddBlockLocatorHeight(height)
	}

	return readElement(r, &msg.HashStop)
}

// SotoEncode encodes the receiver to w using the soter protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgGetBlocks) SotoEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	count := len(msg.BlockLocatorHeight)
	if count > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message "+
			"[count %v, max %v]", count, MaxBlockLocatorsPerMsg)
		return messageError("MsgGetBlocks.SotoEncode", str)
	}

	err := writeElement(w, msg.ProtocolVersion)
	if err != nil {
		return err
	}

	err = WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, hash := range msg.BlockLocatorHeight {
		err = writeElement(w, hash)
		if err != nil {
			return err
		}
	}

	return writeElement(w, &msg.HashStop)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgGetBlocks) Command() string {
	return CmdGetBlocks
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgGetBlocks) MaxPayloadLength(pver uint32) uint32 {
	// Protocol version 4 bytes + num hashes (varInt) + max block locator
	// hashes + hash stop.
	return 4 + MaxVarIntPayload + (4 * MaxBlockLocatorsPerMsg) + chainhash.HashSize
}

// NewMsgGetBlocks returns a new soter getblocks message that conforms to the
// Message interface using the passed parameters and defaults for the remaining
// fields.
func NewMsgGetBlocks(hashStop *chainhash.Hash) *MsgGetBlocks {
	return &MsgGetBlocks{
		ProtocolVersion:    ProtocolVersion,
		BlockLocatorHeight: make([]*int32, 0, MaxBlockLocatorsPerMsg),
		HashStop:           *hashStop,
	}
}

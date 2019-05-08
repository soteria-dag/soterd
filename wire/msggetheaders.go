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

// MsgGetHeaders implements the Message interface and represents a soter
// getheaders message.  It is used to request a list of block headers for
// blocks starting after the last known hash in the slice of block locator
// hashes.  The list is returned via a headers message (MsgHeaders) and is
// limited by a specific hash to stop at or the maximum number of block headers
// per message, which is currently 2000.
//
// Set the HashStop field to the hash at which to stop and use
// AddBlockLocatorHeight to build up the list of block locator hashes.
//
// The algorithm for building the block locator hashes should be to add the
// hashes in reverse order until you reach the genesis block.  In order to keep
// the list of locator hashes to a resonable number of entries, first add the
// most recent 10 block hashes, then double the step each loop iteration to
// exponentially decrease the number of hashes the further away from head and
// closer to the genesis block you get.
type MsgGetHeaders struct {
	ProtocolVersion    uint32
	BlockLocatorHeight []*int32
	HashStop           chainhash.Hash
}

// AddBlockLocatorHeight adds a new block locator hash to the message.
func (msg *MsgGetHeaders) AddBlockLocatorHeight(height *int32) error {
	if len(msg.BlockLocatorHeight)+1 > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message [max %v]",
			MaxBlockLocatorsPerMsg)
		return messageError("MsgGetHeaders.AddBlockLocatorHeight", str)
	}

	msg.BlockLocatorHeight = append(msg.BlockLocatorHeight, height)
	return nil
}

// SotoDecode decodes r using the soter protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgGetHeaders) SotoDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
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
		str := fmt.Sprintf("too many block locator hashes for message "+
			"[count %v, max %v]", count, MaxBlockLocatorsPerMsg)
		return messageError("MsgGetHeaders.SotoDecode", str)
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
func (msg *MsgGetHeaders) SotoEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	// Limit to max block locator hashes per message.
	count := len(msg.BlockLocatorHeight)
	if count > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message "+
			"[count %v, max %v]", count, MaxBlockLocatorsPerMsg)
		return messageError("MsgGetHeaders.SotoEncode", str)
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
		err := writeElement(w, hash)
		if err != nil {
			return err
		}
	}

	return writeElement(w, &msg.HashStop)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgGetHeaders) Command() string {
	return CmdGetHeaders
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgGetHeaders) MaxPayloadLength(pver uint32) uint32 {
	// Version 4 bytes + num block locator hashes (varInt) + max allowed block
	// locators + hash stop.
	return 4 + MaxVarIntPayload + (4 * MaxBlockLocatorsPerMsg) + chainhash.HashSize
}

// NewMsgGetHeaders returns a new soter getheaders message that conforms to
// the Message interface.  See MsgGetHeaders for details.
func NewMsgGetHeaders() *MsgGetHeaders {
	return &MsgGetHeaders{
		BlockLocatorHeight: make([]*int32, 0,
			MaxBlockLocatorsPerMsg),
	}
}

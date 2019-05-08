// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
	"io"
)

// MsgPong implements the Message interface and represents a soter pong
// message which is used primarily to confirm that a connection is still valid
// in response to a soter ping message (MsgPing).
//
// This message was not added until protocol versions AFTER BIP0031Version.
type MsgPong struct {
	// Unique value associated with message that is used to identify
	// specific ping message.
	Nonce uint64
}

// SotoDecode decodes r using the soter protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgPong) SotoDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	// NOTE: <= is not a mistake here.  The BIP0031 was defined as AFTER
	// the version unlike most others.
	if pver <= BIP0031Version {
		str := fmt.Sprintf("pong message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgPong.SotoDecode", str)
	}

	return readElement(r, &msg.Nonce)
}

// SotoEncode encodes the receiver to w using the soter protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgPong) SotoEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	// NOTE: <= is not a mistake here.  The BIP0031 was defined as AFTER
	// the version unlike most others.
	if pver <= BIP0031Version {
		str := fmt.Sprintf("pong message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgPong.SotoEncode", str)
	}

	return writeElement(w, msg.Nonce)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgPong) Command() string {
	return CmdPong
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgPong) MaxPayloadLength(pver uint32) uint32 {
	plen := uint32(0)
	// The pong message did not exist for BIP0031Version and earlier.
	// NOTE: > is not a mistake here.  The BIP0031 was defined as AFTER
	// the version unlike most others.
	if pver > BIP0031Version {
		// Nonce 8 bytes.
		plen += 8
	}

	return plen
}

// NewMsgPong returns a new soter pong message that conforms to the Message
// interface.  See MsgPong for details.
func NewMsgPong(nonce uint64) *MsgPong {
	return &MsgPong{
		Nonce: nonce,
	}
}

// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
)

// MsgGetAddrCache implements the Message interface and represents a soter
// getaddrcache message.  It is used to request a list of known active peers on the
// network from a peer to help identify potential nodes.  The list is returned
// via one or more addr messages (MsgAddrCache).
//
// This message has no payload.
type MsgGetAddrCache struct{}

// SotoDecode decodes r using the soter protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgGetAddrCache) SotoDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return nil
}

// SotoEncode encodes the receiver to w using the soter protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgGetAddrCache) SotoEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgGetAddrCache) Command() string {
	return CmdGetAddrCache
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgGetAddrCache) MaxPayloadLength(pver uint32) uint32 {
	return 0
}

// NewMsgGetAddrCache returns a new soter getaddr message that conforms to the
// Message interface.  See MsgGetAddrCache for details.
func NewMsgGetAddrCache() *MsgGetAddrCache {
	return &MsgGetAddrCache{}
}

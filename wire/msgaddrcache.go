// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
	"io"
)

// MsgAddrCache implements the Message interface and represents a soter
// addrcache message. It is used to provide a list of known active peers on the
// network. An active peer is considered one that has transmitted a message
// within the last 3 hours. Nodes which have not transmitted in that time
// frame should be forgotten. Each message is limited to a maximum number of
// addresses, which is currently 1000.  As a result, multiple messages must
// be used to relay the full list.
//
// Use the AddAddress function to build up the list of known addresses when
// sending an addr message to another peer.
type MsgAddrCache struct {
	AddrList []*NetAddress
}

// AddAddress adds a known active peer to the message.
func (msg *MsgAddrCache) AddAddress(na *NetAddress) error {
	if len(msg.AddrList)+1 > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses in message [max %v]",
			MaxAddrPerMsg)
		return messageError("MsgAddrCache.AddAddress", str)
	}

	msg.AddrList = append(msg.AddrList, na)
	return nil
}

// AddAddresses adds multiple known active peers to the message.
func (msg *MsgAddrCache) AddAddresses(netAddrs ...*NetAddress) error {
	for _, na := range netAddrs {
		err := msg.AddAddress(na)
		if err != nil {
			return err
		}
	}
	return nil
}

// ClearAddresses removes all addresses from the message.
func (msg *MsgAddrCache) ClearAddresses() {
	msg.AddrList = []*NetAddress{}
}

// SotoDecode decodes r using the soter protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgAddrCache) SotoDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

	// Limit to max addresses per message.
	if count > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses for message "+
			"[count %v, max %v]", count, MaxAddrPerMsg)
		return messageError("MsgAddrCache.SotoDecode", str)
	}

	addrList := make([]NetAddress, count)
	msg.AddrList = make([]*NetAddress, 0, count)
	for i := uint64(0); i < count; i++ {
		na := &addrList[i]
		err := readNetAddress(r, pver, na, true)
		if err != nil {
			return err
		}
		_ = msg.AddAddress(na)
	}
	return nil
}

// SotoEncode encodes the receiver to w using the soter protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgAddrCache) SotoEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	// Protocol versions before MultipleAddressVersion only allowed 1 address
	// per message.
	count := len(msg.AddrList)
	if pver < MultipleAddressVersion && count > 1 {
		str := fmt.Sprintf("too many addresses for message of "+
			"protocol version %v [count %v, max 1]", pver, count)
		return messageError("MsgAddrCache.SotoEncode", str)

	}
	if count > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses for message "+
			"[count %v, max %v]", count, MaxAddrPerMsg)
		return messageError("MsgAddrCache.SotoEncode", str)
	}

	err := WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, na := range msg.AddrList {
		err = writeNetAddress(w, pver, na, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgAddrCache) Command() string {
	return CmdAddrCache
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgAddrCache) MaxPayloadLength(pver uint32) uint32 {
	if pver < MultipleAddressVersion {
		// Num addresses (varInt) + a single net addresses.
		return MaxVarIntPayload + maxNetAddressPayload(pver)
	}

	// Num addresses (varInt) + max allowed addresses.
	return MaxVarIntPayload + (MaxAddrPerMsg * maxNetAddressPayload(pver))
}

// NewMsgAddrCache returns a new soter addr message that conforms to the
// Message interface.  See MsgAddrCache for details.
func NewMsgAddrCache() *MsgAddrCache {
	return &MsgAddrCache{
		AddrList: make([]*NetAddress, 0, MaxAddrPerMsg),
	}
}

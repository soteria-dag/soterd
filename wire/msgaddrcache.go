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
	Inbound []*NetAddress
	Outbound []*NetAddress
	Known []*NetAddress
}

// NewMsgAddrCache returns a new soter addr message that conforms to the
// Message interface.  See MsgAddrCache for details.
func NewMsgAddrCache() *MsgAddrCache {
	return &MsgAddrCache{
		Inbound: make([]*NetAddress, 0, MaxAddrPerMsg),
		Outbound: make([]*NetAddress, 0, MaxAddrPerMsg),
		Known: make([]*NetAddress, 0, MaxAddrPerMsg),
	}
}

// Count returns the total number of addresses in the type
func (msg *MsgAddrCache) Count() int {
	return len(msg.Inbound) + len(msg.Outbound) + len(msg.Known)
}

func (msg *MsgAddrCache) Reset() {
	msg.Inbound = make([]*NetAddress, 0, MaxAddrPerMsg)
	msg.Outbound = make([]*NetAddress, 0, MaxAddrPerMsg)
	msg.Known = make([]*NetAddress, 0, MaxAddrPerMsg)
}

func (msg *MsgAddrCache) AddInbound(na *NetAddress) error {
	count := msg.Count()
	if count > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses for message [count %v, max %v]",
			count, MaxAddrPerMsg)
		return messageError("MsgAddrCache.AddInbound", str)
	}

	msg.Inbound = append(msg.Inbound, na)
	return nil
}

func (msg *MsgAddrCache) AddManyInbound(addrs ...*NetAddress) error {
	for _, na := range addrs {
		err := msg.AddInbound(na)
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *MsgAddrCache) AddOutbound(na *NetAddress) error {
	count := msg.Count()
	if count > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses for message [count %v, max %v]",
			count, MaxAddrPerMsg)
		return messageError("MsgAddrCache.AddOutbound", str)
	}

	msg.Outbound = append(msg.Outbound, na)
	return nil
}

func (msg *MsgAddrCache) AddManyOutbound(addrs ...*NetAddress) error {
	for _, na := range addrs {
		err := msg.AddOutbound(na)
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *MsgAddrCache) AddKnown(na *NetAddress) error {
	count := msg.Count()
	if count > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses for message [count %v, max %v]",
			count, MaxAddrPerMsg)
		return messageError("MsgAddrCache.AddKnown", str)
	}

	msg.Known = append(msg.Known, na)
	return nil
}

func (msg *MsgAddrCache) AddManyKnown(addrs ...*NetAddress) error {
	for _, na := range addrs {
		err := msg.AddKnown(na)
		if err != nil {
			return err
		}
	}

	return nil
}

// SotoDecode decodes r using the soter protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgAddrCache) SotoDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	var total = uint64(0)

	// Define a function for reading a slice of addresses from the reader
	readAddrs := func() ([]*NetAddress, error) {
		var addrs []*NetAddress

		count, err := ReadVarInt(r, pver)
		if err != nil {
			return addrs, err
		}

		if (total + count) > MaxAddrPerMsg {
			str := fmt.Sprintf("too many addresses for message [count %v, max %v]",
				count, MaxAddrPerMsg)
			return addrs, messageError("MsgAddrCache.SotoDecode", str)
		}

		addrs = make([]*NetAddress, 0, count)
		for i := uint64(0); i < count; i++ {
			var na NetAddress
			err := readNetAddress(r, pver, &na, true)
			if err != nil {
				return addrs, err
			}

			addrs = append(addrs, &na)
			total++
		}

		return addrs, nil
	}

	// Read Inbound addresses
	addrs, err := readAddrs()
	if err != nil {
		return err
	}
	msg.Inbound = addrs

	// Read Outbound addresses
	addrs, err = readAddrs()
	if err != nil {
		return err
	}
	msg.Outbound = addrs

	// Read Known addresses
	addrs, err = readAddrs()
	if err != nil {
		return err
	}
	msg.Known = addrs

	return nil
}

// SotoEncode encodes the receiver to w using the soter protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgAddrCache) SotoEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	count := msg.Count()
	if count > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses for message [count %v, max %v]",
			count, MaxAddrPerMsg)
		return messageError("MsgAddrCache.SotoEncode", str)
	}

	// Define a function for writing a slice of addresses to the writer
	writeAddr := func(addrs []*NetAddress) error {
		err := WriteVarInt(w, pver, uint64(len(addrs)))
		if err != nil {
			return err
		}

		for _, na := range addrs {
			err = writeNetAddress(w, pver, na, true)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// Write Inbound addresses
	err := writeAddr(msg.Inbound)
	if err != nil {
		return err
	}

	// Write Outbound addresses
	err = writeAddr(msg.Outbound)
	if err != nil {
		return err
	}

	// Write Known addresses
	err = writeAddr(msg.Known)
	if err != nil {
		return err
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
	// (num addresses (varInt) * number of fields) + max allowed addresses between all fields
	return (MaxVarIntPayload * 3) + (MaxAddrPerMsg * maxNetAddressPayload(pver))
}

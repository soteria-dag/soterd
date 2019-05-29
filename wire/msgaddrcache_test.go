// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

// TestAddrCache tests the MsgAddrCache API.
func TestAddrCache(t *testing.T) {
	pver := ProtocolVersion

	// Ensure the command is expected value.
	wantCmd := "addrcache"
	msg := NewMsgAddrCache()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgAddrCache: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	// Num addresses (varInt) + max allowed addresses.
	wantPayload := uint32(30027)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	// Ensure NetAddresses are added properly.
	tcpAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	na := NewNetAddress(tcpAddr, SFNodeNetwork)
	err := msg.AddInbound(na)
	if err != nil {
		t.Errorf("AddInbound: %v", err)
	}
	err = msg.AddOutbound(na)
	if err != nil {
		t.Errorf("AddOutbound: %v", err)
	}
	err = msg.AddKnown(na)
	if err != nil {
		t.Errorf("AddKnown: %v", err)
	}

	if msg.Count() != 3 {
		t.Errorf("Count: got %d, want %d", msg.Count(), 3)
	}

	if msg.Inbound[0] != na {
		t.Errorf("AddInbound: wrong address added - got %v, want %v",
			spew.Sprint(msg.Inbound[0]), spew.Sprint(na))
	}
	if msg.Outbound[0] != na {
		t.Errorf("AddOutbound: wrong address added - got %v, want %v",
			spew.Sprint(msg.Outbound[0]), spew.Sprint(na))
	}
	if msg.Known[0] != na {
		t.Errorf("AddKnown: wrong address added - got %v, want %v",
			spew.Sprint(msg.Known[0]), spew.Sprint(na))
	}

	// Ensure the address lists are cleared properly.
	msg.Reset()
	if msg.Count() != 0 {
		t.Errorf("Reset: address lists not empty; got %d, want %d", msg.Count(), 0)
	}

	// Ensure adding more than the max allowed addresses per message returns
	// error.
	err = nil
	for i := 0; i < MaxAddrPerMsg + 5; i++ {
		err = msg.AddInbound(na)
	}
	if err == nil {
		t.Errorf("AddInbound: expected error on too many addresses not received")
	}
}

// TestAddrCacheWire tests the MsgAddrCache wire encode and decode for various numbers
// of addresses and protocol versions.
func TestAddrCacheWire(t *testing.T) {
	// A couple of NetAddresses to use for testing.
	na := &NetAddress{
		Timestamp: time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}
	na2 := &NetAddress{
		Timestamp: time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("192.168.0.1"),
		Port:      8334,
	}

	// Empty address message.
	noAddr := NewMsgAddrCache()
	noAddrEncoded := []byte{
		0x00, // Varint for number of Inbound addresses
		0x00, // Varint for number of Outbound addresses
		0x00, // Varint for number of Known addresses
	}

	// Address message with multiple addresses.
	multiAddr := NewMsgAddrCache()
	_ = multiAddr.AddInbound(na)
	_ = multiAddr.AddOutbound(na2)
	multiAddrEncoded := []byte{
		0x01,                   // Varint for number of Inbound addresses
		0x29, 0xab, 0x5f, 0x49, // Timestamp
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // SFNodeNetwork
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, // IP 127.0.0.1
		0x20, 0x8d, // Port 8333 in big-endian
		0x01,                   // Varint for number of Outbound addresses
		0x29, 0xab, 0x5f, 0x49, // Timestamp
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // SFNodeNetwork
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xc0, 0xa8, 0x00, 0x01, // IP 192.168.0.1
		0x20, 0x8e, // Port 8334 in big-endian
		0x00, // Varint for number of Known addresses
	}

	tests := []struct {
		in   *MsgAddrCache        // Message to encode
		out  *MsgAddrCache        // Expected decoded message
		buf  []byte          // Wire encoding
		pver uint32          // Protocol version for wire encoding
		enc  MessageEncoding // Message encoding format
	}{
		// Latest protocol version with no addresses.
		{
			noAddr,
			noAddr,
			noAddrEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

		// Latest protocol version with multiple addresses.
		{
			multiAddr,
			multiAddr,
			multiAddrEncoded,
			ProtocolVersion,
			BaseEncoding,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to wire format.
		var buf bytes.Buffer
		err := test.in.SotoEncode(&buf, test.pver, test.enc)
		if err != nil {
			t.Errorf("SotoEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("SotoEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from wire format.
		var msg MsgAddrCache
		rbuf := bytes.NewReader(test.buf)
		err = msg.SotoDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("SotoDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("SotoDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

// TestAddrCacheWireErrors performs negative tests against wire encode and decode
// of MsgAddr to confirm error paths work correctly.
func TestAddrCacheWireErrors(t *testing.T) {
	pver := ProtocolVersion
	wireErr := &MessageError{}

	// A couple of NetAddresses to use for testing.
	na := &NetAddress{
		Timestamp: time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}
	na2 := &NetAddress{
		Timestamp: time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("192.168.0.1"),
		Port:      8334,
	}

	// Address message with multiple addresses.
	baseAddr := NewMsgAddrCache()
	_ = baseAddr.AddManyInbound(na, na2)
	baseAddrEncoded := []byte{
		0x02,                   // Varint for number of Inbound addresses
		0x29, 0xab, 0x5f, 0x49, // Timestamp
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // SFNodeNetwork
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, // IP 127.0.0.1
		0x20, 0x8d, // Port 8333 in big-endian
		0x29, 0xab, 0x5f, 0x49, // Timestamp
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // SFNodeNetwork
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xc0, 0xa8, 0x00, 0x01, // IP 192.168.0.1
		0x20, 0x8e, // Port 8334 in big-endian
		0x00, // Varint for number of Outbound addresses
		0x00, // Varint for number of Known addresses
	}

	// Message that forces an error by having more than the max allowed
	// addresses.
	maxAddr := NewMsgAddrCache()
	for i := 0; i < MaxAddrPerMsg; i++ {
		_ = maxAddr.AddInbound(na)
	}
	maxAddr.Inbound = append(maxAddr.Inbound, na)
	maxAddrEncoded := []byte{
		0xfd, 0x03, 0xe9, // Varint for number of addresses (1001)
	}

	tests := []struct {
		in       *MsgAddrCache   // Value to encode
		buf      []byte          // Wire encoding
		pver     uint32          // Protocol version for wire encoding
		enc      MessageEncoding // Message encoding format
		max      int             // Max size of fixed buffer to induce errors
		writeErr error           // Expected write error
		readErr  error           // Expected read error
	}{
		// Latest protocol version with intentional read/write errors.
		// Force error in addresses count
		{baseAddr, baseAddrEncoded, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
		// Force error in address list.
		{baseAddr, baseAddrEncoded, pver, BaseEncoding, 1, io.ErrShortWrite, io.EOF},
		// Force error with greater than max inventory vectors.
		{maxAddr, maxAddrEncoded, pver, BaseEncoding, 3, wireErr, wireErr},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		w := newFixedWriter(test.max)
		err := test.in.SotoEncode(w, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.writeErr) {
			t.Errorf("SotoEncode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// For errors which are not of type MessageError, check them for
		// equality.
		if _, ok := err.(*MessageError); !ok {
			if err != test.writeErr {
				t.Errorf("SotoEncode #%d wrong error got: %v, "+
					"want: %v", i, err, test.writeErr)
				continue
			}
		}

		// Decode from wire format.
		var msg MsgAddrCache
		r := newFixedReader(test.max, test.buf)
		err = msg.SotoDecode(r, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("SotoDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

		// For errors which are not of type MessageError, check them for
		// equality.
		if _, ok := err.(*MessageError); !ok {
			if err != test.readErr {
				t.Errorf("SotoDecode #%d wrong error got: %v, "+
					"want: %v", i, err, test.readErr)
				continue
			}
		}

	}
}

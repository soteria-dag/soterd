// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
)

// TestGetBlocks tests the MsgGetBlocks API.
func TestGetBlocks(t *testing.T) {
	pver := ProtocolVersion

	// Block 99500 hash.
	hashStr := "000000000002e7ad7b9eef9479e4aabc65cb831269cc20d2632c13684406dee0"

	// Block 100000 hash.
	blockHeight := int32(10000)
	locatorHeight := &blockHeight
	hashStr = "3ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506"
	hashStop, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	// Ensure we get the same data back out.
	msg := NewMsgGetBlocks(hashStop)
	if !msg.HashStop.IsEqual(hashStop) {
		t.Errorf("NewMsgGetBlocks: wrong stop hash - got %v, want %v",
			msg.HashStop, hashStop)
	}

	// Ensure the command is expected value.
	wantCmd := "getblocks"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgGetBlocks: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	// Protocol version 4 bytes + num hashes (varInt) + max block locator
	// heights + hash stop.
	wantPayload := uint32(4 + 9 + (4 * 1) + 32)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	// Ensure block locator hashes are added properly.
	err = msg.AddBlockLocatorHeight(locatorHeight)
	if err != nil {
		t.Errorf("AddBlockLocatorHeight: %v", err)
	}
	if *msg.BlockLocatorHeight[0] != *locatorHeight {
		t.Errorf("AddBlockLocatorHeight: wrong block locator added - "+
			"got %v, want %v",
			spew.Sprint(msg.BlockLocatorHeight[0]),
			spew.Sprint(locatorHeight))
	}

	// Ensure adding more than the max allowed block locator heights per
	// message returns an error.
	for i := 0; i < MaxBlockLocatorsPerMsg; i++ {
		err = msg.AddBlockLocatorHeight(locatorHeight)
	}
	if err == nil {
		t.Errorf("AddBlockLocatorHeight: expected error on too many " +
			"block locator heights added")
	}
}

// TestGetBlocksWire tests the MsgGetBlocks wire encode and decode for various
// numbers of block locator hashes and protocol versions.
func TestGetBlocksWire(t *testing.T) {
	// Set protocol inside getblocks message.
	pver := uint32(60002)

	// Block 99499 hash.
	hashStr := "2710f40c87ec93d010a6fd95f42c59a2cbacc60b18cf6b7957535"
	height := int32(99499)
	locatorHeight := &height

	hashStop, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	// MsgGetBlocks message with no block locators or stop hash.
	noLocators := NewMsgGetBlocks(&chainhash.Hash{})
	noLocators.ProtocolVersion = pver
	noLocatorsEncoded := []byte{
		0x62, 0xea, 0x00, 0x00, // Protocol version 60002
		0x00, // Varint for number of block locator hashes
		0x00, 0x00, 0x00, 0x00, // locator height
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // hashStop
	}

	// MsgGetBlocks message with a block locator height and a stop hash.
	multiLocators := NewMsgGetBlocks(hashStop)
	_ = multiLocators.AddBlockLocatorHeight(locatorHeight)
	multiLocators.ProtocolVersion = pver
	multiLocatorsEncoded := []byte{
		0x62, 0xea, 0x00, 0x00, // Protocol version 60002
		0x01, // Varint for number of block locator hashes
		0xab, 0x84, 0x01, 0x00, // locatorHeight
		0x35, 0x75, 0x95, 0xb7, 0xf6, 0x8c, 0xb1, 0x60,
		0xcc, 0xba, 0x2c, 0x9a, 0xc5, 0x42, 0x5f, 0xd9,
		0x6f, 0x0a, 0x01, 0x3d, 0xc9, 0x7e, 0xc8, 0x40,
		0x0f, 0x71, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // hashStop
	}

	tests := []struct {
		in   *MsgGetBlocks   // Message to encode
		out  *MsgGetBlocks   // Expected decoded message
		buf  []byte          // Wire encoding
		pver uint32          // Protocol version for wire encoding
		enc  MessageEncoding // Message encoding format
	}{
		// Latest protocol version with no block locators.
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

		// Latest protocol version with multiple block locators.
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

		// Protocol version BIP0035Version with no block locators.
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			BIP0035Version,
			BaseEncoding,
		},

		// Protocol version BIP0035Version with multiple block locators.
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			BIP0035Version,
			BaseEncoding,
		},

		// Protocol version BIP0031Version with no block locators.
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			BIP0031Version,
			BaseEncoding,
		},

		// Protocol version BIP0031Versionwith multiple block locators.
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			BIP0031Version,
			BaseEncoding,
		},

		// Protocol version NetAddressTimeVersion with no block locators.
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

		// Protocol version NetAddressTimeVersion multiple block locators.
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

		// Protocol version MultipleAddressVersion with no block locators.
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			MultipleAddressVersion,
			BaseEncoding,
		},

		// Protocol version MultipleAddressVersion multiple block locators.
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			MultipleAddressVersion,
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
		var msg MsgGetBlocks
		rbuf := bytes.NewReader(test.buf)
		err = msg.SotoDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("SotoDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("SotoDecode #%d\n got: %s want: %s", i,
				spew.Sdump(&msg), spew.Sdump(test.out))
			continue
		}
	}
}

// TestGetBlocksWireErrors performs negative tests against wire encode and
// decode of MsgGetBlocks to confirm error paths work correctly.
func TestGetBlocksWireErrors(t *testing.T) {
	// Set protocol inside getheaders message.  Use protocol version 60002
	// specifically here instead of the latest because the test data is
	// using bytes encoded with that protocol version.
	pver := uint32(60002)
	wireErr := &MessageError{}

	// Block 99499 hash.
	hashStr := "2710f40c87ec93d010a6fd95f42c59a2cbacc60b18cf6b7957535"
	height := int32(99499)
	locatorHeight := &height

	hashStop, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	// MsgGetBlocks message with multiple block locators and a stop hash.
	baseGetBlocks := NewMsgGetBlocks(hashStop)
	baseGetBlocks.ProtocolVersion = pver
	err = baseGetBlocks.AddBlockLocatorHeight(locatorHeight)
	if err != nil {
		t.Errorf("AddBlockLocatorHeight: %v", err)
	}
	baseGetBlocksEncoded := []byte{
		0x62, 0xea, 0x00, 0x00, // Protocol version 60002
		0x01, // Varint for number of block locator hashes
		0xab, 0x84, 0x01, 0x00, // locatorHeight
		0x35, 0x75, 0x95, 0xb7, 0xf6, 0x8c, 0xb1, 0x60,
		0xcc, 0xba, 0x2c, 0x9a, 0xc5, 0x42, 0x5f, 0xd9,
		0x6f, 0x0a, 0x01, 0x3d, 0xc9, 0x7e, 0xc8, 0x40,
		0x0f, 0x71, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // hashStop
	}

	// Message that forces an error by having more than the max allowed
	// block locator hashes.
	maxGetBlocks := NewMsgGetBlocks(hashStop)
	for i := 0; i < MaxBlockLocatorsPerMsg; i++ {
		_ = maxGetBlocks.AddBlockLocatorHeight(locatorHeight)
	}
	maxGetBlocks.BlockLocatorHeight = append(maxGetBlocks.BlockLocatorHeight,
		locatorHeight)
	maxGetBlocksEncoded := []byte{
		0x62, 0xea, 0x00, 0x00, // Protocol version 60002
		0xfd, 0xf5, 0x01, // Varint for number of block loc hashes (501)
	}

	tests := []struct {
		in       *MsgGetBlocks   // Value to encode
		buf      []byte          // Wire encoding
		pver     uint32          // Protocol version for wire encoding
		enc      MessageEncoding // Message encoding format
		max      int             // Max size of fixed buffer to induce errors
		writeErr error           // Expected write error
		readErr  error           // Expected read error
	}{
		// Force error in protocol version.
		{baseGetBlocks, baseGetBlocksEncoded, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
		// Force error in block locator height count.
		{baseGetBlocks, baseGetBlocksEncoded, pver, BaseEncoding, 4, io.ErrShortWrite, io.EOF},
		// Force error in block locator height.
		{baseGetBlocks, baseGetBlocksEncoded, pver, BaseEncoding, 5, io.ErrShortWrite, io.EOF},
		// Force error in stop hash.
		{baseGetBlocks, baseGetBlocksEncoded, pver, BaseEncoding, 9, io.ErrShortWrite, io.EOF},
		// Force error with greater than max block locator hashes.
		{maxGetBlocks, maxGetBlocksEncoded, pver, BaseEncoding, 7, wireErr, wireErr},
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
		var msg MsgGetBlocks
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

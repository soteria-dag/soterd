// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterjson_test

import (
	"testing"

	"github.com/soteria-dag/soterd/soterjson"
)

// TestErrorCodeStringer tests the stringized output for the ErrorCode type.
func TestErrorCodeStringer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   soterjson.ErrorCode
		want string
	}{
		{soterjson.ErrDuplicateMethod, "ErrDuplicateMethod"},
		{soterjson.ErrInvalidUsageFlags, "ErrInvalidUsageFlags"},
		{soterjson.ErrInvalidType, "ErrInvalidType"},
		{soterjson.ErrEmbeddedType, "ErrEmbeddedType"},
		{soterjson.ErrUnexportedField, "ErrUnexportedField"},
		{soterjson.ErrUnsupportedFieldType, "ErrUnsupportedFieldType"},
		{soterjson.ErrNonOptionalField, "ErrNonOptionalField"},
		{soterjson.ErrNonOptionalDefault, "ErrNonOptionalDefault"},
		{soterjson.ErrMismatchedDefault, "ErrMismatchedDefault"},
		{soterjson.ErrUnregisteredMethod, "ErrUnregisteredMethod"},
		{soterjson.ErrNumParams, "ErrNumParams"},
		{soterjson.ErrMissingDescription, "ErrMissingDescription"},
		{0xffff, "Unknown ErrorCode (65535)"},
	}

	// Detect additional error codes that don't have the stringer added.
	if len(tests)-1 != int(soterjson.TstNumErrorCodes) {
		t.Errorf("It appears an error code was added without adding an " +
			"associated stringer test")
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

// TestError tests the error output for the Error type.
func TestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   soterjson.Error
		want string
	}{
		{
			soterjson.Error{Description: "some error"},
			"some error",
		},
		{
			soterjson.Error{Description: "human-readable error"},
			"human-readable error",
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.Error()
		if result != test.want {
			t.Errorf("Error #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

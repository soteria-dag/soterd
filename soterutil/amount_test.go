// Copyright (c) 2013, 2014 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterutil_test

import (
	"math"
	"testing"

	. "github.com/soteria-dag/soterd/soterutil"
)

func TestAmountCreation(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		valid    bool
		expected Amount
	}{
		// Positive tests.
		{
			name:     "zero",
			amount:   0,
			valid:    true,
			expected: 0,
		},
		{
			name:     "max producible",
			amount:   21e5,
			valid:    true,
			expected: MaxNanoSoter,
		},
		{
			name:     "min producible",
			amount:   -21e5,
			valid:    true,
			expected: -MaxNanoSoter,
		},
		{
			name:     "exceeds max producible",
			amount:   21e5 + 1e-9,
			valid:    true,
			expected: MaxNanoSoter + 1,
		},
		{
			name:     "exceeds min producible",
			amount:   -21e5 - 1e-9,
			valid:    true,
			expected: -MaxNanoSoter - 1,
		},
		{
			name:     "one hundred",
			amount:   100,
			valid:    true,
			expected: 100 * NanoSoterPerSoter,
		},
		{
			name:     "fraction",
			amount:   0.001234567,
			valid:    true,
			expected: 1234567,
		},
		{
			name:     "rounding up",
			amount:   54.999999999999943157,
			valid:    true,
			expected: 55 * NanoSoterPerSoter,
		},
		{
			name:     "rounding down",
			amount:   55.000000000000056843,
			valid:    true,
			expected: 55 * NanoSoterPerSoter,
		},

		// Negative tests.
		{
			name:   "not-a-number",
			amount: math.NaN(),
			valid:  false,
		},
		{
			name:   "-infinity",
			amount: math.Inf(-1),
			valid:  false,
		},
		{
			name:   "+infinity",
			amount: math.Inf(1),
			valid:  false,
		},
	}

	for _, test := range tests {
		a, err := NewAmount(test.amount)
		switch {
		case test.valid && err != nil:
			t.Errorf("%v: Positive test Amount creation failed with: %v", test.name, err)
			continue
		case !test.valid && err == nil:
			t.Errorf("%v: Negative test Amount creation succeeded (value %v) when should fail", test.name, a)
			continue
		}

		if a != test.expected {
			t.Errorf("%v: Created amount %v does not match expected %v", test.name, a, test.expected)
			continue
		}
	}
}

func TestAmountUnitConversions(t *testing.T) {
	tests := []struct {
		name      string
		amount    Amount
		unit      AmountUnit
		converted float64
		s         string
	}{
		{
			name:      "MSOTER",
			amount:    MaxNanoSoter,
			unit:      AmountMegaSOTER,
			converted: 2.1,
			s:         "2.1 MSOTER",
		},
		{
			name:      "kSOTER",
			amount:    44433322211100,
			unit:      AmountKiloSOTER,
			converted: 44.433322211100,
			s:         "44.4333222111 kSOTER",
		},
		{
			name:      "SOTER",
			amount:    44433322211100,
			unit:      AmountSOTER,
			converted: 44433.322211100,
			s:         "44433.3222111 SOTER",
		},
		{
			name:      "mSOTER",
			amount:    44433322211100,
			unit:      AmountMilliSOTER,
			converted: 44433322.211100,
			s:         "44433322.2111 mSOTER",
		},
		{

			name:      "μSOTER",
			amount:    44433322211100,
			unit:      AmountMicroSOTER,
			converted: 44433322211.100,
			s:         "44433322211.1 μSOTER",
		},
		{

			name:      "nSoter",
			amount:    44433322211100,
			unit:      AmountNanoSoter,
			converted: 44433322211100,
			s:         "44433322211100 nSoter",
		},
		{

			name:      "non-standard unit",
			amount:    44433322211100,
			unit:      AmountUnit(-1),
			converted: 444333.22211100,
			s:         "444333.222111 1e-1 SOTER",
		},
	}

	for _, test := range tests {
		f := test.amount.ToUnit(test.unit)
		if f != test.converted {
			t.Errorf("%v: converted value %v does not match expected %v", test.name, f, test.converted)
			continue
		}

		s := test.amount.Format(test.unit)
		if s != test.s {
			t.Errorf("%v: format '%v' does not match expected '%v'", test.name, s, test.s)
			continue
		}

		// Verify that Amount.ToSOTER works as advertised.
		f1 := test.amount.ToUnit(AmountSOTER)
		f2 := test.amount.ToSOTER()
		if f1 != f2 {
			t.Errorf("%v: ToSOTER does not match ToUnit(AmountSOTER): %v != %v", test.name, f1, f2)
		}

		// Verify that Amount.String works as advertised.
		s1 := test.amount.Format(AmountSOTER)
		s2 := test.amount.String()
		if s1 != s2 {
			t.Errorf("%v: String does not match Format(AmountBitcoin): %v != %v", test.name, s1, s2)
		}
	}
}

func TestAmountMulF64(t *testing.T) {
	tests := []struct {
		name string
		amt  Amount
		mul  float64
		res  Amount
	}{
		{
			name: "Multiply 0.1 SOTER by 2",
			amt:  100e5, // 0.1 SOTER
			mul:  2,
			res:  200e5, // 0.2 SOTER
		},
		{
			name: "Multiply 0.2 SOTER by 0.02",
			amt:  200e5, // 0.2 SOTER
			mul:  1.02,
			res:  204e5, // 0.204 SOTER
		},
		{
			name: "Multiply 0.1 SOTER by -2",
			amt:  100e5, // 0.1 SOTER
			mul:  -2,
			res:  -200e5, // -0.2 SOTER
		},
		{
			name: "Multiply 0.2 SOTER by -0.02",
			amt:  200e5, // 0.2 SOTER
			mul:  -1.02,
			res:  -204e5, // -0.204 SOTER
		},
		{
			name: "Multiply -0.1 SOTER by 2",
			amt:  -100e5, // -0.1 SOTER
			mul:  2,
			res:  -200e5, // -0.2 SOTER
		},
		{
			name: "Multiply -0.2 SOTER by 0.02",
			amt:  -200e5, // -0.2 SOTER
			mul:  1.02,
			res:  -204e5, // -0.204 SOTER
		},
		{
			name: "Multiply -0.1 SOTER by -2",
			amt:  -100e5, // -0.1 SOTER
			mul:  -2,
			res:  200e5, // 0.2 SOTER
		},
		{
			name: "Multiply -0.2 SOTER by -0.02",
			amt:  -200e5, // -0.2 SOTER
			mul:  -1.02,
			res:  204e5, // 0.204 SOTER
		},
		{
			name: "Round down",
			amt:  49, // 49 nSoter
			mul:  0.01,
			res:  0,
		},
		{
			name: "Round up",
			amt:  50, // 50 nSoter
			mul:  0.01,
			res:  1, // 1 nSoter
		},
		{
			name: "Multiply by 0.",
			amt:  1e9, // 1 SOTER
			mul:  0,
			res:  0, // 0 SOTER
		},
		{
			name: "Multiply 1 by 0.5.",
			amt:  1, // 1 nSoter
			mul:  0.5,
			res:  1, // 1 nSoter
		},
		{
			name: "Multiply 100 by 66%.",
			amt:  100, // 100 nSoter
			mul:  0.66,
			res:  66, // 66 nSoter
		},
		{
			name: "Multiply 100 by 66.6%.",
			amt:  100, // 100 nSoter
			mul:  0.666,
			res:  67, // 67 nSoter
		},
		{
			name: "Multiply 100 by 2/3.",
			amt:  100, // 100 nSoter
			mul:  2.0 / 3,
			res:  67, // 67 nSoter
		},
	}

	for _, test := range tests {
		a := test.amt.MulF64(test.mul)
		if a != test.res {
			t.Errorf("%v: expected %v got %v", test.name, test.res, a)
		}
	}
}

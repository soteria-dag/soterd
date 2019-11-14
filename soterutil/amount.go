// Copyright (c) 2013, 2014 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterutil

import (
	"errors"
	"math"
	"strconv"
)

// AmountUnit describes a method of converting an Amount to something
// other than the base unit of a soter token.  The value of the AmountUnit
// is the exponent component of the decadic multiple to convert from
// an amount in soter token to an amount counted in units.
type AmountUnit int

// These constants define various units used when describing a soter token amount.
const (
	AmountMegaSOTO  AmountUnit = 6
	AmountKiloSOTO  AmountUnit = 3
	AmountSOTO      AmountUnit = 0
	AmountMilliSOTO AmountUnit = -3
	AmountMicroSOTO AmountUnit = -6
	AmountNanoSoter AmountUnit = -9
)

// String returns the unit as a string.  For recognized units, the SI
// prefix is used, or "nanoSoter" for the base unit.  For all unrecognized
// units, "1eN SOTO" is returned, where N is the AmountUnit.
func (u AmountUnit) String() string {
	switch u {
	case AmountMegaSOTO:
		return "MSOTO"
	case AmountKiloSOTO:
		return "kSOTO"
	case AmountSOTO:
		return "SOTO"
	case AmountMilliSOTO:
		return "mSOTO"
	case AmountMicroSOTO:
		return "μSOTO"
	case AmountNanoSoter:
		return "nSoter"
	default:
		return "1e" + strconv.FormatInt(int64(u), 10) + " SOTO"
	}
}

// Amount represents the base soter monetary unit (colloquially referred
// to as a `nanoSoter').  A single Amount is equal to 1e-9 of a soter.
type Amount int64

// round converts a floating point number, which may or may not be representable
// as an integer, to the Amount integer type by rounding to the nearest integer.
// This is performed by adding or subtracting 0.5 depending on the sign, and
// relying on integer truncation to round the value to the nearest Amount.
func round(f float64) Amount {
	if f < 0 {
		return Amount(f - 0.5)
	}
	return Amount(f + 0.5)
}

// NewAmount creates an Amount from a floating point value representing
// some value in soter.  NewAmount errors if f is NaN or +-Infinity, but
// does not check that the amount is within the total amount of soter
// producible as f may not refer to an amount at a single moment in time.
//
// NewAmount is for specifically for converting SOTO to nanoSoter.
// For creating a new Amount with an int64 value which denotes a quantity of nanoSoter,
// do a simple type conversion from type int64 to Amount.
// See GoDoc for example: http://godoc.org/github.com/soteria-dag/soterd/soterutil#example-Amount
func NewAmount(f float64) (Amount, error) {
	// The amount is only considered invalid if it cannot be represented
	// as an integer type.  This may happen if f is NaN or +-Infinity.
	switch {
	case math.IsNaN(f):
		fallthrough
	case math.IsInf(f, 1):
		fallthrough
	case math.IsInf(f, -1):
		return 0, errors.New("invalid soter amount")
	}

	return round(f * NanoSoterPerSoter), nil
}

// ToUnit converts a monetary amount counted in soter base units to a
// floating point value representing an amount of soter.
func (a Amount) ToUnit(u AmountUnit) float64 {
	return float64(a) / math.Pow10(int(u+9))
}

// ToSOTO is the equivalent of calling ToUnit with AmountSOTO.
func (a Amount) ToSOTO() float64 {
	return a.ToUnit(AmountSOTO)
}

// Format formats a monetary amount counted in soter base units as a
// string for a given unit.  The conversion will succeed for any unit,
// however, known units will be formatted with an appended label describing
// the units with SI notation, or "nanoSoter" for the base unit.
func (a Amount) Format(u AmountUnit) string {
	units := " " + u.String()
	return strconv.FormatFloat(a.ToUnit(u), 'f', -int(u+9), 64) + units
}

// String is the equivalent of calling Format with AmountSOTO.
func (a Amount) String() string {
	return a.Format(AmountSOTO)
}

// MulF64 multiplies an Amount by a floating point value.  While this is not
// an operation that must typically be done by a full node or wallet, it is
// useful for services that build on top of soter (for example, calculating
// a fee by multiplying by a percentage).
func (a Amount) MulF64(f float64) Amount {
	return round(float64(a) * f)
}

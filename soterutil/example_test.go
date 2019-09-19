package soterutil_test

import (
	"fmt"
	"math"

	"github.com/soteria-dag/soterd/soterutil"
)

func ExampleAmount() {

	a := soterutil.Amount(0)
	fmt.Println("Zero nSoter:", a)

	a = soterutil.Amount(1e9)
	fmt.Println("1,000,000,000 nSoter:", a)

	a = soterutil.Amount(1e6)
	fmt.Println("1,000,000 nSoter:", a)
	// Output:
	// Zero nSoter: 0 SOTER
	// 1,000,000,000 nSoter: 1 SOTER
	// 1,000,000 nSoter: 0.001 SOTER
}

// ExampleNewAmount tests converting a SOTER amount to nanoSoter
func ExampleNewAmount() {
	amountOne, err := soterutil.NewAmount(1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountOne) //Output 1

	amountFraction, err := soterutil.NewAmount(0.01234567)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountFraction) //Output 2

	amountZero, err := soterutil.NewAmount(0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountZero) //Output 3

	amountNaN, err := soterutil.NewAmount(math.NaN())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountNaN) //Output 4

	// Output: 1 SOTER
	// 0.01234567 SOTER
	// 0 SOTER
	// invalid soter amount
}

func ExampleAmount_unitConversions() {
	amount := soterutil.Amount(44433322211100)

	fmt.Println("nSoter to kSOTER:", amount.Format(soterutil.AmountKiloSOTER))
	fmt.Println("nSoter to SOTER:", amount)
	fmt.Println("nSoter to MilliSOTER:", amount.Format(soterutil.AmountMilliSOTER))
	fmt.Println("nSoter to MicroSOTER:", amount.Format(soterutil.AmountMicroSOTER))
	fmt.Println("nSoter to nSoter:", amount.Format(soterutil.AmountNanoSoter))

	// Output:
	// nSoter to kSOTER: 44.4333222111 kSOTER
	// nSoter to SOTER: 44433.3222111 SOTER
	// nSoter to MilliSOTER: 44433322.2111 mSOTER
	// nSoter to MicroSOTER: 44433322211.1 μSOTER
	// nSoter to nSoter: 44433322211100 nSoter
}

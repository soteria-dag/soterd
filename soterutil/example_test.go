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
	// Zero nSoter: 0 SOTO
	// 1,000,000,000 nSoter: 1 SOTO
	// 1,000,000 nSoter: 0.001 SOTO
}

// ExampleNewAmount tests converting a SOTO amount to nanoSoter
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

	// Output: 1 SOTO
	// 0.01234567 SOTO
	// 0 SOTO
	// invalid soter amount
}

func ExampleAmount_unitConversions() {
	amount := soterutil.Amount(44433322211100)

	fmt.Println("nSoter to kSOTO:", amount.Format(soterutil.AmountKiloSOTO))
	fmt.Println("nSoter to SOTO:", amount)
	fmt.Println("nSoter to MilliSOTO:", amount.Format(soterutil.AmountMilliSOTO))
	fmt.Println("nSoter to MicroSOTO:", amount.Format(soterutil.AmountMicroSOTO))
	fmt.Println("nSoter to nSoter:", amount.Format(soterutil.AmountNanoSoter))

	// Output:
	// nSoter to kSOTO: 44.4333222111 kSOTO
	// nSoter to SOTO: 44433.3222111 SOTO
	// nSoter to MilliSOTO: 44433322.2111 mSOTO
	// nSoter to MicroSOTO: 44433322211.1 μSOTO
	// nSoter to nSoter: 44433322211100 nSoter
}

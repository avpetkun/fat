// Package bench compares fat with holiman/uint256 for 256 bits,
// lukechampine.com/uint128 for unsigned 128 bits and math/big for all four
// types. It is a module of its own, so the root module needs none of them.
//
// inputs_test.go builds the inputs once for every side, each type has a file
// with its TestAgree test and its benchmarks by families. The cases run the
// same 2^18 values and name their side impl=fat, impl=holiman,
// impl=lukechampine or impl=big:
//
//	cd bench
//	go test -run '^$' -bench . -benchtime 200ms -count 6 | benchstat -col /impl -
//
// The whole run takes about 40 minutes on an Apple M4 Pro.
//
// A pair is only of operations with the same result on these inputs: holiman
// wraps, so fat runs its wrapping form, lukechampine checks as fat does,
// and math/big runs where the result fits. TestAgree checks that every side
// gives the same values and leaves its inputs as they were
package bench

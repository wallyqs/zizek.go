// A tour of Zizek's draw and configuration surface, one scenario per feature:
//
//  1. bounded integers with a custom shrink target;
//  2. assumptions that reject examples instead of failing them;
//  3. boolean and byte draws shrinking together;
//  4. bounded IEEE-754 floats with portable shrinking; and
//  5. rejection limits (zizek.TooManyRejections).
//
// Every scenario is deterministic: it uses a fixed seed and prints the shrunk
// failure it expects to find.
package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

// shrinkTargets shows ShrinkTowards: failures shrink to the failing value
// closest to the requested target instead of the interval minimum.
func shrinkTargets() (*zizek.Failure, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := zizek.Draw(
			tc,
			zizek.Integers().MinValue(-1000).MaxValue(1000).ShrinkTowards(500),
		)
		if err != nil {
			return err
		}
		tc.Note(fmt.Sprintf("value %d", value))
		if value > 700 {
			return fmt.Errorf("value %d exceeds the budget of 700", value)
		}
		return nil
	}
	err := zizek.Check(property, zizek.Config{Examples: 300, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("expected a failure above 700: %w", err)
	}
	return failure, nil
}

// evenNumbersOnly shows Assume: odd draws are rejected (not failed), so the
// shrunk counterexample is the smallest *even* value above the limit.
func evenNumbersOnly() (*zizek.Failure, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := tc.Integer(0, 100, 0)
		if err != nil {
			return err
		}
		tc.Assume(value%2 == 0)
		tc.Note(fmt.Sprintf("even value %d", value))
		if value > 50 {
			return fmt.Errorf("even value %d exceeds 50", value)
		}
		return nil
	}
	err := zizek.Check(property, zizek.Config{Examples: 300, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("expected an even failure above 50: %w", err)
	}
	return failure, nil
}

// flagsAndBytes shows low-level Boolean and Bytes draws shrinking together:
// the boolean stays true because false passes, and the payload shrinks to the
// single smallest high byte.
func flagsAndBytes() (*zizek.Failure, error) {
	property := func(tc *zizek.TestCase) error {
		compressed, err := tc.Boolean()
		if err != nil {
			return err
		}
		payload, err := tc.Bytes(1, 4)
		if err != nil {
			return err
		}
		tc.Note(fmt.Sprintf("compressed=%t payload=%x", compressed, payload))
		if compressed {
			for _, value := range payload {
				if value >= 0x80 {
					return fmt.Errorf(
						"compressed payload contains high byte %#x",
						value,
					)
				}
			}
		}
		return nil
	}
	err := zizek.Check(property, zizek.Config{Examples: 300, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("expected a high-byte failure: %w", err)
	}
	return failure, nil
}

// boundedFloats shows that float generation, replay, and shrink ordering use
// the compiled float choice. The smallest value outside [0, 1) is exactly 1.
func boundedFloats() (*zizek.Failure, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := zizek.Draw(
			tc,
			zizek.Floats().MinValue(0).MaxValue(10),
		)
		if err != nil {
			return err
		}
		tc.Note(fmt.Sprintf("value %g", value))
		if value >= 1 {
			return fmt.Errorf("float %g is outside [0, 1)", value)
		}
		return nil
	}
	err := zizek.Check(property, zizek.Config{Examples: 300, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("expected a float failure: %w", err)
	}
	return failure, nil
}

// rejectionLimit shows MaxRejections: a property whose assumptions reject
// every example stops with TooManyRejections instead of searching forever.
func rejectionLimit() (*zizek.TooManyRejections, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := tc.Integer(0, 1000, 0)
		if err != nil {
			return err
		}
		tc.Assume(value > 1000) // impossible: every example is rejected
		return nil
	}
	err := zizek.Check(property, zizek.Config{
		Examples:      10,
		MaxRejections: 20,
		Seed:          1,
	})
	var rejected *zizek.TooManyRejections
	if !errors.As(err, &rejected) {
		return nil, fmt.Errorf("expected TooManyRejections, got: %w", err)
	}
	return rejected, nil
}

func main() {
	fmt.Println("== 1. integers with a shrink target ==")
	failure, err := shrinkTargets()
	if err != nil {
		panic(err)
	}
	fmt.Println(failure)

	fmt.Println("\n== 2. assumptions ==")
	if failure, err = evenNumbersOnly(); err != nil {
		panic(err)
	}
	fmt.Println(failure)

	fmt.Println("\n== 3. booleans and bytes ==")
	if failure, err = flagsAndBytes(); err != nil {
		panic(err)
	}
	fmt.Println(failure)

	fmt.Println("\n== 4. bounded floats ==")
	if failure, err = boundedFloats(); err != nil {
		panic(err)
	}
	fmt.Println(failure)

	fmt.Println("\n== 5. rejection limits ==")
	rejected, err := rejectionLimit()
	if err != nil {
		panic(err)
	}
	fmt.Println(rejected)
}

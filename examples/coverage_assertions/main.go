// Coverage assertions: catching the test that passes for the wrong reason.
//
// Demonstrates:
//   - tc.Always, an ordinary failure that shrinks to the boundary it crossed;
//   - tc.Unreachable, shrinking to the simplest input reaching a branch;
//   - tc.Sometimes, failing a run whose interesting branch never executed;
//   - tc.Cover, requiring a rate rather than a single sighting; and
//   - Config.Require, declaring a label whose call site itself may never run.
//
// The code under test is a clamp that forgot its ceiling. Every scenario is
// deterministic: it uses a fixed seed and prints the verdict it expects.
package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

// clamp confines value to [floor, ceiling]. Deliberate bug: the ceiling check
// compares against the floor, so values above the ceiling pass straight
// through.
func clamp(value, floor, ceiling int64) int64 {
	if value < floor {
		return floor
	}
	if value < floor { // BUG: should be `if value > ceiling`
		return ceiling
	}
	return value
}

// alwaysShrinksToTheBoundary uses the per-example half of the vocabulary.
// Always fails like any other failure, so it shrinks: the reported example is
// the *smallest* value that escapes the ceiling, 101, not whatever large value
// generation found first.
func alwaysShrinksToTheBoundary() (*zizek.Failure, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := tc.Integer(0, 1000, 0)
		if err != nil {
			return err
		}
		clamped := clamp(value, 0, 100)
		tc.Note(fmt.Sprintf("clamp(%d) = %d", value, clamped))
		tc.Always("clamped into range", clamped >= 0 && clamped <= 100)
		return nil
	}

	err := zizek.Check(property, zizek.Config{Examples: 300, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("the broken ceiling should be discovered: %w", err)
	}
	return failure, nil
}

// unreachableFindsTheSimplestPath asserts a branch is dead. When it is not,
// the failure shrinks to the simplest input that reaches it.
func unreachableFindsTheSimplestPath() (*zizek.Failure, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := tc.Integer(0, 1000, 0)
		if err != nil {
			return err
		}
		if clamp(value, 0, 100) > 100 {
			tc.Unreachable("value escaped the clamp")
		}
		return nil
	}

	err := zizek.Check(property, zizek.Config{Examples: 300, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("the escape branch should be reached: %w", err)
	}
	return failure, nil
}

// sometimesFailsTheVacuousPass judges the completed run rather than the
// current example. The property itself holds on every draw here — the
// generator never exceeds the ceiling, so the clamp is never exercised where
// it is broken. Without the coverage assertion this run would go green and
// prove nothing.
func sometimesFailsTheVacuousPass() (*zizek.FailedCoverage, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := tc.Integer(0, 100, 0) // never above the ceiling: too narrow
		if err != nil {
			return err
		}
		clamped := clamp(value, 0, 100)
		tc.Sometimes("saturated at the ceiling", clamped == 100 && value > 100)
		if clamped > value {
			return errors.New("clamp raised the value")
		}
		return nil
	}

	err := zizek.Check(property, zizek.Config{Examples: 200, Seed: 1})
	var missed *zizek.FailedCoverage
	if !errors.As(err, &missed) {
		return nil, fmt.Errorf("the run should fail its coverage requirement: %w", err)
	}
	return missed, nil
}

// coverReportsTheObservedRate is Sometimes with a required rate over the run's
// valid examples. Values 0..1000 leave about 90% of draws above the ceiling,
// so demanding that 60% stay below it must fail — and the verdict reports the
// counters it observed.
func coverReportsTheObservedRate() (*zizek.FailedCoverage, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := tc.Integer(0, 1000, 0)
		if err != nil {
			return err
		}
		tc.Cover("left below the ceiling", value <= 100, 60)
		return nil
	}

	err := zizek.Check(property, zizek.Config{Examples: 200, Seed: 1})
	var missed *zizek.FailedCoverage
	if !errors.As(err, &missed) {
		return nil, fmt.Errorf("the run should miss the 60%% requirement: %w", err)
	}
	return missed, nil
}

// requireCatchesTheUnregisteredLabel covers the gap the other four cannot. A
// call site registers its label the first time it *runs*; this Sometimes sits
// inside a branch the generator can never enter, so the run never learns the
// label exists. Declaring it in Config.Require is what turns that silence into
// a verdict.
func requireCatchesTheUnregisteredLabel() (*zizek.FailedCoverage, error) {
	property := func(tc *zizek.TestCase) error {
		value, err := tc.Integer(0, 50, 0)
		if err != nil {
			return err
		}
		if value > 100 { // unreachable with these bounds
			tc.Sometimes("overflow path", true)
		}
		return nil
	}

	err := zizek.Check(property, zizek.Config{
		Examples: 100,
		Seed:     1,
		Require:  []zizek.Coverage{{Label: "overflow path"}},
	})
	var missed *zizek.FailedCoverage
	if !errors.As(err, &missed) {
		return nil, fmt.Errorf("the run should fail the declared requirement: %w", err)
	}
	return missed, nil
}

func main() {
	boundary, err := alwaysShrinksToTheBoundary()
	if err != nil {
		panic(err)
	}
	fmt.Println("== 1. Always: fails and shrinks to the boundary ==")
	fmt.Println(boundary)

	reached, err := unreachableFindsTheSimplestPath()
	if err != nil {
		panic(err)
	}
	fmt.Println("\n== 2. Unreachable: shrinks to the simplest reaching input ==")
	fmt.Println(reached)

	vacuous, err := sometimesFailsTheVacuousPass()
	if err != nil {
		panic(err)
	}
	fmt.Println("\n== 3. Sometimes: the vacuous pass becomes a failure ==")
	fmt.Println(vacuous)

	rate, err := coverReportsTheObservedRate()
	if err != nil {
		panic(err)
	}
	fmt.Println("\n== 4. Cover: a required rate, with observed counters ==")
	fmt.Println(rate)

	declared, err := requireCatchesTheUnregisteredLabel()
	if err != nil {
		panic(err)
	}
	fmt.Println("\n== 5. Require: a label whose call site never ran ==")
	fmt.Println(declared)
}

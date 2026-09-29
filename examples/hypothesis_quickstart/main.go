// Port of Hypothesis's introductory bounded-integer example. The property is
// false from 50 onward, so shrinking should report the first failing boundary.
package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

const portableBoundaryFailure = "AAEAAAAACgEAAAAy"

func property(tc *zizek.TestCase) error {
	value, err := zizek.Draw(tc, zizek.Integers().MinValue(0).MaxValue(200))
	if err != nil {
		return err
	}
	if value >= 50 {
		return fmt.Errorf("%d is not less than 50", value)
	}
	return nil
}

func findAndReplayBoundary() (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{Examples: 100, Seed: 1})
	var found *zizek.Failure
	if !errors.As(err, &found) {
		return nil, fmt.Errorf("boundary bug was not found: %w", err)
	}
	if found.Blob != portableBoundaryFailure {
		return nil, fmt.Errorf("blob = %q, want portable boundary fixture", found.Blob)
	}
	if found.Cause == nil || found.Cause.Error() != "50 is not less than 50" {
		return nil, fmt.Errorf("unexpected failure: %v", found.Cause)
	}

	err = zizek.Check(property, zizek.Config{Reproduce: found.Blob})
	var replayed *zizek.Failure
	if !errors.As(err, &replayed) {
		return nil, fmt.Errorf("boundary failure did not replay: %w", err)
	}
	return replayed, nil
}

func main() {
	failure, err := findAndReplayBoundary()
	if err != nil {
		panic(err)
	}
	fmt.Println(failure)
}

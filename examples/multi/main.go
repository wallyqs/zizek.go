// Port of Hegel's two-assertion example. Shrinking must stay on whichever
// assertion site generation discovered first.
package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wallyqs/zizek.go"
)

const edge int64 = 1_337

func property(tc *zizek.TestCase) error {
	value, err := zizek.Draw(tc, zizek.Integers().MinValue(0).MaxValue(2_000))
	if err != nil {
		return err
	}
	if value == edge {
		return fmt.Errorf("%d hit the forbidden edge", value)
	}
	if value >= edge {
		return fmt.Errorf("%d is not below the edge", value)
	}
	return nil
}

func findAndReplayOneOrigin() (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{Examples: 100, Seed: 1})
	var found *zizek.Failure
	if !errors.As(err, &found) {
		return nil, fmt.Errorf("neither integer bug was found: %w", err)
	}
	message := found.Cause.Error()
	if !strings.Contains(message, "forbidden edge") && !strings.Contains(message, "not below the edge") {
		return nil, fmt.Errorf("unexpected failure origin: %s", message)
	}

	err = zizek.Check(property, zizek.Config{Reproduce: found.Blob})
	var replayed *zizek.Failure
	if !errors.As(err, &replayed) {
		return nil, fmt.Errorf("selected failure did not replay: %w", err)
	}
	if replayed.Cause.Error() != message {
		return nil, fmt.Errorf("replay changed origin from %q to %q", message, replayed.Cause)
	}
	return replayed, nil
}

func main() {
	failure, err := findAndReplayOneOrigin()
	if err != nil {
		panic(err)
	}
	fmt.Println(failure)
}

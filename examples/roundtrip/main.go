// Round-trip property example: run-length encoding.
//
// Demonstrates:
//   - the portable text generator (zizek.Text);
//   - notes attached to the failing example (tc.Note);
//   - shrinking to a minimal counterexample; and
//   - replaying the portable reproduce blob.
package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wallyqs/zizek.go"
)

// The shrunk failure blob is identical in Go, Python, and Ruby.
const portableRoundtripFailure = "AAEAAAAEAQAAADAAAAA="

type run struct {
	count int
	char  byte
}

// encode compresses text into (count, char) runs. It contains a deliberate
// bug: the run in progress is never flushed when the input ends, so the final
// run of every non-empty input is lost.
func encode(text string) []run {
	var runs []run
	count := 0
	var current byte
	for i := 0; i < len(text); i++ {
		char := text[i]
		if count > 0 && char == current {
			count++
			continue
		}
		if count > 0 {
			runs = append(runs, run{count, current})
		}
		current = char
		count = 1
	}
	return runs
}

func decode(runs []run) string {
	var out strings.Builder
	for _, item := range runs {
		out.WriteString(strings.Repeat(string(item.char), item.count))
	}
	return out.String()
}

func property(tc *zizek.TestCase) error {
	text, err := zizek.Draw(tc, zizek.Text().MaxSize(12))
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("text %q encodes to %v", text, encode(text)))
	if decoded := decode(encode(text)); decoded != text {
		return fmt.Errorf("round trip lost data: %q became %q", text, decoded)
	}
	return nil
}

func findAndReplayBug() (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{Examples: 200, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("round-trip bug was not found: %w", err)
	}

	// The blob is a portable description of the shrunk choices: replaying it
	// runs exactly the recorded example, here or in any other Zizek package.
	if err := zizek.Check(
		property,
		zizek.Config{Reproduce: failure.Blob},
	); !errors.As(err, &failure) {
		return nil, fmt.Errorf("failure did not replay: %w", err)
	}
	return failure, nil
}

func main() {
	failure, err := findAndReplayBug()
	if err != nil {
		panic(err)
	}
	fmt.Println(failure)
}

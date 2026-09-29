// The Go half of a cross-language database check: replay a counterexample
// that another language found and stored.
//
// It runs with Examples: 1, which is nowhere near enough to generate this bug,
// and fails anyway — on the stream Python wrote into the shared directory. The
// blob it prints is the one Python and Ruby print, because all three replayed
// the same bytes.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/wallyqs/zizek.go"
)

func pairIsSmall(tc *zizek.TestCase) error {
	left, err := tc.Integer(0, 1000, 0)
	if err != nil {
		return err
	}
	right, err := tc.Integer(0, 1000, 0)
	if err != nil {
		return err
	}
	if left > 100 && right > 100 {
		return errors.New("both above 100")
	}
	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: crosslang_database <database-directory> <key>")
		os.Exit(2)
	}
	err := zizek.Check(pairIsSmall, zizek.Config{
		Examples:    1,
		Seed:        999,
		Database:    zizek.NewDirectoryDatabase(os.Args[1]),
		DatabaseKey: os.Args[2],
	})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		fmt.Fprintf(os.Stderr, "go found no counterexample; the stored entry did not replay (%v)\n", err)
		os.Exit(1)
	}
	if failure.ExamplesRun != 1 {
		fmt.Fprintf(os.Stderr, "go generated instead of replaying: %d examples\n", failure.ExamplesRun)
		os.Exit(1)
	}
	fmt.Println(failure.Blob)
}

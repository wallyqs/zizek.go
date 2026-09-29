// Keyed draws: a reproduce blob that survives editing the property.
//
// Demonstrates:
//   - naming a decision with zizek.Key so a replay finds it by name instead
//     of by position;
//   - a stored blob still reproducing its failure after a new draw is
//     inserted into the property, with the new draw's miss counted and
//     reported instead of returned as an error; and
//   - the positional version of the same edit, where inserting a draw shifts
//     every choice after it and the old blob stops meaning what it recorded.
//
// The scenario is a client's retry policy. Version 1 draws a timeout and a
// retry count; the bug fires when the policy retries more than three times.
// Version 2 inserts a TLS flag between them — the kind of unrelated question
// a growing simulator adds every week.
package main

import (
	"errors"
	"fmt"
	"slices"

	"github.com/wallyqs/zizek.go"
)

func checkPolicy(tc *zizek.TestCase, timeout, retries int64) error {
	tc.Note(fmt.Sprintf("timeout=%ds retries=%d", timeout, retries))
	if retries > 3 {
		return fmt.Errorf("%d retries would stampede the backend", retries)
	}
	return nil
}

// keyedV1 draws every decision under a stable name.
func keyedV1(tc *zizek.TestCase) error {
	timeout, err := tc.IntegerKeyed(1, 30, 0, zizek.Key("timeout"))
	if err != nil {
		return err
	}
	retries, err := tc.IntegerKeyed(0, 9, 0, zizek.Key("retries"))
	if err != nil {
		return err
	}
	return checkPolicy(tc, timeout, retries)
}

// keyedV2 inserts an unrelated keyed draw between the original two.
func keyedV2(tc *zizek.TestCase) error {
	timeout, err := tc.IntegerKeyed(1, 30, 0, zizek.Key("timeout"))
	if err != nil {
		return err
	}
	tls, err := tc.BooleanKeyed(zizek.Key("tls"))
	if err != nil {
		return err
	}
	retries, err := tc.IntegerKeyed(0, 9, 0, zizek.Key("retries"))
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("tls=%t", tls))
	return checkPolicy(tc, timeout, retries)
}

// The same two versions, addressed positionally.

func positionalV1(tc *zizek.TestCase) error {
	timeout, err := tc.Integer(1, 30, 0)
	if err != nil {
		return err
	}
	retries, err := tc.Integer(0, 9, 0)
	if err != nil {
		return err
	}
	return checkPolicy(tc, timeout, retries)
}

func positionalV2(tc *zizek.TestCase) error {
	timeout, err := tc.Integer(1, 30, 0)
	if err != nil {
		return err
	}
	tls, err := tc.Boolean()
	if err != nil {
		return err
	}
	retries, err := tc.Integer(0, 9, 0)
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("tls=%t", tls))
	return checkPolicy(tc, timeout, retries)
}

func findFailure(property zizek.Property) (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{Examples: 300, Seed: 5})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("the retry stampede should be discovered: %w", err)
	}
	return failure, nil
}

func replay(property zizek.Property, blob string) (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{Reproduce: blob})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("the recorded failure should still reproduce: %w", err)
	}
	return failure, nil
}

// keyedBlobSurvivesTheEdit finds the bug in version 1, then replays its blob
// against version 2.
//
// The keyed pair resolves by name, so "timeout" and "retries" read exactly the
// values the blob recorded, wherever they now sit. The new "tls" draw has no
// entry under its name — that is an edit to the property, not a malformed blob
// — so it uses its simplest value and the failure reports one key miss.
func keyedBlobSurvivesTheEdit() (*zizek.Failure, *zizek.Failure, error) {
	found, err := findFailure(keyedV1)
	if err != nil {
		return nil, nil, err
	}
	replayed, err := replay(keyedV2, found.Blob)
	if err != nil {
		return nil, nil, err
	}
	if replayed.KeyMisses != 1 {
		return nil, nil, fmt.Errorf("expected exactly one key miss, got %d", replayed.KeyMisses)
	}
	if len(found.Notes) == 0 || !slices.Contains(replayed.Notes, found.Notes[0]) {
		return nil, nil, fmt.Errorf(
			"the recorded draws must be preserved: %v became %v",
			found.Notes, replayed.Notes,
		)
	}
	return found, replayed, nil
}

// positionalBlobBreaksOnTheEdit runs the same edit under positional
// addressing. The inserted boolean consumes the entry recorded for the retry
// count, every draw after it is off by one, and the stream ends before the
// property stops drawing: ReplayOverrun.
func positionalBlobBreaksOnTheEdit() (*zizek.Failure, *zizek.ReplayOverrun, error) {
	found, err := findFailure(positionalV1)
	if err != nil {
		return nil, nil, err
	}
	replayErr := zizek.Check(positionalV2, zizek.Config{Reproduce: found.Blob})
	var overrun *zizek.ReplayOverrun
	if !errors.As(replayErr, &overrun) {
		return nil, nil, fmt.Errorf(
			"the shifted replay should not run to completion: %v", replayErr,
		)
	}
	return found, overrun, nil
}

func main() {
	found, replayed, err := keyedBlobSurvivesTheEdit()
	if err != nil {
		panic(err)
	}
	fmt.Println("== keyed: the blob outlives the edit ==")
	fmt.Printf("version 1 failure:\n%v\n", found)
	fmt.Printf("\nversion 2 replay of the same blob:\n%v\n", replayed)

	found, overrun, err := positionalBlobBreaksOnTheEdit()
	if err != nil {
		panic(err)
	}
	fmt.Println("\n== positional: the same edit shifts every later draw ==")
	fmt.Printf("version 1 failure:\n%v\n", found)
	fmt.Printf("\nversion 2 replay of the same blob: %T: %v\n", overrun, overrun)
}

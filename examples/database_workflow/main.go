// The example database: a counterexample that outlives the run that found it.
//
// Demonstrates:
//   - Config.Database storing a shrunk failure's raw choice stream on disk;
//   - the next run replaying the stored entry before generating anything, so
//     yesterday's bug reproduces on the first attempt — no search, no seed;
//   - the fixed bug cleaning itself up: the entry that no longer reproduces
//     is deleted by the replay that failed to reproduce it; and
//   - the on-disk layout (<key digest>/<stream digest>), which the Python and
//     Ruby packages write identically, so a directory can be shared.
//
// The whole run works in a temporary directory and leaves nothing behind.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/wallyqs/zizek.go"
)

// databaseKey is a name both sides of a shared directory agree on. Left unset,
// the key is derived from the property's own function name — stable across
// builds, and meaningless to another language's suite.
const databaseKey = "sorted_merge_keeps_every_element"

// mergeBuggy merges two sorted slices. Deliberate bug: the loop stops at the
// end of the shorter input, dropping whatever the longer one still holds.
func mergeBuggy(left, right []int64) []int64 {
	merged := []int64{}
	a, b := 0, 0
	for a < len(left) && b < len(right) {
		if left[a] <= right[b] {
			merged = append(merged, left[a])
			a++
			continue
		}
		merged = append(merged, right[b])
		b++
	}
	return merged // BUG: the unconsumed tail of the longer input is lost
}

// mergeFixed keeps every element. This property only counts elements, so
// concatenation satisfies it.
func mergeFixed(left, right []int64) []int64 {
	return append(append([]int64{}, left...), right...)
}

// sorted draws a short sorted slice. Map records no choices of its own, so
// sorting here costs nothing in the blob.
func sorted() zizek.Generator[[]int64] {
	return zizek.Map(
		zizek.Lists(zizek.Integers().MinValue(0).MaxValue(9)).MaxSize(4),
		func(values []int64) []int64 {
			slices.Sort(values)
			return values
		},
	)
}

// property is parameterized on the merge under test, so the same one can first
// find the bug and then verify the fix.
func property(merge func([]int64, []int64) []int64) zizek.Property {
	return func(tc *zizek.TestCase) error {
		left, err := zizek.Draw(tc, sorted())
		if err != nil {
			return err
		}
		right, err := zizek.Draw(tc, sorted())
		if err != nil {
			return err
		}
		merged := merge(left, right)
		if len(merged) != len(left)+len(right) {
			return fmt.Errorf("merged %v and %v into %v", left, right, merged)
		}
		return nil
	}
}

// storedEntries lists the stream files under a database root, skipping the
// .name file the engine writes for a human reading the checkout.
func storedEntries(root string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(root, "*", "*"))
	if err != nil {
		return nil, err
	}
	entries := []string{}
	for _, path := range matches {
		if !strings.HasPrefix(filepath.Base(path), ".") {
			entries = append(entries, path)
		}
	}
	sort.Strings(entries)
	return entries, nil
}

func checkExpectingFailure(prop zizek.Property, config zizek.Config) (*zizek.Failure, error) {
	err := zizek.Check(prop, config)
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("the dropped-tail bug should be discovered: %w", err)
	}
	return failure, nil
}

// runWorkflow walks the whole cycle: find and store, replay first on the next
// run, and collect the entry once the bug is fixed.
func runWorkflow() (first, second *zizek.Failure, stored string, err error) {
	root, err := os.MkdirTemp("", "zizek-example-db")
	if err != nil {
		return nil, nil, "", err
	}
	defer os.RemoveAll(root)
	database := zizek.NewDirectoryDatabase(root)

	// Run 1: the search finds the bug, shrinks it, and stores the shrunk
	// choice stream under the database key.
	first, err = checkExpectingFailure(property(mergeBuggy), zizek.Config{
		Examples:    300,
		Seed:        1,
		Database:    database,
		DatabaseKey: databaseKey,
	})
	if err != nil {
		return nil, nil, "", err
	}
	entries, err := storedEntries(root)
	if err != nil {
		return nil, nil, "", err
	}
	if len(entries) != 1 {
		return nil, nil, "", fmt.Errorf("one counterexample should be stored, found %d", len(entries))
	}
	// <root>/<key digest>/<stream digest>: both digests are the compiled
	// keyed-draw hash, which is why the layout is portable.
	stored = filepath.Join(
		filepath.Base(filepath.Dir(entries[0])),
		filepath.Base(entries[0]),
	)
	if want := zizek.DatabaseDigest([]byte(databaseKey)); filepath.Dir(stored) != want {
		return nil, nil, "", fmt.Errorf("stored under %q, want the key digest %q", stored, want)
	}

	// Run 2: a different seed, but the stored entry is replayed before any
	// generation — the bug reproduces on the first attempt.
	second, err = checkExpectingFailure(property(mergeBuggy), zizek.Config{
		Examples:    300,
		Seed:        999,
		Database:    database,
		DatabaseKey: databaseKey,
	})
	if err != nil {
		return nil, nil, "", err
	}
	if second.ExamplesRun != 1 {
		return nil, nil, "", fmt.Errorf(
			"the stored entry should be the first attempt, ran %d", second.ExamplesRun,
		)
	}
	if second.Blob != first.Blob {
		return nil, nil, "", fmt.Errorf(
			"both runs should report the same blob: %q then %q", first.Blob, second.Blob,
		)
	}

	// Run 3: the bug is fixed. The stored entry no longer reproduces, so the
	// run deletes it and explores normally — the directory keeps itself
	// current.
	if err := zizek.Check(property(mergeFixed), zizek.Config{
		Examples:    100,
		Seed:        1,
		Database:    database,
		DatabaseKey: databaseKey,
	}); err != nil {
		return nil, nil, "", fmt.Errorf("the fixed merge should pass: %w", err)
	}
	entries, err = storedEntries(root)
	if err != nil {
		return nil, nil, "", err
	}
	if len(entries) != 0 {
		return nil, nil, "", fmt.Errorf(
			"the fixed bug's entry should have been collected, %d remain", len(entries),
		)
	}
	return first, second, stored, nil
}

func main() {
	first, second, stored, err := runWorkflow()
	if err != nil {
		panic(err)
	}
	fmt.Println("== run 1: found, shrunk, and stored ==")
	fmt.Println(first)
	fmt.Printf("stored as %s\n", stored)
	fmt.Printf(
		"\n== run 2 (different seed): replayed from the database on attempt %d ==\n",
		second.ExamplesRun,
	)
	fmt.Println(second)
	fmt.Println("\n== run 3 (bug fixed): the stale entry collected itself ==")
}

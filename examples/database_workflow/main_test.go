package main

import (
	"strings"
	"testing"

	"github.com/wallyqs/zizek.go"
)

func TestDatabaseStoresReplaysAndCollects(t *testing.T) {
	first, second, stored, err := runWorkflow()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.Error(), "merged [] and [0] into []") {
		t.Fatalf("expected the minimal dropped-tail counterexample:\n%s", first)
	}
	if second.ExamplesRun != 1 {
		t.Fatalf("second run used %d examples, want the stored entry first", second.ExamplesRun)
	}
	if second.Blob != first.Blob {
		t.Fatalf("blobs differ across runs: %q then %q", first.Blob, second.Blob)
	}
	// The key digest is the portable half of the layout: another language
	// naming the property the same way writes into this directory.
	keyDigest, _, found := strings.Cut(stored, "/")
	if !found || keyDigest != zizek.DatabaseDigest([]byte(databaseKey)) {
		t.Fatalf("stored as %q, want <key digest>/<stream digest>", stored)
	}
}

package main

import (
	"strings"
	"testing"
)

func TestRoundTripShrinksToOneCharacter(t *testing.T) {
	failure, err := findAndReplayBug()
	if err != nil {
		t.Fatal(err)
	}
	rendered := failure.Error()
	if !strings.Contains(rendered, `"0" became ""`) {
		t.Fatalf(
			"expected the minimal one-character counterexample:\n%s",
			rendered,
		)
	}
	if failure.Blob != portableRoundtripFailure {
		t.Fatalf("blob = %q, want shared round-trip fixture", failure.Blob)
	}
}

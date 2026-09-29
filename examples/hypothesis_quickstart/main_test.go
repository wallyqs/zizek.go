package main

import "testing"

func TestShrinksToFirstFailingBoundary(t *testing.T) {
	failure, err := findAndReplayBoundary()
	if err != nil {
		t.Fatal(err)
	}
	if failure.Blob != portableBoundaryFailure {
		t.Fatalf("blob = %q, want %q", failure.Blob, portableBoundaryFailure)
	}
}

package main

import "testing"

func TestPreservesSelectedFailureOrigin(t *testing.T) {
	if _, err := findAndReplayOneOrigin(); err != nil {
		t.Fatal(err)
	}
}

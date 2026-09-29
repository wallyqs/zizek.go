package main

import (
	"strings"
	"testing"
)

func TestFindsRepeatedMinimumBug(t *testing.T) {
	failure, err := findAndReplayBug()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"step 1: push", "step 2: push", "step 3: pop"} {
		if !strings.Contains(failure.Error(), want) {
			t.Fatalf("failure missing %q:\n%s", want, failure)
		}
	}
	if strings.Contains(failure.Error(), "step 4:") {
		t.Fatalf("failure did not shrink to three steps:\n%s", failure)
	}
}

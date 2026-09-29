package main

import (
	"strings"
	"testing"
)

func TestFindsShrinksAndReplaysFourLitreSolution(t *testing.T) {
	failure, err := findAndReplaySolution()
	if err != nil {
		t.Fatal(err)
	}
	if failure.Blob != portableDieHardFailure {
		t.Fatalf("blob = %q, want shared Die Hard fixture", failure.Blob)
	}
	rendered := failure.Error()
	for _, want := range []string{
		"step 1: fill_big",
		"step 2: pour_big_into_small",
		"step 3: empty_small",
		"step 4: pour_big_into_small",
		"step 5: fill_big",
		"step 6: pour_big_into_small",
		"small / big = 3 / 4",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("failure missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "step 7:") {
		t.Fatalf("solution did not shrink to six steps:\n%s", rendered)
	}
}

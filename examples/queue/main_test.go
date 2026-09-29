package main

import (
	"strings"
	"testing"
)

func TestQueueBugIsFoundShrunkAndReplayed(t *testing.T) {
	failure, err := findAndReplayBug()
	if err != nil {
		t.Fatal(err)
	}
	rendered := failure.Error()
	if !strings.Contains(rendered, "property panicked") {
		t.Fatalf("expected the out-of-range panic to be captured:\n%s", rendered)
	}
	for _, want := range []string{"step 1:", "pop"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("failure trace missing %q:\n%s", want, rendered)
		}
	}
	if failure.Blob != portableQueueFailure {
		t.Fatalf("blob = %q, want shared queue fixture", failure.Blob)
	}
}

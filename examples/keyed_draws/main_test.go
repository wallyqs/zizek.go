package main

import (
	"strings"
	"testing"
)

func TestKeyedBlobSurvivesAnInsertedDraw(t *testing.T) {
	found, replayed, err := keyedBlobSurvivesTheEdit()
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Blob != found.Blob {
		t.Fatalf("replay reported %q, want the recorded blob %q", replayed.Blob, found.Blob)
	}
	if replayed.KeyMisses != 1 {
		t.Fatalf("key misses = %d, want the one inserted draw", replayed.KeyMisses)
	}
	rendered := replayed.Error()
	if !strings.Contains(rendered, "1 keyed draw was not in this blob") {
		t.Fatalf("expected the miss to be reported:\n%s", rendered)
	}
	// The inserted draw missed, so it took its simplest value rather than
	// stealing the entry recorded for the retry count.
	if !strings.Contains(rendered, "tls=false") {
		t.Fatalf("expected the missed draw's simplest value:\n%s", rendered)
	}
}

func TestPositionalBlobBreaksOnTheSameEdit(t *testing.T) {
	_, overrun, err := positionalBlobBreaksOnTheEdit()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(overrun.Error(), "replay ended before") {
		t.Fatalf("expected an exhausted replay, got %q", overrun.Error())
	}
}

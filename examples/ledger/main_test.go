package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/wallyqs/zizek.go"
)

func TestLedgerBugShrinksToThreeSteps(t *testing.T) {
	failure, err := findAndReplayBug()
	if err != nil {
		t.Fatal(err)
	}
	rendered := failure.Error()
	for _, want := range []string{
		"step 1: create_account",
		"step 2: create_account",
		"step 3: transfer",
		"transfer 1",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("failure missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "step 4:") {
		t.Fatalf("ledger did not shrink to three steps:\n%s", rendered)
	}
	var portable *zizek.Failure
	if err := zizek.Check(property, zizek.Config{Reproduce: portableLedgerFailure}); !errors.As(err, &portable) {
		t.Fatalf("shared ledger fixture no longer replays: %v", err)
	}
}

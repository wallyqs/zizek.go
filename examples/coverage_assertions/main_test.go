package main

import (
	"strings"
	"testing"
)

func TestAlwaysShrinksToTheBoundary(t *testing.T) {
	failure, err := alwaysShrinksToTheBoundary()
	if err != nil {
		t.Fatal(err)
	}
	rendered := failure.Error()
	if !strings.Contains(rendered, "clamp(101) = 101") {
		t.Fatalf("expected the smallest escaping value, 101:\n%s", rendered)
	}
	if !strings.Contains(rendered, `always: "clamped into range" did not hold`) {
		t.Fatalf("expected the assertion's label in the failure:\n%s", rendered)
	}
}

func TestUnreachableShrinksToTheSimplestReachingInput(t *testing.T) {
	failure, err := unreachableFindsTheSimplestPath()
	if err != nil {
		t.Fatal(err)
	}
	rendered := failure.Error()
	if !strings.Contains(rendered, `unreachable: "value escaped the clamp" was reached`) {
		t.Fatalf("expected the branch's label in the failure:\n%s", rendered)
	}
	// Both scenarios shrink to 101, so they record the same choice stream.
	if failure.Blob != "AAEAAAAACgEAAABl" {
		t.Fatalf("blob = %q, want the 101 boundary stream", failure.Blob)
	}
}

func TestSometimesFailsAVacuousRun(t *testing.T) {
	missed, err := sometimesFailsTheVacuousPass()
	if err != nil {
		t.Fatal(err)
	}
	if missed.Label != "saturated at the ceiling" || missed.Observed != 0 {
		t.Fatalf("verdict = %+v, want the label observed zero times", missed)
	}
	if missed.Valid != 200 {
		t.Fatalf("valid examples = %d, want the full run", missed.Valid)
	}
}

func TestCoverReportsTheObservedRate(t *testing.T) {
	missed, err := coverReportsTheObservedRate()
	if err != nil {
		t.Fatal(err)
	}
	if missed.MinPercent != 60 {
		t.Fatalf("required percent = %v, want 60", missed.MinPercent)
	}
	if missed.Observed == 0 || missed.Observed >= 120 {
		t.Fatalf("observed = %d, want a rate below the required 60%%", missed.Observed)
	}
}

func TestRequireCatchesALabelThatNeverRegistered(t *testing.T) {
	missed, err := requireCatchesTheUnregisteredLabel()
	if err != nil {
		t.Fatal(err)
	}
	if missed.Label != "overflow path" {
		t.Fatalf("label = %q, want the declared requirement", missed.Label)
	}
}

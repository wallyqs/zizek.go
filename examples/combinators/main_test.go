package main

import (
	"strings"
	"testing"
)

func TestCompositionShrinksEveryFieldAtOnce(t *testing.T) {
	failure, err := deleteWithABody()
	if err != nil {
		t.Fatal(err)
	}
	rendered := failure.Error()
	// Minimal in the fields the failure does not need — a one-character path
	// and zero retries — while the verb and the body stay what it does need.
	if !strings.Contains(rendered, `DELETE /a body="x" retries=0`) {
		t.Fatalf("expected the minimal DELETE-with-body request:\n%s", rendered)
	}
}

func TestDependentDrawsShrinkToTheEarliestCoveringMeeting(t *testing.T) {
	failure, err := meetingAcrossNoon()
	if err != nil {
		t.Fatal(err)
	}
	rendered := failure.Error()
	if !strings.Contains(rendered, "meeting 0:00-12:59") {
		t.Fatalf("expected the earliest meeting covering noon:\n%s", rendered)
	}
}

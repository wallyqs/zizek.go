package main

import (
	"strings"
	"testing"
)

func TestShrinkTargetsLandsOnSmallestFailure(t *testing.T) {
	failure, err := shrinkTargets()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(failure.Error(), "value 701 exceeds") {
		t.Fatalf("expected the shrunk value 701:\n%s", failure)
	}
}

func TestAssumeShrinksToSmallestEvenFailure(t *testing.T) {
	failure, err := evenNumbersOnly()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(failure.Error(), "even value 52 exceeds") {
		t.Fatalf("expected the shrunk even value 52:\n%s", failure)
	}
}

func TestFlagsAndBytesShrinkTogether(t *testing.T) {
	failure, err := flagsAndBytes()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(failure.Error(), "compressed=true payload=80") {
		t.Fatalf("expected true flag and single 0x80 payload:\n%s", failure)
	}
}

func TestBoundedFloatsShrinkToOne(t *testing.T) {
	failure, err := boundedFloats()
	if err != nil {
		t.Fatal(err)
	}
	if failure.Blob != "AAEAAAACAAAAAAAA8D8=" {
		t.Fatalf("expected the portable float-one blob, got %q", failure.Blob)
	}
}

func TestRejectionLimitStopsTheSearch(t *testing.T) {
	rejected, err := rejectionLimit()
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Limit != 20 || rejected.Rejections != 21 {
		t.Fatalf("unexpected rejection accounting: %+v", rejected)
	}
}

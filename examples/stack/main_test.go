package main

import "testing"

func TestStackModelPasses(t *testing.T) {
	if err := verifyStackModel(); err != nil {
		t.Fatal(err)
	}
}

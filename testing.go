package zizek

import "testing"

// T is the generated test context passed to a property run by Test.
//
// It is an alias rather than a wrapper, so every TestCase method remains
// available and generators accepted by Draw work without conversion.
type T = TestCase

// Test runs body as a property test and reports any failure through t.
//
// With no config it uses Check's defaults. At most one Config may be supplied.
// Use T.Always, T.Unreachable, panic, or another assertion that panics to fail
// an individual generated example, and MustDraw to draw without an error
// return. Check remains available for callers that prefer an error-returning
// property or need to inspect the returned Failure.
func Test(t testing.TB, body func(*T), configs ...Config) {
	t.Helper()
	if len(configs) > 1 {
		t.Fatalf("zizek.Test accepts at most one Config, got %d", len(configs))
		return
	}

	config := Config{}
	if len(configs) == 1 {
		config = configs[0]
	}
	if config.Database != nil && config.DatabaseKey == "" {
		config.DatabaseKey = t.Name()
	}

	property := func(tc *TestCase) error {
		body(tc)
		return nil
	}
	if err := Check(property, config); err != nil {
		t.Error(err)
	}
}

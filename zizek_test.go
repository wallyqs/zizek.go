package zizek

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

const crossLanguageIntegerFive = "AAEAAAAACgEAAAAF"

// An explicit ShrinkTowards above max must clamp to max, not to min. Go and Zig
// used to fold this together with the "no target given" default and clamped to
// min instead.
const crossLanguageClampedTarget = "AAEAAAAACgEAAAAK"
const crossLanguageFloatOne = "AAEAAAACAAAAAAAA8D8="

type recordingDatabase struct {
	fetched []string
}

func (db *recordingDatabase) Fetch(key string) [][]byte {
	db.fetched = append(db.fetched, key)
	return nil
}

func (*recordingDatabase) Save(string, []byte)   {}
func (*recordingDatabase) Delete(string, []byte) {}

func TestTestingAdapterRunsAProperty(t *testing.T) {
	runs := 0
	Test(t, func(ht *T) {
		runs++
		ht.Always("adapter callback ran", true)
	}, Config{Examples: 3, Seed: 1})
	if runs < 3 {
		t.Fatalf("property body ran %d times, want at least 3", runs)
	}
}

func TestTestingAdapterUsesTheGoTestNameAsItsDatabaseKey(t *testing.T) {
	database := &recordingDatabase{}
	Test(t, func(*T) {}, Config{Examples: 1, Database: database})
	if !reflect.DeepEqual(database.fetched, []string{t.Name()}) {
		t.Fatalf("database fetches = %q, want [%q]", database.fetched, t.Name())
	}

	database.fetched = nil
	Test(t, func(*T) {}, Config{
		Examples:    1,
		Database:    database,
		DatabaseKey: "portable/property",
	})
	if !reflect.DeepEqual(database.fetched, []string{"portable/property"}) {
		t.Fatalf("database fetches = %q, want [portable/property]", database.fetched)
	}
}

func TestFloatShrinksAndReplaysPortably(t *testing.T) {
	var seen []float64
	property := func(tc *TestCase) error {
		value, err := Draw(tc, Floats().MinValue(0).MaxValue(10))
		if err != nil {
			return err
		}
		seen = append(seen, value)
		if value >= 1 {
			return errors.New("float must be less than one")
		}
		return nil
	}
	var failure *Failure
	if !errors.As(Check(property, Config{Examples: 300, Seed: 1}), &failure) {
		t.Fatal("expected a bounded float failure")
	}
	if failure.Blob != crossLanguageFloatOne {
		t.Fatalf("float shrank to %q, want %q", failure.Blob, crossLanguageFloatOne)
	}
	seen = nil
	if !errors.As(Check(property, Config{Reproduce: failure.Blob}), &failure) {
		t.Fatal("portable float blob did not reproduce")
	}
	if !reflect.DeepEqual(seen, []float64{1}) {
		t.Fatalf("portable float replayed %v, want [1]", seen)
	}
}

// collectFloats runs a passing property that draws perCase floats from
// generator in each of examples cases, and returns every value drawn.
func collectFloats(t *testing.T, generator FloatGenerator, perCase, examples int) []float64 {
	t.Helper()
	var drawn []float64
	property := func(tc *TestCase) error {
		for i := 0; i < perCase; i++ {
			value, err := Draw(tc, generator)
			if err != nil {
				return err
			}
			drawn = append(drawn, value)
		}
		return nil
	}
	if err := Check(property, Config{Examples: examples, Seed: 1}); err != nil {
		t.Fatal(err)
	}
	return drawn
}

// Raw bit patterns essentially never land in a narrow range, so the old draw
// fell back to the bounds: floats(1000, 1001) produced only the two endpoints.
// The default draw now interpolates across the range half the time.
func TestBoundedFloatsReachTheInterior(t *testing.T) {
	distinct := map[float64]bool{}
	for _, value := range collectFloats(t, Floats().MinValue(1000).MaxValue(1001), 1, 300) {
		if value < 1000 || value > 1001 {
			t.Fatalf("drew %v outside [1000, 1001]", value)
		}
		distinct[value] = true
	}
	if len(distinct) <= 100 {
		t.Fatalf("floats(1000, 1001) drew %d distinct values", len(distinct))
	}

	unit := collectFloats(t, Floats().MinValue(0).MaxValue(1), 10, 600)
	interior := 0
	for _, value := range unit {
		if value >= 0.25 && value <= 0.75 {
			interior++
		}
	}
	if share := float64(interior) / float64(len(unit)); share < 0.20 {
		t.Fatalf("%.1f%% of %d floats(0, 1) draws were interior", share*100, len(unit))
	}
}

func TestUnboundedFloatsStillReachSpecialValues(t *testing.T) {
	seen := map[uint64]bool{}
	nan := false
	for _, value := range collectFloats(t, Floats().AllowNaN(true).AllowInfinity(true), 4, 500) {
		nan = nan || math.IsNaN(value)
		seen[math.Float64bits(value)] = true
	}
	if !nan {
		t.Fatal("no NaN drawn")
	}
	for _, special := range []float64{math.Inf(1), math.Inf(-1), 0, math.Copysign(0, -1)} {
		if !seen[math.Float64bits(special)] {
			t.Fatalf("never drew %v (bits %016x)", special, math.Float64bits(special))
		}
	}

	zero := false
	for _, value := range collectFloats(t, Floats(), 4, 500) {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			t.Fatalf("finite floats drew %v", value)
		}
		zero = zero || value == 0
	}
	if !zero {
		t.Fatal("finite floats never drew zero")
	}
}

func TestFailureShrinksAndReplays(t *testing.T) {
	var seen []int64
	property := func(tc *TestCase) error {
		value, err := tc.Integer(-100, 100, 0)
		if err != nil {
			return err
		}
		seen = append(seen, value)
		if value >= 5 {
			return errors.New("value must be less than five")
		}
		return nil
	}
	err := Check(property, Config{Examples: 200, Seed: 1})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("expected Failure, got %v", err)
	}
	seen = nil
	err = Check(property, Config{Reproduce: failure.Blob})
	if !errors.As(err, &failure) {
		t.Fatalf("replay did not fail: %v", err)
	}
	if len(seen) != 1 || seen[0] != 5 {
		t.Fatalf("replayed values = %v, want [5]", seen)
	}
}

// Two panics of the same value type raised from different lines are different
// bugs: the origin token carries the panic site.
func TestShrinkingDistinguishesPanicSites(t *testing.T) {
	for _, seed := range []int64{1, 2, 3} {
		found := ""
		property := func(tc *TestCase) error {
			value, err := tc.Integer(0, 100, 0)
			if err != nil {
				return err
			}
			if value > 50 {
				if found == "" {
					found = "A"
				}
				panic("bug A")
			}
			if value == 7 {
				if found == "" {
					found = "B"
				}
				panic("bug B")
			}
			return nil
		}
		err := Check(property, Config{Examples: 50, Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			t.Fatalf("seed %d: expected Failure, got %v", seed, err)
		}
		want := "bug " + found
		if !strings.Contains(failure.Cause.Error(), want) {
			t.Fatalf(
				"seed %d reported a different bug than the one found: got %q, want %q",
				seed, failure.Cause.Error(), want,
			)
		}
	}
}

func TestBooleanAndBytesReplay(t *testing.T) {
	property := func(tc *TestCase) error {
		enabled, err := tc.Boolean()
		if err != nil {
			return err
		}
		payload, err := tc.Bytes(0, 4)
		if err != nil {
			return err
		}
		if enabled && len(payload) != 0 {
			return errors.New("enabled payload")
		}
		return nil
	}
	err := Check(property, Config{Examples: 200, Seed: 7})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("expected Failure, got %v", err)
	}
	if err := Check(property, Config{Reproduce: failure.Blob}); !errors.As(err, &failure) {
		t.Fatalf("replay did not fail: %v", err)
	}
}

func TestCrossLanguageBlobFixture(t *testing.T) {
	var seen int64
	property := func(tc *TestCase) error {
		value, err := tc.Integer(-100, 100, 0)
		if err != nil {
			return err
		}
		seen = value
		return errors.New("always fails")
	}
	var failure *Failure
	if err := Check(property, Config{Reproduce: crossLanguageIntegerFive}); !errors.As(err, &failure) {
		t.Fatalf("fixture did not replay: %v", err)
	}
	if seen != 5 {
		t.Fatalf("fixture replayed %d, want 5", seen)
	}
}

func TestAssumptionsDoNotCountAsValidExamples(t *testing.T) {
	attempts := 0
	err := Check(func(tc *TestCase) error {
		attempts++
		tc.Assume(attempts%2 == 0)
		return nil
	}, Config{Examples: 3, Seed: 11})
	if err != nil {
		t.Fatalf("assumption run failed: %v", err)
	}
	if attempts != 6 {
		t.Fatalf("attempts = %d, want 6 for three valid examples", attempts)
	}
}

func TestSimplestProbeUsesChoiceSimplestValues(t *testing.T) {
	attempts := 0
	err := Check(func(tc *TestCase) error {
		attempts++
		integer, err := tc.Integer(5, 10, 7)
		if err != nil {
			return err
		}
		boolean, err := tc.Boolean()
		if err != nil {
			return err
		}
		bytes, err := tc.Bytes(2, 4)
		if err != nil {
			return err
		}
		float, err := tc.Float(1, 2, false, false, 0)
		if err != nil {
			return err
		}
		if integer != 7 || boolean || !reflect.DeepEqual(bytes, []byte{0, 0}) || float != 1 {
			return fmt.Errorf(
				"got simplest values (%d, %t, %v, %g)",
				integer, boolean, bytes, float,
			)
		}
		return nil
	}, Config{Examples: 1})
	if err != nil {
		t.Fatalf("simplest probe failed: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want the probe to count as the one valid example", attempts)
	}
}

func TestLargeBaseExampleHealthCheckAndSuppression(t *testing.T) {
	property := func(tc *TestCase) error {
		for i := 0; i < 4097; i++ {
			if _, err := tc.Boolean(); err != nil {
				return err
			}
		}
		return nil
	}
	err := Check(property, Config{Examples: 1})
	var health *FailedHealthCheck
	if !errors.As(err, &health) || health.Check != "large_base_example" {
		t.Fatalf("expected large_base_example health check, got %v", err)
	}
	if err := Check(property, Config{
		Examples:                 1,
		SuppressLargeBaseExample: true,
	}); err != nil {
		t.Fatalf("suppressed large_base_example check failed: %v", err)
	}
}

func TestTooSlowHealthCheckAndSuppression(t *testing.T) {
	property := func(tc *TestCase) error {
		tc.generationMicros = 1_000_001
		return nil
	}
	err := Check(property, Config{Examples: 1})
	var health *FailedHealthCheck
	if !errors.As(err, &health) || health.Check != "too_slow" {
		t.Fatalf("expected too_slow health check, got %v", err)
	}
	if err := Check(property, Config{
		Examples:        1,
		SuppressTooSlow: true,
	}); err != nil {
		t.Fatalf("suppressed too_slow check failed: %v", err)
	}
}

func TestTooManyRejections(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		tc.Assume(false)
		return nil
	}, Config{Examples: 1, MaxRejections: 2})
	var rejected *TooManyRejections
	if !errors.As(err, &rejected) {
		t.Fatalf("expected TooManyRejections, got %v", err)
	}
	if rejected.Rejections != 3 || rejected.Limit != 2 {
		t.Fatalf("unexpected rejection counts: %+v", rejected)
	}
}

func TestFilterTooMuchHealthCheckAndSuppression(t *testing.T) {
	property := func(tc *TestCase) error {
		tc.Assume(false)
		return nil
	}
	err := Check(property, Config{
		Examples:                 1,
		SuppressLargeBaseExample: true,
	})
	var health *FailedHealthCheck
	if !errors.As(err, &health) || health.Check != "filter_too_much" {
		t.Fatalf("expected filter_too_much health check, got %v", err)
	}

	err = Check(property, Config{
		Examples:              1,
		MaxRejections:         50,
		SuppressFilterTooMuch: true,
	})
	var rejected *TooManyRejections
	if !errors.As(err, &rejected) || rejected.Rejections != 51 {
		t.Fatalf("expected suppressed check to reach rejection cap, got %v", err)
	}
}

func TestDataTooLargeHealthCheckAndSuppression(t *testing.T) {
	attempts := 0
	property := func(tc *TestCase) error {
		attempts++
		if attempts <= 20 {
			for i := 0; i <= 8192; i++ {
				if _, err := tc.Boolean(); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err := Check(property, Config{
		Examples:                 1,
		SuppressLargeBaseExample: true,
	})
	var health *FailedHealthCheck
	if !errors.As(err, &health) || health.Check != "data_too_large" {
		t.Fatalf("expected data_too_large health check, got %v", err)
	}

	attempts = 0
	err = Check(property, Config{
		Examples:                 1,
		SuppressDataTooLarge:     true,
		SuppressLargeBaseExample: true,
	})
	if err != nil {
		t.Fatalf("suppressed data_too_large check failed: %v", err)
	}
	if attempts != 21 {
		t.Fatalf("attempts = %d, want 21", attempts)
	}
}

func TestValidExamplesExpandTheRejectionBudget(t *testing.T) {
	attempts := 0
	err := Check(func(tc *TestCase) error {
		attempts++
		tc.Assume(attempts%101 == 0)
		return nil
	}, Config{Examples: 15, Seed: 11, SuppressFilterTooMuch: true})
	if err != nil {
		t.Fatalf("adaptive rejection run failed: %v", err)
	}
	if attempts != 1515 {
		t.Fatalf("attempts = %d, want 1515", attempts)
	}
}

func TestRejectedShrinkCandidateIsNotFailure(t *testing.T) {
	property := func(tc *TestCase) error {
		value, err := tc.Integer(0, 100, 0)
		if err != nil {
			return err
		}
		tc.Assume(value >= 5)
		return errors.New("accepted value fails")
	}
	err := Check(property, Config{Examples: 100, Seed: 3})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("expected Failure, got %v", err)
	}
	var replayed int64
	err = Check(func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 100, 0)
		if drawErr != nil {
			return drawErr
		}
		replayed = value
		tc.Assume(value >= 5)
		return errors.New("accepted value fails")
	}, Config{Reproduce: failure.Blob})
	if !errors.As(err, &failure) {
		t.Fatalf("replay did not fail: %v", err)
	}
	if replayed != 5 {
		t.Fatalf("replayed value = %d, want 5", replayed)
	}
}

func TestReplayPunsAcrossChoiceKinds(t *testing.T) {
	var seen bool
	property := func(tc *TestCase) error {
		value, err := tc.Boolean()
		if err != nil {
			return err
		}
		seen = value
		return errors.New("always fails")
	}
	var failure *Failure
	if err := Check(property, Config{Reproduce: crossLanguageIntegerFive}); !errors.As(err, &failure) {
		t.Fatalf("punned replay did not fail: %v", err)
	}
	if !seen {
		t.Fatal("integer choice should pun to the boolean unit true")
	}
}

func TestForcedDrawSkipsReplayPosition(t *testing.T) {
	raw := choiceListHeader(2)
	raw = append(raw, serializeIntegerChoice(signedBytes(5))...)
	raw = append(raw, serializeIntegerChoice(signedBytes(7))...)
	blob := rawFailureEncode(raw)
	var forced, replayed int64
	property := func(tc *TestCase) error {
		var err error
		forced, err = tc.integerForced(99)
		if err != nil {
			return err
		}
		replayed, err = tc.Integer(0, 10, 0)
		if err != nil {
			return err
		}
		return errors.New("always fails")
	}
	var failure *Failure
	if err := Check(property, Config{Reproduce: blob}); !errors.As(err, &failure) {
		t.Fatalf("forced replay did not fail: %v", err)
	}
	if forced != 99 || replayed != 7 {
		t.Fatalf("got forced=%d replayed=%d, want 99 and 7", forced, replayed)
	}
}

func TestReplayOverrunIsNotFailure(t *testing.T) {
	property := func(tc *TestCase) error {
		if _, err := tc.Integer(0, 10, 0); err != nil {
			return err
		}
		if _, err := tc.Integer(0, 10, 0); err != nil {
			return err
		}
		return errors.New("unreachable failure")
	}
	err := Check(property, Config{Reproduce: crossLanguageIntegerFive})
	var overrun *ReplayOverrun
	if !errors.As(err, &overrun) {
		t.Fatalf("expected ReplayOverrun, got %v", err)
	}
	var failure *Failure
	if errors.As(err, &failure) {
		t.Fatalf("overrun was misclassified as Failure: %v", err)
	}
}

func TestPanickedReplayOverrunIsNotFailure(t *testing.T) {
	property := func(tc *TestCase) error {
		MustDraw(tc, Integers().MinValue(0).MaxValue(10))
		MustDraw(tc, Integers().MinValue(0).MaxValue(10))
		return errors.New("unreachable failure")
	}
	err := Check(property, Config{Reproduce: crossLanguageIntegerFive})
	var overrun *ReplayOverrun
	if !errors.As(err, &overrun) {
		t.Fatalf("expected ReplayOverrun, got %v", err)
	}
	var failure *Failure
	if errors.As(err, &failure) {
		t.Fatalf("panicked overrun was misclassified as Failure: %v", err)
	}
}

func TestPanickedChoiceBudgetOverrunMatchesTheReturnedForm(t *testing.T) {
	drain := func(tc *TestCase) error {
		for i := 0; i < 10_000; i++ {
			if _, err := tc.Integer(0, 100, 0); err != nil {
				return err
			}
		}
		return nil
	}
	returned := Check(func(tc *TestCase) error { return drain(tc) }, Config{Examples: 1, Seed: 7})
	panicked := Check(func(tc *TestCase) error {
		if err := drain(tc); err != nil {
			panic(err)
		}
		return nil
	}, Config{Examples: 1, Seed: 7})
	if fmt.Sprintf("%T: %v", panicked, panicked) != fmt.Sprintf("%T: %v", returned, returned) {
		t.Fatalf("panicked draw error diverged from the returned one:\n%v\nvs\n%v", panicked, returned)
	}
	var failure *Failure
	if errors.As(panicked, &failure) {
		t.Fatalf("choice budget overrun was misclassified as Failure: %v", panicked)
	}
}

func TestMustDrawConfigurationErrorFailsTheProperty(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		MustDraw(tc, Lists(Integers()).MinSize(-1))
		return nil
	}, Config{Examples: 5, Seed: 1})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("expected a configuration error to fail as a counterexample, got %v", err)
	}
}

func TestNestedChoiceSpans(t *testing.T) {
	var spans []ChoiceSpan
	err := Check(func(tc *TestCase) error {
		outer := tc.StartSpan("outer")
		if outer != 0 {
			t.Fatalf("outer span ID = %d, want 0", outer)
		}
		if _, err := tc.Integer(0, 10, 0); err != nil {
			return err
		}
		inner := tc.StartSpan("inner")
		if _, err := tc.Boolean(); err != nil {
			return err
		}
		tc.StopSpan(true)
		if _, err := tc.Bytes(0, 2); err != nil {
			return err
		}
		tc.StopSpan(false)
		spans = tc.Spans()
		if inner != 1 {
			t.Fatalf("inner span ID = %d, want 1", inner)
		}
		return nil
	}, Config{Examples: 1, Seed: 17})
	if err != nil {
		t.Fatalf("span property failed: %v", err)
	}
	want := []ChoiceSpan{
		{Start: 0, End: 3, Label: "outer", Depth: 0, Parent: -1},
		{Start: 1, End: 2, Label: "inner", Depth: 1, Parent: 0, Discarded: true},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("spans = %#v, want %#v", spans, want)
	}
}

func TestListShrinksByDeletingElements(t *testing.T) {
	element := Integers().MinValue(0).MaxValue(1000)
	var seen [][]int64
	property := func(tc *TestCase) error {
		values, err := Draw(tc, Lists[int64](element))
		if err != nil {
			return err
		}
		total := int64(0)
		for _, value := range values {
			total += value
		}
		if total >= 50 {
			return errors.New("sum too large")
		}
		return nil
	}
	var failure *Failure
	if !errors.As(Check(property, Config{Examples: 200, Seed: 7}), &failure) {
		t.Fatal("the list property did not fail")
	}
	replay := func(tc *TestCase) error {
		values, err := Draw(tc, Lists[int64](element))
		if err != nil {
			return err
		}
		seen = append(seen, values)
		return errors.New("replay")
	}
	var replayFailure *Failure
	if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
		t.Fatal("replaying the shrunk blob did not fail")
	}
	// Not merely "shorter": the one element left is the smallest that fails.
	if !reflect.DeepEqual(seen, [][]int64{{50}}) {
		t.Fatalf("shrank to %v, want [[50]]", seen)
	}
}

func TestListDeletesAnElementFromTheMiddle(t *testing.T) {
	element := Integers().MinValue(0).MaxValue(10)
	property := func(tc *TestCase) error {
		values, err := Draw(tc, Lists[int64](element))
		if err != nil {
			return err
		}
		for _, value := range values {
			if value == 3 {
				return errors.New("found three")
			}
		}
		return nil
	}
	var failure *Failure
	if !errors.As(Check(property, Config{Examples: 500, Seed: 3}), &failure) {
		t.Fatal("the list property did not fail")
	}
	var seen [][]int64
	replay := func(tc *TestCase) error {
		values, err := Draw(tc, Lists[int64](element))
		if err != nil {
			return err
		}
		seen = append(seen, values)
		return errors.New("replay")
	}
	var replayFailure *Failure
	if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
		t.Fatal("replaying the shrunk blob did not fail")
	}
	// Every element around the interesting one has to be deletable, not only
	// the trailing ones a truncating shrinker can drop.
	if !reflect.DeepEqual(seen, [][]int64{{3}}) {
		t.Fatalf("shrank to %v, want [[3]]", seen)
	}
}

func TestListRespectsItsSizeBounds(t *testing.T) {
	element := Integers().MinValue(0).MaxValue(10)
	lengths := map[int]bool{}
	err := Check(func(tc *TestCase) error {
		values, err := Draw(tc, Lists[int64](element).MinSize(2).MaxSize(4))
		if err != nil {
			return err
		}
		lengths[len(values)] = true
		return nil
	}, Config{Examples: 200, Seed: 5})
	if err != nil {
		t.Fatalf("bounded list property failed: %v", err)
	}
	for length := range lengths {
		if length < 2 || length > 4 {
			t.Fatalf("bounded list produced length %d", length)
		}
	}
	if len(lengths) < 2 {
		t.Fatalf("bounded list never varied its length: %v", lengths)
	}
}

func TestFixedSizeListDrawsNoLengthChoice(t *testing.T) {
	element := Integers().MinValue(0).MaxValue(10)
	var kinds []string
	err := Check(func(tc *TestCase) error {
		if _, err := Draw(tc, Lists[int64](element).MinSize(2).MaxSize(2)); err != nil {
			return err
		}
		for _, item := range tc.draws {
			kinds = append(kinds, item.kind)
		}
		return nil
	}, Config{Examples: 1, Seed: 1})
	if err != nil {
		t.Fatalf("fixed-size list property failed: %v", err)
	}
	// Coinciding bounds decide nothing, so the stream holds the two elements
	// and no "another element?" booleans at all.
	if !reflect.DeepEqual(kinds, []string{"integer", "integer"}) {
		t.Fatalf("fixed-size list recorded %v, want two integers", kinds)
	}
}

func TestListElementsAreRecordedAsSpans(t *testing.T) {
	element := Integers().MinValue(0).MaxValue(3)
	var spans []ChoiceSpan
	err := Check(func(tc *TestCase) error {
		if _, err := Collect(tc, element, 2, 2, true, "list.element"); err != nil {
			return err
		}
		spans = tc.Spans()
		return nil
	}, Config{Examples: 1, Seed: 1})
	if err != nil {
		t.Fatalf("span property failed: %v", err)
	}
	// One span per element plus the empty span of the iteration that stopped;
	// each element span is exactly the range deletion removes.
	want := []ChoiceSpan{
		{Start: 0, End: 1, Label: "list.element", Depth: 0, Parent: -1},
		{Start: 1, End: 2, Label: "list.element", Depth: 0, Parent: -1},
		{Start: 2, End: 2, Label: "list.element", Depth: 0, Parent: -1, Discarded: true},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("spans = %#v, want %#v", spans, want)
	}
}

func TestSetElementsAreAlwaysDistinct(t *testing.T) {
	element := Integers().MinValue(0).MaxValue(6)
	longest := 0
	err := Check(func(tc *TestCase) error {
		values, err := Draw(tc, Sets[int64](element).MaxSize(6))
		if err != nil {
			return err
		}
		seen := map[int64]bool{}
		for _, value := range values {
			if seen[value] {
				return fmt.Errorf("set repeated %d: %v", value, values)
			}
			seen[value] = true
		}
		if len(values) > longest {
			longest = len(values)
		}
		return nil
	}, Config{Examples: 300, Seed: 3})
	if err != nil {
		t.Fatalf("set property failed: %v", err)
	}
	// That a set grew past one element at all is what proves the element loop
	// kept asking after a duplicate rather than stopping on the first one.
	if longest < 2 {
		t.Fatalf("set never grew past %d elements", longest)
	}
}

func TestDuplicateCandidatesAreDiscardedNotKept(t *testing.T) {
	// Two distinct values in the range and three requested elements, so the
	// third element is guaranteed to burn its candidates and reject.
	element := Integers().MinValue(0).MaxValue(1)
	var spans []ChoiceSpan
	err := Check(func(tc *TestCase) error {
		identity := func(value int64) int64 { return value }
		if _, err := CollectUnique(tc, element, identity, 0, 3, true, "set.element"); err != nil {
			return err
		}
		spans = tc.Spans()
		return nil
	}, Config{Examples: 1, Seed: 1})
	if err != nil {
		t.Fatalf("unique collection property failed: %v", err)
	}
	discarded := false
	for _, span := range spans {
		if span.Discarded && span.End > span.Start {
			discarded = true
		}
	}
	if !discarded {
		t.Fatalf("a duplicate candidate left no discarded span behind: %#v", spans)
	}
}

func TestUnsatisfiableUniquenessRejectsRatherThanHanging(t *testing.T) {
	// Four distinct values cannot come out of a two-value range.
	element := Integers().MinValue(0).MaxValue(1)
	err := Check(func(tc *TestCase) error {
		if _, drawErr := Draw(tc, Sets[int64](element).MinSize(4)); drawErr != nil {
			return drawErr
		}
		return errors.New("unreachable")
	}, Config{Examples: 50, Seed: 1})
	var rejections *TooManyRejections
	var health *FailedHealthCheck
	if !errors.As(err, &rejections) && !errors.As(err, &health) {
		t.Fatalf("expected a rejection outcome, got %v", err)
	}
}

func TestRejectActionMatchesTheGeneratedRule(t *testing.T) {
	// The three actions, read straight off the compiled decision so the shell's
	// branch names cannot drift from what they dispatch on.
	for _, want := range []struct {
		rejections uint64
		count      int64
		minSize    int64
		action     uint64
	}{
		{1, 0, 0, rejectRetry},
		{4, 0, 0, rejectStop},
		{4, 0, 1, rejectInvalid},
		{4, 2, 0, rejectRetry},
	} {
		got := manyRejectAction(want.rejections, want.count, want.minSize)
		if got != want.action {
			t.Fatalf("manyRejectAction(%d, %d, %d) = %d, want %d",
				want.rejections, want.count, want.minSize, got, want.action)
		}
	}
	if budget := manyRejectionBudget(0); budget != 3 {
		t.Fatalf("manyRejectionBudget(0) = %d, want the floor of 3", budget)
	}
}

func TestMapKeysAreDistinctAndValuesAreFree(t *testing.T) {
	keys := Integers().MinValue(0).MaxValue(3)
	values := Integers().MinValue(0).MaxValue(1)
	grew, repeated := false, false
	err := Check(func(tc *TestCase) error {
		mapping, err := Draw(tc, Maps[int64, int64](keys, values).MaxSize(4))
		if err != nil {
			return err
		}
		if len(mapping) > 1 {
			grew = true
			distinct := map[int64]bool{}
			for _, value := range mapping {
				distinct[value] = true
			}
			if len(distinct) < len(mapping) {
				repeated = true
			}
		}
		return nil
	}, Config{Examples: 200, Seed: 4})
	if err != nil {
		t.Fatalf("map property failed: %v", err)
	}
	if !grew {
		t.Fatal("map never held more than one entry")
	}
	// Values repeat freely — uniqueness is judged on the key alone.
	if !repeated {
		t.Fatal("no map ever repeated a value, so the key predicate is too wide")
	}
}

func TestOpenSpanClosesOnPanic(t *testing.T) {
	tc := newRandomCase(rand.New(rand.NewSource(1)))
	_ = tc.StartSpan("open")
	_ = execute(func(tc *TestCase) error {
		if _, err := tc.Boolean(); err != nil {
			return err
		}
		panic("boom")
	}, tc)
	spans := tc.Spans()
	if len(spans) != 1 || spans[0].End != 1 {
		t.Fatalf("open span was not closed at failure: %#v", spans)
	}
}

func TestForcedChoiceHasNoShrinkCandidates(t *testing.T) {
	tc := newRandomCase(rand.New(rand.NewSource(1)))
	if _, err := tc.integerForced(99); err != nil {
		t.Fatal(err)
	}
	if candidates := candidateDraws(tc.draws[0]); len(candidates) != 0 {
		t.Fatalf("forced choice produced shrink candidates: %#v", candidates)
	}
}

func TestSortKeyIsChoiceCountFirstShortlex(t *testing.T) {
	falseDraw := draw{kind: "boolean", boolean: false}
	trueDraw := draw{kind: "boolean", boolean: true}
	if !drawsSimpler([]draw{trueDraw}, []draw{falseDraw, falseDraw}) {
		t.Fatal("a shorter choice stream must be simpler")
	}
	if !drawsSimpler([]draw{falseDraw, trueDraw}, []draw{trueDraw, falseDraw}) {
		t.Fatal("the first lower boolean index must win at equal length")
	}
	if drawsSimpler([]draw{trueDraw, falseDraw}, []draw{falseDraw, trueDraw}) {
		t.Fatal("a higher first boolean index cannot be simpler")
	}
	falseDraw.forced = true
	trueDraw.forced = true
	if !drawsSimpler([]draw{falseDraw}, []draw{trueDraw}) {
		t.Fatal("forced choice values must still participate in ordering")
	}
	bytes8 := make([]byte, 8)
	bytes9 := make([]byte, 9)
	draw8 := draw{kind: "bytes", bytes: bytes8, params: []int64{0, 16}}
	draw9 := draw{kind: "bytes", bytes: bytes9, params: []int64{0, 16}}
	if !drawsSimpler([]draw{draw8}, []draw{draw9}) ||
		drawsSimpler([]draw{draw9}, []draw{draw8}) {
		t.Fatal("full-width byte indexes must compare without u64 saturation")
	}
}

// A narrow range is drawn uniformly, so a draw that indexes a list picks its
// first and last option no more often than any other. The old flat boundary
// boost gave each endpoint of 0..=199 about 1% of draws instead of 0.5%.
func TestNarrowIntegerDrawsAreUniform(t *testing.T) {
	draws := 0
	hits := map[int64]int{0: 0, 199: 0}
	for seed := int64(0); seed < 5_000; seed++ {
		tc := newRandomCase(rand.New(rand.NewSource(seed + 1)))
		for i := 0; i < 40; i++ {
			value, err := tc.Integer(0, 199, 0)
			if err != nil {
				t.Fatal(err)
			}
			draws++
			if _, endpoint := hits[value]; endpoint {
				hits[value]++
			}
		}
	}
	for endpoint, count := range hits {
		share := float64(count) / float64(draws)
		if share <= 0.0035 || share >= 0.0065 {
			t.Fatalf("value %d took %.4f%% of %d narrow draws", endpoint, share*100, draws)
		}
	}
}

func TestBiasedIntegerFindsLedgerBoundary(t *testing.T) {
	property := func(tc *TestCase) error {
		value, err := tc.Integer(0, 1_000_000, 0)
		if err != nil {
			return err
		}
		if value == 1 {
			return errors.New("found one")
		}
		return nil
	}
	var failure *Failure
	if err := Check(property, Config{Examples: 5000, Seed: 23}); !errors.As(err, &failure) {
		t.Fatalf("biased generation did not find one: %v", err)
	}
}

func TestWideIntegerSwarmClustersEndpointOperands(t *testing.T) {
	const cases = 20_000
	endpointPairs := 0
	for seed := int64(0); seed < cases; seed++ {
		tc := newRandomCase(rand.New(rand.NewSource(seed + 1)))
		left, err := tc.Integer(math.MinInt64, math.MaxInt64, 0)
		if err != nil {
			t.Fatal(err)
		}
		right, err := tc.Integer(math.MinInt64, math.MaxInt64, 0)
		if err != nil {
			t.Fatal(err)
		}
		isEndpoint := func(value int64) bool {
			return value == math.MinInt64 || value == math.MinInt64+1 ||
				value == math.MaxInt64-1 || value == math.MaxInt64
		}
		if isEndpoint(left) && isEndpoint(right) {
			endpointPairs++
		}
	}
	// With the old independent fixed bias, two endpoints almost never landed
	// in one case. These generous bounds catch loss of family-level clustering
	// without pinning the host RNG to one exact distribution.
	if endpointPairs < 150 || endpointPairs > 800 {
		t.Fatalf("wide integer swarm produced %d/%d endpoint pairs", endpointPairs, cases)
	}
}

type recursiveTestTree struct {
	left  *recursiveTestTree
	right *recursiveTestTree
}

func (tree recursiveTestTree) height() int {
	if tree.left == nil {
		return 0
	}
	left, right := tree.left.height(), tree.right.height()
	if right > left {
		left = right
	}
	return left + 1
}

func (tree recursiveTestTree) leaves() int {
	if tree.left == nil {
		return 1
	}
	return tree.left.leaves() + tree.right.leaves()
}

func recursiveTestTrees() RecursiveGenerator[recursiveTestTree] {
	return Recursive(
		Just(recursiveTestTree{}),
		func(children Generator[recursiveTestTree]) Generator[recursiveTestTree] {
			return Composite(func(tc *TestCase) (recursiveTestTree, error) {
				left, err := Draw(tc, children)
				if err != nil {
					return recursiveTestTree{}, err
				}
				right, err := Draw(tc, children)
				if err != nil {
					return recursiveTestTree{}, err
				}
				return recursiveTestTree{left: &left, right: &right}, nil
			})
		},
	)
}

func TestRecursiveHonorsDepthAndLeafBudgets(t *testing.T) {
	sawBranch := false
	err := Check(func(tc *TestCase) error {
		tree, err := Draw(tc, recursiveTestTrees().MaxDepth(3).MaxLeaves(4))
		if err != nil {
			return err
		}
		if tree.height() > 3 || tree.leaves() > 4 {
			return fmt.Errorf("recursive bounds violated: height=%d leaves=%d", tree.height(), tree.leaves())
		}
		sawBranch = sawBranch || tree.left != nil
		return nil
	}, Config{Examples: 200, Seed: 0xE3})
	if err != nil {
		t.Fatal(err)
	}
	if !sawBranch {
		t.Fatal("recursive generator never produced a branch")
	}
}

func TestRecursiveDepthCapRecordsForcedLeaf(t *testing.T) {
	tc := newRandomCase(rand.New(rand.NewSource(1)))
	if _, err := Draw(tc, recursiveTestTrees().MaxDepth(0)); err != nil {
		t.Fatal(err)
	}
	if len(tc.draws) < 2 || !tc.draws[len(tc.draws)-1].forced || tc.draws[len(tc.draws)-1].boolean {
		t.Fatalf("depth cap did not record a forced false decision: %#v", tc.draws)
	}
	spans := tc.Spans()
	if len(spans) == 0 || spans[len(spans)-1].Label != "recursive" || spans[len(spans)-1].Discarded {
		t.Fatalf("recursive subtree span missing: %#v", spans)
	}
}

func TestRecursiveZeroLeafBudgetCanGenerateLeaflessBranch(t *testing.T) {
	leafless := Recursive(
		Just(false),
		func(Generator[bool]) Generator[bool] { return Just(true) },
	).MaxLeaves(0)
	err := Check(func(tc *TestCase) error {
		value, err := Draw(tc, leafless)
		if err != nil {
			return err
		}
		if !value {
			return errors.New("zero leaf budget accepted a leaf")
		}
		return nil
	}, Config{Examples: 50, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMustDrawInsideRecursiveBranchRetriesTheLeafBudget(t *testing.T) {
	// The same shape as TestRecursiveHonorsDepthAndLeafBudgets, but the branch
	// composite surfaces the leaf-budget error by panic instead of return. The
	// retry loop must still catch it; escaping, it would report a budget
	// rejection as a counterexample.
	trees := Recursive(
		Just(recursiveTestTree{}),
		func(children Generator[recursiveTestTree]) Generator[recursiveTestTree] {
			return Composite(func(tc *TestCase) (recursiveTestTree, error) {
				left := MustDraw(tc, children)
				right := MustDraw(tc, children)
				return recursiveTestTree{left: &left, right: &right}, nil
			})
		},
	).MaxDepth(3).MaxLeaves(4)
	err := Check(func(tc *TestCase) error {
		tree := MustDraw(tc, trees)
		if tree.height() > 3 || tree.leaves() > 4 {
			return fmt.Errorf("recursive bounds violated: height=%d leaves=%d", tree.height(), tree.leaves())
		}
		return nil
	}, Config{Examples: 200, Seed: 0xE3})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReusableAndConsumedPools(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		values := NewPool[string](tc)
		if err := values.Add(tc, "old"); err != nil {
			return err
		}
		if err := values.Add(tc, "new"); err != nil {
			return err
		}
		reused, err := Draw(tc, values.ValuesReusable())
		if err != nil {
			return err
		}
		if *reused != "old" && *reused != "new" {
			t.Fatalf("unexpected reused value %q", *reused)
		}
		consumed, err := Draw(tc, values.ValuesConsumed())
		if err != nil {
			return err
		}
		if consumed != "old" && consumed != "new" {
			t.Fatalf("unexpected consumed value %q", consumed)
		}
		if values.Len() != 1 {
			t.Fatalf("pool length = %d, want 1", values.Len())
		}
		return nil
	}, Config{Examples: 20, Seed: 11})
	if err != nil {
		t.Fatal(err)
	}
}

// Adding to a pool draws a recorded identifier, and every reference replays
// onto the same value it named when recorded.
//
// Adding used to consume no choice at all and a reference used to be a reverse
// index into the live list, so the pool's whole identity scheme lived in host
// state the blob never saw. Both halves are now in the stream (hegel-c
// 0.32.4), which is what this pins: the identifier draws are real entries, and
// replaying the blob resolves the same variables rather than whatever now sits
// at those positions.
func TestPoolAddRecordsAnIdentifierThatReplays(t *testing.T) {
	var drawn [][2]string
	property := func(tc *TestCase) error {
		values := NewPool[string](tc)
		for _, name := range []string{"a", "b", "c"} {
			if err := values.Add(tc, name); err != nil {
				return err
			}
		}
		var picked [2]string
		for index := range picked {
			value, err := Draw(tc, values.ValuesReusable())
			if err != nil {
				return err
			}
			picked[index] = *value
		}
		drawn = append(drawn, picked)
		return errors.New("always fails")
	}

	var failure *Failure
	if !errors.As(Check(property, Config{Examples: 1, Seed: 5}), &failure) {
		t.Fatal("the always-failing property did not fail")
	}
	recorded := drawn[len(drawn)-1]
	raw, ok := rawFailureDecode([]byte(failure.Blob))
	if !ok {
		t.Fatal("blob did not decode")
	}
	// Three adds and two references: five recorded entries where the old
	// encoding recorded only the two references.
	if count := choiceStreamCount(raw); count != 5 {
		t.Fatalf("choice count = %d, want 5", count)
	}

	drawn = nil
	if !errors.As(Check(property, Config{Reproduce: failure.Blob}), &failure) {
		t.Fatal("replaying the blob did not fail")
	}
	if replayed := drawn[len(drawn)-1]; replayed != recorded {
		t.Fatalf("replay drew %v, want %v", replayed, recorded)
	}
}

// The minimal failure of thirdA under the continue protocol, and the same
// failure as the stop protocol recorded it before the decision was inverted.
const (
	statefulThirdAFailure      = "AXichYgxCgAACIR0Dfr/dyOC1nNRbEApWU5+mM8ADpIAZA=="
	statefulThirdAStopProtocol = "AXiclcaxDQAACMMwd0Xi/3dh4QCGKG6E2tzywQAOIgBh"
	// thirdA's failure behind one integer drawn before the machine starts.
	statefulDrawBeforeThirdAFailure = "AXicfYmxDQAACMLs6uD/56qJTiQwEGgrJsk23PgvwpAGEhwAcA=="
)

// thirdA's rule "a" fails on its third run. Both rules draw an integer first,
// which is what made the legacy stream read back as protocol 2.
type thirdA struct{ seen []string }

func thirdAMachine() Machine[thirdA] {
	rule := func(name string) Rule[thirdA] {
		return Rule[thirdA]{Name: name, Apply: func(m *thirdA, tc *TestCase) error {
			if _, err := tc.Integer(0, 10, 0); err != nil {
				return err
			}
			m.seen = append(m.seen, name)
			runs := 0
			for _, seen := range m.seen {
				if seen == "a" {
					runs++
				}
			}
			if runs >= 3 {
				return errors.New("a ran three times")
			}
			return nil
		}}
	}
	return Machine[thirdA]{Rules: []Rule[thirdA]{rule("a"), rule("b")}}
}

// replayThirdA returns the rules a thirdA blob runs, failing unless it fails.
func replayThirdA(t *testing.T, blob string) []string {
	t.Helper()
	var state *thirdA
	err := Check(func(tc *TestCase) error {
		state = &thirdA{}
		return Run(state, thirdAMachine(), tc)
	}, Config{Reproduce: blob})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("%s did not reproduce a failure: %v", blob, err)
	}
	return state.seen
}

// thirdAAfterDraw draws an integer before running thirdA, as a property that
// draws its own configuration first does.
func thirdAAfterDraw(state *thirdA, tc *TestCase) error {
	if _, err := tc.Integer(0, 10, 0); err != nil {
		return err
	}
	return Run(state, thirdAMachine(), tc)
}

// A draw before the machine starts must not be read as the machine's own.
// The protocol sniffer used to read the head of the stream, so the leading
// integer made a continue-protocol stream look like a stop-protocol one: the
// replay inverted every decision and stopped after one step, the shrinker
// reported the failure as flaky, and the blob it printed did not reproduce.
func TestDrawBeforeMachineShrinksAndReplays(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		return thirdAAfterDraw(&thirdA{}, tc)
	}, Config{Examples: 100, Seed: 1})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("the machine did not fail with a shrunk failure: %v", err)
	}
	if failure.Blob != statefulDrawBeforeThirdAFailure {
		t.Fatalf("blob = %s, want %s", failure.Blob, statefulDrawBeforeThirdAFailure)
	}
	var state *thirdA
	err = Check(func(tc *TestCase) error {
		state = &thirdA{}
		return thirdAAfterDraw(state, tc)
	}, Config{Reproduce: failure.Blob})
	if !errors.As(err, &failure) {
		t.Fatalf("%s did not reproduce a failure: %v", statefulDrawBeforeThirdAFailure, err)
	}
	if seen := strings.Join(state.seen, ","); seen != "a,a,a" {
		t.Fatalf("replay ran %s, want a,a,a", seen)
	}
}

// The per-step continue decision (hegel-c 0.36.1). The next three tests run
// with the same constants in every shell's test file, because the shared
// shell-case table has no way to describe a machine.

// The simplest value of a continue decision stops. Under the old stop decision
// the all-simplest probe ran every one of its steps, so a one-rule machine at
// 5000 steps overran the choice budget and failed the large_base_example
// health check (upstream #433). With one example the probe is the only case
// the run executes.
func TestSimplestProbeRunsOneStepUnderALargeStepCount(t *testing.T) {
	type pusher struct{ pushed []int64 }
	machine := Machine[pusher]{Rules: []Rule[pusher]{{
		Name: "push",
		Apply: func(m *pusher, tc *TestCase) error {
			value, err := tc.Integer(0, 10, 0)
			m.pushed = append(m.pushed, value)
			return err
		},
	}}}
	var counts []int
	err := Check(func(tc *TestCase) error {
		state := &pusher{}
		err := RunSteps(state, machine, tc, 5000)
		counts = append(counts, len(state.pushed))
		return err
	}, Config{Examples: 1, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 1 || counts[0] != 1 {
		t.Fatalf("step counts = %v, want [1]", counts)
	}
}

// A failure the simplest probe cannot reach shrinks and replays. The probe
// used to record the legacy protocol, which the sniffer reads back as protocol
// 2 once a rule draws an integer first, so this run failed with "shrunk
// example no longer fails". Every shell converges on the same three-step
// stream, so the blob is pinned.
func TestThirdRunFailureShrinksToThreeStepsAndReplays(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		return Run(&thirdA{}, thirdAMachine(), tc)
	}, Config{Examples: 100, Seed: 1})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("the machine did not fail: %v", err)
	}
	raw, ok := rawFailureDecode([]byte(failure.Blob))
	if !ok {
		t.Fatal("blob did not decode")
	}
	if protocol := statefulChoiceProtocol(raw, 4, choiceStreamCount(raw)); protocol != 3 {
		t.Fatalf("protocol = %d, want 3", protocol)
	}
	if failure.Blob != statefulThirdAFailure {
		t.Fatalf("blob = %s, want %s", failure.Blob, statefulThirdAFailure)
	}
	if seen := strings.Join(replayThirdA(t, failure.Blob), ","); seen != "a,a,a" {
		t.Fatalf("replay ran %s, want a,a,a", seen)
	}
}

// A continue-protocol blob replays in every shell, and blobs recorded under the
// stop decision keep their meaning. The shared ledger fixture is the other
// stop-protocol blob, gated by examples/ledger's own test.
func TestPinnedBlobsOfBothProtocolsReplay(t *testing.T) {
	for _, pinned := range []struct {
		blob     string
		protocol uint64
	}{
		{statefulThirdAFailure, 3},
		{statefulThirdAStopProtocol, 2},
	} {
		raw, ok := rawFailureDecode([]byte(pinned.blob))
		if !ok {
			t.Fatalf("%s did not decode", pinned.blob)
		}
		if protocol := statefulChoiceProtocol(raw, 4, choiceStreamCount(raw)); protocol != pinned.protocol {
			t.Fatalf("%s: protocol = %d, want %d", pinned.blob, protocol, pinned.protocol)
		}
		if seen := strings.Join(replayThirdA(t, pinned.blob), ","); seen != "a,a,a" {
			t.Fatalf("%s: replay ran %s, want a,a,a", pinned.blob, seen)
		}
	}
}

type bugA struct{ value int64 }

func (e bugA) Error() string { return fmt.Sprintf("bug A: %d", e.value) }

// A run that discovers one bug must not report a counterexample for another
// one it stumbled into while shrinking.
func TestShrinkingStaysOnTheBugThatWasFound(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		found := ""
		property := func(tc *TestCase) error {
			value, err := tc.Integer(0, 100, 0)
			if err != nil {
				return err
			}
			if value > 50 {
				if found == "" {
					found = "A"
				}
				return bugA{value}
			}
			if value == 7 {
				if found == "" {
					found = "B"
				}
				return errors.New("bug B")
			}
			return nil
		}
		err := Check(property, Config{Examples: 50, MaxRejections: 1000, Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			continue
		}
		var reportedA bugA
		if found == "A" && !errors.As(failure.Unwrap(), &reportedA) {
			t.Fatalf("seed %d found bug A but reported %v", seed, failure.Unwrap())
		}
	}
}

type bugB struct{}

func (bugB) Error() string { return "bug B" }

// shellCases is the shell-level counterpart to the golden vector gate. Every
// shell runs the same table from conformance/shell/shell_cases.tsv, so a behavior
// that drifts in one language fails there rather than in review.
//
// The tables live outside this module: they are shared with the other
// language packages so a behavior that drifts in one fails here. They are
// therefore not shipped with it, and where they are absent the cases skip
// and say so rather than passing quietly.
func shellCases(t *testing.T) [][]string {
	t.Helper()
	var cases [][]string
	// The second table is generated by tools/tracegen from the real Hegel
	// engine, and read separately rather than merged into the hand-written one
	// so that it stays regenerable.
	for _, name := range []string{"shell/shell_cases.tsv", "shell/trace_vectors.tsv"} {
		path := filepath.Join("..", "..", "conformance", name)
		raw, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("%s is missing: the shared conformance tables live in the Zizek source tree and are not shipped with this package", path)
		}
		if err != nil {
			t.Fatalf("read shell cases: %v", err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			cases = append(cases, strings.Split(line, "\t"))
		}
	}
	return cases
}

func caseTrace(t *testing.T, fields []string) string {
	envelope, blob, script, expected := fields[0], fields[1], fields[2], fields[3]
	ops := strings.Split(script, ",")
	_ = envelope
	var drawn []string
	err := Check(func(tc *TestCase) error {
		for _, op := range ops {
			value, drawErr := TraceDraw(tc, op)
			if drawErr != nil {
				return drawErr
			}
			drawn = append(drawn, value)
		}
		return errors.New("replay")
	}, Config{Reproduce: blob})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("replaying the engine's blob did not fail: %v", err)
	}
	if got := strings.Join(drawn, ","); got != expected {
		t.Fatalf("replayed the engine's blob as %s, want %s", got, expected)
	}
	return ""
}

func atoi(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatalf("bad number %q: %v", value, err)
	}
	return parsed
}

func TestSharedShellCases(t *testing.T) {
	cases := shellCases(t)
	if len(cases) == 0 {
		t.Fatal("no shared shell cases were run")
	}
	pinned := 0
	for _, shellCase := range cases {
		kind, rest := shellCase[0], shellCase[1:]
		// An optional trailing blob=<base64> pins the reproduce blob the case
		// must emit. It is stripped before dispatch so the handlers keep their
		// positional fields, and compared against what the handler returns.
		// See conformance/shell/shell_cases.tsv.
		expectedBlob := ""
		if len(rest) != 0 && strings.HasPrefix(rest[len(rest)-1], "blob=") {
			expectedBlob = strings.TrimPrefix(rest[len(rest)-1], "blob=")
			rest = rest[:len(rest)-1]
			pinned++
		}
		t.Run(strings.Join(shellCase, " "), func(t *testing.T) {
			emitted, handled := runDeclaredShellCase(t, kind, rest)
			if !handled {
				switch kind {
				case "TRACE":
					caseTrace(t, rest)
				default:
					t.Fatalf("unknown shell case %q", kind)
				}
			}
			if expectedBlob != "" && emitted != expectedBlob {
				t.Fatalf("%s emitted %q, want the shared %q", kind, emitted, expectedBlob)
			}
		})
	}
	if pinned == 0 {
		t.Fatal("no shared case pinned its blob")
	}
}

func TestOutOfRangeShrinkTargetClampsToTheNearerBound(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		value, drawErr := Draw[int64](tc, Integers().MinValue(0).MaxValue(10).ShrinkTowards(99))
		if drawErr != nil {
			return drawErr
		}
		return fmt.Errorf("always fails at %d", value)
	}, Config{Examples: 50, Seed: 7})

	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("expected a failure, got %v", err)
	}
	if got := failure.Cause.Error(); got != "always fails at 10" {
		t.Fatalf("shrank to %q, want the clamped target 10", got)
	}
	if failure.Blob != crossLanguageClampedTarget {
		t.Fatalf("blob %q, want %q", failure.Blob, crossLanguageClampedTarget)
	}
}

// The catalogue combinators, pinned across every package that has them. Just
// and OneOf's chosen branch record no choice of their own, and SampledFrom and
// OneOf record exactly one integer each, so the blob is the check that all four
// shells compose them identically.
const crossLanguageCatalogue = "AAMAAAABAQAKAQAAAAIACgEAAAAA"

func catalogueDraw(tc *TestCase) (bool, int64, int64, error) {
	flag, err := Draw(tc, Booleans())
	if err != nil {
		return false, 0, 0, err
	}
	pick, err := Draw(tc, SampledFrom([]int64{2, 3, 5, 7}))
	if err != nil {
		return false, 0, 0, err
	}
	extra, err := Draw(tc, OneOf(Just[int64](0), Integers().MinValue(1).MaxValue(9)))
	if err != nil {
		return false, 0, 0, err
	}
	return flag, pick, extra, nil
}

func TestCatalogueGeneratorsShrinkAndReplayPortably(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		flag, pick, _, drawErr := catalogueDraw(tc)
		if drawErr != nil {
			return drawErr
		}
		if flag && pick >= 5 {
			return errors.New("flagged and large")
		}
		return nil
	}, Config{Examples: 300, Seed: 1})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("the catalogue property did not fail: %v", err)
	}
	if failure.Blob != crossLanguageCatalogue {
		t.Fatalf("catalogue blob = %q, want %q", failure.Blob, crossLanguageCatalogue)
	}

	var seen [][3]int64
	replayErr := Check(func(tc *TestCase) error {
		flag, pick, extra, drawErr := catalogueDraw(tc)
		if drawErr != nil {
			return drawErr
		}
		flagged := int64(0)
		if flag {
			flagged = 1
		}
		seen = append(seen, [3]int64{flagged, pick, extra})
		return errors.New("replay")
	}, Config{Reproduce: failure.Blob})
	if !errors.As(replayErr, &failure) {
		t.Fatalf("replaying the catalogue blob did not fail: %v", replayErr)
	}
	// OneOf shrinks towards its first branch, so the minimal example takes
	// Just(0) rather than paying for an integer draw.
	if len(seen) != 1 || seen[0] != [3]int64{1, 5, 0} {
		t.Fatalf("shrank to %v, want [[1 5 0]]", seen)
	}
}

// flat_map, pinned across every package that has it. The drawn scale decides
// the inner generator's bounds, so the two draws only make sense together;
// the blob is the check that all nine shells compose and shrink the bind
// identically.
const crossLanguageFlatMap = "AXicY2IAAi5GBjjFCgABCgAe"

func flatMapScaled() Generator[int64] {
	return FlatMap(Integers().MinValue(0).MaxValue(3), func(scale int64) Generator[int64] {
		return Integers().MinValue(scale * 10).MaxValue(scale*10 + 100)
	})
}

func TestFlatMapShrinksAndReplaysPortably(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		value, drawErr := Draw(tc, flatMapScaled())
		if drawErr != nil {
			return drawErr
		}
		if value >= 5 {
			return errors.New("value reached 5")
		}
		return nil
	}, Config{Examples: 300, Seed: 1})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("the flat_map property did not fail: %v", err)
	}
	if failure.Blob != crossLanguageFlatMap {
		t.Fatalf("flat_map blob = %q, want %q", failure.Blob, crossLanguageFlatMap)
	}

	var seen []int64
	replayErr := Check(func(tc *TestCase) error {
		value, drawErr := Draw(tc, flatMapScaled())
		if drawErr != nil {
			return drawErr
		}
		seen = append(seen, value)
		return errors.New("replay")
	}, Config{Reproduce: failure.Blob})
	if !errors.As(replayErr, &failure) {
		t.Fatalf("replaying the flat_map blob did not fail: %v", replayErr)
	}
	// The minimal example takes scale 0, then the smallest value the property
	// notices.
	if len(seen) != 1 || seen[0] != 5 {
		t.Fatalf("shrank to %v, want [5]", seen)
	}
}

func TestTargetingSteersGenerationTowardHighScores(t *testing.T) {
	// With 200 examples the compiled schedule fires the hill climber at 100
	// valid examples; unguided random search hits the failing band in roughly
	// one example per ten thousand.
	err := Check(func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 100_000, 0)
		if drawErr != nil {
			return drawErr
		}
		tc.Target(float64(value))
		if value >= 99_990 {
			return errors.New("reached the top")
		}
		return nil
	}, Config{Examples: 200, Seed: 5})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("the targeting property did not fail: %v", err)
	}

	var seen []int64
	replayErr := Check(func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 100_000, 0)
		if drawErr != nil {
			return drawErr
		}
		seen = append(seen, value)
		return errors.New("replay")
	}, Config{Reproduce: failure.Blob})
	if !errors.As(replayErr, &failure) {
		t.Fatalf("replaying the targeting blob did not fail: %v", replayErr)
	}
	if len(seen) != 1 || seen[0] != 99_990 {
		t.Fatalf("shrank to %v, want [99990]", seen)
	}
}

func TestTargetingClimbsEachLabelSeparately(t *testing.T) {
	// The failing band is reachable only through the *second* label in
	// sorted order, so finding it requires the per-label bookkeeping: "high"
	// is climbed first, discovers nothing, and the climb moves on to "low".
	err := Check(func(tc *TestCase) error {
		low, drawErr := tc.Integer(0, 20_000, 0)
		if drawErr != nil {
			return drawErr
		}
		high, drawErr := tc.Integer(0, 20_000, 0)
		if drawErr != nil {
			return drawErr
		}
		tc.TargetLabelled(float64(low), "low")
		tc.TargetLabelled(float64(high), "high")
		if low >= 19_990 {
			return errors.New("low climbed")
		}
		return nil
	}, Config{Examples: 600, Seed: 7})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("the two-label targeting property did not fail: %v", err)
	}

	var seen [][2]int64
	replayErr := Check(func(tc *TestCase) error {
		low, drawErr := tc.Integer(0, 20_000, 0)
		if drawErr != nil {
			return drawErr
		}
		high, drawErr := tc.Integer(0, 20_000, 0)
		if drawErr != nil {
			return drawErr
		}
		seen = append(seen, [2]int64{low, high})
		return errors.New("replay")
	}, Config{Reproduce: failure.Blob})
	if !errors.As(replayErr, &failure) {
		t.Fatalf("replaying the two-label blob did not fail: %v", replayErr)
	}
	if len(seen) != 1 || seen[0] != [2]int64{19_990, 0} {
		t.Fatalf("shrank to %v, want [[19990 0]]", seen)
	}
}

func TestTargetValidatesScoresAndLabels(t *testing.T) {
	tc := newRandomCase(rand.New(rand.NewSource(1)))
	tc.Target(3.5)
	func() {
		defer func() {
			message := fmt.Sprint(recover())
			if !strings.Contains(message, "at most once") {
				t.Fatalf("duplicate label panicked with %q", message)
			}
		}()
		tc.Target(4.0)
	}()
	func() {
		defer func() {
			message := fmt.Sprint(recover())
			if !strings.Contains(message, "finite") {
				t.Fatalf("NaN score panicked with %q", message)
			}
		}()
		tc.TargetLabelled(math.NaN(), "other")
	}()
	tc.TargetLabelled(1.0, "second")
	targets := tc.root().targets
	if len(targets) != 2 || targets[""] != 3.5 || targets["second"] != 1.0 {
		t.Fatalf("targets = %v, want map[:3.5 second:1]", targets)
	}
}

func TestMapAndFilterComposeWithoutRecordingAChoice(t *testing.T) {
	doubled := Map(Integers().MinValue(0).MaxValue(10), func(v int64) int64 { return v * 2 })
	evens := Filter(Integers().MinValue(0).MaxValue(20), func(v int64) bool { return v%2 == 0 })
	err := Check(func(tc *TestCase) error {
		first, drawErr := Draw(tc, doubled)
		if drawErr != nil {
			return drawErr
		}
		second, drawErr := Draw(tc, evens)
		if drawErr != nil {
			return drawErr
		}
		if first%2 != 0 || second%2 != 0 {
			return fmt.Errorf("drew odd values (%d, %d)", first, second)
		}
		return nil
	}, Config{Examples: 50, Seed: 3})
	if err != nil {
		t.Fatalf("map/filter run failed: %v", err)
	}
}

func TestTuplesAndJustDrawInOrder(t *testing.T) {
	var seen [][]any
	err := Check(func(tc *TestCase) error {
		values, drawErr := Draw(tc, Tuples(
			AsAny(Just[int64](1)),
			AsAny(Booleans()),
		))
		if drawErr != nil {
			return drawErr
		}
		seen = append(seen, values)
		return nil
	}, Config{Examples: 1})
	if err != nil {
		t.Fatalf("tuples run failed: %v", err)
	}
	if len(seen) != 1 || len(seen[0]) != 2 || seen[0][0] != int64(1) || seen[0][1] != false {
		t.Fatalf("drew %v, want [[1 false]]", seen)
	}
}

// ---- coverage assertions ----

func TestSometimesNeverHeldFailsTheRun(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 10, 0)
		if drawErr != nil {
			return drawErr
		}
		tc.Sometimes("value above range", value > 1000)
		return nil
	}, Config{Examples: 20, Seed: 5})
	var missed *FailedCoverage
	if !errors.As(err, &missed) {
		t.Fatalf("vacuous run reported %v, want FailedCoverage", err)
	}
	if missed.Label != "value above range" || missed.Observed != 0 || missed.Valid != 20 {
		t.Fatalf("verdict = %+v, want the never-held label with 0 of 20", missed)
	}
	if missed.MinPercent != 0 {
		t.Fatalf("Sometimes reported MinPercent %v, want 0", missed.MinPercent)
	}
}

func TestSometimesHeldOncePasses(t *testing.T) {
	attempts := 0
	err := Check(func(tc *TestCase) error {
		attempts++
		tc.Sometimes("first example", attempts == 1)
		return nil
	}, Config{Examples: 20, Seed: 5})
	if err != nil {
		t.Fatalf("covered run failed: %v", err)
	}
}

func TestDeclaredLabelNeverExecutedFailsTheRun(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		return nil
	}, Config{
		Examples: 10,
		Require:  []Coverage{{Label: "unreached branch"}},
	})
	var missed *FailedCoverage
	if !errors.As(err, &missed) {
		t.Fatalf("undeclared branch reported %v, want FailedCoverage", err)
	}
	if missed.Label != "unreached branch" || missed.Observed != 0 || missed.Valid != 10 {
		t.Fatalf("verdict = %+v, want the declared label with 0 of 10", missed)
	}
}

func TestDeclaredLabelObservedAtACallSitePasses(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		tc.Sometimes("reached", true)
		return nil
	}, Config{
		Examples: 10,
		Require:  []Coverage{{Label: "reached"}},
	})
	if err != nil {
		t.Fatalf("declared and observed run failed: %v", err)
	}
}

func TestDeclaredLabelsAreCheckedBeforeObservedOnes(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		tc.Sometimes("observed but missed", false)
		return nil
	}, Config{
		Examples: 5,
		Require:  []Coverage{{Label: "declared and missed"}},
	})
	var missed *FailedCoverage
	if !errors.As(err, &missed) {
		t.Fatalf("run reported %v, want FailedCoverage", err)
	}
	if missed.Label != "declared and missed" {
		t.Fatalf("reported %q, want the declared label first", missed.Label)
	}
}

func TestCoverCountsOncePerExampleAndEnforcesTheRate(t *testing.T) {
	attempts := 0
	err := Check(func(tc *TestCase) error {
		attempts++
		// Two hits at the same call-site label in one example count once.
		tc.Cover("first example", attempts == 1, 100)
		tc.Cover("first example", attempts == 1, 100)
		return nil
	}, Config{Examples: 4, Seed: 5})
	var missed *FailedCoverage
	if !errors.As(err, &missed) {
		t.Fatalf("25%% coverage at a 100%% requirement reported %v, want FailedCoverage", err)
	}
	if missed.Observed != 1 || missed.Valid != 4 {
		t.Fatalf("counted %d of %d, want exactly 1 of 4", missed.Observed, missed.Valid)
	}
	if missed.MinPercent != 100 {
		t.Fatalf("MinPercent = %v, want 100", missed.MinPercent)
	}
}

func TestRejectedExamplesDiscardCoverageObservations(t *testing.T) {
	attempts := 0
	err := Check(func(tc *TestCase) error {
		attempts++
		// The condition holds only on attempts an assumption then rejects, so
		// a shell that commits observations from rejected examples passes.
		tc.Sometimes("rejected attempts only", attempts%2 == 1)
		tc.Assume(attempts%2 == 0)
		return nil
	}, Config{Examples: 10, Seed: 5})
	var missed *FailedCoverage
	if !errors.As(err, &missed) {
		t.Fatalf("run reported %v, want FailedCoverage", err)
	}
	if missed.Observed != 0 {
		t.Fatalf("observed %d examples, want 0 — a rejected example's observations leaked", missed.Observed)
	}
}

func TestCoverageIsNotCheckedWhenARunFindsACounterexample(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		tc.Sometimes("never held", false)
		return errors.New("real bug")
	}, Config{Examples: 10, Seed: 5})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("failing run reported %v, want the counterexample", err)
	}
}

func TestReproduceSkipsCoverageVerdicts(t *testing.T) {
	property := func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 10, 0)
		if drawErr != nil {
			return drawErr
		}
		_ = value
		return errors.New("always fails")
	}
	var failure *Failure
	if err := Check(property, Config{Examples: 5, Seed: 5}); !errors.As(err, &failure) {
		t.Fatalf("seed run reported %v, want a failure to reproduce", err)
	}
	err := Check(property, Config{
		Reproduce: failure.Blob,
		Require:   []Coverage{{Label: "impossible during replay"}},
	})
	if !errors.As(err, &failure) {
		t.Fatalf("replay reported %v, want the reproduced failure rather than FailedCoverage", err)
	}
}

func TestRequiredCoveragePercentIsValidated(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		return nil
	}, Config{
		Examples: 1,
		Require:  []Coverage{{Label: "rate", Percent: 101}},
	})
	if err == nil || !strings.Contains(err.Error(), "percent") {
		t.Fatalf("out-of-range percent reported %v, want a validation error", err)
	}
}

func TestAlwaysFailureShrinksToTheSimplestViolation(t *testing.T) {
	property := func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 100, 0)
		if drawErr != nil {
			return drawErr
		}
		tc.Always("value stays small", value < 5)
		return nil
	}
	err := Check(property, Config{Examples: 100, Seed: 3})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("violated Always reported %v, want a failure", err)
	}
	if !strings.Contains(failure.Cause.Error(), `always: "value stays small" did not hold`) {
		t.Fatalf("failure message %q does not name the assertion", failure.Cause.Error())
	}
	var replayed int64
	err = Check(func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 100, 0)
		if drawErr != nil {
			return drawErr
		}
		replayed = value
		tc.Always("value stays small", value < 5)
		return nil
	}, Config{Reproduce: failure.Blob})
	if !errors.As(err, &failure) {
		t.Fatalf("replay did not fail: %v", err)
	}
	if replayed != 5 {
		t.Fatalf("shrunk violation = %d, want 5", replayed)
	}
}

func TestAlwaysThatHoldsPassesAndCoversItsLabel(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		tc.Always("invariant", true)
		return nil
	}, Config{
		Examples: 10,
		Require:  []Coverage{{Label: "invariant"}},
	})
	if err != nil {
		t.Fatalf("holding Always failed the run: %v", err)
	}
}

func TestRequiredAlwaysSiteThatNeverRunsFailsTheRun(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 10, 0)
		if drawErr != nil {
			return drawErr
		}
		if value > 1000 {
			tc.Always("guarded invariant", true)
		}
		return nil
	}, Config{
		Examples: 20,
		Seed:     5,
		Require:  []Coverage{{Label: "guarded invariant"}},
	})
	var missed *FailedCoverage
	if !errors.As(err, &missed) {
		t.Fatalf("unreached Always site reported %v, want FailedCoverage", err)
	}
	if missed.Label != "guarded invariant" {
		t.Fatalf("verdict names %q, want the required label", missed.Label)
	}
}

func TestUnreachableFailsWithTheSimplestReachingInput(t *testing.T) {
	property := func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 100, 0)
		if drawErr != nil {
			return drawErr
		}
		if value > 5 {
			tc.Unreachable("dead branch")
		}
		return nil
	}
	err := Check(property, Config{Examples: 100, Seed: 3})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("reached Unreachable reported %v, want a failure", err)
	}
	if !strings.Contains(failure.Cause.Error(), `unreachable: "dead branch" was reached`) {
		t.Fatalf("failure message %q does not name the assertion", failure.Cause.Error())
	}
	var replayed int64
	err = Check(func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 100, 0)
		if drawErr != nil {
			return drawErr
		}
		replayed = value
		if value > 5 {
			tc.Unreachable("dead branch")
		}
		return nil
	}, Config{Reproduce: failure.Blob})
	if !errors.As(err, &failure) {
		t.Fatalf("replay did not fail: %v", err)
	}
	if replayed != 6 {
		t.Fatalf("shrunk reaching input = %d, want 6", replayed)
	}
}

func TestUnreachableNeverReachedPasses(t *testing.T) {
	err := Check(func(tc *TestCase) error {
		value, drawErr := tc.Integer(0, 10, 0)
		if drawErr != nil {
			return drawErr
		}
		if value > 1000 {
			tc.Unreachable("dead branch")
		}
		return nil
	}, Config{Examples: 20, Seed: 5})
	if err != nil {
		t.Fatalf("unreached Unreachable failed the run: %v", err)
	}
}

// cloneProperty draws one integer from the stream it is handed, one from a
// clone of it, and one more from the parent — the shape that makes the clone's
// choices unreachable to every flat pass.
func cloneProperty(threshold int64, seen *[3]int64) Property {
	return func(tc *TestCase) error {
		outer, err := tc.Integer(0, 100, 0)
		if err != nil {
			return err
		}
		child, err := tc.CloneStream()
		if err != nil {
			return err
		}
		inner, err := child.Integer(0, 100, 0)
		if err != nil {
			return err
		}
		tail, err := tc.Integer(0, 100, 0)
		if err != nil {
			return err
		}
		*seen = [3]int64{outer, inner, tail}
		if inner >= threshold {
			return fmt.Errorf("inner too big: %d", inner)
		}
		return nil
	}
}

func TestCloneStreamIsFramedAndReplays(t *testing.T) {
	var seen [3]int64
	err := Check(cloneProperty(40, &seen), Config{Examples: 200, Seed: 7})
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("cloned draw reported %v, want a failure", err)
	}
	raw, ok := rawFailureDecode([]byte(failure.Blob))
	if !ok {
		t.Fatal("blob did not decode")
	}
	if !choiceStreamValidate(raw) {
		t.Fatal("a blob carrying a clone did not validate")
	}
	// Three top-level entries: the outer draw, the clone, the trailing draw.
	if count := choiceStreamCount(raw); count != 3 {
		t.Fatalf("top-level choice count = %d, want 3", count)
	}
	offset, found := choiceOffsetAt(raw, 1)
	if !found {
		t.Fatal("no offset for the clone entry")
	}
	if raw[offset] != 5 {
		t.Fatalf("entry 1 has tag %d, want the clone tag 5", raw[offset])
	}
	inner, present := cloneInnerStream(raw, offset)
	if !present {
		t.Fatal("the clone entry carries no stream")
	}
	if count := choiceStreamCount(inner); count != 1 {
		t.Fatalf("clone child count = %d, want 1", count)
	}
	seen = [3]int64{}
	if !errors.As(Check(cloneProperty(40, &seen), Config{Reproduce: failure.Blob}), &failure) {
		t.Fatal("replaying a blob with a clone did not fail")
	}
	if seen[1] < 40 {
		t.Fatalf("replayed inner draw = %d, want the recorded failing value", seen[1])
	}
}

func TestCloneStreamShrinksItsOwnChoices(t *testing.T) {
	// Every seed must reach the same minimum: the clone's draw at exactly the
	// threshold, everything around it at zero. Without a nested pass the flat
	// passes step over the clone as one atomic entry and report whatever value
	// generation found.
	var blob string
	for _, seed := range []int64{1, 2, 3, 7, 11} {
		var seen [3]int64
		err := Check(cloneProperty(40, &seen), Config{Examples: 200, Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			t.Fatalf("seed %d: %v, want a failure", seed, err)
		}
		if seen != [3]int64{0, 40, 0} {
			t.Fatalf("seed %d shrank to %v, want [0 40 0]", seed, seen)
		}
		if blob == "" {
			blob = failure.Blob
		} else if failure.Blob != blob {
			t.Fatalf("seed %d emitted %q, want the same blob as every other seed (%q)",
				seed, failure.Blob, blob)
		}
	}
}

func TestCloneStreamDrawsIndependentlyOfItsParent(t *testing.T) {
	// A clone gets its own stream, so the parent's later draws sit where they
	// did before the clone was made: the trailing draw reads the trailing
	// entry, not whatever the child drew.
	var outer, inner, tail int64
	property := func(tc *TestCase) error {
		var err error
		if outer, err = tc.Integer(0, 1000, 0); err != nil {
			return err
		}
		child, err := tc.CloneStream()
		if err != nil {
			return err
		}
		for i := 0; i < 4; i++ {
			if inner, err = child.Integer(0, 1000, 0); err != nil {
				return err
			}
		}
		if tail, err = tc.Integer(0, 1000, 0); err != nil {
			return err
		}
		return errors.New("always fails")
	}
	var failure *Failure
	if !errors.As(Check(property, Config{Examples: 1, Seed: 5}), &failure) {
		t.Fatal("property did not fail")
	}
	outer, inner, tail = -1, -1, -1
	if !errors.As(Check(property, Config{Reproduce: failure.Blob}), &failure) {
		t.Fatal("replay did not fail")
	}
	if outer != 0 || inner != 0 || tail != 0 {
		t.Fatalf("shrunk to outer=%d inner=%d tail=%d, want all zero", outer, inner, tail)
	}
}

func TestCloneReplayPunsToAnEmptyChild(t *testing.T) {
	// A blob whose clone position holds an ordinary integer — the shape a
	// property that grew a clone since the blob was written produces. Upstream
	// puns it to an empty child, which overruns on its first draw.
	stream := choiceListHeader(2)
	stream = append(stream, serializeIntegerChoice(signedBytes(3))...)
	stream = append(stream, serializeIntegerChoice(signedBytes(4))...)
	blob := rawFailureEncode(stream)

	var childErr error
	err := Check(func(tc *TestCase) error {
		if _, drawErr := tc.Integer(0, 100, 0); drawErr != nil {
			return drawErr
		}
		child, drawErr := tc.CloneStream()
		if drawErr != nil {
			return drawErr
		}
		_, childErr = child.Integer(0, 100, 0)
		return childErr
	}, Config{Reproduce: blob})
	var overrun *ReplayOverrun
	if !errors.As(childErr, &overrun) {
		t.Fatalf("child draw returned %v, want an overrun", childErr)
	}
	if !errors.As(err, &overrun) {
		t.Fatalf("run reported %v, want the child's overrun", err)
	}
}

func TestClonesShareTheFamilyBudgetAndReport(t *testing.T) {
	var childCase *TestCase
	err := Check(func(tc *TestCase) error {
		child, drawErr := tc.CloneStream()
		if drawErr != nil {
			return drawErr
		}
		childCase = child
		child.Note("from the clone")
		child.Cost("faults", 2)
		if _, drawErr = child.Integer(0, 10, 0); drawErr != nil {
			return drawErr
		}
		if len(tc.Notes()) != 1 || tc.Notes()[0] != "from the clone" {
			return fmt.Errorf("parent notes = %v, want the clone's", tc.Notes())
		}
		if tc.Costs()["faults"] != 2 {
			return fmt.Errorf("parent costs = %v, want the clone's", tc.Costs())
		}
		return nil
	}, Config{Examples: 5, Seed: 1})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if childCase.root().flatDraws == 0 {
		t.Fatal("the family counted no draws")
	}
}

func TestNestedClonesShrinkAtEveryDepth(t *testing.T) {
	// A clone inside a clone: splicing a candidate this deep needs one
	// extraction per level, which is where the recursion the IR forbids lives
	// in the shell instead.
	var seen [2]int64
	property := func(tc *TestCase) error {
		child, err := tc.CloneStream()
		if err != nil {
			return err
		}
		outer, err := child.Integer(0, 100, 0)
		if err != nil {
			return err
		}
		grandchild, err := child.CloneStream()
		if err != nil {
			return err
		}
		inner, err := grandchild.Integer(0, 100, 0)
		if err != nil {
			return err
		}
		seen = [2]int64{outer, inner}
		if outer >= 10 && inner >= 20 {
			return fmt.Errorf("both too big: %d %d", outer, inner)
		}
		return nil
	}
	var failure *Failure
	if !errors.As(Check(property, Config{Examples: 300, Seed: 4}), &failure) {
		t.Fatal("nested clones did not fail")
	}
	if seen != [2]int64{10, 20} {
		t.Fatalf("shrank to %v, want [10 20]", seen)
	}
	seen = [2]int64{}
	if !errors.As(Check(property, Config{Reproduce: failure.Blob}), &failure) {
		t.Fatal("replaying nested clones did not fail")
	}
	if seen != [2]int64{10, 20} {
		t.Fatalf("replayed %v, want [10 20]", seen)
	}
}

func TestCloningPastTheNestingLimitRejectsTheExample(t *testing.T) {
	// The wire format refuses to validate a stream nested deeper than 100, so
	// the shell refuses to build one. Upstream concludes the family invalid
	// there, which is a rejection rather than a property failure.
	err := Check(func(tc *TestCase) error {
		at := tc
		for i := 0; i <= maxCloneDepth; i++ {
			child, drawErr := at.CloneStream()
			if drawErr != nil {
				return drawErr
			}
			at = child
		}
		return errors.New("reached past the nesting limit")
	}, Config{Examples: 5, Seed: 1, SuppressFilterTooMuch: true})
	var rejected *TooManyRejections
	if !errors.As(err, &rejected) {
		t.Fatalf("cloning past the limit reported %v, want rejected examples", err)
	}
}

// The example database: storage, the reuse phase, and what it collects.
//
// The shared table in TestDatabaseCases is the cross-language gate; these are
// the Go surface's own — the derived key and the best-effort promise that a
// database which cannot be written never fails a run.

func thresholdProperty(lo, hi, threshold int64) Property {
	return func(tc *TestCase) error {
		value, err := tc.Integer(lo, hi, 0)
		if err != nil {
			return err
		}
		if value >= threshold {
			return fmt.Errorf("value reached %d", threshold)
		}
		return nil
	}
}

func harmlessProperty(lo, hi int64) Property {
	return func(tc *TestCase) error {
		_, err := tc.Integer(lo, hi, 0)
		return err
	}
}

func failureOf(t *testing.T, err error) *Failure {
	t.Helper()
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("want a Failure, got %v", err)
	}
	return failure
}

func TestDatabaseIsOffUntilAskedFor(t *testing.T) {
	// A library call that wrote into the working directory unasked would be a
	// surprise, so the zero Config stores nothing and reads nothing.
	if (Config{}).Database != nil {
		t.Fatal("the zero Config must have no database")
	}
}

func TestDatabaseDigestIsTheCompiledKeyHash(t *testing.T) {
	// The whole cross-language claim rests on this being the same hash a keyed
	// draw uses, not a host one.
	want := fmt.Sprintf("%016x", Key("split_is_exact"))
	if got := DatabaseDigest([]byte("split_is_exact")); got != want {
		t.Fatalf("digest %q, want %q", got, want)
	}
}

func TestDatabaseReplacesALargerStoredCounterexample(t *testing.T) {
	// An entry stored by a weaker shrinker, or by an older build, is replayed,
	// shrunk again, and replaced by what it shrank to.
	database := NewInMemoryDatabase()
	larger, err := hex.DecodeString("01000000000a0100000032")
	if err != nil {
		t.Fatal(err)
	}
	database.Save("case", larger)
	failure := failureOf(t, Check(thresholdProperty(0, 100, 5), Config{
		Examples: 200, Seed: 1, Database: database, DatabaseKey: "case",
	}))
	shrunk, ok := rawFailureDecode([]byte(failure.Blob))
	if !ok {
		t.Fatalf("blob %q did not decode", failure.Blob)
	}
	stored := database.Fetch("case")
	if len(stored) != 1 || !bytes.Equal(stored[0], shrunk) {
		t.Fatalf("stored %x, want just the shrunk %x", stored, shrunk)
	}
	if bytes.Equal(shrunk, larger) {
		t.Fatal("the stored entry was never shrunk")
	}
}

func TestDatabaseIsUntouchedByAReproduceRun(t *testing.T) {
	// Replaying a blob by hand is a manual operation; one that quietly rewrote
	// a shared directory would be a surprise.
	found := failureOf(t, Check(thresholdProperty(0, 100, 5), Config{Examples: 200, Seed: 1}))
	database := NewInMemoryDatabase()
	failureOf(t, Check(thresholdProperty(0, 100, 5), Config{
		Reproduce: found.Blob, Database: database, DatabaseKey: "case",
	}))
	if entries := database.Fetch("case"); len(entries) != 0 {
		t.Fatalf("a reproduce run stored %d entries", len(entries))
	}
}

func TestDatabaseThatCannotBeWrittenDoesNotFailTheRun(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	database := NewDirectoryDatabase(blocked)
	failureOf(t, Check(thresholdProperty(0, 100, 5), Config{
		Examples: 200, Seed: 1, Database: database, DatabaseKey: "case",
	}))
	if entries := database.Fetch("case"); len(entries) != 0 {
		t.Fatalf("an unwritable database reported %d entries", len(entries))
	}
}

func TestDatabaseKeyDirectoryIgnoresWhatItDidNotWrite(t *testing.T) {
	database := NewDirectoryDatabase(t.TempDir())
	database.Save("case", []byte{0, 0, 0, 0})
	keyDir := database.KeyPath("case")
	for _, name := range []string{"README", ".gitignore"} {
		if err := os.WriteFile(filepath.Join(keyDir, name), []byte("checked in on purpose"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries := database.Fetch("case")
	if len(entries) != 1 || !bytes.Equal(entries[0], []byte{0, 0, 0, 0}) {
		t.Fatalf("fetched %x, want the one stored stream", entries)
	}
}

func TestDatabaseSavingTheSameStreamTwiceStoresItOnce(t *testing.T) {
	for _, database := range []Database{NewDirectoryDatabase(t.TempDir()), NewInMemoryDatabase()} {
		database.Save("case", []byte{0, 0, 0, 0})
		database.Save("case", []byte{0, 0, 0, 0})
		if entries := database.Fetch("case"); len(entries) != 1 {
			t.Fatalf("%T stored %d copies of one stream", database, len(entries))
		}
	}
}

func TestDatabaseEmptiedKeyLeavesNoDirectoryBehind(t *testing.T) {
	database := NewDirectoryDatabase(t.TempDir())
	database.Save("case", []byte{0, 0, 0, 0})
	database.Delete("case", []byte{0, 0, 0, 0})
	if _, err := os.Stat(database.KeyPath("case")); !os.IsNotExist(err) {
		t.Fatalf("the key directory outlived its last entry: %v", err)
	}
}

func TestDatabaseDeletingOneOfSeveralKeepsTheDirectoryLabelled(t *testing.T) {
	// The .name file goes with the last entry, not the first: a directory that
	// still holds counterexamples must keep the label that says whose they are.
	database := NewDirectoryDatabase(t.TempDir())
	database.Save("case", []byte{0, 0, 0, 0})
	database.Save("case", []byte{1, 0, 0, 0})
	database.Delete("case", []byte{0, 0, 0, 0})
	name, err := os.ReadFile(filepath.Join(database.KeyPath("case"), ".name"))
	if err != nil {
		t.Fatalf("the label went with the first entry: %v", err)
	}
	if string(name) != "case" {
		t.Fatalf(".name holds %q, want %q", name, "case")
	}
	if entries := database.Fetch("case"); len(entries) != 1 {
		t.Fatalf("fetched %d entries, want the one that was not deleted", len(entries))
	}
}

func TestDatabaseDeletingWhatIsNotThereIsQuiet(t *testing.T) {
	for _, database := range []Database{NewDirectoryDatabase(t.TempDir()), NewInMemoryDatabase()} {
		database.Delete("case", []byte{0, 0, 0, 0})
		if entries := database.Fetch("case"); len(entries) != 0 {
			t.Fatalf("%T invented %d entries", database, len(entries))
		}
	}
}

func TestDatabaseRunWithoutAKeyNamesItselfAfterTheProperty(t *testing.T) {
	database := NewInMemoryDatabase()
	property := thresholdProperty(0, 100, 5)
	failureOf(t, Check(property, Config{Examples: 200, Seed: 1, Database: database}))
	if entries := database.Fetch(derivedDatabaseKey(property)); len(entries) != 1 {
		t.Fatalf("the derived key holds %d entries, want 1", len(entries))
	}
}

// TestDatabaseCases is the cross-language gate for the example database.
//
// See conformance/database/database_cases.tsv, which every database-enabled shell runs: a
// digest or a reuse rule that drifts in one language fails here rather than in
// a shared directory three months later.
func TestDatabaseCases(t *testing.T) {
	path := filepath.Join("..", "..", "conformance", "database", "database_cases.tsv")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Skipf("%s is missing: the shared conformance tables live in the Zizek source tree and are not shipped with this package", path)
	}
	if err != nil {
		t.Fatalf("read database cases: %v", err)
	}
	var cases [][]string
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cases = append(cases, strings.Split(line, "\t"))
	}
	if len(cases) == 0 {
		t.Fatal("no shared database cases were run")
	}
	pinned := 0
	for _, databaseCase := range cases {
		kind, rest := databaseCase[0], databaseCase[1:]
		expectedBlob := ""
		if len(rest) != 0 && strings.HasPrefix(rest[len(rest)-1], "blob=") {
			expectedBlob = strings.TrimPrefix(rest[len(rest)-1], "blob=")
			rest = rest[:len(rest)-1]
			pinned++
		}
		t.Run(strings.Join(databaseCase, " "), func(t *testing.T) {
			emitted, handled := runDeclaredDatabaseCase(t, kind, rest)
			if !handled {
				t.Fatalf("unknown database case %q", kind)
			}
			if expectedBlob != "" && emitted != expectedBlob {
				t.Fatalf("%s emitted %q, want the shared %q", kind, emitted, expectedBlob)
			}
		})
	}
	if pinned == 0 {
		t.Fatal("no shared database case pinned its blob")
	}
}

// A compressed blob may inflate to at most 16 MiB, and a choice stream larger
// than that goes out raw, so every blob the encoder writes still decodes. The
// golden vectors pin the bound at small limits; this is the full-size edge.
func TestFailureBlobInflationIsBounded(t *testing.T) {
	limit := int(maxDecompressedLen())
	if out, ok := zlibDecompress(zlibCompress(make([]byte, limit))); !ok || len(out) != limit {
		t.Fatalf("a stream inflating to exactly the bound decoded to %d bytes, ok=%v", len(out), ok)
	}
	if _, ok := zlibDecompress(zlibCompress(make([]byte, limit+1))); ok {
		t.Fatal("a stream inflating one byte past the bound decoded")
	}

	stream := append(choiceListHeader(1), serializeBytesChoice(make([]byte, limit))...)
	blob := rawFailureEncode(stream)
	if payload, ok := base64Decode([]byte(blob)); !ok || payload[0] != 0 {
		t.Fatalf("an over-bound stream was not emitted raw: %q...", blob[:16])
	}
	decoded, ok := rawFailureDecode([]byte(blob))
	if !ok || !bytes.Equal(decoded, stream) {
		t.Fatal("an over-bound stream's blob did not decode back to it")
	}
}

// Code generated from conformance/{draw_ops,shell_cases}.zir by emitter/emit.go; DO NOT EDIT.

package zizek

import (
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
)

// declaredBugA and declaredBugB are TWO_BUG's two failures. A returned
// error's bug origin is its unwrapped type, so the two bugs are two types
// rather than two errors.New values of one type.
type declaredBugA struct{}

func (declaredBugA) Error() string { return "bug A" }

type declaredBugB struct{}

func (declaredBugB) Error() string { return "bug B" }

// runDeclaredShellCase returns handled=false for TRACE and future undeclared cases.
func runDeclaredShellCase(t *testing.T, kind string, fields []string) (emitted string, handled bool) {
	t.Helper()
	switch kind {
	case "PROBE_EXTEND":
		if len(fields) != 6 {
			t.Fatalf("PROBE_EXTEND has %d fields", len(fields))
		}
		blob := fields[0]
		lo := atoi(t, fields[1])
		hi := atoi(t, fields[2])
		expected := fields[3]
		total := atoi(t, fields[4])
		probe_seed := atoi(t, fields[5])
		var recorded []int64
		if expected != "-" {
			for _, field := range strings.Split(expected, ",") { recorded = append(recorded, atoi(t, field)) }
		}
		stream, ok := rawFailureDecode([]byte(blob))
		if !ok { t.Fatalf("%q is not a readable reproduce blob", blob) }
		bare := newReplayCase(stream)
		overran := false
		for i := int64(0); i < total; i++ {
			if _, err := bare.Integer(lo, hi, 0); err != nil {
				var overrun *ReplayOverrun
				if !errors.As(err, &overrun) { t.Fatalf("bare replay draw %d: want a replay overrun, got %v", i, err) }
				overran = true
				break
			}
		}
		if !overran { t.Fatalf("bare replay of a %d-draw prefix served %d draws", len(recorded), total) }
		probe := newProbeCase(stream, rand.New(rand.NewSource(probe_seed)))
		for i := int64(0); i < total; i++ {
			value, err := probe.Integer(lo, hi, 0)
			if err != nil { t.Fatalf("probe draw %d: %v", i, err) }
			if i < int64(len(recorded)) && value != recorded[i] { t.Fatalf("probe replayed %d at %d, want the pinned %d", value, i, recorded[i]) }
		}
		if int64(len(probe.draws)) != total { t.Fatalf("probe holds %d draws, want %d", len(probe.draws), total) }
		return "", true
	case "SHRINK_DESCENDANT":
		if len(fields) != 4 {
			t.Fatalf("SHRINK_DESCENDANT has %d fields", len(fields))
		}
		hi := atoi(t, fields[0])
		threshold := atoi(t, fields[1])
		seed := atoi(t, fields[2])
		examples := atoi(t, fields[3])
		var chain func(*TestCase) ([]int64, error)
		chain = func(tc *TestCase) ([]int64, error) {
			tc.StartSpan("node")
			defer tc.StopSpan(false)
			value, err := tc.Integer(0, hi, 0)
			if err != nil { return nil, err }
			values := []int64{value}
			more, err := tc.Boolean()
			if err != nil { return nil, err }
			if more {
				rest, err := chain(tc)
				if err != nil { return nil, err }
				values = append(values, rest...)
			}
			return values, nil
		}
		property := func(tc *TestCase) error {
			values, err := chain(tc)
			if err != nil { return err }
			if values[len(values)-1] >= threshold { return errors.New("the deepest node is too large") }
			return nil
		}
		var failure *Failure
		if !errors.As(Check(property, Config{Examples: int(examples), MaxRejections: int(2000), Seed: seed}), &failure) { t.Fatal("no failing chain was found") }
		var seen []int64
		replay := func(tc *TestCase) error { var err error; seen, err = chain(tc); if err != nil { return err }; return errors.New("replay") }
		var replayed *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayed) { t.Fatal("replaying the shrunk blob did not fail") }
		if !slices.Equal(seen, []int64{threshold}) { t.Fatalf("shrank to %v instead of the minimal [%d]", seen, threshold) }
		return failure.Blob, true
	case "SHRINK_TRIVIAL":
		if len(fields) != 3 {
			t.Fatalf("SHRINK_TRIVIAL has %d fields", len(fields))
		}
		hi := atoi(t, fields[0])
		seed := atoi(t, fields[1])
		examples := atoi(t, fields[2])
		draw := func(tc *TestCase) ([4]int64, error) {
			guard, err := tc.Integer(0, hi, 0); if err != nil { return [4]int64{}, err }
			tc.StartSpan("pair"); defer tc.StopSpan(false)
			first, err := tc.Integer(0, hi, 0); if err != nil { return [4]int64{}, err }
			flag, err := tc.Boolean(); if err != nil { return [4]int64{}, err }
			second, err := tc.Integer(0, hi, 0); if err != nil { return [4]int64{}, err }
			flagged := int64(0); if flag { flagged = 1 }
			return [4]int64{guard, first, flagged, second}, nil
		}
		property := func(tc *TestCase) error {
			values, err := draw(tc); if err != nil { return err }
			if values[0] >= 1 && (values[1] == 0) == (values[3] == 0) { return errors.New("guarded pair") }
			return nil
		}
		var failure *Failure
		if !errors.As(Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed}), &failure) { t.Fatal("no failing example found") }
		var seen [][4]int64
		replay := func(tc *TestCase) error { values, err := draw(tc); if err != nil { return err }; seen = append(seen, values); return errors.New("replay") }
		var replayed *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayed) { t.Fatal("replaying the trivial span did not fail") }
		if len(seen) != 1 || seen[0] != [4]int64{1, 0, 0, 0} { t.Fatalf("trivial span shrank to %v", seen) }
		return failure.Blob, true
	case "KEY_SEQUENCE":
		if len(fields) != 5 {
			t.Fatalf("KEY_SEQUENCE has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		seed := atoi(t, fields[2])
		examples := atoi(t, fields[3])
		expected_misses := atoi(t, fields[4])
		sim := func(insert bool, countdown *int) Property {
			return func(tc *TestCase) error {
				var reads, coins []int64
				for step := 0; step < 6; step++ {
					if step%2 == 0 {
						value, err := tc.IntegerKeyed(lo, hi, 0, tc.NextKey("read"))
						if err != nil { return err }
						reads = append(reads, value)
						if insert && step == 0 {
							extra, err := tc.IntegerKeyed(lo, hi, 0, tc.NextKey("read"))
							if err != nil { return err }
							reads = append(reads, extra)
						}
					} else {
						value, err := tc.IntegerKeyed(0, 1, 0, tc.NextKey("coin"))
						if err != nil { return err }
						coins = append(coins, value)
					}
				}
				tc.Note(fmt.Sprintf("%v|%v", reads, coins))
				*countdown++
				if *countdown >= 4 { return errors.New("boom") }
				return nil
			}
		}
		first := 0
		var failure *Failure
		if !errors.As(Check(sim(false, &first), Config{Examples: int(examples), MaxRejections: int(1000), MaxShrinks: int(1), Seed: seed}), &failure) { return "", true }
		recorded := strings.SplitN(failure.Notes[0], "|", 2)
		second := 99
		var replayed *Failure
		if !errors.As(Check(sim(true, &second), Config{Reproduce: failure.Blob}), &replayed) { t.Fatal("replaying the keyed sequence did not fail") }
		got := strings.SplitN(replayed.Notes[0], "|", 2)
		wantReads := recorded[0][:len(recorded[0])-1] + fmt.Sprintf(" %d]", lo)
		if got[0] != wantReads { t.Fatalf("reads resolved to %s, want %s", got[0], wantReads) }
		if got[1] != recorded[1] { t.Fatalf("coins moved: %s, want %s", got[1], recorded[1]) }
		if replayed.Puns != 0 || replayed.KeyMisses != int(expected_misses) { t.Fatalf("puns=%d keyMisses=%d", replayed.Puns, replayed.KeyMisses) }
		return "", true
	case "KEYED_REPLAY":
		if len(fields) != 7 {
			t.Fatalf("KEYED_REPLAY has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		threshold := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		expected := atoi(t, fields[5])
		expected_misses := atoi(t, fields[6])
		firstKey, secondKey := Key("read", 1), Key("read", 2)
		freshKey := Key("retry", 1)
		recorded := func(tc *TestCase) error {
			first, err := tc.IntegerKeyed(lo, hi, 0, firstKey)
			if err != nil { return err }
			second, err := tc.IntegerKeyed(lo, hi, 0, secondKey)
			if err != nil { return err }
			if first >= threshold && second >= threshold { return errors.New("both large") }
			return nil
		}
		var failure *Failure
		if !errors.As(Check(recorded, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed}), &failure) { t.Fatal("the keyed property did not fail") }
		var seen [][2]int64
		evolved := func(tc *TestCase) error {
			second, err := tc.IntegerKeyed(lo, hi, 0, secondKey)
			if err != nil { return err }
			if _, err := tc.IntegerKeyed(lo, hi, 0, freshKey); err != nil { return err }
			first, err := tc.IntegerKeyed(lo, hi, 0, firstKey)
			if err != nil { return err }
			seen = append(seen, [2]int64{first, second})
			return errors.New("replay")
		}
		var replayed *Failure
		if !errors.As(Check(evolved, Config{Reproduce: failure.Blob}), &replayed) { t.Fatal("replaying the keyed blob did not fail") }
		want := [][2]int64{{expected, expected}}
		if !slices.Equal(seen, want) { t.Fatalf("keyed draws resolved to %v, want %v", seen, want) }
		if replayed.KeyMisses != int(expected_misses) { t.Fatalf("reported %d key misses, want %d", replayed.KeyMisses, expected_misses) }
		return failure.Blob, true
	case "KEYED_DEEP_CLONE":
		if len(fields) != 4 {
			t.Fatalf("KEYED_DEEP_CLONE has %d fields", len(fields))
		}
		depth := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		seed := atoi(t, fields[2])
		examples := atoi(t, fields[3])
		deepKey := Key("deep")
		var innermost func(*TestCase, int64) (int64, error)
		innermost = func(tc *TestCase, levels int64) (int64, error) {
			if levels == 0 { return tc.IntegerKeyed(0, hi, 0, deepKey) }
			child, err := tc.CloneStream()
			if err != nil { return 0, err }
			return innermost(child, levels-1)
		}
		property := func(tc *TestCase) error {
			value, err := innermost(tc, depth)
			if err != nil { return err }
			if value >= hi { return errors.New("the deepest keyed draw is too large") }
			return nil
		}
		var failure *Failure
		if !errors.As(Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed}), &failure) { t.Fatal("the deep keyed property did not fail") }
		if _, ok := rawFailureDecode([]byte(failure.Blob)); !ok { t.Fatalf("%q is not a readable reproduce blob", failure.Blob) }
		var seen []int64
		replay := func(tc *TestCase) error { value, err := innermost(tc, depth); if err != nil { return err }; seen = append(seen, value); return errors.New("replay") }
		var replayed *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayed) { t.Fatal("replaying the deep keyed blob did not fail") }
		if !slices.Equal(seen, []int64{hi}) { t.Fatalf("the deep keyed draw replayed as %v, want [%d]", seen, hi) }
		return failure.Blob, true
	case "KEY_DIGEST":
		if len(fields) != 2 {
			t.Fatalf("KEY_DIGEST has %d fields", len(fields))
		}
		name := fields[0]
		expected := fields[1]
		if got := fmt.Sprintf("%016x", Key(name)); got != expected { t.Fatalf("key digest %s, want %s", got, expected) }
		return "", true
	case "COST_FIRST":
		if len(fields) != 6 {
			t.Fatalf("COST_FIRST has %d fields", len(fields))
		}
		sites := atoi(t, fields[0])
		bound := atoi(t, fields[1])
		threshold := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		expected_faults := atoi(t, fields[5])
		property := func(tc *TestCase) error {
			fired, total := int64(0), int64(0)
			for site := int64(0); site < sites; site++ {
				inject, err := tc.BooleanKeyed(tc.NextKey("inject", site))
				if err != nil { return err }
				if inject { fired++; tc.Cost("fault", 1); continue }
				value, err := tc.IntegerKeyed(0, bound, 0, tc.NextKey("payload", site))
				if err != nil { return err }
				total += value
			}
			tc.Note(fmt.Sprintf("%d", fired))
			if fired > 0 && total >= threshold { return errors.New("boom") }
			return nil
		}
		var failure *Failure
		if !errors.As(Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed}), &failure) { t.Fatal("the cost property did not fail") }
		if failure.Notes[0] != fmt.Sprintf("%d", expected_faults) || failure.Costs["fault"] != expected_faults { t.Fatalf("cost report mismatch: notes=%v costs=%v", failure.Notes, failure.Costs) }
		return "", true
	case "TWO_BUG":
		if len(fields) != 6 {
			t.Fatalf("TWO_BUG has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		a_threshold := atoi(t, fields[2])
		b_value := atoi(t, fields[3])
		seed := atoi(t, fields[4])
		examples := atoi(t, fields[5])
		found := ""
		bugA := declaredBugA{}
		bugB := declaredBugB{}
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			if draw0 > a_threshold {
				if found == "" { found = "A" }
				return bugA
			}
			if draw0 == b_value {
				if found == "" { found = "B" }
				return bugB
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		if err == nil { return "", true }
		reported := "B"
		if errors.Is(err, bugA) { reported = "A" }
		if reported != found { t.Fatalf("found bug %s but reported a counterexample for bug %s", found, reported) }
		return "", true
	case "SHRINK_MIN":
		if len(fields) != 6 {
			t.Fatalf("SHRINK_MIN has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		threshold := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		expected := atoi(t, fields[5])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			if draw0 >= threshold {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 []int64
		replay := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || seen0[0] != expected {
			t.Fatalf("draw 0 shrank to %v instead of %d", seen0, expected)
		}
		return failure.Blob, true
	case "SHRINK_SPARSE":
		if len(fields) != 5 {
			t.Fatalf("SHRINK_SPARSE has %d fields", len(fields))
		}
		hi := atoi(t, fields[0])
		period := atoi(t, fields[1])
		dense := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(0, hi, 0)
			if err != nil {
				return err
			}
			if draw0 >= period && (draw0%period == 0 || draw0 >= dense) {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 []int64
		replay := func(tc *TestCase) error {
			draw0, err := tc.Integer(0, hi, 0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || seen0[0] != period {
			t.Fatalf("draw 0 shrank to %v instead of %d", seen0, period)
		}
		return failure.Blob, true
	case "SHRINK_DECLARED_LEN":
		if len(fields) != 4 {
			t.Fatalf("SHRINK_DECLARED_LEN has %d fields", len(fields))
		}
		hi := atoi(t, fields[0])
		threshold := atoi(t, fields[1])
		seed := atoi(t, fields[2])
		examples := atoi(t, fields[3])
		property := func(tc *TestCase) error {
			element0 := Integers().MinValue(0).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 10, true, "list.element")
			if err != nil {
				return err
			}
			draw1, err := tc.Integer(0, 10, 0)
			if err != nil {
				return err
			}
			if draw1 == int64(len(draw0)) && slices.ContainsFunc(draw0, func(value int64) bool { return value >= threshold }) {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 [][]int64
		var seen1 []int64
		replay := func(tc *TestCase) error {
			element0 := Integers().MinValue(0).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 10, true, "list.element")
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			draw1, err := tc.Integer(0, 10, 0)
			if err != nil {
				return err
			}
			seen1 = append(seen1, draw1)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || len(seen0[0]) != 1 || seen0[0][0] != threshold {
			t.Fatalf("draw 0 shrank to %v instead of [[%d]]", seen0, threshold)
		}
		if len(seen1) != 1 || seen1[0] != 1 {
			t.Fatalf("draw 1 shrank to %v instead of %d", seen1, 1)
		}
		return failure.Blob, true
	case "SHRINK_GATE":
		if len(fields) != 4 {
			t.Fatalf("SHRINK_GATE has %d fields", len(fields))
		}
		hi := atoi(t, fields[0])
		threshold := atoi(t, fields[1])
		seed := atoi(t, fields[2])
		examples := atoi(t, fields[3])
		draw := func(tc *TestCase) ([]int64, error) {
			coin, err := tc.Integer(0, hi, 0); if err != nil { return nil, err }
			drawn := []int64{coin}
			if coin < threshold {
				pick, err := tc.Integer(0, 2, 0); if err != nil { return nil, err }
				drawn = append(drawn, pick)
			}
			value, err := tc.Integer(-1000, 1000, 0); if err != nil { return nil, err }
			return append(drawn, value), nil
		}
		property := func(tc *TestCase) error {
			if _, err := draw(tc); err != nil { return err }
			return errors.New("the gate always fails")
		}
		var failure *Failure
		if !errors.As(Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed}), &failure) { t.Fatal("no failing example found") }
		var seen [][]int64
		replay := func(tc *TestCase) error { drawn, err := draw(tc); if err != nil { return err }; seen = append(seen, drawn); return errors.New("replay") }
		var replayed *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayed) { t.Fatal("replaying the raised gate did not fail") }
		if len(seen) != 1 || !slices.Equal(seen[0], []int64{threshold, 0}) { t.Fatalf("gate shrank to %v instead of [%d 0]", seen, threshold) }
		return failure.Blob, true
	case "SHRINK_LIST":
		if len(fields) != 6 {
			t.Fatalf("SHRINK_LIST has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		threshold := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		expected := atoi(t, fields[5])
		property := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 0, false, "list.element")
			if err != nil {
				return err
			}
			for _, value := range draw0 {
				if value >= threshold {
					return errors.New("declared failure")
				}
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 [][]int64
		replay := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 0, false, "list.element")
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || len(seen0[0]) != 1 || seen0[0][0] != expected {
			t.Fatalf("draw 0 shrank to %v instead of [[%d]]", seen0, expected)
		}
		return failure.Blob, true
	case "SHRINK_SET":
		if len(fields) != 6 {
			t.Fatalf("SHRINK_SET has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		threshold := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		expected := atoi(t, fields[5])
		property := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := CollectUnique(tc, element0, func(item int64) int64 { return item }, int(0), 0, false, "set.element")
			if err != nil {
				return err
			}
			for _, value := range draw0 {
				if value >= threshold {
					return errors.New("declared failure")
				}
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 [][]int64
		replay := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := CollectUnique(tc, element0, func(item int64) int64 { return item }, int(0), 0, false, "set.element")
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || len(seen0[0]) != 1 || seen0[0][0] != expected {
			t.Fatalf("draw 0 shrank to %v instead of [[%d]]", seen0, expected)
		}
		return failure.Blob, true
	case "SHRINK_CLONE":
		if len(fields) != 4 {
			t.Fatalf("SHRINK_CLONE has %d fields", len(fields))
		}
		hi := atoi(t, fields[0])
		threshold := atoi(t, fields[1])
		seed := atoi(t, fields[2])
		examples := atoi(t, fields[3])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(0, hi, 0)
			if err != nil {
				return err
			}
			child1, err := tc.CloneStream()
			if err != nil {
				return err
			}
			draw1, err := child1.Integer(0, hi, 0)
			if err != nil {
				return err
			}
			draw2, err := tc.Integer(0, hi, 0)
			if err != nil {
				return err
			}
			_ = draw0
			_ = draw2
			if draw1 >= threshold {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 []int64
		var seen1 []int64
		var seen2 []int64
		replay := func(tc *TestCase) error {
			draw0, err := tc.Integer(0, hi, 0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			child1, err := tc.CloneStream()
			if err != nil {
				return err
			}
			draw1, err := child1.Integer(0, hi, 0)
			if err != nil {
				return err
			}
			seen1 = append(seen1, draw1)
			draw2, err := tc.Integer(0, hi, 0)
			if err != nil {
				return err
			}
			seen2 = append(seen2, draw2)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || seen0[0] != 0 {
			t.Fatalf("draw 0 shrank to %v instead of %d", seen0, 0)
		}
		if len(seen1) != 1 || seen1[0] != threshold {
			t.Fatalf("draw 1 shrank to %v instead of %d", seen1, threshold)
		}
		if len(seen2) != 1 || seen2[0] != 0 {
			t.Fatalf("draw 2 shrank to %v instead of %d", seen2, 0)
		}
		return failure.Blob, true
	case "SHRINK_DUPLICATE":
		if len(fields) != 5 {
			t.Fatalf("SHRINK_DUPLICATE has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		seed := atoi(t, fields[2])
		examples := atoi(t, fields[3])
		expected := atoi(t, fields[4])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			draw1, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			if draw0 == draw1 && draw0 != 0 {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 []int64
		var seen1 []int64
		replay := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			draw1, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			seen1 = append(seen1, draw1)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || seen0[0] != expected {
			t.Fatalf("draw 0 shrank to %v instead of %d", seen0, expected)
		}
		if len(seen1) != 1 || seen1[0] != expected {
			t.Fatalf("draw 1 shrank to %v instead of %d", seen1, expected)
		}
		return failure.Blob, true
	case "SHRINK_PAIR":
		if len(fields) != 7 {
			t.Fatalf("SHRINK_PAIR has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		gap := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		expected_first := atoi(t, fields[5])
		expected_second := atoi(t, fields[6])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			draw1, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			if draw0-draw1 == gap {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 []int64
		var seen1 []int64
		replay := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			draw1, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			seen1 = append(seen1, draw1)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || seen0[0] != expected_first {
			t.Fatalf("draw 0 shrank to %v instead of %d", seen0, expected_first)
		}
		if len(seen1) != 1 || seen1[0] != expected_second {
			t.Fatalf("draw 1 shrank to %v instead of %d", seen1, expected_second)
		}
		return failure.Blob, true
	case "ALWAYS_SHRINK":
		if len(fields) != 6 {
			t.Fatalf("ALWAYS_SHRINK has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		threshold := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		expected := atoi(t, fields[5])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			tc.Always("stays small", draw0 < threshold)
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 []int64
		replay := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || seen0[0] != expected {
			t.Fatalf("draw 0 shrank to %v instead of %d", seen0, expected)
		}
		return failure.Blob, true
	case "SHRINK_REORDER":
		if len(fields) != 6 {
			t.Fatalf("SHRINK_REORDER has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		first := atoi(t, fields[2])
		second := atoi(t, fields[3])
		seed := atoi(t, fields[4])
		examples := atoi(t, fields[5])
		property := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 0, false, "list.element")
			if err != nil {
				return err
			}
			members := make(map[int64]struct{})
			for _, value := range draw0 { members[value] = struct{}{} }
			_, hasFirst := members[first]
			_, hasSecond := members[second]
			if len(members) == 2 && hasFirst && hasSecond {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 [][]int64
		replay := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 0, false, "list.element")
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || !slices.Equal(seen0[0], []int64{first, second}) {
			t.Fatalf("draw 0 shrank to %v instead of the declared list", seen0)
		}
		return failure.Blob, true
	case "SHRINK_TEXT":
		if len(fields) != 7 {
			t.Fatalf("SHRINK_TEXT has %d fields", len(fields))
		}
		alphabet := fields[0]
		min_size := atoi(t, fields[1])
		max_size := atoi(t, fields[2])
		threshold := atoi(t, fields[3])
		seed := atoi(t, fields[4])
		examples := atoi(t, fields[5])
		expected := fields[6]
		generator0 := Text().MinSize(int(min_size)).MaxSize(int(max_size))
		if alphabet != "-" {
			characters, err := Alphabet(alphabet)
			if err != nil { t.Fatalf("bad alphabet %q: %v", alphabet, err) }
			generator0 = generator0.Alphabet(characters)
		}
		property := func(tc *TestCase) error {
			draw0, err := Draw(tc, generator0)
			if err != nil {
				return err
			}
			if int64(len([]rune(draw0))) >= threshold {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 []string
		replay := func(tc *TestCase) error {
			draw0, err := Draw(tc, generator0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || seen0[0] != expected {
			t.Fatalf("draw 0 shrank to %q instead of %q", seen0, expected)
		}
		return failure.Blob, true
	case "SHRINK_CHARS":
		if len(fields) != 5 {
			t.Fatalf("SHRINK_CHARS has %d fields", len(fields))
		}
		alphabet := fields[0]
		simplest := fields[1]
		expected := fields[2]
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		generator0 := Text().MinSize(int(2)).MaxSize(int(5))
		if alphabet != "-" {
			characters, err := Alphabet(alphabet)
			if err != nil { t.Fatalf("bad alphabet %q: %v", alphabet, err) }
			generator0 = generator0.Alphabet(characters)
		}
		property := func(tc *TestCase) error {
			draw0, err := Draw(tc, generator0)
			if err != nil {
				return err
			}
			counts := make(map[rune]int)
			for _, character := range draw0 { counts[character]++ }
			for character, count := range counts {
				if count > 1 && string(character) != simplest { return errors.New("declared failure") }
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(2000), Seed: seed})
		var failure *Failure
		if !errors.As(err, &failure) {
			return "", true
		}
		var seen0 []string
		replay := func(tc *TestCase) error {
			draw0, err := Draw(tc, generator0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			return errors.New("replay")
		}
		var replayFailure *Failure
		if !errors.As(Check(replay, Config{Reproduce: failure.Blob}), &replayFailure) {
			t.Fatal("replaying the shrunk blob did not fail")
		}
		if len(seen0) != 1 || seen0[0] != expected {
			t.Fatalf("draw 0 shrank to %q instead of %q", seen0, expected)
		}
		return failure.Blob, true
	case "UNIQUE_EXHAUSTED":
		if len(fields) != 5 {
			t.Fatalf("UNIQUE_EXHAUSTED has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		min_size := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		property := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := CollectUnique(tc, element0, func(item int64) int64 { return item }, int(min_size), 0, false, "set.element")
			if err != nil {
				return err
			}
			fails := true
			for _, value := range draw0 {
				if !(value >= lo) { fails = false }
			}
			if fails { return errors.New("declared failure") }
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var failure *Failure
		if errors.As(err, &failure) { t.Fatalf("declared rejecting case reported a counterexample: %v", err) }
		var rejections *TooManyRejections
		var health *FailedHealthCheck
		if !errors.As(err, &rejections) && !errors.As(err, &health) { t.Fatalf("expected a rejection outcome, got %v", err) }
		return "", true
	case "REPRODUCE_FAILS":
		if len(fields) != 4 {
			t.Fatalf("REPRODUCE_FAILS has %d fields", len(fields))
		}
		blob := fields[0]
		lo := atoi(t, fields[1])
		hi := atoi(t, fields[2])
		expected := atoi(t, fields[3])
		var seen0 []int64
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			seen0 = append(seen0, draw0)
			if draw0 <= hi {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Reproduce: blob})
		var reproduced *Failure
		if !errors.As(err, &reproduced) { t.Fatalf("portable failure blob did not reproduce a property failure: %v", err) }
		if len(seen0) != 1 || seen0[0] != expected { t.Fatalf("portable failure replayed %v, want [%d]", seen0, expected) }
		return "", true
	case "REPRODUCE_PASSES":
		if len(fields) != 3 {
			t.Fatalf("REPRODUCE_PASSES has %d fields", len(fields))
		}
		blob := fields[0]
		lo := atoi(t, fields[1])
		hi := atoi(t, fields[2])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			if draw0 >= hi {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Reproduce: blob})
		var replayErr *InvalidReplay
		if !errors.As(err, &replayErr) { t.Fatalf("reproducing a passing blob gave %v, want an invalid replay", err) }
		return "", true
	case "PUN_REPORT":
		if len(fields) != 4 {
			t.Fatalf("PUN_REPORT has %d fields", len(fields))
		}
		blob := fields[0]
		lo := atoi(t, fields[1])
		hi := atoi(t, fields[2])
		expected_puns := atoi(t, fields[3])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			if draw0 <= hi {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Reproduce: blob})
		var reproduced *Failure
		if !errors.As(err, &reproduced) { t.Fatalf("portable failure blob did not reproduce a property failure: %v", err) }
		if reproduced.Puns != int(expected_puns) { t.Fatalf("reported %d puns, want %d", reproduced.Puns, expected_puns) }
		if strings.Contains(reproduced.Error(), "warning:") != (expected_puns != 0) { t.Fatalf("pun warning disagrees with the count: %s", reproduced.Error()) }
		return "", true
	case "SOMETIMES_MISSED":
		if len(fields) != 5 {
			t.Fatalf("SOMETIMES_MISSED has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		threshold := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(lo, hi, 0)
			if err != nil {
				return err
			}
			tc.Sometimes("above threshold", draw0 > threshold)
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var missed *FailedCoverage
		if !errors.As(err, &missed) { t.Fatalf("vacuous run reported %v, want the coverage-failure outcome", err) }
		if missed.Label != "above threshold" || missed.Observed != int(0) || missed.Valid != int(examples) { t.Fatalf("unexpected coverage verdict: %+v", missed) }
		return "", true
	case "REQUIRE_MISSED":
		if len(fields) != 1 {
			t.Fatalf("REQUIRE_MISSED has %d fields", len(fields))
		}
		examples := atoi(t, fields[0])
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(0, 0, 0)
			if err != nil {
				return err
			}
			if draw0 >= 1 {
				return errors.New("declared failure")
			}
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: 1, Require: []Coverage{{Label: "unreached"}}})
		var missed *FailedCoverage
		if !errors.As(err, &missed) { t.Fatalf("vacuous run reported %v, want the coverage-failure outcome", err) }
		if missed.Label != "unreached" || missed.Observed != int(0) || missed.Valid != int(examples) { t.Fatalf("unexpected coverage verdict: %+v", missed) }
		return "", true
	case "COVER_RATE":
		if len(fields) != 2 {
			t.Fatalf("COVER_RATE has %d fields", len(fields))
		}
		examples := atoi(t, fields[0])
		percent := atoi(t, fields[1])
		attempts := 0
		property := func(tc *TestCase) error {
			draw0, err := tc.Integer(0, 0, 0)
			if err != nil {
				return err
			}
			_ = draw0
			attempts++
			held := attempts == 1
			tc.Cover("first example", held, float64(percent))
			tc.Cover("first example", held, float64(percent))
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: 1})
		var missed *FailedCoverage
		if !errors.As(err, &missed) { t.Fatalf("vacuous run reported %v, want the coverage-failure outcome", err) }
		if missed.Label != "first example" || missed.Observed != int(1) || missed.Valid != int(examples) { t.Fatalf("unexpected coverage verdict: %+v", missed) }
		return "", true
	case "COVER_STEERS":
		if len(fields) != 8 {
			t.Fatalf("COVER_STEERS has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		floor := atoi(t, fields[2])
		copies := atoi(t, fields[3])
		percent := atoi(t, fields[4])
		seed := atoi(t, fields[5])
		examples := atoi(t, fields[6])
		min_observed := atoi(t, fields[7])
		property := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 0, false, "list.element")
			if err != nil {
				return err
			}
			held := false
			for _, value := range draw0 {
				if value < floor {
					continue
				}
				seen := 0
				for _, other := range draw0 {
					if other == value {
						seen++
					}
				}
				if int64(seen) >= copies {
					held = true
				}
			}
			tc.Cover("repeat", held, float64(percent))
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var missed *FailedCoverage
		if !errors.As(err, &missed) { t.Fatalf("covered run reported %v, want the coverage-failure outcome", err) }
		if missed.Label != "repeat" || missed.Observed < int(min_observed) || missed.Valid != int(examples) { t.Fatalf("unexpected coverage verdict: %+v", missed) }
		return "", true
	case "RARE_STEERS":
		if len(fields) != 7 {
			t.Fatalf("RARE_STEERS has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		floor := atoi(t, fields[2])
		copies := atoi(t, fields[3])
		percent := atoi(t, fields[4])
		seed := atoi(t, fields[5])
		examples := atoi(t, fields[6])
		property := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 0, false, "list.element")
			if err != nil {
				return err
			}
			held := false
			for _, value := range draw0 {
				if value < floor {
					continue
				}
				seen := 0
				for _, other := range draw0 {
					if other == value {
						seen++
					}
				}
				if int64(seen) >= copies {
					held = true
				}
			}
			tc.Cover("repeat", held, float64(percent))
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		if err != nil { t.Fatalf("declared passing run reported %v", err) }
		return "", true
	case "FLAKY_FAILURE":
		if len(fields) != 4 {
			t.Fatalf("FLAKY_FAILURE has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		seed := atoi(t, fields[2])
		examples := atoi(t, fields[3])
		executions := 0
		property := func(tc *TestCase) error {
			if _, err := tc.Integer(lo, hi, 0); err != nil { return err }
			executions++
			if executions == 1 { return errors.New("failed on the first execution") }
			return nil
		}
		var flaky *Flaky
		if err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed}); !errors.As(err, &flaky) { t.Fatalf("a property that failed once reported %v, want Flaky", err) }
		if flaky.Cause == nil || flaky.Cause.Error() != "failed on the first execution" { t.Fatalf("Flaky carries %v, want the original failure", flaky.Cause) }
		if message := flaky.Error(); !strings.Contains(message, "flaky") || !strings.Contains(message, "failed on the first execution") { t.Fatalf("Flaky message %q does not name the original failure", message) }
		return flaky.Blob, true
	case "STEER_KEEPS_RANDOM":
		if len(fields) != 6 {
			t.Fatalf("STEER_KEEPS_RANDOM has %d fields", len(fields))
		}
		lo := atoi(t, fields[0])
		hi := atoi(t, fields[1])
		percent := atoi(t, fields[2])
		seed := atoi(t, fields[3])
		examples := atoi(t, fields[4])
		min_observed := atoi(t, fields[5])
		property := func(tc *TestCase) error {
			element0 := Integers().MinValue(lo).MaxValue(hi)
			draw0, err := Collect(tc, element0, int(0), 0, false, "list.element")
			if err != nil {
				return err
			}
			if len(draw0) > 0 {
				tc.Cover(fmt.Sprintf("bucket %d", draw0[0]%10), true, 0)
			}
			tc.Cover("empty", len(draw0) == 0, float64(percent))
			return nil
		}
		err := Check(property, Config{Examples: int(examples), MaxRejections: int(1000), Seed: seed})
		var missed *FailedCoverage
		if !errors.As(err, &missed) { t.Fatalf("covered run reported %v, want the coverage-failure outcome", err) }
		if missed.Label != "empty" || missed.Observed < int(min_observed) || missed.Valid != int(examples) { t.Fatalf("unexpected coverage verdict: %+v", missed) }
		return "", true
	default:
		return "", false
	}
}

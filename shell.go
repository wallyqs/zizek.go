package zizek

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Property is a single generated test case. Returning a non-nil error marks it
// as failing; panics are also captured as failures.
type Property func(*TestCase) error

var errAssumptionRejected = errors.New("zizek: assumption rejected")

// InvalidReplay means a blob is malformed or does not match the property's
// sequence and types of draws.
type InvalidReplay struct{ Message string }

func (e *InvalidReplay) Error() string { return e.Message }

// ReplayOverrun means the property requested a draw after its replay prefix
// was exhausted.
type ReplayOverrun struct{ Message string }

func (e *ReplayOverrun) Error() string { return e.Message }

type choiceOverrun struct{}

func (e *choiceOverrun) Error() string { return "generated example exceeded the choice budget" }

// Failure is the first failing example after shrinking.
type Failure struct {
	Blob        string
	Cause       error
	ExamplesRun int
	Rejections  int
	Notes       []string
	// Puns counts replayed draws that could not use their recorded value. See
	// ir/replay.zir: a pun is legal but silent, so it is counted rather than
	// returned as an error.
	Puns int
	// KeyMisses counts keyed draws whose name the blob did not carry.
	KeyMisses int
	// Costs is what the reported example spent, per label.
	Costs map[string]int64
}

func (f *Failure) Error() string {
	message := fmt.Sprintf("%v\nreproduce with zizek.Check(property, zizek.Config{Reproduce: %q})", f.Cause, f.Blob)
	if f.KeyMisses != 0 {
		noun := "draws were"
		if f.KeyMisses == 1 {
			noun = "draw was"
		}
		message += fmt.Sprintf(
			"\nwarning: %d keyed %s not in this blob and used the simplest value instead",
			f.KeyMisses,
			noun,
		)
	}
	if f.Puns != 0 {
		// The values reported above are not the ones the blob recorded. Say
		// so: a punned replay is a different run wearing the same blob.
		noun := "choices"
		if f.Puns == 1 {
			noun = "choice"
		}
		message += fmt.Sprintf(
			"\nwarning: %d replayed %s did not fit this property and fell back to"+
				" a default; the example above is not the one the blob recorded",
			f.Puns,
			noun,
		)
	}
	if len(f.Notes) != 0 {
		message += "\n\nTrace:"
		for _, note := range f.Notes {
			message += "\n- " + note
		}
	}
	return message
}

func (f *Failure) Unwrap() error { return f.Cause }

// Flaky means a failing example did not fail the same way when it was replayed:
// the property depends on something other than the choices it drew. There is
// no shrunk example to trust, so it carries the failure as first found and the
// blob of the stream that produced it. Unlike a Failure's, that blob need not
// fail when reproduced.
type Flaky struct {
	Blob        string
	Cause       error
	ExamplesRun int
	Rejections  int
}

func (f *Flaky) Error() string {
	return fmt.Sprintf(
		"flaky test: the property failed, then did not fail the same way when that example was replayed\n"+
			"original failure: %v\nreproduce with zizek.Check(property, zizek.Config{Reproduce: %q})",
		f.Cause, f.Blob,
	)
}

func (f *Flaky) Unwrap() error { return f.Cause }

// errFlakyReplay is internal control flow: the shrinker replayed an example
// that had failed and it did not fail the same way.
var errFlakyReplay = errors.New("zizek: failing example did not fail when replayed")

// Config controls a property run.
type Config struct {
	Examples int
	// MaxRejections is the initial rejection allowance; each valid example
	// earns 100 more. Zero selects the default of 458.
	MaxRejections int
	// SuppressFilterTooMuch disables the early 50-rejection health check.
	SuppressFilterTooMuch bool
	// SuppressDataTooLarge disables the early choice-budget-overrun health check.
	SuppressDataTooLarge bool
	// SuppressLargeBaseExample disables the all-simplest probe size check.
	SuppressLargeBaseExample bool
	// SuppressTooSlow disables the cumulative generation-time health check.
	SuppressTooSlow bool
	// MaxShrinks and MaxStall bound shrinking. Zero selects the defaults
	// (500 and 200); a negative value disables that cap.
	MaxShrinks int
	MaxStall   int
	Seed       int64
	Reproduce  string
	// Require declares labels the completed run must have covered, whether or
	// not any call site for them ever executed. See Cover.
	Require []Coverage
	// Database stores counterexamples between runs, replaying them before
	// generating anything. Nil stores nothing and reads nothing: a library
	// call that wrote into the working directory unasked would be a surprise.
	// See NewDirectoryDatabase.
	Database Database
	// DatabaseKey is the name a stored counterexample is filed under. A name
	// chosen here is portable, so another language's suite replays the same
	// entry; left empty, one is derived from the property's own function name
	// and stays local to Go.
	DatabaseKey string
}

// Coverage declares a label a completed run must have observed. A zero
// Percent requires the label's condition to have held in at least one valid
// example; a positive Percent requires at least that share of them.
type Coverage struct {
	Label   string
	Percent float64
}

// TooManyRejections means assumptions rejected too many generated examples
// before Check could run the requested number of valid examples.
type TooManyRejections struct {
	Rejections int
	Overruns   int
	Limit      int
}

func (e *TooManyRejections) Error() string {
	if e.Overruns != 0 {
		return fmt.Sprintf(
			"too many invalid generated examples: got %d rejections and %d overruns (limit %d)",
			e.Rejections, e.Overruns, e.Limit,
		)
	}
	return fmt.Sprintf(
		"too many rejected examples: got %d rejections (limit %d)",
		e.Rejections,
		e.Limit,
	)
}

// FailedHealthCheck reports a generation pattern that weakens testing.
type FailedHealthCheck struct {
	Check      string
	Rejections int
	Overruns   int
	Valid      int
}

func (e *FailedHealthCheck) Error() string {
	if e.Check == "data_too_large" {
		return fmt.Sprintf(
			"%s: %d inputs exceeded the choice budget while only %d valid inputs were generated",
			e.Check, e.Overruns, e.Valid,
		)
	}
	if e.Check == "large_base_example" {
		return "large_base_example: the simplest natural input is too large"
	}
	if e.Check == "too_slow" {
		return fmt.Sprintf(
			"too_slow: input generation exceeded one cumulative second while only %d valid inputs were generated",
			e.Valid,
		)
	}
	return fmt.Sprintf(
		"%s: %d inputs were rejected by assumptions while only %d valid inputs were generated",
		e.Check, e.Rejections, e.Valid,
	)
}

// FailedCoverage reports a coverage requirement the completed run missed.
//
// There is no counterexample and so no blob or shrinking: the failure is the
// absence of an example, not an example. The counters are a pure function of
// the run's choice streams, so rerunning with the same seed reproduces the
// identical verdict.
type FailedCoverage struct {
	Label    string
	Observed int
	Valid    int
	// MinPercent is the strongest rate required for the label across its call
	// sites and Config.Require entries; zero means "at least once".
	MinPercent float64
}

func (e *FailedCoverage) Error() string {
	if e.MinPercent == 0 {
		return fmt.Sprintf(
			"coverage: %q never held in %d valid examples",
			e.Label, e.Valid,
		)
	}
	return fmt.Sprintf(
		"coverage: %q held in %d of %d valid examples, below the required %v%%",
		e.Label, e.Observed, e.Valid, e.MinPercent,
	)
}

// coverageCounter accumulates one label's run-level coverage. Declared
// requirements are registered before the first example and observed-only
// labels in first-observation order, which is also the order verdicts are
// checked in — so shells that agree on the counters agree on the label a
// failure names.
type coverageCounter struct {
	label    string
	observed int
	permille uint64
}

type draw struct {
	kind    string
	encoded []byte
	forced  bool
	integer int64
	boolean bool
	bytes   []byte
	params  []int64
	float   float64
	floats  [3]float64
	flags   [2]bool
	// alphabet is the interval set a string draw was made over; bytes holds
	// that draw's packed u32-LE codepoints.
	alphabet []byte
	// key is the hash of this draw's name; keyed says whether it has one.
	// encoded stays the inner entry either way, so indexing, shrinking, and
	// replay resolution never have to know about keys.
	key   uint64
	keyed bool
	// child is the stream a clone draw carries; nil for every other kind. Its
	// entry is built at serialization time rather than here, because the child
	// keeps drawing after the clone is recorded.
	child *TestCase
}

// climbableDraw answers whether the targeting hill climber can step a draw.
// Strings have no sensible "one larger", clones are streams rather than
// values, and a forced draw is not this stream's to change.
func climbableDraw(d draw) bool {
	if d.forced || d.child != nil {
		return false
	}
	switch d.kind {
	case "integer", "boolean", "float", "bytes":
		return true
	}
	return false
}

// stepDraw answers the draw stepped by delta, or false when the stepped value
// is not representable within the draw's own constraints. Which values are
// reachable is compiled (ir/targeting.zir targeting_step_*); this only
// re-encodes the answer as a stream entry.
func stepDraw(d draw, delta int64) (draw, bool) {
	switch d.kind {
	case "integer":
		stepped, ok := targetingStepInteger(
			signedBytes(d.params[0]), signedBytes(d.params[1]),
			signedBytes(d.integer), delta,
		)
		if !ok {
			return draw{}, false
		}
		value, fits := signedInt(stepped)
		if !fits {
			return draw{}, false
		}
		d.integer = value
		d.encoded = serializeIntegerChoice(stepped)
	case "boolean":
		stepped, ok := targetingStepBoolean(d.boolean, delta)
		if !ok {
			return draw{}, false
		}
		d.boolean = stepped
		d.encoded = serializeBooleanChoice(stepped)
	case "float":
		stepped, ok := targetingStepFloat(
			d.floats[0], d.floats[1], d.flags[0], d.flags[1], d.floats[2],
			d.float, delta,
		)
		if !ok {
			return draw{}, false
		}
		d.float = stepped
		d.encoded = serializeFloatChoice(stepped)
	case "bytes":
		stepped, ok := targetingStepBytes(d.params[0], d.params[1], d.bytes, delta)
		if !ok {
			return draw{}, false
		}
		d.bytes = stepped
		d.encoded = serializeBytesChoice(stepped)
	default:
		return draw{}, false
	}
	d.forced = false
	return d, true
}

// entry is the stream entry for a draw: its value, wrapped if it has a name.
func (d draw) entry() []byte {
	encoded := d.encoded
	if d.kind == "clone" {
		var children []draw
		if d.child != nil {
			children = d.child.draws
		}
		encoded = serializeCloneHeader(uint64(len(children)))
		for _, item := range children {
			encoded = append(encoded, item.entry()...)
		}
	}
	if !d.keyed {
		return encoded
	}
	return serializeKeyedChoice(d.key, encoded)
}

// flatDrawCount counts a draw list the way Hegel's flattened length does: a
// clone contributes itself plus everything inside it.
func flatDrawCount(draws []draw) uint64 {
	total := uint64(0)
	for _, item := range draws {
		total++
		if item.child != nil {
			total += flatDrawCount(item.child.draws)
		}
	}
	return total
}

// Key hashes a draw's name into the 64-bit key a stream records.
//
// Components may be strings, byte slices, or integers, in any mix — a
// simulator name like Key("read", conn, seq) is the shape this is for. The
// encoding is compiled (ir/keys.zir), so the same name hashes identically in
// every language.
func Key(components ...any) uint64 {
	return keyHash(keyEncode(components))
}

func keyEncode(components []any) []byte {
	var encoded []byte
	for _, component := range components {
		switch value := component.(type) {
		case string:
			encoded = keyAppendText(encoded, []byte(value))
		case []byte:
			encoded = keyAppendText(encoded, value)
		case int:
			encoded = keyAppendInt(encoded, signedBytes(int64(value)))
		case int64:
			encoded = keyAppendInt(encoded, signedBytes(value))
		case uint64:
			encoded = keyAppendInt(encoded, signedBytes(int64(value)))
		default:
			panic("zizek: key components must be string, []byte, or an integer")
		}
	}
	return encoded
}

// ChoiceSpan describes a labelled half-open range of recorded choices.
type ChoiceSpan struct {
	Start     int
	End       int
	Label     string
	Depth     int
	Parent    int
	Discarded bool
}

// TestCase supplies typed draws to a property.
type TestCase struct {
	// parent is the stream this one was cloned from; nil for the case the
	// property was handed. Everything a run reports — notes, costs, coverage,
	// puns, the choice budget — belongs to the family rather than to one
	// stream, so those route through root.
	parent    *TestCase
	rng       *rand.Rand
	replay    []byte
	simplest  bool
	position  int64
	remaining uint64
	draws     []draw
	// flatDraws counts every draw in the family, including those inside clones.
	// Maintained on the root so the choice budget stays O(1) per draw.
	flatDraws        uint64
	integerSwarm     uint64
	integerSwarmSet  bool
	floatSwarm       uint64
	floatSwarmSet    bool
	spans            []ChoiceSpan
	spanStack        []int
	notes            []string
	generationDepth  int
	generationMicros uint64
	puns             int
	// Identifiers handed out by FreshID, family-wide so an identifier is
	// unique across every stream of the case. Only ever grows: that is what
	// keeps the recorded windows monotone within a run, so consuming a value
	// cannot destabilize the identifiers recorded after it.
	freshIDs map[int64]bool
	// Keyed draws whose name the stream did not carry. Like a pun this is a
	// silent substitution — the draw falls back to its simplest value — so it
	// is counted rather than returned as an error.
	keyMisses int
	keyed     map[uint64][]int64
	// Per-name ordinals for NextKey. Execution-local by construction: a
	// counter that outlives one example produces different keys on the replay
	// than on the run that recorded it, and every draw misses.
	keyCounts map[string]int64
	// What this example spent, per label. The shrinker minimizes the total
	// ahead of the choice stream.
	costs map[string]int64
	// Targeting observations for this example, per label. The run keeps the
	// best-scoring example per label and hill-climbs it on a schedule.
	targets map[string]float64
	// Coverage observations buffered for this example. Check commits them into
	// its run-level counters only when the example turns out valid, so a
	// rejected or overrun example contributes nothing — the same convention
	// discarded draws follow.
	coverageOrder    []string
	coverageHeld     map[string]bool
	coveragePermille map[string]uint64
}

// root is the case the property was handed: the family every clone belongs to.
func (tc *TestCase) root() *TestCase {
	for tc.parent != nil {
		tc = tc.parent
	}
	return tc
}

// depth is how many clones deep this stream sits.
func (tc *TestCase) depth() int {
	levels := 0
	for tc.parent != nil {
		levels++
		tc = tc.parent
	}
	return levels
}

// record appends a draw to this stream and counts it against the family's
// choice budget.
func (tc *TestCase) record(item draw) {
	tc.draws = append(tc.draws, item)
	tc.root().flatDraws++
}

// Cost records that this example spent something scarce.
//
// The shrinker orders candidates by total cost first and by the choice stream
// second, so a candidate injecting one fewer fault wins even when its stream is
// longer. A property that never calls this is ordered exactly as before.
func (tc *TestCase) Cost(label string, amount int64) {
	if amount < 0 {
		panic("zizek: cost amount must be >= 0")
	}
	family := tc.root()
	if family.costs == nil {
		family.costs = map[string]int64{}
	}
	family.costs[label] += amount
}

// Costs returns a snapshot of what this example spent.
func (tc *TestCase) Costs() map[string]int64 {
	out := map[string]int64{}
	for label, amount := range tc.root().costs {
		out[label] = amount
	}
	return out
}

// Target records a targeting observation under the empty label: generation is
// steered towards examples that maximize the score. The run keeps, per label,
// the best-scoring valid example, and hill-climbs those examples on a
// schedule between ordinary generation. The score must be finite.
func (tc *TestCase) Target(score float64) {
	tc.TargetLabelled(score, "")
}

// TargetLabelled records a targeting observation under an explicit label. A
// label may be observed at most once per example — a property targeting
// several quantities names them apart.
func (tc *TestCase) TargetLabelled(score float64, label string) {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		panic(fmt.Sprintf("zizek: target(%v, %q) requires a finite score", score, label))
	}
	family := tc.root()
	if family.targets == nil {
		family.targets = map[string]float64{}
	}
	if _, seen := family.targets[label]; seen {
		panic(fmt.Sprintf(
			"zizek: target(%v, %q) would overwrite an earlier observation; "+
				"each label can be observed at most once per example", score, label))
	}
	family.targets[label] = score
}

func (tc *TestCase) totalCost() uint64 {
	var total int64
	for _, amount := range tc.root().costs {
		total += amount
	}
	return uint64(total)
}

// NextKey is the key naming the next draw in a sequence.
//
// It appends an ordinal counted *within this name*, so NextKey("read", conn)
// numbers that connection's reads independently of every other sequence. An
// insertion then shifts only its own sequence.
func (tc *TestCase) NextKey(components ...any) uint64 {
	scope := keyEncode(components)
	if tc.keyCounts == nil {
		tc.keyCounts = map[string]int64{}
	}
	ordinal := tc.keyCounts[string(scope)]
	tc.keyCounts[string(scope)] = ordinal + 1
	return keyHash(keyAppendInt(scope, signedBytes(ordinal)))
}

type generationTimer struct {
	start time.Time
	outer bool
}

func (tc *TestCase) startGeneration() generationTimer {
	family := tc.root()
	timer := generationTimer{outer: family.generationDepth == 0}
	family.generationDepth++
	if timer.outer {
		timer.start = time.Now()
	}
	return timer
}

func (tc *TestCase) finishGeneration(timer generationTimer) {
	family := tc.root()
	family.generationDepth--
	if !timer.outer {
		return
	}
	elapsed := uint64(time.Since(timer.start) / time.Microsecond)
	if ^uint64(0)-family.generationMicros < elapsed {
		family.generationMicros = ^uint64(0)
	} else {
		family.generationMicros += elapsed
	}
}

// Assume rejects the current generated example when condition is false.
//
// Rejection is control flow, not a property failure: Check continues until it
// has run the requested number of valid examples or reaches MaxRejections.
func (tc *TestCase) Assume(condition bool) {
	if !condition {
		panic(errAssumptionRejected)
	}
}

// Always asserts that condition holds every time this call site runs. A false
// condition fails the current example — an ordinary property failure that
// shrinks and replays like any other, with this call site as the bug's
// origin. A passing evaluation also counts as covering label, so a call site
// that may legitimately never execute is fine on its own ("always or
// unreachable"), while pairing the label with Config.Require additionally
// fails a run that never reached it.
func (tc *TestCase) Always(label string, condition bool) {
	if !condition {
		panic(fmt.Sprintf("always: %q did not hold", label))
	}
	tc.Cover(label, true, 0)
}

// Unreachable fails the current example when it runs at all: the assertion is
// that no example reaches this call site. The failure shrinks and replays
// like any other, so the reported counterexample is the simplest input that
// reaches the branch.
func (tc *TestCase) Unreachable(label string) {
	panic(fmt.Sprintf("unreachable: %q was reached", label))
}

// Sometimes asserts that condition holds in at least one valid example of the
// run. A property passes vacuously when the branch it exists to test never
// executes; Sometimes turns that silent gap into a failure. The verdict is
// per run, not per example, so a false condition here never fails the current
// example — only a completed run in which the condition never held.
func (tc *TestCase) Sometimes(label string, condition bool) {
	tc.Cover(label, condition, 0)
}

// Cover asserts that condition holds in at least percent% of the run's valid
// examples — and at least once, whatever the rate. A label counts once per
// example in which its condition held, however many times the call site ran.
// Percent zero is Sometimes. Labels a run must observe even when the call
// site itself may never execute are declared up front via Config.Require.
func (tc *TestCase) Cover(label string, condition bool, percent float64) {
	if !(percent >= 0 && percent <= 100) {
		panic("zizek: cover percent must be between 0 and 100")
	}
	family := tc.root()
	if family.coverageHeld == nil {
		family.coverageHeld = map[string]bool{}
		family.coveragePermille = map[string]uint64{}
	}
	held, seen := family.coverageHeld[label]
	if !seen {
		family.coverageOrder = append(family.coverageOrder, label)
	}
	family.coverageHeld[label] = held || condition
	if permille := uint64(percent * 10); permille > family.coveragePermille[label] {
		family.coveragePermille[label] = permille
	}
}

// Note attaches a line of diagnostic context to this example.
func (tc *TestCase) Note(message string) {
	family := tc.root()
	family.notes = append(family.notes, message)
}

// Notes returns a snapshot of the current diagnostic trace.
func (tc *TestCase) Notes() []string {
	return append([]string(nil), tc.root().notes...)
}

// StartSpan opens a labelled structural span at the current choice position
// and returns its span ID.
func (tc *TestCase) StartSpan(label string) int {
	parent := -1
	if len(tc.spanStack) != 0 {
		parent = tc.spanStack[len(tc.spanStack)-1]
	}
	id := len(tc.spans)
	tc.spans = append(tc.spans, ChoiceSpan{
		Start:  len(tc.draws),
		End:    len(tc.draws),
		Label:  label,
		Depth:  len(tc.spanStack),
		Parent: parent,
	})
	tc.spanStack = append(tc.spanStack, id)
	return id
}

// StopSpan closes the innermost open span. A discarded span records a
// rejected/abandoned attempt while preserving its structural range.
func (tc *TestCase) StopSpan(discarded bool) {
	if len(tc.spanStack) == 0 {
		return
	}
	last := len(tc.spanStack) - 1
	id := tc.spanStack[last]
	tc.spanStack = tc.spanStack[:last]
	tc.spans[id].End = len(tc.draws)
	tc.spans[id].Discarded = discarded
}

// Spans returns a snapshot of the spans recorded so far.
func (tc *TestCase) Spans() []ChoiceSpan {
	return append([]ChoiceSpan(nil), tc.spans...)
}

func (tc *TestCase) closeOpenSpans() {
	for len(tc.spanStack) != 0 {
		tc.StopSpan(false)
	}
	// A clone has its own span structure, and the property may leave one open
	// on a child exactly as it may on the stream it was handed.
	for _, item := range tc.draws {
		if item.child != nil {
			item.child.closeOpenSpans()
		}
	}
}

func newRandomCase(rng *rand.Rand) *TestCase {
	return &TestCase{rng: rng, position: 4}
}

func newSimplestCase() *TestCase {
	return &TestCase{simplest: true, position: 4}
}

func newReplayCase(replay []byte) *TestCase {
	return &TestCase{
		replay:    replay,
		position:  4,
		remaining: choiceStreamCount(replay),
	}
}

func newProbeCase(replay []byte, rng *rand.Rand) *TestCase {
	tc := newReplayCase(replay)
	tc.rng = rng
	return tc
}

func signedBytes(value int64) []byte {
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, uint64(value))
	return canonicalizeSignedLe(raw)
}

func positiveBytes(value uint64) []byte {
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, value)
	if value>>63 != 0 {
		raw = append(raw, 0)
	}
	return canonicalizeSignedLe(raw)
}

func signedInt(value []byte) (int64, bool) {
	canonical := canonicalizeSignedLe(value)
	if len(canonical) > 8 {
		return 0, false
	}
	fill := byte(0)
	if canonical[len(canonical)-1]&0x80 != 0 {
		fill = 0xff
	}
	raw := [8]byte{}
	for i := range raw {
		raw[i] = fill
	}
	copy(raw[:], canonical)
	return int64(binary.LittleEndian.Uint64(raw[:])), true
}

func positiveUint(value []byte) (uint64, bool) {
	canonical := canonicalizeSignedLe(value)
	if canonical[len(canonical)-1]&0x80 != 0 || len(canonical) > 9 {
		return 0, false
	}
	if len(canonical) == 9 && canonical[8] != 0 {
		return 0, false
	}
	raw := [8]byte{}
	copy(raw[:], canonical)
	return binary.LittleEndian.Uint64(raw[:]), true
}

func (tc *TestCase) advanceReplay(kind string) (int64, error) {
	for {
		if tc.replay == nil || tc.remaining == 0 {
			return 0, &ReplayOverrun{Message: "replay ended before " + kind + " draw"}
		}
		start := tc.position
		end := choiceEntryEnd(tc.replay, start, 0)
		if end == 0 {
			return 0, &InvalidReplay{Message: fmt.Sprintf("invalid choice at offset %d", start)}
		}
		tc.position = end
		tc.remaining--
		// A named entry belongs to the draw that names it, wherever that draw
		// happens to fall. The positional cursor steps over it, so adding or
		// removing keyed draws cannot shift unkeyed ones.
		if _, named := keyedEntryKey(tc.replay, start); !named {
			return start, nil
		}
	}
}

// keyOffsets indexes the replay stream by key, innermost entry offset first.
func (tc *TestCase) keyOffsets() map[uint64][]int64 {
	if tc.keyed == nil {
		offsets := map[uint64][]int64{}
		position := int64(4)
		for remaining := choiceStreamCount(tc.replay); remaining > 0; remaining-- {
			end := choiceEntryEnd(tc.replay, position, 0)
			if end == 0 {
				break
			}
			if key, named := keyedEntryKey(tc.replay, position); named {
				offsets[key] = append(offsets[key], keyedEntryInner(tc.replay, position))
			}
			position = end
		}
		tc.keyed = offsets
	}
	return tc.keyed
}

// resolve reports where to read this draw from, and whether to use its
// simplest value instead.
//
// An unkeyed draw takes the next positional entry and overruns when there is
// none. A keyed draw takes the next entry recorded under its name and falls
// back to the simplest value when the stream has none — a missing name is an
// edit to the property, not a malformed blob.
func (tc *TestCase) resolve(kind string, key uint64, keyed bool) (int64, bool, error) {
	if tc.simplest {
		return 0, true, nil
	}
	if tc.replay == nil {
		return 0, false, nil
	}
	if !keyed {
		start, err := tc.advanceReplay(kind)
		return start, false, err
	}
	pending := tc.keyOffsets()[key]
	if len(pending) == 0 {
		tc.root().keyMisses++
		return 0, true, nil
	}
	tc.keyed[key] = pending[1:]
	return pending[0], false, nil
}

func (tc *TestCase) checkChoiceBudget() error {
	if tc.remaining == 0 && tc.rng != nil {
		tc.replay = nil
	}
	// The budget belongs to the family, not to one stream: upstream's cap is on
	// the whole pieced-together choice sequence, so a clone cannot buy another
	// 8,192 draws by starting a stream of its own.
	if tc.replay == nil && choiceBudgetExhausted(tc.root().flatDraws) {
		return &choiceOverrun{}
	}
	return nil
}

// randomUniformInt64 draws without the boundary catalogue. Every narrow range
// (below integerSwarmIsWide) takes this path: a uniform draw already lands on
// each endpoint of a narrow range often enough, and a boundary boost on top
// of it over-picked the ends — most visibly the first and last option of a
// list indexed by an integer draw. Recursive target sizes use it for wide
// ranges too, so a target sampled from [2, maxLeaves] really is uniform.
func randomUniformInt64(rng *rand.Rand, minValue, maxValue int64) int64 {
	span := uint64(maxValue) - uint64(minValue) + 1
	if span == 0 {
		return int64(rng.Uint64())
	}
	limit := ^uint64(0) - (^uint64(0) % span)
	for {
		value := rng.Uint64()
		if value < limit {
			return int64(uint64(minValue) + value%span)
		}
	}
}

// randomSwarmInt64 applies one family-wide mixture to every wide integer in
// an example. The mixture word is deliberately not recorded: only the chosen
// value enters the stream, so replay and shrinking retain their old contract.
func (tc *TestCase) randomSwarmInt64(minValue, maxValue int64) int64 {
	if !integerSwarmIsWide(minValue, maxValue) {
		return randomUniformInt64(tc.rng, minValue, maxValue)
	}
	root := tc.root()
	if !root.integerSwarmSet {
		root.integerSwarm = tc.rng.Uint64()
		root.integerSwarmSet = true
	}
	category := integerSwarmCategory(root.integerSwarm, uint64(tc.rng.Intn(1000)))
	switch category {
	case 0:
		if value, ok := integerSwarmEndpointCandidate(minValue, maxValue, tc.rng.Uint64()%4); ok {
			return value
		}
	case 1:
		count := integerBoundaryCount(minValue, maxValue)
		if count > 2 {
			if value, ok := integerBoundaryCandidate(minValue, maxValue, 2+tc.rng.Uint64()%(count-2)); ok {
				return value
			}
		}
	case 2:
		if value, ok := integerSwarmDiffuseCandidate(
			minValue, maxValue, tc.rng.Uint64()%integerSwarmDiffuseCount(),
		); ok {
			return value
		}
	}
	return randomUniformInt64(tc.rng, minValue, maxValue)
}

// randomSwarmFloat draws a float from one family-wide category mixture: a
// roll picks the category, and a category the spec rules out yields to the
// default draw, so its mass never goes to a neighbouring category. Like the
// integer swarm, the mixture word is never recorded.
func (tc *TestCase) randomSwarmFloat(
	minValue, maxValue float64,
	allowNaN, allowInfinity bool,
	smallestNonzeroMagnitude float64,
) float64 {
	root := tc.root()
	if !root.floatSwarmSet {
		root.floatSwarm = tc.rng.Uint64()
		root.floatSwarmSet = true
	}
	category := floatSwarmCategory(root.floatSwarm, tc.rng.Uint64()%floatSwarmRollSpan())
	if value, ok := floatCategoryCandidate(
		category, minValue, maxValue, allowNaN, allowInfinity, smallestNonzeroMagnitude,
		tc.rng.Uint64(), tc.rng.Uint64(), tc.rng.Uint64(), tc.rng.Uint64(), tc.rng.Uint64(),
	); ok {
		return value
	}
	return floatDefaultCandidate(
		minValue, maxValue, allowNaN, allowInfinity, smallestNonzeroMagnitude,
		tc.rng.Uint64(), tc.rng.Uint64(), tc.rng.Uint64(), tc.rng.Uint64(),
	)
}

// Integer draws an int64 in [minValue, maxValue], shrinking toward
// shrinkTowards clamped to that interval.
func (tc *TestCase) Integer(minValue, maxValue, shrinkTowards int64) (int64, error) {
	return tc.integer(minValue, maxValue, shrinkTowards, 0, false, false)
}

// IntegerKeyed draws an integer addressed by name rather than by position.
// The key comes from Key or NextKey.
func (tc *TestCase) IntegerKeyed(minValue, maxValue, shrinkTowards int64, key uint64) (int64, error) {
	return tc.integer(minValue, maxValue, shrinkTowards, key, true, false)
}

// integerUniform records and replays an ordinary integer choice but samples a
// fresh value uniformly. It is intentionally internal: user integer draws use
// the discovery-biased sampler, while engine parameters must not.
func (tc *TestCase) integerUniform(minValue, maxValue, shrinkTowards int64) (int64, error) {
	return tc.integer(minValue, maxValue, shrinkTowards, 0, false, true)
}

func (tc *TestCase) integer(minValue, maxValue, shrinkTowards int64, key uint64, keyed bool, uniform bool) (int64, error) {
	if minValue > maxValue {
		return 0, errors.New("minValue must be <= maxValue")
	}
	if err := tc.checkChoiceBudget(); err != nil {
		return 0, err
	}
	start, useSimplest, err := tc.resolve("integer", key, keyed)
	if err != nil {
		return 0, err
	}
	var value int64
	if useSimplest {
		valueBytes := integerSimplest(
			signedBytes(minValue), signedBytes(maxValue), signedBytes(shrinkTowards),
		)
		value, _ = signedInt(valueBytes)
	} else if tc.replay == nil {
		if uniform {
			value = randomUniformInt64(tc.rng, minValue, maxValue)
		} else {
			value = tc.randomSwarmInt64(minValue, maxValue)
		}
	} else {
		low, high := signedBytes(minValue), signedBytes(maxValue)
		if replayIntegerPuns(tc.replay, start, low, high) {
			tc.root().puns++
		}
		value, _ = signedInt(
			replayInteger(tc.replay, start, low, high, signedBytes(shrinkTowards)),
		)
	}
	encoded := serializeIntegerChoice(signedBytes(value))
	tc.record(draw{
		kind: "integer", encoded: encoded, integer: value,
		params: []int64{minValue, maxValue, shrinkTowards},
		key:    key, keyed: keyed,
	})
	return value, nil
}

// freshIDRegistry is the family's set of identifiers ever drawn.
func (tc *TestCase) freshIDRegistry() map[int64]bool {
	family := tc.root()
	if family.freshIDs == nil {
		family.freshIDs = map[int64]bool{}
	}
	return family.freshIDs
}

// sortedIdentifiers lists a registry ascending. The fallback fold below is
// defined over the identifiers in ascending order, and Go's map iteration is
// deliberately not.
func sortedIdentifiers(used map[int64]bool) []int64 {
	ordered := make([]int64, 0, len(used))
	for id := range used {
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered
}

// identifierMaxPlusOne anchors both identifier windows: one more than the
// largest identifier the family has ever drawn, or zero before the first draw.
func identifierMaxPlusOne(ordered []int64) uint64 {
	if len(ordered) == 0 {
		return 0
	}
	return uint64(ordered[len(ordered)-1]) + 1
}

// drawIdentifier records one identifier choice: an integer in [0, windowHi].
//
// Identifier draws differ from Integer only in how a replayed value is
// resolved — an ordinary integer puns to the nearest legal value, while an
// identifier is repaired onto one the pool still holds — so what reaches the
// stream is an ordinary integer entry and nothing downstream (the wire format,
// the shrinker, the indexers) has to know these exist. The repaired value is
// what gets recorded, which is what lets a replayed stream heal rather than
// drift.
func (tc *TestCase) drawIdentifier(
	windowHi int64,
	simplestValue int64,
	generate func() int64,
	repair func(int64) int64,
) (int64, error) {
	if err := tc.checkChoiceBudget(); err != nil {
		return 0, err
	}
	start, useSimplest, err := tc.resolve("integer", 0, false)
	if err != nil {
		return 0, err
	}
	var value int64
	if useSimplest {
		value = simplestValue
	} else if tc.replay == nil {
		value = generate()
	} else {
		low, high := signedBytes(0), signedBytes(windowHi)
		// A recorded value outside the window is punned, not repaired. Repair
		// is for a value that named a member which has since gone, and it
		// slides towards an earlier one; a value that was never in the window
		// names nothing, so it falls back to the simplest member the way every
		// other out-of-range replay value does. Reading replayInteger first
		// would lose the distinction — it clamps to the window, which would
		// send a runaway value to the largest member rather than the smallest.
		if replayIntegerPuns(tc.replay, start, low, high) {
			tc.root().puns++
			value = simplestValue
		} else {
			raw, _ := signedInt(
				replayInteger(tc.replay, start, low, high, signedBytes(0)),
			)
			value = repair(raw)
		}
	}
	encoded := serializeIntegerChoice(signedBytes(value))
	tc.record(draw{
		kind: "integer", encoded: encoded, integer: value,
		params: []int64{0, windowHi, 0},
	})
	return value, nil
}

// FreshID draws an identifier that is arbitrary but unique within this family.
//
// The identifier is recorded by value, so it keeps meaning the same variable
// when unrelated earlier draws are deleted during shrinking — which is the
// whole point of it, and what a positional index cannot do. Generation always
// hands out the smallest unused identifier, and a replayed identifier that is
// already in use is repaired to that same smallest unused one.
func (tc *TestCase) FreshID() (int64, error) {
	used := tc.freshIDRegistry()
	ordered := sortedIdentifiers(used)
	windowHi := int64(statefulFreshIdWindow(identifierMaxPlusOne(ordered)))
	fallback := uint64(0)
	for _, usedID := range ordered {
		fallback = statefulFreshIdFallbackStep(fallback, uint64(usedID))
	}
	smallestUnused := int64(fallback)

	value, err := tc.drawIdentifier(
		windowHi,
		smallestUnused,
		func() int64 { return smallestUnused },
		func(raw int64) int64 {
			if statefulFreshIdAccepts(uint64(raw), uint64(windowHi), used[raw]) {
				return raw
			}
			return smallestUnused
		},
	)
	if err != nil {
		return 0, err
	}
	used[value] = true
	return value, nil
}

// FromSet draws one of members, recording the member itself.
//
// Recording the member rather than an index into the live set is what makes a
// reference survive the deletion of other members: while the member it names
// is still there it keeps naming it, and when that member goes the reference
// slides to an earlier one rather than to whatever now occupies some shifted
// position.
func (tc *TestCase) FromSet(members []int64) (int64, error) {
	if len(members) == 0 {
		return 0, errors.New("FromSet requires at least one member")
	}
	seen := make(map[int64]bool, len(members))
	for _, member := range members {
		seen[member] = true
	}
	ordered := sortedIdentifiers(seen)
	if ordered[0] < 0 {
		return 0, errors.New("FromSet requires nonnegative members")
	}
	smallest := ordered[0]
	windowHi := int64(statefulSetChoiceWindow(
		identifierMaxPlusOne(sortedIdentifiers(tc.freshIDRegistry())),
	))

	return tc.drawIdentifier(
		windowHi,
		smallest,
		func() int64 { return ordered[tc.rng.Intn(len(ordered))] },
		func(raw int64) int64 {
			if !statefulSetChoiceAccepts(uint64(raw), uint64(windowHi)) {
				return smallest
			}
			best := smallest
			for _, candidate := range ordered {
				if statefulSetChoiceBetter(uint64(candidate), uint64(raw), uint64(best)) {
					best = candidate
				}
			}
			return best
		},
	)
}

// Boolean draws a boolean, shrinking toward false.
func (tc *TestCase) Boolean() (bool, error) { return tc.booleanDraw(0.5, false, 0, false) }

// BooleanKeyed draws a boolean addressed by name rather than by position.
func (tc *TestCase) BooleanKeyed(key uint64) (bool, error) {
	return tc.booleanDraw(0.5, false, key, true)
}

// WeightedKeyed draws a weighted boolean addressed by name.
func (tc *TestCase) WeightedKeyed(probability float64, key uint64) (bool, error) {
	return tc.booleanDraw(probability, true, key, true)
}

func (tc *TestCase) booleanDraw(probability float64, weighted bool, key uint64, keyed bool) (bool, error) {
	if err := tc.checkChoiceBudget(); err != nil {
		return false, err
	}
	start, useSimplest, err := tc.resolve("boolean", key, keyed)
	if err != nil {
		return false, err
	}
	var value bool
	if useSimplest {
		value = false
	} else if tc.replay == nil {
		if weighted {
			value = tc.rng.Float64() < probability
		} else {
			value = tc.rng.Intn(2) != 0
		}
	} else {
		if replayBooleanPuns(tc.replay, start) {
			tc.root().puns++
		}
		value = replayBoolean(tc.replay, start)
	}
	tc.record(draw{
		kind: "boolean", encoded: serializeBooleanChoice(value), boolean: value,
		key: key, keyed: keyed,
	})
	return value, nil
}

// Weighted draws a boolean that is true with the given probability. It is
// recorded as an ordinary boolean choice: the weight biases generation only,
// so a blob replays the same whatever probability produced it.
func (tc *TestCase) Weighted(probability float64) (bool, error) {
	return tc.booleanDraw(probability, true, 0, false)
}

// Bytes draws a byte slice whose length is in [minSize, maxSize].
func (tc *TestCase) Bytes(minSize, maxSize int) ([]byte, error) {
	return tc.bytesDraw(minSize, maxSize, 0, false)
}

// BytesKeyed draws a byte slice addressed by name rather than by position.
func (tc *TestCase) BytesKeyed(minSize, maxSize int, key uint64) ([]byte, error) {
	return tc.bytesDraw(minSize, maxSize, key, true)
}

func (tc *TestCase) bytesDraw(minSize, maxSize int, key uint64, keyed bool) ([]byte, error) {
	if minSize < 0 || minSize > maxSize {
		return nil, errors.New("require 0 <= minSize <= maxSize")
	}
	if err := tc.checkChoiceBudget(); err != nil {
		return nil, err
	}
	start, useSimplest, err := tc.resolve("bytes", key, keyed)
	if err != nil {
		return nil, err
	}
	var value []byte
	if useSimplest {
		value = bytesSimplest(int64(minSize))
	} else if tc.replay == nil {
		size := minSize + tc.rng.Intn(maxSize-minSize+1)
		value = make([]byte, size)
		_, _ = tc.rng.Read(value)
	} else {
		if replayBytesPuns(tc.replay, start, int64(minSize), int64(maxSize)) {
			tc.root().puns++
		}
		value = replayBytes(tc.replay, start, int64(minSize), int64(maxSize))
	}
	value = append([]byte(nil), value...)
	tc.record(draw{
		kind: "bytes", encoded: serializeBytesChoice(value), bytes: value,
		params: []int64{int64(minSize), int64(maxSize)},
		key:    key, keyed: keyed,
	})
	return append([]byte(nil), value...), nil
}

// Unicode is the default text alphabet: every Unicode scalar value, as
// Hypothesis uses. Generation is biased towards the simple end of the shrink
// order, so most drawn characters are still ASCII (see ir/generation.zir).
var Unicode = alphabetRange(0, 0x10FFFF)

// Alphabet returns the alphabet holding exactly these characters.
//
// Alphabets are interval sets, canonical by construction: the order the
// characters are written in and any repeats among them do not change the
// result, so Alphabet("cab") and Alphabet("abc") are the same alphabet and
// index their members identically.
func Alphabet(characters string) ([]byte, error) {
	encoded := []byte{}
	for _, character := range characters {
		if !alphabetCodepointAllowed(uint64(character)) {
			return nil, fmt.Errorf("%q cannot be an alphabet member", character)
		}
		encoded = alphabetInsert(encoded, uint64(character))
	}
	return encoded, nil
}

// AlphabetRange returns the alphabet holding every scalar value in a codepoint
// range. Surrogates are not scalar values and are dropped from the range
// rather than rejected, so AlphabetRange(0, 0x10FFFF) is the whole of Unicode.
func AlphabetRange(minCodepoint, maxCodepoint uint64) []byte {
	return alphabetRange(minCodepoint, maxCodepoint)
}

func packCodepoints(value string) []byte {
	packed := make([]byte, 0, 4*len(value))
	for _, character := range value {
		packed = append(packed, byte(character), byte(character>>8), byte(character>>16), byte(character>>24))
	}
	return packed
}

func unpackCodepoints(packed []byte) string {
	runes := make([]rune, 0, len(packed)/4)
	for at := 0; at+4 <= len(packed); at += 4 {
		runes = append(runes, rune(uint32(packed[at])|uint32(packed[at+1])<<8|
			uint32(packed[at+2])<<16|uint32(packed[at+3])<<24))
	}
	return string(runes)
}

// String draws text of length [minSize, maxSize] over an alphabet. A nil
// alphabet means every Unicode scalar value.
//
// Shrinking is shortlex over the alphabet's shrink order, which puts '0'
// through 'Z' first, so a minimal counterexample reads as "0" rather than as a
// control character.
func (tc *TestCase) String(minSize, maxSize int, alphabet []byte) (string, error) {
	return tc.stringDraw(minSize, maxSize, alphabet, 0, false)
}

// StringKeyed draws text addressed by name rather than by position.
func (tc *TestCase) StringKeyed(minSize, maxSize int, alphabet []byte, key uint64) (string, error) {
	return tc.stringDraw(minSize, maxSize, alphabet, key, true)
}

func (tc *TestCase) stringDraw(
	minSize, maxSize int, characters []byte, key uint64, keyed bool,
) (string, error) {
	if characters == nil {
		characters = Unicode
	}
	if minSize < 0 || minSize > maxSize {
		return "", errors.New("require 0 <= minSize <= maxSize")
	}
	if !alphabetValid(characters) {
		return "", errors.New("alphabet must be a canonical non-empty interval set")
	}
	if err := tc.checkChoiceBudget(); err != nil {
		return "", err
	}
	start, useSimplest, err := tc.resolve("string", key, keyed)
	if err != nil {
		return "", err
	}
	var value []byte
	if useSimplest {
		value = stringSimplest(characters, int64(minSize))
	} else if tc.replay == nil {
		size := minSize + tc.rng.Intn(maxSize-minSize+1)
		biased := uint64(tc.rng.Intn(1000)) < textBiasThreshold()
		span := textRankSpan(characters, biased)
		value = make([]byte, 0, 4*size)
		for i := 0; i < size; i++ {
			codepoint, _ := alphabetShrinkCodepoint(characters, uint64(tc.rng.Int63n(int64(span))))
			value = append(value, byte(codepoint), byte(codepoint>>8), byte(codepoint>>16), byte(codepoint>>24))
		}
	} else {
		if replayStringPuns(tc.replay, start, characters, int64(minSize), int64(maxSize)) {
			tc.root().puns++
		}
		value = replayString(tc.replay, start, characters, int64(minSize), int64(maxSize))
	}
	value = append([]byte(nil), value...)
	tc.record(draw{
		kind: "string", encoded: serializeStringChoicePacked(uint64(len(value)/4), value),
		bytes: value, alphabet: characters,
		params: []int64{int64(minSize), int64(maxSize)},
		key:    key, keyed: keyed,
	})
	return unpackCodepoints(value), nil
}

// Float draws an IEEE-754 binary64 value with portable replay and shrinking.
// Bounds use sign-aware ordering, so -0.0 is strictly below +0.0.
func (tc *TestCase) Float(
	minValue, maxValue float64,
	allowNaN, allowInfinity bool,
	smallestNonzeroMagnitude float64,
) (float64, error) {
	return tc.floatDraw(minValue, maxValue, allowNaN, allowInfinity, smallestNonzeroMagnitude, 0, false)
}

// FloatKeyed draws a float addressed by name rather than by position.
func (tc *TestCase) FloatKeyed(
	minValue, maxValue float64,
	allowNaN, allowInfinity bool,
	smallestNonzeroMagnitude float64,
	key uint64,
) (float64, error) {
	return tc.floatDraw(minValue, maxValue, allowNaN, allowInfinity, smallestNonzeroMagnitude, key, true)
}

func (tc *TestCase) floatDraw(
	minValue, maxValue float64,
	allowNaN, allowInfinity bool,
	smallestNonzeroMagnitude float64,
	key uint64, keyed bool,
) (float64, error) {
	if !floatGenerationSpecValid(
		minValue, maxValue, allowNaN, allowInfinity, smallestNonzeroMagnitude,
	) {
		return 0, errors.New("invalid float bounds")
	}
	unit, ok := floatUnit(
		minValue, maxValue, allowNaN, allowInfinity, smallestNonzeroMagnitude,
	)
	if !ok {
		return 0, errors.New("float configuration admits no values")
	}
	if err := tc.checkChoiceBudget(); err != nil {
		return 0, err
	}

	start, useSimplest, err := tc.resolve("float", key, keyed)
	if err != nil {
		return 0, err
	}
	var value float64
	if useSimplest {
		value, _ = floatSimplest(
			minValue, maxValue, allowNaN, allowInfinity, smallestNonzeroMagnitude,
		)
	} else if tc.replay != nil {
		if replayFloatPuns(
			tc.replay, start,
			minValue, maxValue, allowNaN, allowInfinity,
			smallestNonzeroMagnitude,
		) {
			tc.root().puns++
		}
		replayed, resolved := replayFloat(
			tc.replay, start,
			minValue, maxValue, allowNaN, allowInfinity,
			smallestNonzeroMagnitude,
		)
		if resolved {
			value = replayed
		} else {
			value = unit
		}
	} else {
		value = tc.randomSwarmFloat(
			minValue, maxValue, allowNaN, allowInfinity, smallestNonzeroMagnitude,
		)
	}

	tc.record(draw{
		kind: "float", encoded: serializeFloatChoice(value), float: value,
		floats: [3]float64{minValue, maxValue, smallestNonzeroMagnitude},
		flags:  [2]bool{allowNaN, allowInfinity},
		key:    key, keyed: keyed,
	})
	return value, nil
}

// maxCloneDepth bounds how deeply clones may nest. It is Hegel's
// MAX_CLONE_DEPTH and the wire format's own nesting limit (ir/wire.zir
// @choice_entry_end): a stream nested deeper than this cannot be validated, so
// refusing to build one keeps the shell from emitting a blob nothing can read.
const maxCloneDepth = 100

// CloneStream returns a child test case that draws from a stream of its own.
//
// The clone occupies one choice position in this stream and gets its own
// prefix, RNG, and span structure, so its draws neither shift nor are shifted
// by the parent's — which is what makes it the seam a concurrent runner would
// use. Everything the run reports is shared: notes, costs, coverage, and the
// choice budget all belong to the family rather than to one stream.
//
// This is Hegel's hegel_test_case_clone without threads. The child is drawn
// from sequentially, but the wire framing (`[5][u32 child count][children]`)
// and the replay rules are the engine's, so a blob carrying a clone crosses
// between the two.
//
// On replay a clone value at this position hands the child its recorded
// stream. Any other value puns to an empty child, which overruns on its first
// draw in a bare replay and extends randomly under a probe — the engine's
// rule, and the reason a blob whose draws have changed shape still replays.
func (tc *TestCase) CloneStream() (*TestCase, error) {
	if err := tc.checkChoiceBudget(); err != nil {
		return nil, err
	}
	if tc.depth()+1 > maxCloneDepth {
		// Upstream concludes the family invalid here rather than failing the
		// property, so this is a rejection: the same path a false Assume takes.
		panic(errAssumptionRejected)
	}
	start, useSimplest, err := tc.resolve("clone", 0, false)
	if err != nil {
		return nil, err
	}
	child := &TestCase{parent: tc, position: 4}
	switch {
	case useSimplest:
		child.simplest = true
	case tc.replay == nil:
		child.rng = spawnRng(tc.rng)
	default:
		inner, ok := cloneInnerStream(tc.replay, uint64(start))
		if !ok {
			// A non-clone entry where a clone belongs: pun to an empty child.
			tc.root().puns++
			inner = choiceListHeader(0)
		}
		child.replay = inner
		child.remaining = choiceStreamCount(inner)
		child.rng = spawnRng(tc.rng)
	}
	tc.record(draw{kind: "clone", child: child})
	return child, nil
}

// spawnRng gives a clone a stream of its own, drawn from its parent's, so two
// streams never hand back the same values. A parent without an RNG (a bare
// replay, or the all-simplest probe) has none to give.
func spawnRng(rng *rand.Rand) *rand.Rand {
	if rng == nil {
		return nil
	}
	return rand.New(rand.NewSource(rng.Int63()))
}

// integerForced records value at the next choice position. During replay the
// prefix entry at that position is skipped, matching Hegel's forced-draw
// cursor accounting.
func (tc *TestCase) integerForced(value int64) (int64, error) {
	if err := tc.checkChoiceBudget(); err != nil {
		return 0, err
	}
	if tc.replay != nil {
		if _, err := tc.advanceReplay("forced integer"); err != nil {
			return 0, err
		}
	}
	tc.record(draw{
		kind: "integer", encoded: serializeIntegerChoice(signedBytes(value)), forced: true, integer: value,
		params: []int64{value, value, value},
	})
	return value, nil
}

// booleanForced records value at the next choice position, skipping the
// replay entry there the way integerForced does.
func (tc *TestCase) booleanForced(value bool) (bool, error) {
	if err := tc.checkChoiceBudget(); err != nil {
		return false, err
	}
	if tc.replay != nil {
		if _, err := tc.advanceReplay("forced boolean"); err != nil {
			return false, err
		}
	}
	tc.record(draw{
		kind: "boolean", encoded: serializeBooleanChoice(value), forced: true, boolean: value,
	})
	return value, nil
}

func (tc *TestCase) finish() error {
	return nil
}

func serializeDraws(draws []draw) []byte {
	raw := choiceListHeader(uint64(len(draws)))
	for _, item := range draws {
		raw = append(raw, item.entry()...)
	}
	return raw
}

type choiceTree struct {
	terminal bool
	children map[byte]*choiceTree
}

func (tree *choiceTree) contains(path []byte) bool {
	node := tree
	for _, value := range path {
		child := node.children[value]
		if child == nil {
			return false
		}
		node = child
	}
	return node.terminal
}

func (tree *choiceTree) insert(path []byte) {
	node := tree
	for _, value := range path {
		if node.children == nil {
			node.children = make(map[byte]*choiceTree)
		}
		if node.children[value] == nil {
			node.children[value] = &choiceTree{}
		}
		node = node.children[value]
	}
	node.terminal = true
}

func spanMutationCandidates(draws []draw, spans []ChoiceSpan, rng *rand.Rand) [][]byte {
	for _, item := range draws {
		if item.keyed {
			return nil
		}
	}
	grouped := map[string]map[[2]int]struct{}{}
	for _, span := range spans {
		if span.Discarded || span.End <= span.Start || span.End > len(draws) {
			continue
		}
		if grouped[span.Label] == nil {
			grouped[span.Label] = map[[2]int]struct{}{}
		}
		grouped[span.Label][[2]int{span.Start, span.End}] = struct{}{}
	}
	labels := make([]string, 0, len(grouped))
	for label, ranges := range grouped {
		if len(ranges) >= 2 {
			labels = append(labels, label)
		}
	}
	sort.Strings(labels)
	if len(labels) == 0 {
		return nil
	}
	original := serializeDraws(draws)
	var candidates [][]byte
	for attempt := uint64(0); attempt < mutationAttemptCount(); attempt++ {
		randomized := mutationAttemptRandomized(attempt)
		groupIndex := uint64(0)
		if randomized {
			var ok bool
			groupIndex, ok = mutationIndex(uint64(len(labels)), rng.Uint64())
			if !ok {
				break
			}
		}
		label := labels[groupIndex]
		var ranges [][2]int
		for item := range grouped[label] {
			ranges = append(ranges, item)
		}
		sort.Slice(ranges, func(i, j int) bool {
			return ranges[i][0] < ranges[j][0] || ranges[i][0] == ranges[j][0] && ranges[i][1] < ranges[j][1]
		})
		first, second := uint64(0), uint64(1)
		if randomized {
			first, _ = mutationIndex(uint64(len(ranges)), rng.Uint64())
			var ok bool
			second, ok = mutationDistinctIndex(uint64(len(ranges)), first, rng.Uint64())
			if !ok {
				continue
			}
		}
		a, b := ranges[first], ranges[second]
		if a[0] > b[0] || a[0] == b[0] && a[1] > b[1] {
			a, b = b, a
		}
		useA := mutationUseFirst(attempt)
		if randomized {
			useA = mutationUseFirst(rng.Uint64())
		}
		raw, ok := choiceMutateRanges(original, uint64(a[0]), uint64(a[1]), uint64(b[0]), uint64(b[1]), useA)
		if !ok || bytes.Equal(raw, original) {
			continue
		}
		seen := false
		for _, candidate := range candidates {
			seen = seen || bytes.Equal(candidate, raw)
		}
		if !seen {
			candidates = append(candidates, raw)
		}
	}
	return candidates
}

func (tc *TestCase) serialized() []byte { return serializeDraws(tc.draws) }

// panicFailure keeps the source location of a panic, which a plain error value
// does not carry, so the shrinker can tell one panicking bug from another.
type panicFailure struct {
	value any
	site  string
}

func (e *panicFailure) Error() string { return fmt.Sprintf("property panicked: %v", e.value) }

// panicSite reports the first frame outside this package, which is where the
// property panicked.
func panicSite() string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		// Skip this package and the runtime's own panic machinery; the first
		// frame past them is where the property panicked.
		inShell := strings.HasSuffix(frame.File, "/shell.go")
		inRuntime := strings.HasPrefix(frame.Function, "runtime.")
		if frame.File != "" && !inShell && !inRuntime {
			return fmt.Sprintf("%s:%d", frame.File, frame.Line)
		}
		if !more {
			return ""
		}
	}
}

// Outcome tags are defined by ir/lifecycle.zir. Deriving one from a Go error
// is this shell's job; deciding what it means is the core's.
const (
	outcomeValid uint64 = iota
	outcomeRejected
	outcomeOverrun
	outcomeInvalid
	outcomeInteresting
)

func classify(err error) uint64 {
	if err == nil {
		return outcomeValid
	}
	if errors.Is(err, errAssumptionRejected) {
		return outcomeRejected
	}
	var overrun *ReplayOverrun
	if errors.As(err, &overrun) {
		return outcomeOverrun
	}
	var generatedOverrun *choiceOverrun
	if errors.As(err, &generatedOverrun) {
		return outcomeOverrun
	}
	var replayErr *InvalidReplay
	if errors.As(err, &replayErr) {
		return outcomeInvalid
	}
	return outcomeInteresting
}

// interestingOrigin identifies which bug a failure represents. A panic is
// identified by where it happened; a returned error only by its concrete type,
// because Go error values carry no location. The message is deliberately
// excluded: one quoting a drawn value would change on every shrink step and
// stop the search immediately. Distinct error types or panics therefore give
// sharper attribution than repeated calls to errors.New.
func interestingOrigin(err error) string {
	if err == nil {
		return ""
	}
	var panicked *panicFailure
	if errors.As(err, &panicked) {
		return "panic@" + panicked.site
	}
	unwrapped := err
	for {
		next := errors.Unwrap(unwrapped)
		if next == nil {
			break
		}
		unwrapped = next
	}
	return fmt.Sprintf("%T", unwrapped)
}

// isFailure reports whether an outcome is a failure of the bug being shrunk.
// Accepting any failure lets the search migrate to an unrelated bug that
// happens to be easier to reach, and then report a minimal counterexample for a
// bug the run never found.
func isFailure(err error, origin string) bool {
	outcome := classify(err)
	if !outcomeIsFailure(outcome) {
		return false
	}
	return shrinkAccept(outcome, origin == "" || interestingOrigin(err) == origin)
}

func execute(property Property, tc *TestCase) (failure error) {
	defer func() {
		tc.closeOpenSpans()
		if recovered := recover(); recovered != nil {
			if recovered == errAssumptionRejected {
				failure = errAssumptionRejected
				return
			}
			// A body without an error return (zizek.Test, MustDraw) can only
			// surface a draw error by panicking with it. When that error is
			// the run's own control flow — a rejection, an overrun, a replay
			// mismatch — it must come back out as itself: wrapped as a panic
			// it would classify as interesting and turn a discarded example
			// into a reported bug.
			if err, ok := recovered.(error); ok && !outcomeIsFailure(classify(err)) {
				failure = err
				return
			}
			failure = &panicFailure{value: recovered, site: panicSite()}
		}
	}()
	if err := property(tc); err != nil {
		return err
	}
	return tc.finish()
}

// drawSortIndex returns the choice's unconstrained index for shortlex
// comparison.
func drawSortIndex(item draw) ([]byte, bool) {
	switch item.kind {
	case "boolean":
		return booleanToIndex(item.boolean), true
	case "integer":
		minValue, maxValue, target := item.params[0], item.params[1], item.params[2]
		return integerToIndex(
			signedBytes(minValue), signedBytes(maxValue), signedBytes(target),
			signedBytes(item.integer),
		), true
	case "bytes":
		return bytesToIndex(item.params[0], item.bytes), true
	case "string":
		return stringToIndex(item.alphabet, item.params[0], item.bytes), true
	case "float":
		index, ok := floatChoiceToIndex(
			item.floats[0], item.floats[1], item.flags[0], item.flags[1],
			item.floats[2], item.float,
		)
		if !ok {
			return nil, false
		}
		return index, true
	}
	return nil, false
}

// drawIndex reports where a draw currently sits in its own shrink schedule.
// The compiled schedule is defined over u64 indexes; byte draws can sit
// beyond that, and saturating keeps their earliest (most useful) candidates
// reachable.
func drawIndex(item draw) (uint64, bool) {
	if item.forced {
		return 0, false
	}
	index, ok := drawSortIndex(item)
	if !ok {
		return 0, false
	}
	return saturatedUint(index)
}

// raiseBounds reports where a draw sits and the largest index it could move
// to, for the raise pass, or false for a draw the pass leaves alone. Upstream
// raises every node but a sequence, so strings and byte strings are skipped —
// their index bundles length and content, and a larger one is a longer value —
// and so are clones, which carry a stream rather than a value, and forced
// draws, which are not this stream's to change.
func raiseBounds(item draw) (uint64, uint64, bool) {
	if item.forced || item.child != nil {
		return 0, 0, false
	}
	var largest []byte
	switch item.kind {
	case "boolean":
		largest = booleanMaxIndex()
	case "integer":
		largest = integerMaxIndex(signedBytes(item.params[0]), signedBytes(item.params[1]))
	case "float":
		largest = floatChoiceMaxIndex()
	default:
		return 0, 0, false
	}
	current, ok := drawIndex(item)
	if !ok {
		return 0, 0, false
	}
	maximum, _ := saturatedUint(largest)
	return current, maximum, true
}

// drawGroupKey reports what makes two draws duplicates of each other.
//
// Upstream groups by (choice type, value) and lowers the group using the first
// member's constraints, accepting that a member with different constraints may
// reject the shared value. Grouping by the *index* instead is the same idea in
// the coordinate the schedule already walks: two draws with equal bounds share
// an index exactly when they share a value, and a member whose bounds differ is
// rebuilt for its own bounds or misses. A forced draw is never a member — it has
// no index and may not be rewritten.
func drawGroupKey(item draw) (string, uint64, bool) {
	index, ok := drawIndex(item)
	if !ok {
		return "", 0, false
	}
	return item.kind, index, true
}

// trivialSpanCandidate rebuilds every choice in a span at its simplest value,
// and reports how many of them the pass was actually able to move.
//
// This is upstream's try_trivial_spans. Per-choice descent walks one draw at a
// time and stops the moment a candidate stops failing, so a span whose choices
// only fail *together* at their simplest values is out of its reach — and the
// all-simplest probe does not help either, because it collapses the whole
// example rather than one span of it. A forced draw is left alone, as in every
// other pass, and a draw already at index 0 is copied rather than rebuilt.
func trivialSpanCandidate(draws []draw, span ChoiceSpan) ([]draw, uint64) {
	trial := append([]draw(nil), draws...)
	rewritable := uint64(0)
	for at := span.Start; at < span.End && at < len(draws); at++ {
		index, ok := drawIndex(draws[at])
		if !ok || index == 0 {
			continue
		}
		simplest, legal := drawAtIndex(draws[at], 0)
		if !legal {
			continue
		}
		trial[at] = simplest
		rewritable++
	}
	return trial, rewritable
}

// shortenString rebuilds a string draw with the character at one position
// removed.
//
// The whole-value index bundles length and content into one number, and the
// downward search over it converges on the wrong end of the interval: from
// "abb" over an "abcdef" alphabet it probes eleven candidates and never reaches
// "bb", which sits far below. Removing a character is the candidate that search
// cannot produce.
func shortenString(item draw, position uint64) (draw, bool) {
	if item.kind != "string" || item.forced {
		return draw{}, false
	}
	value, ok := stringDropCharacter(item.bytes, item.params[0], int64(position))
	if !ok {
		return draw{}, false
	}
	candidate := item
	candidate.bytes = value
	candidate.encoded = serializeStringChoicePacked(uint64(len(value)/4), value)
	return candidate, true
}

// lowerCharacters rebuilds a string draw with every copy of one repeated
// character replaced by a lower one.
//
// A string draw carries a single whole-value index, so the per-choice schedule
// can only slide the entire value along one linear order. A property that
// depends on two *characters* of that value agreeing is the case it cannot
// serve — M12's shape, one level down and inside a single draw.
func lowerCharacters(item draw, ordinal uint64) (draw, bool) {
	if item.kind != "string" || item.forced {
		return draw{}, false
	}
	rank, ok := stringDuplicateRank(item.alphabet, item.bytes)
	if !ok {
		return draw{}, false
	}
	target, more := shrinkCandidateAt(rank, ordinal)
	if !more {
		return draw{}, false
	}
	value, rewritten := stringReplaceRank(item.alphabet, item.bytes, rank, target)
	if !rewritten {
		return draw{}, false
	}
	candidate := item
	candidate.bytes = value
	candidate.encoded = serializeStringChoicePacked(uint64(len(value)/4), value)
	return candidate, true
}

// descendantSpan reports the ordinal-th span strictly nested inside spans[at].
//
// Upstream's pass_to_descendant picks a random descendant; a cursor machine has
// no chooser, so the ancestor is the cursor and the descendant is the ordinal.
// Spans are recorded in the order they open, so scanning forward from the
// ancestor gives a stable enumeration in every language.
func descendantSpan(spans []ChoiceSpan, at int, ordinal uint64) (ChoiceSpan, bool) {
	ancestor := spans[at]
	if ancestor.Discarded || ancestor.End <= ancestor.Start {
		return ChoiceSpan{}, false
	}
	seen := uint64(0)
	for other := at + 1; other < len(spans); other++ {
		candidate := spans[other]
		if candidate.Start < ancestor.Start || candidate.End > ancestor.End {
			continue
		}
		if candidate.Discarded || candidate.Depth <= ancestor.Depth {
			continue
		}
		if !shrinkDescendantWorthwhile(
			uint64(ancestor.End-ancestor.Start), uint64(candidate.End-candidate.Start),
		) {
			continue
		}
		if seen == ordinal {
			return candidate, true
		}
		seen++
	}
	return ChoiceSpan{}, false
}

// paidExtent answers the choices a paid deletion of spans[at] removes, and
// whether that extent is wider than the span itself. A span of two or more
// choices is deleted exactly, as the deletion pass already tried on its own. A
// single-choice span takes upstream's extent: the span with the spanless choice
// before it, or else stretched to the start of the next span. A discarded span
// is never a candidate — its choices are a rejected candidate's debris or a
// collection's stop marker, and paying for their deletion buys nothing — and
// neither is an extent covering the whole stream.
func paidExtent(draws []draw, spans []ChoiceSpan, at int) (start, end int, widened, ok bool) {
	span := spans[at]
	if span.Discarded || span.End <= span.Start || span.End > len(draws) {
		return 0, 0, false, false
	}
	start, end = span.Start, span.End
	if end-start < 2 {
		widened = true
		preceded := start > 0
		for _, other := range spans {
			if other.End == start && other.Start < start {
				preceded = false
				break
			}
		}
		if preceded {
			start--
		} else {
			end = len(draws)
			for _, other := range spans {
				if other.Start >= span.End && other.Start < end {
					end = other.Start
				}
			}
			if end <= span.End {
				return 0, 0, false, false
			}
		}
	}
	if start == 0 && end == len(draws) {
		return 0, 0, false, false
	}
	return start, end, widened, true
}

// paidFollower answers the draw that pays for deleting spans[at]: the first one
// after the structure enclosing it. Upstream takes the innermost ancestor span
// reaching past the deletion. A zizek collection opens no span of its own — its
// elements are a run of adjacent sibling spans sharing a label, ending in the
// discarded span of its final "no more" choice — so that run is what plays the
// part of the list, and only a span outside any such run falls back to its
// ancestors.
func paidFollower(spans []ChoiceSpan, at, end, drawCount int) (int, bool) {
	span := spans[at]
	boundary := -1
	runEnd, extended := span.End, false
	for other := at + 1; other < len(spans); other++ {
		candidate := spans[other]
		if candidate.Parent != span.Parent {
			continue
		}
		if candidate.Label != span.Label || candidate.Start != runEnd {
			break
		}
		runEnd, extended = candidate.End, true
	}
	if extended {
		boundary = runEnd
	} else {
		for parent := span.Parent; parent >= 0; parent = spans[parent].Parent {
			if spans[parent].End > end {
				boundary = spans[parent].End
				break
			}
		}
	}
	if boundary < end || boundary >= drawCount {
		return 0, false
	}
	return boundary, true
}

// nudgeDraw answers the paying draw moved one step: an integer down or up, a
// boolean to the side the step names, a string one character shorter. False
// when this draw cannot take this step, so the pass moves to the next one
// without running the property.
func nudgeDraw(item draw, step int64) (draw, bool) {
	if item.forced {
		return draw{}, false
	}
	switch item.kind {
	case "integer":
		return stepDraw(item, step)
	case "boolean":
		stepped, ok := stepDraw(item, step)
		if !ok || stepped.boolean == item.boolean {
			return draw{}, false
		}
		return stepped, true
	case "string":
		if step < 0 {
			return shortenString(item, uint64(len(item.bytes)/4-1))
		}
	}
	return draw{}, false
}

// nextSibling reports the span that should trade places with spans[at], which is
// the next one under the same parent carrying the same label.
//
// Upstream groups a span's children by label and sorts the group; taking only
// the *next* member makes each request one adjacent swap, and repeating it until
// nothing moves sorts the group the same way. A discarded span is never a
// member: its choices are a rejected candidate's debris, so its position carries
// no meaning to canonicalize.
func nextSibling(spans []ChoiceSpan, at int) (int, bool) {
	span := spans[at]
	if span.Discarded || span.End <= span.Start {
		return 0, false
	}
	for other := at + 1; other < len(spans); other++ {
		candidate := spans[other]
		if candidate.Discarded || candidate.Parent != span.Parent ||
			candidate.Label != span.Label {
			continue
		}
		if candidate.End <= candidate.Start || candidate.Start < span.End {
			continue
		}
		return other, true
	}
	return 0, false
}

// pairIndex reports where a draw sits in its schedule, for the pair pass.
//
// Upstream's lower_integers_together takes integer nodes only, and so does
// this: the shift it makes is meaningful in value space precisely because an
// integer's index is its distance from its own shrink target. A byte string's
// index is a shortlex position, and shifting two of those by a shared amount
// preserves nothing a property could depend on.
func pairIndex(item draw) (uint64, bool) {
	if item.kind != "integer" {
		return 0, false
	}
	return drawIndex(item)
}

func saturatedUint(value []byte) (uint64, bool) {
	if index, ok := positiveUint(value); ok {
		return index, true
	}
	canonical := canonicalizeSignedLe(value)
	if canonical[len(canonical)-1]&0x80 != 0 {
		return 0, false
	}
	return ^uint64(0), true
}

func compareDrawIndexes(left, right []byte) int {
	return int(signedCompareLe(left, right))
}

// drawsSimpler implements choice-count-first shortlex.
func drawsSimpler(candidate, current []draw) bool {
	return drawsCompare(candidate, current) < 0
}

// drawsCompare orders two draw lists: negative when candidate is the simpler
// of the two, positive when it is not, zero when neither is.
//
// The count that decides first is the *flattened* one, so a candidate that
// removes a choice from inside a clone is simpler even though the parent's own
// entry count has not moved — without that, a nested shrink could never make
// progress the accept rule would keep. A clone is then compared by its own
// stream, recursively, which is the same rule one level down.
func drawsCompare(candidate, current []draw) int {
	if left, right := flatDrawCount(candidate), flatDrawCount(current); left != right {
		if left < right {
			return -1
		}
		return 1
	}
	if len(candidate) != len(current) {
		if len(candidate) < len(current) {
			return -1
		}
		return 1
	}
	for i := range candidate {
		if candidate[i].child != nil || current[i].child != nil {
			if candidate[i].child == nil || current[i].child == nil {
				return 1
			}
			order := drawsCompare(candidate[i].child.draws, current[i].child.draws)
			if order != 0 {
				return order
			}
			continue
		}
		candidateIndex, candidateOK := drawSortIndex(candidate[i])
		currentIndex, currentOK := drawSortIndex(current[i])
		if !candidateOK || !currentOK {
			return 1
		}
		if order := compareDrawIndexes(candidateIndex, currentIndex); order != 0 {
			return order
		}
	}
	return 0
}

// drawAtIndex rebuilds a draw at a schedule index, or reports a miss.
func drawAtIndex(item draw, index uint64) (draw, bool) {
	switch item.kind {
	case "boolean":
		value, ok := booleanFromIndex(positiveBytes(index))
		if !ok {
			return draw{}, false
		}
		candidate := item
		candidate.boolean = value
		candidate.encoded = serializeBooleanChoice(value)
		return candidate, true
	case "integer":
		minValue, maxValue, target := item.params[0], item.params[1], item.params[2]
		valueBytes, ok := integerFromIndex(
			signedBytes(minValue), signedBytes(maxValue), signedBytes(target),
			positiveBytes(index),
		)
		if !ok {
			return draw{}, false
		}
		value, fits := signedInt(valueBytes)
		if !fits {
			return draw{}, false
		}
		candidate := item
		candidate.integer = value
		candidate.encoded = serializeIntegerChoice(valueBytes)
		return candidate, true
	case "bytes":
		minSize, maxSize := item.params[0], item.params[1]
		value, ok := bytesFromIndex(minSize, maxSize, positiveBytes(index))
		if !ok {
			return draw{}, false
		}
		candidate := item
		candidate.bytes = value
		candidate.encoded = serializeBytesChoice(value)
		return candidate, true
	case "string":
		minSize, maxSize := item.params[0], item.params[1]
		value, ok := stringFromIndex(item.alphabet, minSize, maxSize, positiveBytes(index))
		if !ok {
			return draw{}, false
		}
		candidate := item
		candidate.bytes = value
		candidate.encoded = serializeStringChoicePacked(uint64(len(value)/4), value)
		return candidate, true
	case "float":
		value, ok := floatChoiceFromIndex(
			item.floats[0], item.floats[1], item.flags[0], item.flags[1],
			item.floats[2], positiveBytes(index),
		)
		if !ok {
			return draw{}, false
		}
		candidate := item
		candidate.float = value
		candidate.encoded = serializeFloatChoice(value)
		return candidate, true
	}
	return draw{}, false
}

// candidateDraws lists every shrink candidate for a draw, in schedule order.
func candidateDraws(item draw) []draw {
	current, ok := drawIndex(item)
	if !ok {
		return nil
	}
	var out []draw
	for ordinal := uint64(0); ; ordinal++ {
		index, more := shrinkCandidateAt(current, ordinal)
		if !more {
			return out
		}
		if candidate, legal := drawAtIndex(item, index); legal {
			out = append(out, candidate)
		}
	}
}

// integerExtraDraw answers an integer draw at the ordinal-th extra lowering
// candidate, or false once there are none. The index schedule bisects, which
// finds the edge of an interval; a sparse failing set such as the positive
// multiples of 1000 is not one, and from 10^9 the bisection stops at 875000.
// The divisions and trailing-zeros moves that carry on from there are compiled
// (ir/choices.zir integer_shrink_extra); this only re-encodes the value.
func integerExtraDraw(item draw, ordinal uint64) (draw, bool) {
	if item.kind != "integer" || item.forced {
		return draw{}, false
	}
	valueBytes, ok := integerShrinkExtra(
		signedBytes(item.params[0]), signedBytes(item.params[1]), signedBytes(item.params[2]),
		signedBytes(item.integer), ordinal,
	)
	if !ok {
		return draw{}, false
	}
	value, fits := signedInt(valueBytes)
	if !fits {
		return draw{}, false
	}
	item.integer = value
	item.encoded = serializeIntegerChoice(valueBytes)
	return item, true
}

// Request kinds and feedback codes are defined by ir/lifecycle.zir.
const (
	requestDone uint64 = iota
	requestBaseline
	requestDeleteSpan
	requestLower
	requestVerify
	requestLowerDuplicates
	requestLowerPair
	requestReorder
	requestTrivialSpan
	requestDescendant
	requestLowerCharacters
	requestShortenString
	requestDeletePaid
	requestRaise
)

const (
	feedbackSkip uint64 = iota
	feedbackRejected
	feedbackAccepted
)

// shrinkRun is one execution of a candidate: the whole family it produced, and
// the case whose draws the candidate stream became. The two differ only when
// the level being shrunk is a clone's stream rather than the family's own.
type shrinkRun struct {
	family  *TestCase
	subject *TestCase
	err     error
}

// A shrinkLevel is one stream the driver can minimize. The driver rewrites
// this level's choices; the level knows how to put them back into the family
// and read the result out again.
type shrinkLevel struct {
	run func(candidate []byte) shrinkRun
}

// familyLevel shrinks the stream the property was handed.
func familyLevel(property Property) shrinkLevel {
	return shrinkLevel{run: func(candidate []byte) shrinkRun {
		tc := newReplayCase(candidate)
		err := execute(property, tc)
		return shrinkRun{family: tc, subject: tc, err: err}
	}}
}

// shrinkStream serves the generated shrink schedule over one level until it
// reports done.
//
// The search order lives in the core; this loop only runs the property, maps
// the host's outcome onto a feedback code, and turns a request into a
// concrete candidate. Zero caps leave the search unbounded. The last result is
// how many shrinks the sweep accepted, which is what its cap counts.
func shrinkStream(
	property Property,
	level shrinkLevel,
	family *TestCase,
	subject *TestCase,
	origin string,
	maxShrinks uint64,
	maxStall uint64,
) (error, *TestCase, *TestCase, bool, uint64) {
	draws := append([]draw(nil), subject.draws...)
	spans := append([]ChoiceSpan(nil), subject.spans...)
	familyDraws := append([]draw(nil), family.draws...)
	cost := family.totalCost()
	best, bestSubject := family, subject
	accepted := false
	var verifiedErr error
	var verifiedCase *TestCase
	scheduleAt := -1
	scheduleIndex := uint64(0)
	scheduleOK := false
	scheduleCount := uint64(0)

	// The accept rule is about the whole example, not this level of it: a
	// candidate is kept when the family still fails and the family's cost and
	// choice stream have not got worse. Inside a clone that reduces to the
	// child's own comparison, since nothing outside it moved.
	adopt := func(run shrinkRun) {
		// accepted means *improved*, not merely adopted: the baseline replays
		// the example the caller already had, and a caller that treated that as
		// progress would descend into the same clone forever.
		if run.family.totalCost() < cost || drawsCompare(run.family.draws, familyDraws) < 0 {
			accepted = true
		}
		draws = append([]draw(nil), run.subject.draws...)
		spans = append([]ChoiceSpan(nil), run.subject.spans...)
		familyDraws = append([]draw(nil), run.family.draws...)
		cost = run.family.totalCost()
		best, bestSubject = run.family, run.subject
		scheduleAt = -1
	}

	var phase, spanCursor, duplicateCursor, position, pairCursor uint64
	var reorderCursor, trivialCursor, descendantCursor, characterCursor uint64
	var shortenCursor, paidCursor, raiseCursor, ordinal, shrinks, stall uint64
	for {
		kind := shrinkRequestKind(
			phase, spanCursor, duplicateCursor, position, pairCursor, reorderCursor,
			trivialCursor, descendantCursor, characterCursor, shortenCursor, paidCursor,
			raiseCursor, uint64(len(draws)), uint64(len(spans)),
		)
		if kind == requestDone {
			break
		}

		feedback := feedbackSkip
		switch kind {
		case requestBaseline:
			// The baseline replays a stream that has already failed — the
			// example as found, or a candidate that failed when it was adopted
			// — so one that does not fail now is the property, not the search.
			run := level.run(serializeDraws(draws))
			if !isFailure(run.err, origin) {
				return errFlakyReplay, nil, nil, accepted, shrinks
			}
			adopt(run)
			feedback = feedbackAccepted
		case requestDeleteSpan:
			span := spans[shrinkRequestSpan(spanCursor, uint64(len(spans)))]
			var candidate []byte
			var ok bool
			if span.End > span.Start {
				candidate, ok = choiceDeleteRange(
					serializeDraws(draws), uint64(span.Start), uint64(span.End),
				)
			}
			if ok {
				run := level.run(candidate)
				if isFailure(run.err, origin) && shrinkCostProgress(
					run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
				) {
					adopt(run)
					feedback = feedbackAccepted
				} else {
					feedback = feedbackRejected
				}
			}
		case requestDeletePaid:
			// A deletion the plain pass could not make, paid for by moving the
			// draw after the enclosing structure one step: a list's declared
			// length, an index into it, a parity flag drawn after it.
			at := int(shrinkRequestSpan(paidCursor, uint64(len(spans))))
			start, end, widened, ok := paidExtent(draws, spans, at)
			follower, paid := 0, false
			step, stepped := int64(0), false
			if ok {
				follower, paid = paidFollower(spans, at, end, len(draws))
				if paid || widened {
					step, stepped = shrinkDeletePaidVariant(ordinal, widened)
				}
			}
			if stepped {
				trial := append(append([]draw(nil), draws[:start]...), draws[end:]...)
				legal := true
				if step != 0 {
					var nudged draw
					if paid {
						nudged, legal = nudgeDraw(draws[follower], step)
					} else {
						legal = false
					}
					if legal {
						trial[follower-(end-start)] = nudged
					}
				}
				if !legal {
					// This draw cannot take this step; try the next one without
					// running the property.
					feedback = feedbackRejected
				} else {
					run := level.run(serializeDraws(trial))
					if isFailure(run.err, origin) && shrinkCostProgress(
						run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
					) {
						adopt(run)
						feedback = feedbackAccepted
					} else {
						feedback = feedbackRejected
					}
				}
			}
		case requestLower:
			if int(position) != scheduleAt {
				scheduleAt = int(position)
				scheduleIndex, scheduleOK = drawIndex(draws[position])
				scheduleCount = 0
				if scheduleOK {
					scheduleCount = shrinkCandidateCount(scheduleIndex)
				}
			}
			var candidate draw
			more, legal := false, false
			if scheduleOK {
				if index, scheduled := shrinkCandidateAt(scheduleIndex, ordinal); scheduled {
					candidate, legal = drawAtIndex(draws[position], index)
					more = true
				} else {
					// Past the index schedule an integer still has the
					// value-space moves bisection cannot make.
					candidate, more = integerExtraDraw(draws[position], ordinal-scheduleCount)
					legal = more
				}
			}
			if more {
				if !legal {
					// No legal value here; step the schedule without running.
					feedback = feedbackRejected
				} else {
					trial := append([]draw(nil), draws...)
					trial[position] = candidate
					run := level.run(serializeDraws(trial))
					if isFailure(run.err, origin) && shrinkCostProgress(
						run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
					) {
						adopt(run)
						feedback = feedbackAccepted
					} else {
						feedback = feedbackRejected
					}
				}
			}
		case requestLowerDuplicates:
			// Every draw that currently sits where this one does, lowered as one.
			// Neither member can move alone when the property depends on them
			// being equal: dropping either breaks the failure, so the per-choice
			// pass rejects every candidate and leaves the pair where it was.
			kindName, groupIndex, indexed := drawGroupKey(draws[duplicateCursor])
			members := []uint64{}
			if indexed {
				for at, item := range draws {
					if name, index, ok := drawGroupKey(item); ok &&
						name == kindName && index == groupIndex {
						members = append(members, uint64(at))
					}
				}
			}
			// One position drives each group, so a group of n is tried once and
			// not n times.
			leads := len(members) > 0 && members[0] == duplicateCursor
			index, more := uint64(0), false
			if leads && shrinkDuplicateWorthwhile(uint64(len(members)), groupIndex) {
				index, more = shrinkCandidateAt(groupIndex, ordinal)
			}
			if more {
				trial := append([]draw(nil), draws...)
				missed := false
				for _, at := range members {
					candidate, legal := drawAtIndex(draws[at], index)
					if !legal {
						missed = true
						break
					}
					trial[at] = candidate
				}
				if missed {
					// No legal value for some member; step the schedule without
					// running the property.
					feedback = feedbackRejected
				} else {
					run := level.run(serializeDraws(trial))
					if isFailure(run.err, origin) && shrinkCostProgress(
						run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
					) {
						adopt(run)
						feedback = feedbackAccepted
					} else {
						feedback = feedbackRejected
					}
				}
			}
		case requestLowerPair:
			// Two adjacent integer draws, lowered by one shared amount. This is
			// the difference-preserving move: a property that fails on x-y==5
			// rejects every candidate the per-choice pass offers, because
			// lowering either member alone changes the difference.
			left, right := draws[pairCursor], draws[pairCursor+1]
			leftIndex, leftOK := pairIndex(left)
			rightIndex, rightOK := pairIndex(right)
			index, more := uint64(0), false
			base := uint64(0)
			if leftOK && rightOK && shrinkPairWorthwhile(leftIndex, rightIndex) {
				base = shrinkPairBase(leftIndex, rightIndex)
				index, more = shrinkCandidateAt(base, ordinal)
			}
			if more {
				trial := append([]draw(nil), draws...)
				missed := false
				for offset, item := range []draw{left, right} {
					at := pairCursor + uint64(offset)
					current := leftIndex
					if offset == 1 {
						current = rightIndex
					}
					shifted, ok := shrinkPairCandidate(current, base, index)
					if !ok {
						missed = true
						break
					}
					candidate, legal := drawAtIndex(item, shifted)
					if !legal {
						missed = true
						break
					}
					trial[at] = candidate
				}
				if missed {
					// No legal value for one member; step the schedule without
					// running the property.
					feedback = feedbackRejected
				} else {
					run := level.run(serializeDraws(trial))
					if isFailure(run.err, origin) && shrinkCostProgress(
						run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
					) {
						adopt(run)
						feedback = feedbackAccepted
					} else {
						feedback = feedbackRejected
					}
				}
			}
		case requestTrivialSpan:
			// A whole span at its simplest values, in one execution.
			span := spans[trivialCursor]
			if !span.Discarded && span.End > span.Start {
				trial, rewritable := trivialSpanCandidate(draws, span)
				if shrinkTrivialWorthwhile(rewritable) {
					run := level.run(serializeDraws(trial))
					if isFailure(run.err, origin) && shrinkCostProgress(
						run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
					) {
						adopt(run)
						feedback = feedbackAccepted
					} else {
						feedback = feedbackRejected
					}
				}
			}
		case requestDescendant:
			// A span replaced by one of its own descendants. Deletion removes a
			// whole subtree; this keeps the subtree and discards the structure
			// wrapped around it, which is the move a nested failure needs when
			// the interesting part is buried rather than beside something.
			if inner, ok := descendantSpan(spans, int(descendantCursor), ordinal); ok {
				outer := spans[descendantCursor]
				stream := serializeDraws(draws)
				// Two deletions rather than a new primitive: drop the tail after
				// the descendant, then the head before it. Trailing first, so
				// the leading offsets are still the ones measured above.
				after, okAfter := choiceDeleteRange(stream, uint64(inner.End), uint64(outer.End))
				if okAfter {
					candidate, okBefore := choiceDeleteRange(
						after, uint64(outer.Start), uint64(inner.Start),
					)
					if okBefore {
						run := level.run(candidate)
						if isFailure(run.err, origin) && shrinkCostProgress(
							run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
						) {
							adopt(run)
							feedback = feedbackAccepted
						} else {
							feedback = feedbackRejected
						}
					}
				}
			}
		case requestReorder:
			// Two sibling spans trade places. Nothing about the example changes
			// except which order its parts were drawn in, so this reports a
			// canonical arrangement rather than the one generation stumbled on.
			if sibling, ok := nextSibling(spans, int(reorderCursor)); ok {
				first, second := spans[reorderCursor], spans[sibling]
				candidate, swapped := choiceSwapRanges(
					serializeDraws(draws),
					uint64(first.Start), uint64(first.End),
					uint64(second.Start), uint64(second.End),
				)
				if swapped {
					run := level.run(candidate)
					if isFailure(run.err, origin) && shrinkCostProgress(
						run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
					) {
						adopt(run)
						feedback = feedbackAccepted
					} else {
						feedback = feedbackRejected
					}
				}
			}
		case requestShortenString:
			// One character removed from a string draw.
			if candidate, ok := shortenString(draws[shortenCursor], ordinal); ok {
				trial := append([]draw(nil), draws...)
				trial[shortenCursor] = candidate
				run := level.run(serializeDraws(trial))
				if isFailure(run.err, origin) && shrinkCostProgress(
					run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
				) {
					adopt(run)
					feedback = feedbackAccepted
				} else {
					feedback = feedbackRejected
				}
			}
		case requestLowerCharacters:
			// Every copy of one repeated character in a string draw, lowered
			// together.
			if candidate, ok := lowerCharacters(draws[characterCursor], ordinal); ok {
				trial := append([]draw(nil), draws...)
				trial[characterCursor] = candidate
				run := level.run(serializeDraws(trial))
				if isFailure(run.err, origin) && shrinkCostProgress(
					run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
				) {
					adopt(run)
					feedback = feedbackAccepted
				} else {
					feedback = feedbackRejected
				}
			}
		case requestRaise:
			// One draw made *less* simple, kept only when the run that comes back
			// is smaller: a gate past its threshold, so the draws behind it are
			// never made. The candidate is larger as written, so the accept rule
			// judges the executed run alone.
			if current, largest, ok := raiseBounds(draws[raiseCursor]); ok {
				if index, raised := shrinkRaiseCandidate(current, largest, ordinal); raised {
					candidate, legal := drawAtIndex(draws[raiseCursor], index)
					if !legal {
						// No legal value there; try the next raise without running.
						feedback = feedbackRejected
					} else {
						trial := append([]draw(nil), draws...)
						trial[raiseCursor] = candidate
						run := level.run(serializeDraws(trial))
						if isFailure(run.err, origin) && shrinkCostProgress(
							run.family.totalCost(), cost, drawsSimpler(run.family.draws, familyDraws),
						) {
							adopt(run)
							feedback = feedbackAccepted
						} else {
							feedback = feedbackRejected
						}
					}
				}
			}
		case requestVerify:
			run := level.run(serializeDraws(draws))
			if !isFailure(run.err, origin) {
				return errFlakyReplay, nil, nil, accepted, shrinks
			}
			verifiedErr, verifiedCase = run.err, run.family
			best, bestSubject = run.family, run.subject
			feedback = feedbackAccepted
		}

		phase, spanCursor, duplicateCursor, position, pairCursor,
			reorderCursor, trivialCursor, descendantCursor, characterCursor,
			shortenCursor, paidCursor, raiseCursor, ordinal, shrinks, stall =
			shrinkNextPhase(
				phase, feedback, spanCursor, duplicateCursor, position, pairCursor,
				reorderCursor, trivialCursor, descendantCursor, characterCursor, shortenCursor,
				paidCursor, raiseCursor, shrinks, stall,
				maxShrinks, maxStall, uint64(len(draws)), uint64(len(spans)),
			),
			shrinkNextSpanCursor(phase, feedback, spanCursor),
			shrinkNextDuplicateCursor(phase, feedback, duplicateCursor),
			shrinkNextPosition(phase, feedback, position),
			shrinkNextPairCursor(phase, feedback, pairCursor),
			shrinkNextReorderCursor(phase, feedback, reorderCursor),
			shrinkNextTrivialCursor(phase, feedback, trivialCursor),
			shrinkNextDescendantCursor(phase, feedback, descendantCursor),
			shrinkNextCharacterCursor(phase, feedback, characterCursor),
			shrinkNextShortenCursor(phase, feedback, shortenCursor),
			shrinkNextPaidCursor(phase, feedback, paidCursor),
			shrinkNextRaiseCursor(phase, feedback, raiseCursor),
			shrinkNextOrdinal(phase, feedback, ordinal),
			shrinkNextShrinks(phase, feedback, shrinks),
			shrinkNextStall(stall, feedback)
	}

	if verifiedCase == nil {
		return errors.New("internal error: shrink finished without verifying"),
			best, bestSubject, accepted, shrinks
	}
	return verifiedErr, best, bestSubject, accepted, shrinks
}

// clonePaths lists every clone position in a family, outermost first, as the
// index path from the family's own stream down to the clone.
//
// A path rather than a pointer because each candidate re-runs the property and
// so builds a whole new family: the case a path names survives that, a pointer
// into the previous run does not.
func clonePaths(draws []draw, prefix []int) [][]int {
	var paths [][]int
	for at, item := range draws {
		if item.child == nil {
			continue
		}
		path := append(append([]int(nil), prefix...), at)
		paths = append(paths, path)
		paths = append(paths, clonePaths(item.child.draws, path)...)
	}
	return paths
}

// caseAt walks a path from a family to the clone stream it names.
func caseAt(family *TestCase, path []int) (*TestCase, bool) {
	at := family
	for _, index := range path {
		if index >= len(at.draws) || at.draws[index].child == nil {
			return nil, false
		}
		at = at.draws[index].child
	}
	return at, true
}

// spliceAt rebuilds stream with the clone named by path carrying inner.
//
// This is the extract-and-splice the structural core is built around
// (ir/structure.zir): the bytes inside a clone entry are already a well-formed
// choice stream, so descending is one extraction per level and the recursion
// lives here rather than in the IR, which forbids it.
func spliceAt(stream []byte, path []int, inner []byte) ([]byte, bool) {
	offset, ok := choiceOffsetAt(stream, uint64(path[0]))
	if !ok {
		return nil, false
	}
	if len(path) > 1 {
		nested, present := cloneInnerStream(stream, offset)
		if !present {
			return nil, false
		}
		rebuilt, spliced := spliceAt(nested, path[1:], inner)
		if !spliced {
			return nil, false
		}
		inner = rebuilt
	}
	return cloneReplaceInner(stream, offset, inner)
}

// cloneLevel shrinks the stream inside one clone.
//
// Every candidate is spliced into the family's stream at the clone's position
// and replayed through the whole property, so what is judged is the family's
// outcome — the child's choices matter only through the failure they still
// produce. The realized child stream is then read back out of that run.
func cloneLevel(property Property, family *TestCase, path []int) shrinkLevel {
	stream := serializeDraws(family.draws)
	return shrinkLevel{run: func(candidate []byte) shrinkRun {
		spliced, ok := spliceAt(stream, path, candidate)
		if !ok {
			return shrinkRun{family: family, subject: family, err: nil}
		}
		tc := newReplayCase(spliced)
		err := execute(property, tc)
		subject, present := caseAt(tc, path)
		if !present {
			// The candidate changed the shape of the run: whatever it is, it is
			// not this clone's stream any more, so there is nothing to adopt.
			return shrinkRun{family: tc, subject: tc, err: nil}
		}
		return shrinkRun{family: tc, subject: subject, err: err}
	}}
}

// shrinkCloneStreams runs the whole schedule inside each clone of a family.
//
// The flat passes treat a clone entry as one atomic choice — they can delete it
// or move it, never reach the choices inside it — so a failure that depends on
// a value drawn from a clone would otherwise be reported exactly as generation
// found it. Each clone gets its own run of the same compiled driver over its
// own stream, which is upstream's shrink_clone_streams.
func shrinkCloneStreams(
	property Property,
	family *TestCase,
	origin string,
	maxShrinks uint64,
	maxStall uint64,
) (*TestCase, bool, error) {
	improved := false
	for _, path := range clonePaths(family.draws, nil) {
		subject, present := caseAt(family, path)
		if !present || len(subject.draws) == 0 {
			continue
		}
		err, best, _, accepted, _ := shrinkStream(
			property, cloneLevel(property, family, path),
			family, subject, origin, maxShrinks, maxStall,
		)
		if err == errFlakyReplay {
			return family, improved, err
		}
		if accepted && best != nil {
			family = best
			improved = true
		}
	}
	return family, improved, nil
}

// shrink minimizes a failing example: the family's own stream first, then the
// stream inside each of its clones, to a fixpoint.
//
// The two alternate because they feed each other — a deletion at the top can
// expose a clone worth descending into, and a smaller child stream can make a
// parent candidate legal that was not before. Each accepted candidate is
// strictly simpler than the one it replaces, so the alternation terminates.
func shrink(
	property Property,
	initial *TestCase,
	origin string,
	maxShrinks uint64,
	maxStall uint64,
) (error, *TestCase) {
	level := familyLevel(property)
	for {
		verifiedErr, best, _, accepted, shrinks := shrinkStream(
			property, level, initial, initial, origin, maxShrinks, maxStall,
		)
		if verifiedErr == nil || best == nil {
			return verifiedErr, best
		}
		next, improved, err := shrinkCloneStreams(property, best, origin, maxShrinks, maxStall)
		if err != nil {
			return err, nil
		}
		// A sweep that improved the stream goes round again even when no clone
		// did, unless it stopped at its shrink cap: a later pass's accept can
		// make an earlier pass's candidate legal, and one sweep never revisits
		// it. A state machine needs this most, because a continue decision
		// misread after a deletion stops the run, so lowering a rule is often
		// what lets a whole step go.
		if !improved && !shrinkSweepAgain(accepted, shrinks, maxShrinks) {
			return verifiedErr, best
		}
		initial = next
	}
}

// Check runs a property, shrinks its first failure, and returns a Failure.
func Check(property Property, config Config) error {
	if config.Reproduce != "" {
		raw, ok := rawFailureDecode([]byte(config.Reproduce))
		if !ok {
			return &InvalidReplay{Message: "invalid or unsupported reproduce blob"}
		}
		tc := newReplayCase(raw)
		err := execute(property, tc)
		if err == nil {
			return &InvalidReplay{Message: "reproduced example did not fail"}
		}
		if errors.Is(err, errAssumptionRejected) {
			return &InvalidReplay{Message: "reproduced example was rejected by an assumption"}
		}
		var overrun *ReplayOverrun
		if errors.As(err, &overrun) {
			return overrun
		}
		var replayErr *InvalidReplay
		if errors.As(err, &replayErr) {
			return replayErr
		}
		return &Failure{
			Blob:        config.Reproduce,
			Cause:       err,
			ExamplesRun: 1,
			Notes:       tc.Notes(),
			Puns:        tc.puns,
			KeyMisses:   tc.keyMisses,
			Costs:       tc.Costs(),
		}
	}
	examples := config.Examples
	if examples == 0 {
		examples = 100
	}
	if examples < 0 {
		return errors.New("examples must be non-negative")
	}
	maxRejections := config.MaxRejections
	if maxRejections == 0 {
		maxRejections = 458
	}
	if maxRejections < 0 {
		return errors.New("max rejections must be non-negative")
	}
	maxShrinks := config.MaxShrinks
	if maxShrinks == 0 {
		maxShrinks = 500
	}
	maxStall := config.MaxStall
	if maxStall == 0 {
		maxStall = 200
	}
	shrinkLimit := uint64(0)
	if maxShrinks > 0 {
		shrinkLimit = uint64(maxShrinks)
	}
	stallLimit := uint64(0)
	if maxStall > 0 {
		stallLimit = uint64(maxStall)
	}
	var coverage []*coverageCounter
	coverageIndex := map[string]int{}
	for _, required := range config.Require {
		if !(required.Percent >= 0 && required.Percent <= 100) {
			return errors.New("required coverage percent must be between 0 and 100")
		}
		permille := uint64(required.Percent * 10)
		if at, seen := coverageIndex[required.Label]; seen {
			if permille > coverage[at].permille {
				coverage[at].permille = permille
			}
			continue
		}
		coverageIndex[required.Label] = len(coverage)
		coverage = append(coverage, &coverageCounter{
			label:    required.Label,
			permille: permille,
		})
	}
	database := config.Database
	databaseKey := config.DatabaseKey
	if database != nil && databaseKey == "" {
		databaseKey = derivedDatabaseKey(property)
	}
	rng := rand.New(rand.NewSource(config.Seed))
	valid := 0
	attempts := 0
	// Database replays are attempts but not generation. The schedule is told
	// generation's own count, so a stale entry cannot take the all-simplest
	// probe's place as the first request.
	replays := 0
	rejections := 0
	overruns := 0
	generationMicros := uint64(0)
	var pendingMutations [][]byte
	// Consecutive executed mutation requests; the schedule caps it at one
	// brood so fresh random generation keeps its share of the run.
	mutationStreak := uint64(0)
	choiceTree := &choiceTree{}
	// Per-label best score and the stream that produced it, for the targeting
	// hill climber. The schedule, budgets, record and accept rules, stepping,
	// and the step-size search are all compiled (ir/targeting.zir); this loop
	// holds the maps and runs the trials.
	type targetEntry struct {
		score  float64
		stream []byte
	}
	targetingBest := map[string]targetEntry{}
	targetingStep := targetingScheduleStep(uint64(examples))
	targetingNextAt := targetingStep
	targetingBudget := targetingTrialBudget(uint64(examples))

	// recordTargets folds a valid example's observations into the per-label
	// bests.
	recordTargets := func(tc *TestCase) {
		for label, score := range tc.root().targets {
			entry, seen := targetingBest[label]
			if targetingRecord(seen, entry.score, score) {
				targetingBest[label] = targetEntry{score: score, stream: tc.serialized()}
			}
		}
	}

	// counterexample shrinks a failing case, stores what it shrank to, and
	// builds the failure to return. A replay during the search that stops
	// failing is reported as Flaky, carrying the failure as found rather than a
	// shrunk one that cannot be trusted; nothing is stored or collected for it.
	counterexample := func(tc *TestCase, err error, replaced []byte) error {
		shrunkError, shrunk := shrink(
			property,
			tc,
			interestingOrigin(err),
			shrinkLimit,
			stallLimit,
		)
		if shrunkError == errFlakyReplay {
			return &Flaky{
				Blob:        rawFailureEncode(tc.serialized()),
				Cause:       err,
				ExamplesRun: attempts,
				Rejections:  rejections,
			}
		}
		stream := shrunk.serialized()
		if database != nil {
			database.Save(databaseKey, stream)
			// The entry that reproduced is superseded by whatever it shrank
			// to, so it goes; an entry that shrank to itself is already
			// stored.
			if replaced != nil && !bytes.Equal(replaced, stream) {
				database.Delete(databaseKey, replaced)
			}
		}
		return &Failure{
			Blob:        rawFailureEncode(stream),
			Cause:       shrunkError,
			ExamplesRun: attempts,
			Rejections:  rejections,
			Notes:       shrunk.Notes(),
			Puns:        shrunk.puns,
			KeyMisses:   shrunk.keyMisses,
			Costs:       shrunk.Costs(),
		}
	}

	// targetingTrial runs one climb trial: replay a stepped stream with
	// fresh-RNG extension — the execution mode mutation probes already use —
	// counting it as an ordinary attempt. Answers done=false when the
	// firing's budget is spent.
	targetingTrial := func(stream []byte) (error, *TestCase, uint64, bool) {
		if uint64(valid) >= uint64(examples) || uint64(attempts) >= targetingBudget {
			return nil, nil, 0, false
		}
		attempts++
		tc := newProbeCase(stream, rng)
		err := execute(property, tc)
		choiceTree.insert(tc.serialized())
		if err == nil {
			valid++
			recordTargets(tc)
		}
		return err, tc, classify(err), true
	}

	// targetingClimb hill-climbs one label's best-known stream: walk its
	// realized draws from the end backwards, stepping each climbable draw
	// through the compiled step-size search, in the +k direction and then -k.
	// A failing trial comes back as interesting for the ordinary
	// counterexample path. Mirrors hegel-c/src/native/targeting.rs::hill_climb.
	targetingClimb := func(label string, allowance uint64) (uint64, error, *TestCase) {
		err, current, outcome, ran := targetingTrial(targetingBest[label].stream)
		if !ran {
			return 0, nil, nil
		}
		if outcome == outcomeInteresting {
			return 0, err, current
		}
		if err != nil {
			return 0, nil, nil
		}
		currentScore := math.Inf(-1)
		if score, seen := current.root().targets[label]; seen {
			currentScore = score
		}
		improvements := uint64(0)
		examined := map[int]bool{}
		at := len(current.draws) - 1
		previousLength := len(current.draws)
		for at >= 0 && improvements <= allowance {
			if len(current.draws) != previousLength {
				at = len(current.draws) - 1
				previousLength = len(current.draws)
				continue
			}
			if examined[at] {
				at--
				continue
			}
			examined[at] = true
			if climbableDraw(current.draws[at]) {
				lengthBefore := len(current.draws)
				for _, direction := range []int64{1, -1} {
					// The -k direction only makes sense while the draw
					// survived the +k climb at the same index with the stream
					// length unchanged.
					if direction == -1 &&
						(at >= len(current.draws) || len(current.draws) != lengthBefore) {
						break
					}
					mode, a, b := uint64(0), uint64(1), uint64(0)
					for {
						// A committed improvement may have shortened the
						// stream below this index; the outer walk restarts
						// from the new end.
						if at >= len(current.draws) {
							break
						}
						probe, more := findIntegerProbe(mode, a, b)
						if !more {
							break
						}
						committed := false
						if probe <= targetingStepCap() {
							if stepped, ok := stepDraw(current.draws[at], direction*int64(probe)); ok {
								draws := append([]draw(nil), current.draws...)
								draws[at] = stepped
								trialErr, trial, trialOutcome, ran := targetingTrial(serializeDraws(draws))
								if !ran {
									break
								}
								if trialOutcome == outcomeInteresting {
									return improvements, trialErr, trial
								}
								if trialErr == nil {
									trialScore := math.Inf(-1)
									if score, seen := trial.root().targets[label]; seen {
										trialScore = score
									}
									accept := targetingAccept(
										trialScore, currentScore,
										flatDrawCount(trial.draws), flatDrawCount(current.draws),
									)
									if accept != 0 {
										current = trial
										currentScore = trialScore
										if accept == 2 {
											improvements++
										}
										committed = true
									}
								}
							}
						}
						mode, a, b = findIntegerNextMode(mode, a, b, committed),
							findIntegerNextA(mode, a, b, committed),
							findIntegerNextB(mode, a, b, committed)
					}
				}
			}
			at--
		}
		return improvements, nil, nil
	}

	// optimiseTargets climbs every label until a full round improves nothing
	// or the budget is spent. Mirrors targeting.rs::optimise_targets, with
	// the improvement allowance doubling each round.
	optimiseTargets := func() (error, *TestCase) {
		round := uint64(0)
		for {
			callsBefore := attempts
			anyImprovements := false
			allowance := targetingImprovementAllowance(round)
			labels := make([]string, 0, len(targetingBest))
			for label := range targetingBest {
				labels = append(labels, label)
			}
			sort.Strings(labels)
			for _, label := range labels {
				improvements, err, tc := targetingClimb(label, allowance)
				if tc != nil {
					return err, tc
				}
				if improvements > 0 {
					anyImprovements = true
				}
			}
			round++
			if !anyImprovements || attempts == callsBefore {
				return nil, nil
			}
		}
	}

	// Reuse before generate: replay what earlier runs stored under this name,
	// simplest first. An entry that still fails is the answer; one that does
	// not is a bug that was fixed, and collecting it now keeps every future
	// run from paying for it. Replays are counted as attempts but not as valid
	// examples, so a stored corpus never eats the exploration budget, and they
	// feed neither the coverage verdict nor the health checks — both are
	// statements about what this run explored.
	if database != nil {
		stored := database.Fetch(databaseKey)
		sortDatabaseEntries(stored)
		for _, entry := range stored {
			if !choiceStreamValidate(entry) {
				database.Delete(databaseKey, entry)
				continue
			}
			attempts++
			replays++
			tc := newReplayCase(entry)
			err := execute(property, tc)
			if classify(err) != outcomeInteresting {
				database.Delete(databaseKey, entry)
				continue
			}
			return counterexample(tc, err, entry)
		}
	}

	for generationShouldContinue(uint64(valid), uint64(examples)) {
		request := generationRequestKind(uint64(attempts-replays), uint64(len(pendingMutations)), mutationStreak)
		simplestProbe := request == 1
		var mutation []byte
		if request == 2 {
			mutation, pendingMutations = pendingMutations[0], pendingMutations[1:]
		}
		mutationProbe := request == 2
		novelty := generationNoveltyAction(request, mutationProbe && choiceTree.contains(mutation))
		if novelty == 0 {
			continue
		}
		if novelty == 2 {
			choiceTree.insert(mutation)
		}
		attempts++
		if mutationProbe {
			mutationStreak++
		} else {
			mutationStreak = 0
		}
		tc := newSimplestCase()
		if !simplestProbe {
			if mutationProbe {
				tc = newProbeCase(mutation, rng)
			} else {
				tc = newRandomCase(rng)
			}
		}
		err := execute(property, tc)
		choiceTree.insert(tc.serialized())
		if ^uint64(0)-generationMicros < tc.generationMicros {
			generationMicros = ^uint64(0)
		} else {
			generationMicros += tc.generationMicros
		}
		outcome := classify(err)
		var probeOverrun *choiceOverrun
		if simplestProbe && healthLargeBaseExample(
			errors.As(err, &probeOverrun),
			outcome,
			uint64(len(tc.draws)),
			config.SuppressLargeBaseExample,
		) {
			return &FailedHealthCheck{
				Check: "large_base_example",
				Valid: valid,
			}
		}
		healthValid := valid
		if err == nil {
			healthValid++
		}
		if outcome != outcomeInteresting && healthTooSlow(
			generationMicros, uint64(healthValid), config.SuppressTooSlow,
		) {
			return &FailedHealthCheck{
				Check: "too_slow",
				Valid: healthValid,
			}
		}
		if err == nil {
			valid++
			// An example that reached a label the run has barely covered is on
			// the frontier, so its mutation proposals jump the queue ahead of
			// ones bred from the common path, and may be bred at all when the
			// base rule declines.
			heldRare := false
			for _, label := range tc.coverageOrder {
				at, seen := coverageIndex[label]
				if !seen {
					at = len(coverage)
					coverageIndex[label] = at
					coverage = append(coverage, &coverageCounter{label: label})
				}
				counter := coverage[at]
				if permille := tc.coveragePermille[label]; permille > counter.permille {
					counter.permille = permille
				}
				if tc.coverageHeld[label] {
					counter.observed++
					if generationLabelRare(uint64(counter.observed), uint64(valid), counter.permille) {
						heldRare = true
					}
				}
			}
			switch generationMutationAdmission(
				request, uint64(valid), uint64(len(pendingMutations)), heldRare,
			) {
			case 1:
				pendingMutations = append(pendingMutations, spanMutationCandidates(tc.draws, tc.spans, rng)...)
			case 2:
				pendingMutations = append(spanMutationCandidates(tc.draws, tc.spans, rng), pendingMutations...)
			}
			// Targeting: fold this example's observations in, and fire the
			// hill climber when the schedule says so. A failing trial is a
			// counterexample like any other.
			recordTargets(tc)
			if len(targetingBest) > 0 && targetingShouldFire(uint64(valid), targetingNextAt) {
				targetingNextAt = targetingNextThreshold(targetingNextAt, targetingStep)
				if foundErr, found := optimiseTargets(); found != nil {
					return counterexample(found, foundErr, nil)
				}
			}
			continue
		}
		if errors.Is(err, errAssumptionRejected) {
			rejections++
			rejectionLimit := generationRejectionLimit(
				uint64(maxRejections), uint64(valid),
			)
			if uint64(rejections+overruns) > rejectionLimit {
				return &TooManyRejections{
					Rejections: rejections,
					Overruns:   overruns,
					Limit:      int(rejectionLimit),
				}
			}
			if healthFilterTooMuch(
				uint64(rejections), uint64(valid), config.SuppressFilterTooMuch,
			) {
				return &FailedHealthCheck{
					Check:      "filter_too_much",
					Rejections: rejections,
					Valid:      valid,
				}
			}
			continue
		}
		var generatedOverrun *choiceOverrun
		if errors.As(err, &generatedOverrun) {
			overruns++
			rejectionLimit := generationRejectionLimit(
				uint64(maxRejections), uint64(valid),
			)
			if uint64(rejections+overruns) > rejectionLimit {
				return &TooManyRejections{
					Rejections: rejections,
					Overruns:   overruns,
					Limit:      int(rejectionLimit),
				}
			}
			if healthDataTooLarge(
				uint64(overruns), uint64(valid), config.SuppressDataTooLarge,
			) {
				return &FailedHealthCheck{
					Check:    "data_too_large",
					Overruns: overruns,
					Valid:    valid,
				}
			}
			continue
		}
		var replayErr *InvalidReplay
		if errors.As(err, &replayErr) {
			return replayErr
		}
		return counterexample(tc, err, nil)
	}
	// The coverage verdict runs only on a run that completed fully valid: a
	// counterexample or an aborted run reports that instead, and a verdict
	// about a truncated run would be noise. The accept rule is compiled
	// (ir/lifecycle.zir @coverage_failed) so every shell agrees on it.
	for _, counter := range coverage {
		if coverageFailed(uint64(counter.observed), uint64(valid), counter.permille) {
			return &FailedCoverage{
				Label:      counter.label,
				Observed:   counter.observed,
				Valid:      valid,
				MinPercent: float64(counter.permille) / 10,
			}
		}
	}
	return nil
}

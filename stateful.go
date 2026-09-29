package zizek

import "fmt"

// DefaultStatefulStepCount is how many successful rules a stateful case runs
// unless Run is told otherwise.
const DefaultStatefulStepCount uint64 = 50

// statefulStopProbability is the chance a free per-step "stop here?" boolean
// comes up true under the stop protocols (1 and 2), which only a replay of an
// older blob still records, and which a probe extends from the RNG once that
// blob runs out. New cases record a "continue?" boolean instead, at
// statefulContinueProbability().
const statefulStopProbability = 1.0 / 65536.0

// Rule is a named state-machine action or invariant.
type Rule[M any] struct {
	Name  string
	Apply func(*M, *TestCase) error
}

// Machine explicitly registers a state machine's rules and invariants.
type Machine[M any] struct {
	Rules      []Rule[M]
	Invariants []Rule[M]
}

type statefulFeatureFlags struct {
	probability float64
	known       []bool
	disabled    []bool
	candidates  map[int]struct{}
}

func newStatefulFeatureFlags(tc *TestCase, ruleCount int) (*statefulFeatureFlags, error) {
	var raw int64
	var err error
	if tc.replay != nil {
		raw, err = tc.Integer(0, 254, 0)
	} else {
		raw, err = tc.integerForced(int64(statefulFeatureGenerationRaw()))
	}
	if err != nil {
		return nil, err
	}
	candidates := make(map[int]struct{}, ruleCount)
	for index := 0; index < ruleCount; index++ {
		candidates[index] = struct{}{}
	}
	return &statefulFeatureFlags{
		probability: statefulFeatureProbability(uint64(raw)),
		known:       make([]bool, ruleCount),
		disabled:    make([]bool, ruleCount),
		candidates:  candidates,
	}, nil
}

func (flags *statefulFeatureFlags) enabled(tc *TestCase, index int) (bool, error) {
	tc.StartSpan("stateful.feature_flag")
	_, isCandidate := flags.candidates[index]
	forced, isForced := statefulFeatureForced(
		uint64(len(flags.candidates)),
		isCandidate,
		flags.known[index],
		flags.disabled[index],
	)
	var disabled bool
	var err error
	if isForced {
		disabled, err = tc.booleanForced(forced)
	} else {
		disabled, err = tc.Weighted(flags.probability)
	}
	if err != nil {
		tc.StopSpan(false)
		return false, err
	}
	flags.known[index] = true
	flags.disabled[index] = disabled
	if !disabled {
		clear(flags.candidates)
	}
	delete(flags.candidates, index)
	tc.StopSpan(false)
	return !disabled, nil
}

func drawStatefulRuleIndex(tc *TestCase, ruleCount int) (int, error) {
	index, err := tc.Integer(0, int64(ruleCount-1), 0)
	return int(index), err
}

// probeStatefulRule draws one rule index and consults its flag unless the rule
// is already known to be disabled.
//
// A probe — the rule index and the flag it consults — is one span, so the
// shrinker can delete a probe that landed on a disabled rule without
// misaligning the draws after it. A misaligned continue decision reads as a
// stop, so that deletion is the one that reaches the minimal sequence; the span
// tree is not part of the blob.
func probeStatefulRule(
	tc *TestCase,
	ruleCount int,
	flags *statefulFeatureFlags,
	knownBad map[int]struct{},
) (int, bool, error) {
	tc.StartSpan("stateful.rule_probe")
	defer tc.StopSpan(false)
	index, err := drawStatefulRuleIndex(tc, ruleCount)
	if err != nil {
		return 0, false, err
	}
	if _, bad := knownBad[index]; bad {
		return index, false, nil
	}
	enabled, err := flags.enabled(tc, index)
	return index, enabled, err
}

func selectStatefulRule(
	tc *TestCase,
	ruleCount int,
	flags **statefulFeatureFlags,
) (int, error) {
	if *flags == nil {
		created, err := newStatefulFeatureFlags(tc, ruleCount)
		if err != nil {
			return 0, err
		}
		*flags = created
	}
	knownBad := make(map[int]struct{})
	for attempt := uint64(0); attempt < statefulRuleProbeCount(); attempt++ {
		index, enabled, err := probeStatefulRule(tc, ruleCount, *flags, knownBad)
		if err != nil {
			return 0, err
		}
		if enabled {
			return index, nil
		}
		knownBad[index] = struct{}{}
	}

	maxGood := statefulFallbackMaxGood(uint64(ruleCount), uint64(len(knownBad)))
	speculative, err := tc.Integer(0, int64(maxGood-1), 0)
	if err != nil {
		return 0, err
	}
	allowed := make([]int, 0, maxGood)
	for index := 0; index < ruleCount; index++ {
		if _, bad := knownBad[index]; bad {
			continue
		}
		enabled, err := (*flags).enabled(tc, index)
		if err != nil {
			return 0, err
		}
		if enabled {
			allowed = append(allowed, index)
			if statefulFallbackAccept(uint64(len(allowed)), uint64(speculative)) {
				forced, err := tc.integerForced(int64(index))
				return int(forced), err
			}
		}
	}
	offset, err := tc.Integer(0, int64(len(allowed)-1), 0)
	if err != nil {
		return 0, err
	}
	forced, err := tc.integerForced(int64(allowed[offset]))
	return int(forced), err
}

func checkInvariants[M any](
	state *M,
	invariants []Rule[M],
	tc *TestCase,
) error {
	for _, invariant := range invariants {
		tc.Note("invariant: " + invariant.Name)
		if err := invariant.Apply(state, tc); err != nil {
			return err
		}
	}
	return nil
}

func applyRule[M any](
	state *M,
	rule Rule[M],
	tc *TestCase,
) (err error, rejected bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if recovered == errAssumptionRejected {
				err = nil
				rejected = true
				return
			}
			panic(recovered)
		}
	}()
	err = rule.Apply(state, tc)
	return err, false
}

// statefulStopDecision draws one round's decision and reports whether the case
// stops. Protocol 3 records "continue?"; protocols 1 and 2 recorded "stop
// here?", and a replay of one of their blobs keeps drawing that.
func statefulStopDecision(
	tc *TestCase,
	protocol, stepsDrawn, stepsRejected, stepCount uint64,
) (bool, error) {
	if protocol == 3 {
		forced, isForced := statefulContinueForced(stepsDrawn, stepsRejected, stepCount)
		var proceed bool
		var err error
		if isForced {
			proceed, err = tc.booleanForced(forced)
		} else {
			proceed, err = tc.Weighted(statefulContinueProbability())
		}
		return !proceed, err
	}
	forced, isForced := statefulStopForced(stepsDrawn, stepsRejected, stepCount)
	if isForced {
		return tc.booleanForced(forced)
	}
	return tc.Weighted(statefulStopProbability)
}

// Run executes a manually registered state machine for DefaultStatefulStepCount
// successful steps.
func Run[M any](state *M, machine Machine[M], tc *TestCase) error {
	return RunSteps(state, machine, tc, DefaultStatefulStepCount)
}

// RunSteps executes a manually registered state machine for at most stepCount
// successful steps.
//
// The case runs at least one step and at most stepCount successful ones. There
// is no step budget drawn up front: before each step the case draws one
// "continue?" boolean, which statefulContinueForced either forces or leaves to
// the choice stream. Its simplest value stops, so the simplest case runs
// exactly one step and the shrinker can end a run at any step by lowering one
// boolean. A rule that rejects via Assume spends an attempt but not a step, so
// a machine with selective assumptions still gets its full run.
//
// A replayed blob recorded before the decision was inverted draws a "stop
// here?" boolean instead; statefulChoiceProtocol tells them apart.
func RunSteps[M any](state *M, machine Machine[M], tc *TestCase, stepCount uint64) error {
	tc.Assume(len(machine.Rules) != 0)
	tc.Note("initial invariant check")
	if err := checkInvariants(state, machine.Invariants, tc); err != nil {
		return err
	}

	var stepsDrawn, stepsRejected uint64
	// Generation and the simplest probe record protocol 3. The probe used to
	// record the legacy protocol, whose stream the sniffer reads back as
	// protocol 2 whenever the first rule's first draw is an integer, so its
	// failures did not replay.
	protocol := uint64(3)
	if tc.replay != nil {
		protocol = statefulChoiceProtocol(tc.replay, tc.position, tc.remaining)
	}
	// The first decision, which is forced, and the feature probability sit
	// before any step's span; each step's span then holds its rule probes, the
	// rule's own draws, and the decision that follows it. That makes every step,
	// the first included, one span the shrinker can delete without misaligning
	// the steps after it. Upstream's span starts at the decision instead, which
	// leaves the first step undeletable here: deleting it takes the forced
	// decision and the probability with it, and a misread decision stops the
	// run. The span tree is not part of the blob.
	stop, err := statefulStopDecision(tc, protocol, stepsDrawn, stepsRejected, stepCount)
	if err != nil || stop {
		return err
	}
	var featureFlags *statefulFeatureFlags
	if protocol != 1 {
		if featureFlags, err = newStatefulFeatureFlags(tc, len(machine.Rules)); err != nil {
			return err
		}
	}
	for {
		tc.StartSpan("stateful.rule")
		var index int
		if featureFlags == nil {
			index, err = drawStatefulRuleIndex(tc, len(machine.Rules))
		} else {
			index, err = selectStatefulRule(tc, len(machine.Rules), &featureFlags)
		}
		if err != nil {
			tc.StopSpan(false)
			return err
		}
		rule := machine.Rules[index]
		stepsDrawn++
		tc.Note(fmt.Sprintf("step %d: %s", stepsDrawn, rule.Name))
		err, rejected := applyRule(state, rule, tc)
		if rejected {
			stepsRejected++
			tc.Note("rule rejected by assumption")
		} else if err == nil {
			err = checkInvariants(state, machine.Invariants, tc)
		}
		if err == nil {
			stop, err = statefulStopDecision(tc, protocol, stepsDrawn, stepsRejected, stepCount)
		}
		if err != nil {
			tc.StopSpan(false)
			return err
		}
		tc.StopSpan(rejected)
		if stop {
			return nil
		}
	}
}

// Pool stores arbitrary execution-local values behind portable choices.
//
// A variable's identity is a recorded choice rather than a host-side counter,
// and a draw from the pool records the identifier it selected rather than a
// position in the live list. That is what lets the shrinker delete an earlier
// addition without changing what every later draw refers to — see
// ir/stateful.zir's stable-identifier section. Before that, deleting one
// addition renumbered every reference after it, so most deletions produced a
// different rule sequence that no longer failed and shrinking stalled far from
// minimal.
type Pool[T any] struct {
	active []int64
	values map[int64]*T
}

// NewPool creates an empty pool associated with the current test execution.
func NewPool[T any](_ *TestCase) *Pool[T] {
	return &Pool[T]{values: make(map[int64]*T)}
}

func (p *Pool[T]) Len() int {
	return len(p.active)
}

func (p *Pool[T]) IsEmpty() bool {
	return len(p.active) == 0
}

// Add stores a value under a freshly drawn identifier.
//
// It takes the case because adding is now a draw: the identifier is recorded
// in the choice stream rather than handed out by a host-side counter, which is
// what lets a later reference to this variable keep meaning this variable when
// an earlier addition is shrunk away.
func (p *Pool[T]) Add(tc *TestCase, value T) error {
	id, err := tc.FreshID()
	if err != nil {
		return err
	}
	p.active = append(p.active, id)
	p.values[id] = &value
	return nil
}

func (p *Pool[T]) ValuesReusable() ReusableValues[T] {
	return ReusableValues[T]{pool: p}
}

func (p *Pool[T]) ValuesConsumed() ConsumedValues[T] {
	return ConsumedValues[T]{pool: p}
}

// ReusableValues draws a pointer without removing it from the pool.
type ReusableValues[T any] struct {
	pool *Pool[T]
}

func (g ReusableValues[T]) Draw(tc *TestCase) (*T, error) {
	tc.Assume(!g.pool.IsEmpty())
	id, err := tc.FromSet(g.pool.active)
	if err != nil {
		return nil, err
	}
	return g.pool.values[id], nil
}

// ConsumedValues removes and returns a selected pool value.
type ConsumedValues[T any] struct {
	pool *Pool[T]
}

func (g ConsumedValues[T]) Draw(tc *TestCase) (T, error) {
	var zero T
	tc.Assume(!g.pool.IsEmpty())
	id, err := tc.FromSet(g.pool.active)
	if err != nil {
		return zero, err
	}
	for position, active := range g.pool.active {
		if active == id {
			g.pool.active = append(
				g.pool.active[:position],
				g.pool.active[position+1:]...,
			)
			break
		}
	}
	value := *g.pool.values[id]
	delete(g.pool.values, id)
	return value, nil
}

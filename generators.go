package zizek

import (
	"errors"
	"math"
)

// Generator records the decisions needed to produce a value of type T.
type Generator[T any] interface {
	Draw(*TestCase) (T, error)
}

// Draw produces a value through a typed generator.
func Draw[T any](tc *TestCase, generator Generator[T]) (T, error) {
	timer := tc.startGeneration()
	defer tc.finishGeneration(timer)
	return generator.Draw(tc)
}

// MustDraw is Draw for bodies without an error return, such as the one
// zizek.Test runs: instead of returning a generator error it panics with it.
// The runner recognizes a panicked control-flow error (the replay ran out, the
// choice budget is spent, the example was rejected) and discards the example
// exactly as if the property had returned the error; any other generator
// error, such as an invalid configuration, fails the property.
func MustDraw[T any](tc *TestCase, generator Generator[T]) T {
	value, err := Draw(tc, generator)
	if err != nil {
		panic(err)
	}
	return value
}

// IntegerGenerator builds bounded int64 draws.
type IntegerGenerator struct {
	min int64
	max int64
	// targetSet distinguishes "shrink towards 0" from "no target given". Folding
	// the two together is what made Go and Zig clamp an explicit out-of-range
	// target while the other four shells passed it through.
	target    int64
	targetSet bool
}

// Integers generates signed 64-bit integers.
func Integers() IntegerGenerator {
	return IntegerGenerator{min: -1 << 63, max: 1<<63 - 1}
}

func (g IntegerGenerator) MinValue(value int64) IntegerGenerator {
	g.min = value
	return g
}

func (g IntegerGenerator) MaxValue(value int64) IntegerGenerator {
	g.max = value
	return g
}

func (g IntegerGenerator) ShrinkTowards(value int64) IntegerGenerator {
	g.target = value
	g.targetSet = true
	return g
}

func (g IntegerGenerator) Draw(tc *TestCase) (int64, error) {
	// An explicit target is passed through unclamped: integerSimplest already
	// clamps it to the nearer bound, and clamping again here would send a target
	// above max down to min instead.
	target := g.target
	if !g.targetSet {
		target = integerDefaultShrinkTarget(g.min, g.max)
	}
	return tc.Integer(g.min, g.max, target)
}

// FloatGenerator builds IEEE-754 binary64 draws.
type FloatGenerator struct {
	min             float64
	max             float64
	allowNaN        bool
	allowInfinity   bool
	smallestNonzero float64
}

// Floats generates finite binary64 values by default.
func Floats() FloatGenerator {
	return FloatGenerator{min: math.Inf(-1), max: math.Inf(1)}
}

func (g FloatGenerator) MinValue(value float64) FloatGenerator {
	g.min = value
	return g
}

func (g FloatGenerator) MaxValue(value float64) FloatGenerator {
	g.max = value
	return g
}

func (g FloatGenerator) AllowNaN(value bool) FloatGenerator {
	g.allowNaN = value
	return g
}

func (g FloatGenerator) AllowInfinity(value bool) FloatGenerator {
	g.allowInfinity = value
	return g
}

func (g FloatGenerator) SmallestNonzeroMagnitude(value float64) FloatGenerator {
	g.smallestNonzero = value
	return g
}

func (g FloatGenerator) Draw(tc *TestCase) (float64, error) {
	return tc.Float(
		g.min, g.max, g.allowNaN, g.allowInfinity, g.smallestNonzero,
	)
}

// Collect draws a variable-length sequence of values from one generator.
//
// The length is never a choice of its own. Each iteration draws a single
// "another element?" boolean inside a span that also holds that element's own
// choices, which is what makes a collection structurally shrinkable: deleting
// the span removes exactly one element and leaves the rest of the stream
// framed correctly. Truncation falls out of the same shape — lowering one of
// those booleans to false ends the sequence there.
//
// The decision itself lives in the core (ir/many.zir); this loop only supplies
// entropy and records spans.
func Collect[T any](
	tc *TestCase, element Generator[T], minSize, maxSize int, hasMax bool, label string,
) ([]T, error) {
	if minSize < 0 {
		return nil, errors.New("minSize must be >= 0")
	}
	bound := 0
	if hasMax {
		bound = maxSize
		if bound < minSize {
			return nil, errors.New("maxSize must be >= minSize")
		}
	}
	probability := lengthPContinue(int64(minSize), int64(bound), hasMax)
	drawsChoice := manyDrawsChoice(int64(minSize), int64(bound), hasMax)
	values := []T{}
	for {
		tc.StartSpan(label)
		forced, isForced := manyForcedValue(
			false, int64(len(values)), int64(minSize), int64(bound), hasMax, probability,
		)
		var more bool
		var err error
		switch {
		case !drawsChoice:
			// Bounds coincide: upstream draws nothing at all here.
			more = len(values) < minSize
		case isForced:
			more, err = tc.booleanForced(forced)
		default:
			more, err = tc.Weighted(probability)
		}
		if err != nil {
			// An open span is closed for us when the execution unwinds.
			return nil, err
		}
		if !more {
			tc.StopSpan(true)
			return values, nil
		}
		value, err := Draw(tc, element)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		tc.StopSpan(false)
	}
}

// What a rejected element costs a unique collection. Defined by
// ir/many.zir::many_reject_action.
const (
	rejectRetry   uint64 = 0
	rejectStop    uint64 = 1
	rejectInvalid uint64 = 2
)

// CollectUnique draws a variable-length sequence whose elements are all
// distinct under key.
//
// Same pacing as Collect, with one extra decision layered on: what a duplicate
// costs. Upstream spells a unique collection as a *filtered* element draw inside
// the same "another element?" loop, so this does too. Each element gets up to
// filterRetryAllowed candidates, each in its own span that is discarded when the
// candidate is already present — the choices stay in the stream, so a replay
// walks the same positions, but the shrinker may delete them freely.
//
// When every candidate for one element is a duplicate the *element* is rejected
// rather than the example, and manyRejectAction decides what that costs: retry,
// stop the collection where it is, or — when the collection cannot reach its own
// minSize — reject the example the way a failed assumption does. Both rules live
// in ir/many.zir.
func CollectUnique[T any, K comparable](
	tc *TestCase,
	element Generator[T],
	key func(T) K,
	minSize, maxSize int,
	hasMax bool,
	label string,
) ([]T, error) {
	if key == nil {
		return nil, errors.New("key must not be nil")
	}
	if minSize < 0 {
		return nil, errors.New("minSize must be >= 0")
	}
	bound := 0
	if hasMax {
		bound = maxSize
		if bound < minSize {
			return nil, errors.New("maxSize must be >= minSize")
		}
	}
	probability := lengthPContinue(int64(minSize), int64(bound), hasMax)
	drawsChoice := manyDrawsChoice(int64(minSize), int64(bound), hasMax)
	values := []T{}
	seen := map[K]struct{}{}
	rejections := uint64(0)
	forceStop := false
	for {
		tc.StartSpan(label)
		forced, isForced := manyForcedValue(
			forceStop, int64(len(values)), int64(minSize), int64(bound), hasMax, probability,
		)
		var more bool
		var err error
		switch {
		case !drawsChoice:
			// Bounds coincide: upstream draws nothing at all here.
			more = len(values) < minSize
		case isForced:
			more, err = tc.booleanForced(forced)
		default:
			more, err = tc.Weighted(probability)
		}
		if err != nil {
			// An open span is closed for us when the execution unwinds.
			return nil, err
		}
		if !more {
			tc.StopSpan(true)
			return values, nil
		}
		var chosen T
		var fingerprint K
		accepted := false
		for attempts := uint64(0); !accepted && filterRetryAllowed(attempts); attempts++ {
			tc.StartSpan("collection.candidate")
			candidate, err := Draw(tc, element)
			if err != nil {
				return nil, err
			}
			mark := key(candidate)
			if _, duplicate := seen[mark]; duplicate {
				tc.StopSpan(true)
				continue
			}
			tc.StopSpan(false)
			chosen, fingerprint, accepted = candidate, mark, true
		}
		if !accepted {
			tc.StopSpan(true)
			rejections++
			switch manyRejectAction(rejections, int64(len(values)), int64(minSize)) {
			case rejectInvalid:
				tc.Assume(false)
			case rejectStop:
				forceStop = true
			}
			continue
		}
		seen[fingerprint] = struct{}{}
		values = append(values, chosen)
		tc.StopSpan(false)
	}
}

// ListGenerator draws a variable-length list of values from one element
// generator.
type ListGenerator[T any] struct {
	element Generator[T]
	min     int
	max     int
	hasMax  bool
}

// Lists generates a list whose elements each occupy a deletable span.
//
// Unlike the scalar generators this one has structure the shrinker can
// reduce: it deletes whole elements before it simplifies the values inside
// them, so a failing list minimizes towards the shortest list that still
// fails rather than towards small elements in a long list.
func Lists[T any](element Generator[T]) ListGenerator[T] {
	return ListGenerator[T]{element: element}
}

func (g ListGenerator[T]) MinSize(size int) ListGenerator[T] {
	g.min = size
	return g
}

func (g ListGenerator[T]) MaxSize(size int) ListGenerator[T] {
	g.max = size
	g.hasMax = true
	return g
}

func (g ListGenerator[T]) Draw(tc *TestCase) ([]T, error) {
	return Collect(tc, g.element, g.min, g.max, g.hasMax, "list.element")
}

// UniqueListGenerator draws a list with no two elements sharing a key.
//
// Uniqueness needs a comparable key type that the element type does not supply,
// and a Go method cannot introduce a type parameter — so this is a generator of
// its own rather than a UniqueBy method on ListGenerator.
type UniqueListGenerator[T any, K comparable] struct {
	element Generator[T]
	key     func(T) K
	label   string
	min     int
	max     int
	hasMax  bool
}

// UniqueLists generates a list whose elements have distinct keys.
//
// A duplicate costs the element, not the example: the candidate's choices are
// discarded and another is drawn. See CollectUnique.
func UniqueLists[T any, K comparable](
	element Generator[T], key func(T) K,
) UniqueListGenerator[T, K] {
	return UniqueListGenerator[T, K]{element: element, key: key, label: "list.element"}
}

// Sets generates a list of distinct values.
//
// Go has no set type, so a set is a slice whose elements are pairwise distinct —
// which is also the shape the shrinker can delete elements from.
func Sets[T comparable](element Generator[T]) UniqueListGenerator[T, T] {
	return UniqueListGenerator[T, T]{
		element: element,
		key:     func(value T) T { return value },
		label:   "set.element",
	}
}

func (g UniqueListGenerator[T, K]) MinSize(size int) UniqueListGenerator[T, K] {
	g.min = size
	return g
}

func (g UniqueListGenerator[T, K]) MaxSize(size int) UniqueListGenerator[T, K] {
	g.max = size
	g.hasMax = true
	return g
}

func (g UniqueListGenerator[T, K]) Draw(tc *TestCase) ([]T, error) {
	return CollectUnique(tc, g.element, g.key, g.min, g.max, g.hasMax, g.label)
}

// MapEntry is one key/value pair drawn by MapGenerator. The pair is drawn as a
// unit so that deleting its span deletes both halves together.
type MapEntry[K comparable, V any] struct {
	Key   K
	Value V
}

type mapEntryGenerator[K comparable, V any] struct {
	keys   Generator[K]
	values Generator[V]
}

func (g mapEntryGenerator[K, V]) Draw(tc *TestCase) (MapEntry[K, V], error) {
	key, err := Draw(tc, g.keys)
	if err != nil {
		return MapEntry[K, V]{}, err
	}
	value, err := Draw(tc, g.values)
	if err != nil {
		return MapEntry[K, V]{}, err
	}
	return MapEntry[K, V]{Key: key, Value: value}, nil
}

// MapGenerator draws a map with distinct keys.
type MapGenerator[K comparable, V any] struct {
	entries mapEntryGenerator[K, V]
	min     int
	max     int
	hasMax  bool
}

// Maps generates a map whose keys are distinct. Uniqueness is judged on the key
// alone, so the value generator is free to repeat.
func Maps[K comparable, V any](keys Generator[K], values Generator[V]) MapGenerator[K, V] {
	return MapGenerator[K, V]{entries: mapEntryGenerator[K, V]{keys: keys, values: values}}
}

func (g MapGenerator[K, V]) MinSize(size int) MapGenerator[K, V] {
	g.min = size
	return g
}

func (g MapGenerator[K, V]) MaxSize(size int) MapGenerator[K, V] {
	g.max = size
	g.hasMax = true
	return g
}

func (g MapGenerator[K, V]) Draw(tc *TestCase) (map[K]V, error) {
	entries, err := CollectUnique(
		tc,
		g.entries,
		func(entry MapEntry[K, V]) K { return entry.Key },
		g.min, g.max, g.hasMax, "map.entry",
	)
	if err != nil {
		return nil, err
	}
	out := make(map[K]V, len(entries))
	for _, entry := range entries {
		out[entry.Key] = entry.Value
	}
	return out, nil
}

// TextGenerator produces text over an alphabet, by default every Unicode
// scalar value. Values shrink shortlex over the alphabet's shrink order —
// shorter first, then towards '0', the digits, and the uppercase letters — so
// a minimal failing string is one a reader can retype.
type TextGenerator struct {
	min      int
	max      int
	hasMax   bool
	alphabet []byte
}

// Text creates a text generator.
func Text() TextGenerator {
	return TextGenerator{}
}

func (g TextGenerator) MinSize(value int) TextGenerator {
	g.min = value
	return g
}

func (g TextGenerator) MaxSize(value int) TextGenerator {
	g.max = value
	g.hasMax = true
	return g
}

// Alphabet restricts the characters drawn to an interval set built by
// Alphabet or AlphabetRange.
func (g TextGenerator) Alphabet(characters []byte) TextGenerator {
	g.alphabet = characters
	return g
}

func (g TextGenerator) Draw(tc *TestCase) (string, error) {
	max := g.max
	if !g.hasMax {
		max = int(textWidenedMax(uint64(g.min)))
	}
	return tc.String(g.min, max, g.alphabet)
}

// ---------- scalars, constants, and combinators ----------
//
// None of the wrappers below records a choice of its own — only SampledFrom
// and OneOf do, and each records exactly one integer — so composing them
// leaves the reproduce blob byte-compatible with the other packages.
//
// The semantics are shared across languages and the spelling is native:
// Go has no method-level type parameters, so Map and Filter are functions
// taking the source generator rather than methods on it, and a heterogeneous
// tuple is a []any rather than a typed product.

type booleanGenerator struct{}

func (booleanGenerator) Draw(tc *TestCase) (bool, error) { return tc.Boolean() }

// Booleans generates booleans, shrinking towards false.
func Booleans() Generator[bool] { return booleanGenerator{} }

type weightedBooleanGenerator struct{ probability float64 }

func (g weightedBooleanGenerator) Draw(tc *TestCase) (bool, error) {
	if math.IsNaN(g.probability) || g.probability < 0 || g.probability > 1 {
		return false, errors.New("weighted_booleans requires a probability in [0, 1]")
	}
	return tc.Weighted(g.probability)
}

// WeightedBooleans generates booleans that are true with the given
// probability. Recorded as an ordinary boolean choice: the weight biases
// generation only, so a blob replays the same whatever probability produced
// it.
func WeightedBooleans(probability float64) Generator[bool] {
	return weightedBooleanGenerator{probability: probability}
}

// BytesGenerator draws byte strings, shrinking towards shorter and lower
// values.
type BytesGenerator struct {
	minimum int
	maximum int
}

// Binary generates byte strings between 0 and 16 bytes long by default.
func Binary() BytesGenerator { return BytesGenerator{minimum: 0, maximum: 16} }

// MinSize returns a copy drawing at least n bytes.
func (g BytesGenerator) MinSize(n int) BytesGenerator { g.minimum = n; return g }

// MaxSize returns a copy drawing at most n bytes.
func (g BytesGenerator) MaxSize(n int) BytesGenerator { g.maximum = n; return g }

// Draw implements Generator.
func (g BytesGenerator) Draw(tc *TestCase) ([]byte, error) { return tc.Bytes(g.minimum, g.maximum) }

type justGenerator[T any] struct{ value T }

func (g justGenerator[T]) Draw(*TestCase) (T, error) { return g.value, nil }

// Just always produces one value, recording no choice. Go spells the `none`
// of the other packages as Just[any](nil).
func Just[T any](value T) Generator[T] { return justGenerator[T]{value: value} }

type sampledGenerator[T any] struct{ values []T }

func (g sampledGenerator[T]) Draw(tc *TestCase) (T, error) {
	// A real integer choice, so the index shrinks and replays like any other.
	index, err := tc.Integer(0, int64(len(g.values)-1), 0)
	if err != nil {
		var zero T
		return zero, err
	}
	return g.values[index], nil
}

// SampledFrom picks one of a fixed sequence, shrinking towards the first
// element. It panics on an empty sequence, which is a programming error rather
// than a property failure.
func SampledFrom[T any](values []T) Generator[T] {
	if len(values) == 0 {
		panic("zizek: SampledFrom requires at least one value")
	}
	return sampledGenerator[T]{values: values}
}

type oneOfGenerator[T any] struct{ choices []Generator[T] }

func (g oneOfGenerator[T]) Draw(tc *TestCase) (T, error) {
	index, err := tc.Integer(0, int64(len(g.choices)-1), 0)
	if err != nil {
		var zero T
		return zero, err
	}
	return g.choices[index].Draw(tc)
}

// OneOf draws from one of several generators, shrinking towards the first.
func OneOf[T any](choices ...Generator[T]) Generator[T] {
	if len(choices) == 0 {
		panic("zizek: OneOf requires at least one generator")
	}
	return oneOfGenerator[T]{choices: choices}
}

type anyGenerator[T any] struct{ source Generator[T] }

func (g anyGenerator[T]) Draw(tc *TestCase) (any, error) {
	value, err := g.source.Draw(tc)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// AsAny erases a generator's element type so it can join a Tuples list. Go has
// no variadic type parameters, so a heterogeneous tuple has to go through any.
func AsAny[T any](source Generator[T]) Generator[any] { return anyGenerator[T]{source: source} }

type tupleGenerator struct{ parts []Generator[any] }

func (g tupleGenerator) Draw(tc *TestCase) ([]any, error) {
	values := make([]any, 0, len(g.parts))
	for _, part := range g.parts {
		value, err := part.Draw(tc)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Tuples draws each generator in order and collects the values.
func Tuples(parts ...Generator[any]) Generator[[]any] { return tupleGenerator{parts: parts} }

type mappedGenerator[T, U any] struct {
	source    Generator[T]
	transform func(T) U
}

func (g mappedGenerator[T, U]) Draw(tc *TestCase) (U, error) {
	value, err := g.source.Draw(tc)
	if err != nil {
		var zero U
		return zero, err
	}
	return g.transform(value), nil
}

// Map applies a function to each value a generator draws.
func Map[T, U any](source Generator[T], transform func(T) U) Generator[U] {
	return mappedGenerator[T, U]{source: source, transform: transform}
}

type flatMappedGenerator[T, U any] struct {
	source    Generator[T]
	transform func(T) Generator[U]
}

func (g flatMappedGenerator[T, U]) Draw(tc *TestCase) (U, error) {
	tc.StartSpan("flat_map")
	value, err := g.source.Draw(tc)
	if err != nil {
		// An open span is closed for us when the execution unwinds.
		var zero U
		return zero, err
	}
	result, err := g.transform(value).Draw(tc)
	if err != nil {
		var zero U
		return zero, err
	}
	tc.StopSpan(false)
	return result, nil
}

// FlatMap builds the next generator from each drawn value — monadic bind.
// Both draws share one "flat_map" span, so the shrinker can delete or
// simplify the whole unit together: shrinking the source value changes which
// inner generator runs, and the inner draws replay or pun by the ordinary
// prefix rules.
func FlatMap[T, U any](source Generator[T], transform func(T) Generator[U]) Generator[U] {
	return flatMappedGenerator[T, U]{source: source, transform: transform}
}

type filteredGenerator[T any] struct {
	source    Generator[T]
	predicate func(T) bool
}

func (g filteredGenerator[T]) Draw(tc *TestCase) (T, error) {
	var zero T
	// The attempt budget is the same one a unique collection spends per
	// element, so it comes from ir/many.zir rather than a constant here.
	for attempts := uint64(0); filterRetryAllowed(attempts); attempts++ {
		value, err := g.source.Draw(tc)
		if err != nil {
			return zero, err
		}
		if g.predicate(value) {
			return value, nil
		}
	}
	// A rejection discards the example rather than failing the property.
	tc.Assume(false)
	return zero, errAssumptionRejected
}

// Filter keeps only values satisfying a predicate, rejecting the example if a
// bounded number of attempts all fail it.
func Filter[T any](source Generator[T], predicate func(T) bool) Generator[T] {
	return filteredGenerator[T]{source: source, predicate: predicate}
}

type compositeGenerator[T any] struct{ build func(*TestCase) (T, error) }

func (g compositeGenerator[T]) Draw(tc *TestCase) (T, error) { return g.build(tc) }

// Composite builds a generator from a function that draws as it goes. Go
// already hands the test case to a property, so the natural spelling takes the
// case directly rather than the draw callback the dynamic packages use.
func Composite[T any](build func(*TestCase) (T, error)) Generator[T] {
	return compositeGenerator[T]{build: build}
}

// RecursiveGenerator builds recursively-defined values from a leaf generator
// and a function which combines generators for child subtrees.
type RecursiveGenerator[T any] struct {
	leaf      Generator[T]
	branch    func(Generator[T]) Generator[T]
	maxDepth  uint64
	maxLeaves uint64
}

// Recursive generates trees and other recursively-defined values. Every
// subtree occupies a "recursive" span, so shrinking can replace an ancestor
// with a simpler descendant and hoist a deep witness to the root.
func Recursive[T any](
	leaf Generator[T], branch func(Generator[T]) Generator[T],
) RecursiveGenerator[T] {
	if leaf == nil || branch == nil {
		panic("zizek: Recursive requires a leaf generator and branch function")
	}
	return RecursiveGenerator[T]{
		leaf: leaf, branch: branch, maxDepth: 32, maxLeaves: 100,
	}
}

// MaxDepth sets the maximum nesting depth of branch nodes. A value of zero
// generates leaves only; with MaxLeaves zero the draw rejects.
func (g RecursiveGenerator[T]) MaxDepth(value uint64) RecursiveGenerator[T] {
	g.maxDepth = value
	return g
}

// MaxLeaves sets the maximum leaf count for one generated value.
func (g RecursiveGenerator[T]) MaxLeaves(value uint64) RecursiveGenerator[T] {
	if value > math.MaxInt64 {
		panic("max leaves must fit in int64")
	}
	g.maxLeaves = value
	return g
}

type recursiveState struct {
	maxDepth       uint64
	maxLeaves      uint64
	target         uint64
	attempt        uint64
	leaves         uint64
	startPrice     float64
	closedChildren uint64
	closedBranches uint64
	openBranches   []recursiveOpenBranch
	reprices       uint64
}

type recursiveOpenBranch struct {
	depth    uint64
	children uint64
}

func (s *recursiveState) observeDecision(depth uint64) {
	for len(s.openBranches) != 0 && s.openBranches[len(s.openBranches)-1].depth >= depth {
		last := s.openBranches[len(s.openBranches)-1]
		s.openBranches = s.openBranches[:len(s.openBranches)-1]
		s.closedChildren += last.children
		s.closedBranches++
	}
	if depth > 0 && len(s.openBranches) != 0 {
		s.openBranches[len(s.openBranches)-1].children++
	}
}

func (s *recursiveState) observeBranch(depth uint64) {
	s.openBranches = append(s.openBranches, recursiveOpenBranch{depth: depth})
}

func (s *recursiveState) closeBranches() {
	for len(s.openBranches) != 0 {
		last := s.openBranches[len(s.openBranches)-1]
		s.openBranches = s.openBranches[:len(s.openBranches)-1]
		s.closedChildren += last.children
		s.closedBranches++
	}
}

type recursiveSubtreeGenerator[T any] struct {
	owner RecursiveGenerator[T]
	state *recursiveState
	depth uint64
}

var errRecursiveLeafBudget = errors.New("zizek: recursive leaf budget exceeded")

func (g recursiveSubtreeGenerator[T]) Draw(tc *TestCase) (T, error) {
	var zero T
	tc.StartSpan("recursive")
	g.state.observeDecision(g.depth)
	probability := recursiveDecisionProbability(
		g.state.maxDepth, g.depth, g.state.target, g.state.attempt,
		g.state.closedChildren, g.state.closedBranches,
	)
	var branch bool
	var err error
	if g.depth >= g.state.maxDepth {
		branch, err = tc.booleanForced(false)
	} else {
		branch, err = tc.Weighted(probability)
	}
	if err != nil {
		return zero, err
	}
	var value T
	if branch {
		g.state.observeBranch(g.depth)
		child := recursiveSubtreeGenerator[T]{owner: g.owner, state: g.state, depth: g.depth + 1}
		value, err = g.owner.branch(child).Draw(tc)
	} else {
		if g.state.leaves >= g.state.maxLeaves {
			return zero, errRecursiveLeafBudget
		}
		g.state.leaves++
		value, err = g.owner.leaf.Draw(tc)
	}
	if err != nil {
		return zero, err
	}
	// Keep the root span open until the completed-attempt pricing check. A
	// mispriced value then becomes one discarded structural unit.
	if g.depth != 0 {
		tc.StopSpan(false)
	}
	return value, nil
}

// drawRecursiveRoot draws one attempt at a whole tree, converting a panicked
// leaf-budget error back into the returned form the retry loop matches on.
// The budget error crosses user code on its way up — the branch composite
// hands it back — and a composite built on MustDraw hands it back as a panic
// instead of a return.
func drawRecursiveRoot[T any](
	g RecursiveGenerator[T], state *recursiveState, tc *TestCase,
) (value T, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if e, ok := recovered.(error); ok && errors.Is(e, errRecursiveLeafBudget) {
				err = e
				return
			}
			panic(recovered)
		}
	}()
	root := recursiveSubtreeGenerator[T]{owner: g, state: state}
	return root.Draw(tc)
}

func (g RecursiveGenerator[T]) Draw(tc *TestCase) (T, error) {
	var zero T
	baseDepth := len(tc.spanStack)
	target := uint64(1)
	if recursiveTargetDrawn(g.maxLeaves) {
		large, err := tc.Weighted(recursiveLargeTargetProbability())
		if err != nil {
			return zero, err
		}
		if large {
			value, err := tc.integerUniform(2, int64(g.maxLeaves), 0)
			if err != nil {
				return zero, err
			}
			target = uint64(value)
		}
	}
	state := &recursiveState{maxDepth: g.maxDepth, maxLeaves: g.maxLeaves, target: target}
	state.startPrice = recursivePricedProbability(0, 0, 0, target)
	for {
		value, err := drawRecursiveRoot(g, state, tc)
		if errors.Is(err, errRecursiveLeafBudget) {
			for len(tc.spanStack) > baseDepth {
				tc.StopSpan(true)
			}
			state.openBranches = nil
			state.leaves = 0
			state.attempt++
			if !recursiveRetryAllowed(state.attempt) {
				tc.Assume(false)
				return zero, errAssumptionRejected
			}
			state.startPrice = recursivePricedProbability(
				state.closedChildren, state.closedBranches, state.attempt, state.target,
			)
			continue
		}
		if err != nil {
			return zero, err
		}
		state.closeBranches()
		mispriced := recursiveFinishMispriced(
			state.maxDepth, state.target, state.attempt, state.leaves,
			state.closedChildren, state.closedBranches, state.startPrice, state.reprices,
		)
		if !mispriced {
			tc.StopSpan(false)
			return value, nil
		}
		for len(tc.spanStack) > baseDepth {
			tc.StopSpan(true)
		}
		state.leaves = 0
		state.reprices++
		state.startPrice = recursivePricedProbability(
			state.closedChildren, state.closedBranches, state.attempt, state.target,
		)
	}
}

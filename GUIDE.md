# Zizek for Go — feature guide

A walkthrough of every feature the Go runner currently supports, with runnable
examples for each. See [README.md](README.md) for installation and
[dev.md](dev.md) for the core/shell architecture; this guide is about *using*
the API.

Everything below is one package:

```go
import "github.com/wallyqs/zizek.go"
```

The guide runs from the outside in: how a run is configured and what it
returns, then how values are drawn, then the assertions and diagnostics layered
on top, then the features a simulator reaches for — keyed draws, the example
database, costs, targeting, spans, clones — and finally stateful testing. Each
section names the runnable example that demonstrates it; the table at the end
lists them all.

## Properties and `Check`

A property is a function from a test case to an error. Returning a non-nil
error fails the example; panics are captured and fail it too, so an
`index out of range` deep inside the code under test is a counterexample rather
than a crashed run.

```go
func property(tc *zizek.TestCase) error {
	value, err := tc.Integer(-100, 100, 0)
	if err != nil {
		return err
	}
	if value >= 5 {
		return fmt.Errorf("value must be less than five: %d", value)
	}
	return nil
}

err := zizek.Check(property, zizek.Config{Examples: 200, Seed: 1})
```

The `err` a draw returns is not a property failure — it is the run telling you
this example is over (the replay ran out, the choice budget is spent). Return
it unchanged; `Check` knows what to do with it. That is why every draw in this
guide is followed by `if err != nil { return err }`. A body with no error
return — the one `zizek.Test` runs — surfaces the error by panicking with it
instead: `zizek.MustDraw` does exactly that, and the run recognizes a panicked
draw error as the same control flow, not as a counterexample.

`zizek.Config` controls the run. Every field is optional, and a zero `Config`
runs 100 examples with the defaults below:

| Field | Default | Meaning |
|-------|---------|---------|
| `Examples` | 100 | Valid (non-rejected) examples to run before declaring success. |
| `MaxRejections` | 458 | Initial rejection allowance; each valid example earns 100 more. |
| `MaxShrinks` | 500 | Accepted shrink candidates before shrinking stops. Negative disables the cap. |
| `MaxStall` | 200 | Consecutive rejected shrink candidates before shrinking stops. Negative disables the cap. |
| `Seed` | 0 | Seed for random exploration. Fixed seeds make a run reproducible on one machine; portable reproduction uses blobs instead. |
| `Reproduce` | `""` | A failure blob. When set, `Check` replays exactly that example and runs nothing else. |
| `Require` | `nil` | Coverage labels the completed run must have observed, whether or not their call sites ever ran. |
| `Database` | `nil` | Where counterexamples are stored between runs. Nil reads and writes nothing. |
| `DatabaseKey` | `""` | The name a stored counterexample is filed under. Empty derives one from the property's function name. |
| `SuppressFilterTooMuch` | `false` | Disable the early excessive-filtering health check. |
| `SuppressDataTooLarge` | `false` | Disable the early choice-budget-overrun health check. |
| `SuppressLargeBaseExample` | `false` | Allow an all-simplest input that uses more than half the choice budget. |
| `SuppressTooSlow` | `false` | Disable the one-second cumulative generator-time health check. |

### The verdict is a value

`Check` returns nil on success, or one of six errors — the verdict is data you
can match on rather than a message you have to parse:

| Error | Means |
|-------|-------|
| `*zizek.Failure` | A property failed. Already shrunk, carries a reproduce blob. |
| `*zizek.Flaky` | A property failed, then did not fail the same way when that example was replayed. Carries the original error and the unshrunk example's blob. |
| `*zizek.TooManyRejections` | Assumptions rejected too many examples to finish the run. |
| `*zizek.FailedCoverage` | A completed run missed a coverage requirement. |
| `*zizek.FailedHealthCheck` | A generation pattern that weakens testing (see [Health checks](#health-checks)). |
| `*zizek.InvalidReplay` / `*zizek.ReplayOverrun` | A reproduce blob is malformed, or ends before the property stops drawing. |

Match them with `errors.As`:

```go
var failure *zizek.Failure
if errors.As(err, &failure) {
	fmt.Println(failure.Blob, failure.Cause, failure.Notes)
}
```

`*zizek.Failure` is the one you will read most often:

| Field | What it holds |
|-------|---------------|
| `Blob` | The portable reproduce blob for the shrunk example. |
| `Cause` | The error the property returned (or a wrapper around the panic). `Unwrap` returns it, so `errors.Is` and `errors.As` reach through. |
| `Notes` | The notes the *shrunk* example recorded. |
| `ExamplesRun` | Attempts spent before the failure was found. |
| `Rejections` | Examples assumptions threw away. |
| `Puns`, `KeyMisses` | Replay substitutions — see [Failures, blobs, and replay](#failures-blobs-and-replay) and [Keyed draws](#keyed-draws). |
| `Costs` | What the reported example spent, per label — see [Shrink costs](#shrink-costs). |

Printing a failure renders the cause, the reproduce line, any pun or key-miss
warnings, and the notes as a `Trace:` section:

```
value must be less than five: 5
reproduce with zizek.Check(property, zizek.Config{Reproduce: "AAEAAAAACgEAAAAF"})

Trace:
- value 5
```

### Repeating a search versus replaying an example

Two settings look similar and do opposite things:

- `Seed` repeats a *search*. The same seed, the same property, and the same
  build explore the same examples in the same order — useful while you are
  iterating, and worth nothing to anyone on another machine or in another
  language.
- `Reproduce` replays one *example*. The blob is the recorded choice sequence,
  so it reproduces bit-exactly anywhere, and `Check` runs that one example and
  stops.

For a bug you want to keep, neither is the durable answer: put the
counterexample in the [example database](#the-example-database), or paste the
blob into a regression test.

### Running properties under `go test`

`zizek.Test` runs a property inside a Go test. Its body takes `*zizek.T`, an
alias of `*zizek.TestCase` rather than a wrapper, so every method and every
generator works unchanged, and a failure is reported through the test's
`t.Error` with the reproduce line and trace:

```go
func TestIntegerBound(t *testing.T) {
	zizek.Test(t, func(zt *zizek.T) {
		value := zizek.MustDraw(zt, zizek.Integers().MinValue(-100).MaxValue(100))
		zt.Always("less than five", value < 5)
	}, zizek.Config{Examples: 200, Seed: 1})
}
```

The body has no error return, so it fails an example by panicking: with
`zt.Always` or `zt.Unreachable`, a plain `panic`, or any assertion helper that
panics. `zizek.MustDraw` draws without an error return; see
[Properties and `Check`](#properties-and-check) for why its panics are not
counterexamples. `Test` takes at most one `Config`, and with none it uses
`Check`'s defaults. When the `Config` sets a `Database` but no `DatabaseKey`,
the test's name (`t.Name()`) becomes the key.

The package's [Hegel-derived GoDoc examples](hegel_examples_test.go) show
filtering, assumptions, mapping, dependent and flat-mapped draws, targeting,
and bounded recursive trees through the same adapter.

`Check` is the other way in, for an error-returning property, a run outside a
test, or direct access to the returned `Failure`. Since a property is a
function and `Check` returns an error, an ordinary test is all it needs:

```go
func TestSplitIsExact(t *testing.T) {
	if err := zizek.Check(splitIsExact, zizek.Config{Examples: 500}); err != nil {
		t.Fatal(err) // the rendered failure carries the blob and the trace
	}
}
```

Table tests work the way they always do, since a property is just a closure:

```go
func TestCodecsRoundTrip(t *testing.T) {
	for _, codec := range []Codec{JSON, MsgPack, CBOR} {
		t.Run(codec.Name(), func(t *testing.T) {
			property := func(tc *zizek.TestCase) error {
				text, err := zizek.Draw(tc, zizek.Text().MaxSize(32))
				if err != nil {
					return err
				}
				if got := codec.Decode(codec.Encode(text)); got != text {
					return fmt.Errorf("%q round-tripped to %q", text, got)
				}
				return nil
			}
			if err := zizek.Check(property, zizek.Config{Examples: 200}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
```

Two habits worth keeping: give the property a name (`Config.DatabaseKey`) if
you turn the database on, and pin a rediscovered bug with its blob so the
regression test costs one example instead of five hundred.

## Low-level draws

`TestCase` exposes the recorded primitive choices directly. These are the
draws everything else is built from:

| Draw | Produces | Shrinks towards |
|------|----------|-----------------|
| `tc.Integer(min, max, shrinkTowards)` | `int64` in `[min, max]` | `shrinkTowards`, clamped into the interval |
| `tc.Boolean()` | `bool` | `false` |
| `tc.Weighted(probability)` | `bool`, true with that probability | `false` |
| `tc.Bytes(minSize, maxSize)` | `[]byte` of that length | shorter, then lower bytes |
| `tc.String(minSize, maxSize, alphabet)` | `string` over an alphabet (nil means all of Unicode) | shorter, then `'0'`, the digits, the uppercase letters |
| `tc.Float(min, max, allowNaN, allowInfinity, smallestNonzeroMagnitude)` | `float64` | simpler magnitudes, then zero |
| `tc.FreshID()` | an `int64` unique within this example | the smallest unused identifier |
| `tc.FromSet(members)` | one member, recorded by value | the smallest member |

```go
value, err := tc.Integer(0, 100, 0) // int64 in [0, 100], shrinks toward 0
flag, err := tc.Boolean()           // shrinks toward false
payload, err := tc.Bytes(1, 4)      // 1..4 bytes, shrinks shortlex-first
```

Every draw is recorded, which is what makes shrinking and blob replay work.
Narrow integer ranges (fewer than 256 values) are drawn uniformly, so a draw
that indexes a list picks every option equally often. Wide ranges use a
per-case swarm: the generated core supplies candidates such as the interval
endpoints and values near zero, so bugs at boundaries (like `1` in
`0..=1_000_000`) are found reliably. Each generated case selects one swarm
profile, concentrating related operands around endpoints, interesting values,
diffuse fixed-width constants, or the ordinary middle. The selected values —
not the profile entropy — enter the replay stream, so blobs keep the same
replay contract.

Alphabets are built once and reused. `zizek.Alphabet` returns an error only for
codepoints that cannot be alphabet members, so a literal alphabet is safe to
build in a package-level `var`:

```go
lower, err := zizek.Alphabet("abcdefghijklmnopqrstuvwxyz")
printable := zizek.AlphabetRange(0x20, 0x7E)
name, err := tc.String(1, 8, lower)
anything, err := tc.String(0, 8, nil) // nil is zizek.Unicode
```

Alphabets are interval sets, canonical by construction: `Alphabet("cab")` and
`Alphabet("abc")` are the same alphabet and index their members identically, so
a blob written against one replays against the other.

`FreshID` and `FromSet` are the two draws that exist for *references* rather
than values. A fresh identifier names a thing your test created; `FromSet`
picks one of the identifiers still live. Both record the identifier itself
rather than a position, which is what lets the shrinker delete the third of ten
creations without renaming the other nine. [Pools](#pools) are these two draws
wrapped around a slice of Go values, and most tests meet them that way.

## Typed generators

`zizek.Draw` routes a typed generator through a test case:

```go
value, err := zizek.Draw(tc, zizek.Integers().MinValue(-100).MaxValue(100))
name, err := zizek.Draw(tc, zizek.Text().MinSize(1).MaxSize(8))
```

A generator is any value with a `Draw(*TestCase) (T, error)` method
(`zizek.Generator[T]`), so your own types can join the library's. Generators
are values, not builders with hidden state: every configuration call returns a
copy, so building one once and reusing it everywhere is safe and cheap.

```go
var octet = zizek.Integers().MinValue(0).MaxValue(255) // reuse freely
```

### Scalars

- **`zizek.Integers()`** — full-range `int64` until bounded with `MinValue` /
  `MaxValue`; `ShrinkTowards(target)` moves the shrink destination (see
  [Shrink targets](#shrink-targets)).
- **`zizek.Floats()`** — IEEE-754 binary64, finite by default. `MinValue` /
  `MaxValue` bound it; `AllowNaN(true)` and `AllowInfinity(true)` opt the
  special values back in; `SmallestNonzeroMagnitude(x)` excludes magnitudes
  below `x`, which is how you ask for "no subnormals" without writing a filter.

  ```go
  ratio, err := zizek.Draw(tc, zizek.Floats().MinValue(0).MaxValue(1))
  wild, err := zizek.Draw(tc, zizek.Floats().AllowNaN(true).AllowInfinity(true))
  ```

- **`zizek.Text()`** — text over an alphabet, by default every Unicode scalar
  value. `MinSize` / `MaxSize` bound the length **in characters**, not bytes,
  and `Alphabet(characters)` narrows what is drawn. Values shrink towards
  `"0"`, then the digits and uppercase letters, so a minimal failing string is
  one you can retype.
- **`zizek.Booleans()`** and **`zizek.WeightedBooleans(p)`** — the boolean
  draws as generators. The weight biases generation only: a weighted boolean is
  recorded as an ordinary boolean choice, so a blob replays the same whatever
  probability produced it.
- **`zizek.Binary()`** — byte strings, 0 to 16 bytes by default, bounded with
  `MinSize` / `MaxSize`.

### Collections

Collections are the generators with structure the shrinker can *reduce*: each
element occupies its own span, so whole elements are deleted — including ones
in the middle — before the values inside them are simplified. A 30-element list
shrinks to the one element that matters rather than to 30 small elements.

```go
packet, err := zizek.Draw(tc, zizek.Lists(octet).MinSize(1).MaxSize(12))
ports, err := zizek.Draw(tc, zizek.Sets(octet).MinSize(1).MaxSize(5))
headers, err := zizek.Draw(tc, zizek.Maps(zizek.Text().MinSize(1).MaxSize(6), octet))
```

- **`zizek.Lists(element)`** — a variable-length `[]T`, bounded with `MinSize` /
  `MaxSize`.
- **`zizek.Sets(element)`** — the same, with pairwise-distinct elements. Go has
  no set type, so the result is a `[]T`, which is also the shape the shrinker
  deletes from.
- **`zizek.Maps(keys, values)`** — a `map[K]V` with distinct keys and values
  that may repeat.
- **`zizek.UniqueLists(element, key)`** — distinctness under any projection:

  ```go
  users, err := zizek.Draw(tc, zizek.UniqueLists(
      userGenerator,
      func(user User) string { return user.Email }, // distinct by email
  ).MaxSize(4))
  ```

The length is never a choice of its own. Each iteration draws one "another
element?" boolean inside the element's own span, which is what makes deletion
and truncation both fall out of ordinary shrinking.

Distinctness costs the *element*, not the example: a value repeating one
already drawn has its choices discarded and another candidate is drawn. Asking
for more distinct values than the element generator can produce rejects the
example rather than looping, so
`zizek.Sets(zizek.Integers().MinValue(0).MaxValue(1)).MinSize(5)` ends the run
with the `filter_too_much` health check instead of hanging.

`zizek.Collect` and `zizek.CollectUnique` are the same machinery exposed
directly, for a custom collection type that wants element spans without going
through `Lists`:

```go
values, err := zizek.Collect(tc, octet, 1, 4, true, "packet.byte")
```

The `hasMax` flag distinguishes "at most 4" from "unbounded"; the label names
the span each element occupies.

### Combinators

None of the wrappers below records a choice of its own except `SampledFrom` and
`OneOf`, which each record exactly one integer. Composing them therefore leaves
the reproduce blob byte-compatible with the other language packages, and leaves
shrinking as sharp as it was on the parts.

| Combinator | Purpose |
|------------|---------|
| `zizek.Just(value)` | Always this value, recording nothing. `Just[any](nil)` is the `none` of the dynamic packages. |
| `zizek.SampledFrom(values)` | One of a fixed slice, shrinking towards the first. |
| `zizek.OneOf(generators...)` | One of several generators, shrinking towards the first. |
| `zizek.Map(source, transform)` | Apply a function to each drawn value. |
| `zizek.Filter(source, predicate)` | Keep only values satisfying a predicate. |
| `zizek.FlatMap(source, transform)` | Build the next generator from the drawn value — monadic bind. |
| `zizek.Tuples(parts...)` / `zizek.AsAny(g)` | A heterogeneous `[]any`, since Go has no variadic type parameters. |
| `zizek.Composite(build)` | Draw as you go, in a function that gets the test case. |

The semantics are shared across languages and the spelling is native: Go has
no method-level type parameters, so `Map` and `Filter` are functions taking the source generator
rather than methods on it.

```go
verbs := zizek.SampledFrom([]string{"GET", "PUT", "DELETE"})
paths := zizek.Map(zizek.Text().MinSize(1).MaxSize(8), func(name string) string {
	return "/" + name
})
evenRetries := zizek.Filter(
	zizek.Integers().MinValue(0).MaxValue(9),
	func(value int64) bool { return value%2 == 0 },
)
optionalBody := zizek.OneOf(
	zizek.Just[*string](nil), // shrinks towards the first branch: no body
	zizek.Map(zizek.Text().MaxSize(6), func(text string) *string { return &text }),
)
```

`Filter` redraws a bounded number of times and then *rejects the example* the
way a failed assumption does — it never fails the property, and it never loops
forever. A predicate that passes rarely is better expressed by generating from
a narrower source: filtering `Integers()` down to "even" is fine, filtering it
down to "prime and above 10^9" will spend the run's rejection budget.

`FlatMap` is for a shape that depends on a drawn value. Both draws share one
`flat_map` span, so the shrinker can simplify or delete the unit together:

```go
matrices := zizek.FlatMap(
	zizek.Integers().MinValue(1).MaxValue(4),
	func(width int64) zizek.Generator[[]int64] {
		return zizek.Lists(octet).MinSize(int(width)).MaxSize(int(width))
	},
)
```

`Composite` is the general escape hatch and usually the most readable one in
Go: the build function receives the test case, so it can draw, assume, and note
exactly like a property. It is also how you build a struct, since Go has no
`builds`:

```go
type Window struct{ Low, High int64 }

var windows = zizek.Composite(func(tc *zizek.TestCase) (Window, error) {
	low, err := zizek.Draw(tc, zizek.Integers().MinValue(0).MaxValue(100))
	if err != nil {
		return Window{}, err
	}
	high, err := zizek.Draw(tc, zizek.Integers().MinValue(low).MaxValue(100))
	return Window{Low: low, High: high}, err
})
```

Run [`examples/combinators`](examples/combinators) to watch a four-field
request shrink through all of this at once: minimal path, minimal retry count,
and the verb and body the failure actually needs left standing.

### Recursive generators

`zizek.Recursive(leaf, branch)` builds trees and other recursively-defined
values. `leaf` produces a base value. `branch` receives a generator for child
subtrees and returns a generator that combines one or more of them into the
same result type:

```go
type Tree struct {
	Value       int64
	Left, Right *Tree
}

leaves := zizek.Map(
	zizek.Integers().MinValue(0).MaxValue(9),
	func(value int64) *Tree { return &Tree{Value: value} },
)

trees := zizek.Recursive(
	leaves,
	func(subtrees zizek.Generator[*Tree]) zizek.Generator[*Tree] {
		return zizek.Composite(func(tc *zizek.TestCase) (*Tree, error) {
			left, err := zizek.Draw(tc, subtrees)
			if err != nil {
				return nil, err
			}
			right, err := zizek.Draw(tc, subtrees)
			if err != nil {
				return nil, err
			}
			return &Tree{Left: left, Right: right}, nil
		})
	},
).MaxDepth(6).MaxLeaves(32)

tree, err := zizek.Draw(tc, trees)
```

The defaults are a maximum branch depth of 32 and at most 100 leaves.
`MaxDepth(0)` produces leaves only; `MaxLeaves` bounds the total base values,
not the number of branch nodes. An attempt that exceeds the leaf budget is
discarded and retried with a lower branching probability. If bounded retries
cannot construct a value, the example is rejected like a failed assumption.

Every subtree occupies a `recursive` span. The shrinker can therefore replace
an outer tree with one of its descendants, hoisting a deeply nested witness to
the root before simplifying its leaves. A leaf forced by `MaxDepth` is still a
recorded decision, so replay and the shrinker's choice positions stay aligned.

### Shrink targets

By default integers shrink toward 0 (clamped into the interval). With
`ShrinkTowards`, the reported counterexample is the failing value closest to
your target rather than the smallest one:

```go
value, err := zizek.Draw(
	tc,
	zizek.Integers().MinValue(-1000).MaxValue(1000).ShrinkTowards(500),
)
// if the property fails for value > 700, the failure shrinks to exactly 701
```

Reach for it when zero is not the interesting origin — a timestamp shrinking
towards "now", a temperature towards freezing, a version towards the one you
ship. Run [`examples/tour`](examples/tour) to watch this happen.

## Assumptions

`tc.Assume(condition)` rejects the current example when the condition is
false. Rejection is control flow, not failure: `Check` keeps generating until
it has run `Examples` valid examples or exceeds the adaptive rejection budget
`MaxRejections + 100 * valid`, at which point it returns
`*zizek.TooManyRejections`. Because every valid example earns more allowance,
a property whose valid rate is roughly 1% keeps making progress.

```go
value, err := tc.Integer(0, 100, 0)
if err != nil {
	return err
}
tc.Assume(value%2 == 0) // odd draws are discarded, not failed
```

Shrinking respects assumptions too: a shrink candidate that gets rejected is
skipped, so the shrunk counterexample always satisfies every assumption.

`Assume` panics internally with a sentinel that `Check` recovers, which is why
it needs no error return and why code after a false `Assume` does not run. Two
consequences worth knowing: don't wrap a property body in a `recover()` that
swallows unknown panics, and prefer narrowing the generator over assuming when
you can — an assumption that rejects 99 examples in 100 is a health-check
failure waiting to happen, while a bound on the generator costs nothing.

## Coverage assertions

A property can pass vacuously: the branch it exists to test never executes,
and the run goes green. Four calls, split by what they judge — two judge the
current example, two judge the completed run.

### `Always` and `Unreachable` — the example

```go
tc.Always("balance is conserved", total == expected)
if len(queue) > capacity {
	tc.Unreachable("queue overflow")
}
```

`Always` fails the example the moment its condition does not hold, and
`Unreachable` fails the example that reaches it — both are ordinary property
failures that shrink and replay, with the call site as the bug's origin. So
`Unreachable`'s counterexample is the *simplest input reaching the branch*,
which is exactly the question "can this happen?" deserves as an answer. A
holding `Always` also counts as covering its label.

Both abort the example the way a panic does, so the report reads
`property panicked: always: "clamped into range" did not hold` — the label is
what names the assertion in the failure. On its own an `Always` call site may
legitimately never execute ("always or unreachable"); listing its label in
`Config.Require` additionally fails a run that never reached it.

### `Sometimes` and `Cover` — the run

```go
value, err := tc.Integer(0, 1000, 0)
if err != nil {
	return err
}
tc.Sometimes("large input", value > 900)      // must hold in at least one example
tc.Cover("small input", value < 100, 20)      // ...in at least 20% of them
```

The verdict is per run, not per example: a false condition never fails the
current example, only a completed run in which it never held:

```
coverage: "large input" never held in 200 valid examples
```

A label counts once per example however many times its call site runs, and
observations made by rejected or overrun examples are discarded with the rest
of the example.

### `Config.Require` — the label that never registered

A call site registers its label when it first runs, so a `Sometimes` placed
*inside* the untested branch cannot report the branch that never executed.
Declare those up front instead:

```go
err := zizek.Check(property, zizek.Config{
	Require: []zizek.Coverage{
		{Label: "overflow path"},          // must hold at least once
		{Label: "cache hit", Percent: 30}, // in at least 30% of valid examples
	},
})
```

A missed requirement is reported as `*zizek.FailedCoverage` naming the label
and the observed counters:

```
coverage: "left below the ceiling" held in 33 of 200 valid examples, below the required 60%
```

There is no counterexample and so no blob or shrinking — the failure is the
absence of an example; rerunning with the same seed reproduces the identical
verdict. A run that finds a real counterexample reports that instead, and
reproduce mode skips coverage entirely. The accept rule is compiled from
`ir/lifecycle.zir`, so every language agrees on it.

A holding `Always` counts as covering its label, which is what separates it
from Antithesis's `Always`: the bare call permits a site that never runs
("always or unreachable"), and listing the label in `Require` additionally
demands the run reach it.

[`examples/coverage_assertions`](examples/coverage_assertions) runs all five
scenarios against one broken clamp, including the vacuous pass that only
`Sometimes` catches and the dead branch only `Require` can name.

## Notes

`tc.Note(message)` attaches a diagnostic line to the current example. Notes
are collected per example and reported on the shrunk failure — both in
`failure.Notes` and in the rendered `Trace:` section — so they describe the
*minimal* example, not the noisy first one.

```go
tc.Note(fmt.Sprintf("transfer %d from %q to %q", amount, from, to))
```

Notes are the cheapest debugging tool here, and they cost nothing on a passing
run. Note the *decisions* rather than the values you can already see in the
blob: which branch was taken, what the model expected, why the rule was
skipped. `tc.Notes()` returns the current example's notes if a stateful runner
or generator wants to inspect them.

## Failures, blobs, and replay

A shrunk failure carries a portable, base64 reproduce blob:

```go
var failure *zizek.Failure
if errors.As(err, &failure) {
	// Replays exactly the recorded choices — nothing is random.
	err = zizek.Check(property, zizek.Config{Reproduce: failure.Blob})
}
```

Blobs encode the recorded choice sequence, not the RNG seed, so they replay
bit-exactly across machines *and languages*: a blob printed by the Go runner
replays in the Python, Ruby, and Rust packages (the shared fixture
`AAEAAAAACgEAAAAF` replays the integer `5` in all four test suites, and the
ledger/queue examples pin cross-language blob fixtures of their own).

Replay is defensive:

- a structurally invalid blob returns `*zizek.InvalidReplay`;
- a blob that ends before the property stops drawing returns
  `*zizek.ReplayOverrun`;
- a choice of the wrong kind (or out of range) puns to the requested draw's
  unit value instead of failing, so replay stays deterministic while you edit
  the property.

A pun keeps the replay running, but it means the values you get back are not
the ones the blob recorded — and nothing in the values themselves says so. It
bites hardest when a draw's bounds depend on earlier draws, because then the
same blob can fit one run and pun on the next. Every replay therefore counts
its puns: `failure.Puns` reports how many, and a non-zero count adds a warning
line to the rendered failure. Treat it as "this blob no longer reproduces what
it recorded", not as a hard error.

The next section is the way out of that trade-off.

## Keyed draws

Positional replay gives a draw its meaning from how many draws came before it,
which is exactly right until the property changes shape: insert one new draw
and every recorded choice after it answers the wrong question. A keyed draw is
addressed by a *name* instead, so a blob keeps meaning what it recorded while
the property around it evolves:

```go
timeout, err := tc.IntegerKeyed(1, 30, 0, zizek.Key("timeout"))
retries, err := tc.IntegerKeyed(0, 9, 0, zizek.Key("retries"))
```

Every primitive draw has a keyed twin: `IntegerKeyed`, `BooleanKeyed`,
`WeightedKeyed`, `BytesKeyed`, `StringKeyed`, `FloatKeyed`. The key itself is a
`uint64` from one of two helpers:

- **`zizek.Key(components...)`** names a decision made at most once per
  example. Components are strings, byte slices, or integers in any mix —
  `zizek.Key("read", conn, seq)` is the shape this is for — hashed by the
  compiled encoding (`ir/keys.zir`), so the same name means the same entry in
  every language package.
- **`tc.NextKey(components...)`** appends an execution-local ordinal, counted
  *within that name*, for a decision made repeatedly. `tc.NextKey("read", conn)`
  numbers that connection's reads independently of every other sequence, so
  inserting a read on one connection shifts nothing on the others.

```go
connection, err := tc.IntegerKeyed(1, 4, 0, zizek.Key("connection"))
if err != nil {
	return err
}
for range 3 {
	payload, err := tc.BytesKeyed(0, 16, tc.NextKey("read", connection))
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("read %d bytes", len(payload)))
}
```

On replay a keyed draw looks its name up wherever it sits in the stream, and
the positional cursor steps over keyed entries, so the two addressing modes
never shift each other. A name the blob does not carry means the property
changed, not that the blob is malformed: the draw uses its simplest value, and
the failure counts and reports the miss (`failure.KeyMisses`, plus a warning
line) instead of returning an error.

`NextKey`'s counter is execution-local by construction. A counter that outlived
one example would produce different keys on the replay than on the run that
recorded it, and every draw would miss.

[`examples/keyed_draws`](examples/keyed_draws) runs the whole contrast: the
same edit to a property — one new draw inserted between two existing ones —
leaves the keyed blob reproducing its failure with one counted miss, while the
positional blob shifts by one and dies with `ReplayOverrun`.

## The example database

Set `Database` and a failure outlives the run that found it:

```go
func TestSplitIsExact(t *testing.T) {
	err := zizek.Check(splitIsExact, zizek.Config{
		Database:    zizek.NewDirectoryDatabase(".zizek/examples"),
		DatabaseKey: "split_is_exact",
	})
	if err != nil {
		t.Fatal(err)
	}
}
```

The next run replays every stored entry for that key *before* it generates
anything, so a bug found yesterday reproduces on the first example — no search,
no seed. An entry that no longer reproduces — because the bug was fixed — is
deleted on the replay that failed to reproduce it, so the directory keeps
itself current. So is one that no longer parses.

The database is off until asked for: a library call that wrote into your
working directory unasked would be a surprise. Two backends ship —
`zizek.NewDirectoryDatabase(path)` (an empty path selects
`zizek.DefaultDatabasePath`, `.zizek/examples`) and
`zizek.NewInMemoryDatabase()`, which forgets when the process exits and is what
a test *about* the database wants.

`zizek.Database` is a three-method interface over property *names*, so a
backend of your own can key however its store prefers:

```go
type Database interface {
	Fetch(key string) [][]byte
	Save(key string, value []byte)
	Delete(key string, value []byte)
}
```

None of the three returns an error, deliberately: every operation is
best-effort, because a database that cannot be read or written must not fail a
test that would otherwise pass.

What makes this more than a cache is the layout. What is stored is the **raw
choice stream**, not the base64 blob, in `<root>/<key digest>/<stream digest>`,
with both digests being the FNV-1a hash `ir/keys.zir` already computes for
keyed draws — `zizek.DatabaseDigest` exposes it:

```
.zizek/examples/
  61343319301144f4/          # the key's digest
    .name                    # the property's name, for a human
    15b066cccee0c4f2         # a stored stream
```

The Python and Ruby packages write that directory identically, so a
counterexample your Go service's suite found replays in your Ruby client's
suite out of one checked-in directory — provided both name the property the
same with `DatabaseKey`. Left unset, the key is derived from the property's own
Go function name: stable across builds, and meaningless to another language.

Replays count as attempts but not toward `Examples`, the coverage verdict, or
the health checks, all of which are statements about what the run *explored*. A
`Reproduce` run touches the database not at all.

[`examples/database_workflow`](examples/database_workflow) runs the whole cycle
in a temporary directory — find and store, replay on the first attempt of the
next run under a different seed, and watch the fixed bug's entry collect
itself. [`examples/crosslang_database`](examples/crosslang_database) is the
other half: it replays, in Go, a stream that Python wrote.

## Shrink costs

The shrinker's default order answers "what is the simplest *input* that
fails". A fault-injecting property usually wants a different question answered
first: "how few *faults* suffice?" Under the default order, turning one
injected fault off is worth exactly as much as making one integer a little
smaller, so the search may keep four faults because the stream around them
happens to be shorter.

`tc.Cost(label, amount)` declares what an example spent:

```go
inject, err := tc.BooleanKeyed(tc.NextKey("inject", site))
if err != nil {
	return err
}
if inject {
	tc.Cost("reorder", 1) // this example spent one fault
	reorder(queue)
}
```

The shrinker then orders candidates by **total cost first** — a candidate
injecting one fewer fault wins even when its choice stream is longer — with the
ordinary stream order as the tiebreak. A property that never calls `Cost`
shrinks exactly as it did before the feature existed. The amount must be
non-negative (a negative one panics; it is a programming error, not a property
failure), and costs accumulate across a whole example, including anything spent
inside [cloned streams](#cloned-streams-advanced).

`failure.Costs` reports the per-label breakdown of the example finally
reported, and `tc.Costs()` returns a snapshot mid-example.

## Targeted generation

`Cost` orders shrinking after a failure; `Target` steers generation toward one.
A property records one finite score per label per example, and the run steers
generation toward examples that score higher:

```go
value, err := tc.Integer(0, 100_000, 0)
if err != nil {
	return err
}
tc.Target(float64(value))
if value >= 99_990 {
	return errors.New("overflowed")
}
```

The run keeps the best-scoring valid example per label and, on a schedule,
hill-climbs its recorded choices — so a failing band at the far end of the
scored quantity is found in a fraction of the examples random search needs. A
failing climb trial shrinks and reports like any other counterexample, and
nothing new enters the blob.

Target several quantities with `tc.TargetLabelled(score, "depth")`. Each label
may be observed at most once per example, and every score must be finite —
both violations panic, because a NaN score or a silently overwritten
observation would corrupt the search rather than fail one example. Score the
quantity you want *more* of: to find small values, target the negation.

## Structural spans (advanced)

Framework-level code can group choices into labelled ranges so the shrinker
can delete whole semantic units at once:

```go
tc.StartSpan("stateful.rule") // returns the span id, rarely needed
// ... draws ...
tc.StopSpan(false)     // pass true for a discarded/rejected attempt
spans := tc.Spans()    // snapshot for inspection
```

The state-machine runner uses spans for each rule, and `zizek.Lists` uses one
per element, which is why a 40-step failing run shrinks to the three or six
steps that matter and a 30-element list shrinks to the one element that does.
Most tests never call these directly.

Spans do one more thing: after ten valid examples, discovery may copy one
same-label span into another, which makes equality bugs between independent
64-bit draws practical to find at all.

```go
values := make([]int64, 2)
for index := range values {
	tc.StartSpan("operand")
	value, err := tc.Integer(math.MinInt64, math.MaxInt64, 0)
	tc.StopSpan(err != nil)
	if err != nil {
		return err
	}
	values[index] = value
}
if values[0] == values[1] && values[0] != 0 {
	return errors.New("equal nonzero operands")
}
```

Mutation participates only in discovery: the reported counterexample is a
normal portable choice stream, and normal shrinking minimizes it.

## Cloned streams (advanced)

`tc.CloneStream()` returns a child test case that draws from a stream of its
own. The clone takes one choice position in the parent and is framed there, so
the child's draws neither shift nor are shifted by the parent's:

```go
child, err := tc.CloneStream()
if err != nil {
	return err
}
value, err := child.Integer(0, 100, 0) // recorded inside the clone entry
```

Everything a run reports belongs to the *family* rather than to one stream:
notes, costs, coverage observations, and the 8,192-choice budget are shared, so
a clone cannot buy a budget of its own and everything it records reaches the
failure report. Clones may nest 100 deep, which is both Hegel's
`MAX_CLONE_DEPTH` and the wire format's own limit; a property that clones past
it has its example rejected rather than failed.

Shrinking descends into a clone rather than treating it as one atomic choice:
after the flat passes converge, each clone's stream is run through the same
compiled schedule, with every candidate spliced back into the parent and
replayed through the whole property. Without that pass, a failure that depends
on a cloned draw would report whatever value generation happened to find.

This is the framing and replay seam a concurrent runner would use — today the
child is drawn from sequentially, in the same goroutine. On replay, a clone
value at that position hands the child its recorded stream; any other value
puns to an empty child, which overruns on its first draw, so a property that
grew a clone after a blob was written reports an overrun rather than confident,
wrong values.

## Stateful testing

State machines are registered explicitly with typed rules and invariants. A
rule is a name and a function over your state and the test case; an invariant
has the same shape and is checked between rules:

```go
type Counter struct{ value int }

func increment(state *Counter, tc *zizek.TestCase) error {
	step, err := zizek.Draw(tc, zizek.Integers().MinValue(1).MaxValue(10))
	if err != nil {
		return err
	}
	state.value += int(step)
	return nil
}

func nonnegative(state *Counter, _ *zizek.TestCase) error {
	if state.value < 0 {
		return fmt.Errorf("counter went negative: %d", state.value)
	}
	return nil
}

func property(tc *zizek.TestCase) error {
	state := Counter{}
	machine := zizek.Machine[Counter]{
		Rules: []zizek.Rule[Counter]{
			{Name: "increment", Apply: increment},
		},
		Invariants: []zizek.Rule[Counter]{
			{Name: "nonnegative", Apply: nonnegative},
		},
	}
	return zizek.Run(&state, machine, tc)
}
```

Method expressions keep this tidy when the state is a struct with methods:
`{Name: "push", Apply: (*queueTest).push}`.

`Run` checks every invariant once before the first rule, then draws steps until
it stops, picking a rule per step and re-checking every invariant after each
successful rule. There is no step budget drawn up front: each iteration draws
one "continue?" boolean, which is what makes the sequence shrinkable step by
step — its simplest value stops, so the simplest case runs a single step and
lowering any one of them truncates the run there. `zizek.Run` runs at most `zizek.DefaultStatefulStepCount` (50) successful
steps; `zizek.RunSteps(state, machine, tc, n)` sets that bound yourself.

A rule that calls `tc.Assume(false)` is recorded as rejected and skipped
without failing the run — it spends an attempt but not a step, so a machine
whose rules have preconditions still gets its full run. That is how "pop is
only legal on a non-empty queue" is written:

```go
func (state *queueTest) pop(tc *zizek.TestCase) error {
	tc.Assume(len(state.model) != 0)
	// ...
}
```

Each rule runs inside its own `stateful.rule` span, so structural shrinking
deletes whole steps: a 40-step failing run reduces to the three that matter,
and the notes `Run` records (`step 3: pop`) describe that minimal sequence.
Model-based testing falls out naturally — keep a simple reference structure
beside the real one and let an invariant compare them, as
[`examples/queue`](examples/queue) does against a plain slice.

### Pools

`zizek.NewPool[T](tc)` holds values produced by earlier rules behind portable
choices, so shrinking and replay stay deterministic even though the values are
arbitrary Go data:

```go
accounts := zizek.NewPool[string](tc)
if err := accounts.Add(tc, "a"); err != nil {
	return err
}

account, err := zizek.Draw(tc, accounts.ValuesReusable()) // *T, stays in pool
ticket, err := zizek.Draw(tc, accounts.ValuesConsumed())  // T, removed
```

`Add` takes the test case because adding is a *draw*: the identifier comes from
`tc.FreshID()` and is recorded in the choice stream rather than handed out by a
host-side counter. Selecting is `tc.FromSet` over the live identifiers, which
records the identifier itself rather than a position. Together that is what
lets the shrinker delete an earlier addition without changing what every later
reference means — before stable identifiers, deleting one addition renumbered
every reference after it, so most deletions produced a different rule sequence
that no longer failed and shrinking stalled far from minimal.

Drawing from an empty pool rejects the current rule attempt (via `Assume`), so
rules that need prerequisites simply retry on another step. `pool.Len()` and
`pool.IsEmpty()` are there when a rule wants to decide for itself.

[`examples/ledger`](examples/ledger) is the end-to-end version: accounts in a
pool, transfers between drawn account names, an invariant on the conserved
total, and a cross-language blob fixture.

## Health checks

Four checks fail a run early rather than let it look like it tested something:

| Check | Fires when | Suppress with |
|-------|-----------|---------------|
| `filter_too_much` | 50 rejections before 10 valid examples | `SuppressFilterTooMuch` |
| `data_too_large` | 20 choice-budget overruns before 10 valid examples | `SuppressDataTooLarge` |
| `large_base_example` | the all-simplest input uses more than half the 8,192-choice budget, or overruns | `SuppressLargeBaseExample` |
| `too_slow` | generation takes more than one cumulative second in the first ten valid examples | `SuppressTooSlow` |

`Check` runs the all-simplest input first, as a normal attempt, which is what
`large_base_example` judges. `too_slow` counts monotonic time spent inside
`zizek.Draw` calls, not inside the code under test.

They arrive as `*zizek.FailedHealthCheck`, whose `Check` field carries the name
above. Each is a statement about the *generator*, not the code under test:
`filter_too_much` means the assumptions are doing work the generator should do,
`large_base_example` means even the simplest input is enormous. Suppress one
when you have decided the pattern is intended — a genuinely expensive generator
under `too_slow`, say — not to quiet a signal you have not read.

## What the Go package does not have

- **`Text()` alphabets by Unicode category.** Alphabets are interval sets built
  with `zizek.Alphabet` and `zizek.AlphabetRange`; "all letters" is not a
  built-in.
- **Concurrent clones.** `CloneStream` is the framing and replay seam a
  concurrent runner would use, but this shell draws the child inline.
- **A `builds`-style struct combinator.** `Composite` is the Go spelling; Go
  has no keyword arguments to build one from.
- **Typed heterogeneous tuples.** `Tuples` returns `[]any` because Go has no
  variadic type parameters.
- **Integrations with third-party test frameworks.** `zizek.Test` covers the
  standard `testing` package, and `Check` returns an error that any framework
  can report.

## Runnable examples

Run from the package root:

| Example | Features | Run |
|---------|----------|-----|
| [`examples/roundtrip`](examples/roundtrip) | text generator, notes, minimal shrinking, blob replay | `go run ./examples/roundtrip` |
| [`examples/tour`](examples/tour) | shrink targets, assumptions, boolean+bytes draws, bounded floats, rejection limits | `go run ./examples/tour` |
| [`examples/queue`](examples/queue) | state machine, rule assumptions, invariants, panic capture, span shrinking | `go run ./examples/queue` |
| [`examples/ledger`](examples/ledger) | full stateful slice: pools, text, cross-language blob fixture | `go run ./examples/ledger` |
| [`examples/die_hard`](examples/die_hard) | Hegel water-jug port, sequence discovery, Zizek-portable six-step fixture | `go run ./examples/die_hard` |
| [`examples/coverage_assertions`](examples/coverage_assertions) | `Always`, `Unreachable`, `Sometimes`, `Cover`, `Config.Require` | `go run ./examples/coverage_assertions` |
| [`examples/keyed_draws`](examples/keyed_draws) | keyed vs positional replay: the blob that survives editing the property | `go run ./examples/keyed_draws` |
| [`examples/database_workflow`](examples/database_workflow) | the example database: store, replay-first, self-clean | `go run ./examples/database_workflow` |
| [`examples/combinators`](examples/combinators) | `SampledFrom`, `Map`, `Filter`, `OneOf`, `Composite`, shrinking through composition | `go run ./examples/combinators` |
| [`examples/crosslang_database`](examples/crosslang_database) | replaying a stream another language stored | `go run ./examples/crosslang_database <directory> <key>` |
| [`examples/hypothesis_quickstart`](examples/hypothesis_quickstart), [`examples/min_stack`](examples/min_stack), [`examples/stack`](examples/stack), [`examples/multi`](examples/multi) | ports of the Hypothesis and Hegel introductory examples | `go run ./examples/<name>` |

Each example is deterministic, asserts its own expected outcome in an
accompanying test (`go test ./examples/...`), and prints the shrunk failure
with its reproduce blob. `roundtrip`, `tour`, and `queue` each shrink to a blob
that is identical in Go, Python, and Ruby. `crosslang_database` is the
exception: it has no test, and it expects a database directory that another
language package has already written, under the key it is given. The Die Hard port shares its exact shrunk blob with
Ruby, Rust, and Zig. Hegel's own replay blobs use a separate, version-specific
encoding.

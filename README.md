<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/zizek-go-logo-dark.svg">
    <img src="docs/assets/zizek-go-logo.svg" alt="Zizek.go" width="320">
  </picture>
</p>

# Zizek for Go

This module is the native Go runtime for Zizek property testing, based on
[Hegel](https://hegel.dev) and
[Hypothesis](https://hypothesis.readthedocs.io/en/latest/) and keeping them
as oracle implementations. It explores generated examples, records every
choice, shrinks the first failure, and returns a failure carrying a portable
reproduce blob.

Roughly three quarters of the core behavior is compiled from the Hegel engine
into `zizek_gen.go`; the rest is a curated shell that aims to be idiomatic Go.
[dev.md](dev.md) describes the split.

> ⚠️ **Status: experimental.** Behavior and the public API can still change and
> be tweaked between releases.

## Install

```bash
go get github.com/wallyqs/zizek.go
```

The module path ends in `.go` but the package is named `zizek`, so the import
stays bare and the identifier is `zizek`:

```go
import "github.com/wallyqs/zizek.go"
```

## Quick start

A property is a function from a test case to an error. `zizek.Check` runs it,
and returns a shrunk `*zizek.Failure` when it finds a counterexample:

```go
package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

func property(tc *zizek.TestCase) error {
	value, err := zizek.Draw(tc, zizek.Integers().MinValue(-100).MaxValue(100))
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("value %d", value))
	if value >= 5 {
		return fmt.Errorf("value must be less than five: %d", value)
	}
	return nil
}

func main() {
	err := zizek.Check(property, zizek.Config{Examples: 200, Seed: 1})

	var failure *zizek.Failure
	if errors.As(err, &failure) {
		fmt.Println(failure)

		// Runs exactly the recorded, shrunk example and fails the same way.
		err = zizek.Check(property, zizek.Config{Reproduce: failure.Blob})
	}
}
```

The failure says which values failed, not just that something did:

```
value must be less than five: 5
reproduce with zizek.Check(property, zizek.Config{Reproduce: "AAEAAAAACgEAAAAF"})

Trace:
- value 5
```

## Generators

Generators compose with `Map`, `Filter`, `OneOf`, and `Composite`, and add no
hidden choices, so a composed value shrinks and replays like a plain one:

```go
type Request struct {
	Method   string
	Resource *string
	Retries  int64
}

var requests = zizek.Composite(func(tc *zizek.TestCase) (Request, error) {
	method, err := zizek.Draw(tc, zizek.SampledFrom([]string{"GET", "POST", "DELETE"}))
	if err != nil {
		return Request{}, err
	}
	resource, err := zizek.Draw(tc, zizek.OneOf(
		zizek.Just[*string](nil),
		zizek.Map(zizek.Text().MinSize(1).MaxSize(8), func(s string) *string { return &s }),
	))
	if err != nil {
		return Request{}, err
	}
	retries, err := zizek.Draw(tc, zizek.Filter(
		zizek.Integers().MinValue(0).MaxValue(10),
		func(n int64) bool { return n%2 == 0 },
	))
	return Request{Method: method, Resource: resource, Retries: retries}, err
})

err := zizek.Check(func(tc *zizek.TestCase) error {
	request, err := zizek.Draw(tc, requests)
	if err != nil {
		return err
	}
	if request.Method == "DELETE" && request.Resource == nil {
		return errors.New("DELETE must name a resource")
	}
	return nil
}, zizek.Config{Examples: 200, Seed: 7})
```

## Go tests

Inside a Go test, `zizek.Test` runs a body that takes `*zizek.T`, an alias of
`*zizek.TestCase`, and reports the shrunk failure through `testing.T`.
`zizek.MustDraw` draws without an error return:

```go
func TestSplitIsExact(t *testing.T) {
	zizek.Test(t, func(zt *zizek.T) {
		total := zizek.MustDraw(zt, zizek.Integers().MinValue(0).MaxValue(10_000))
		people := zizek.MustDraw(zt, zizek.Integers().MinValue(1).MaxValue(50))

		var sum int64
		for _, share := range Split(total, people) {
			sum += share
		}
		zt.Always("the split is exact", sum == total)
	})
}
```

A `Split` that drops the remainder fails as
`property panicked: always: "the split is exact" did not hold`, followed by
the reproduce line.

## Stateful testing

A state machine is a typed set of rules and invariants over your own state; a
failure shrinks to the shortest sequence of steps that still fails:

```go
type Counter struct{ Value int64 }

func increment(state *Counter, tc *zizek.TestCase) error {
	amount, err := zizek.Draw(tc, zizek.Integers().MinValue(0).MaxValue(10))
	if err != nil {
		return err
	}
	state.Value += amount
	return nil
}

func decrement(state *Counter, tc *zizek.TestCase) error {
	tc.Assume(state.Value > 0)
	state.Value--
	return nil
}

func nonnegative(state *Counter, _ *zizek.TestCase) error {
	if state.Value < 0 {
		return errors.New("counter went negative")
	}
	return nil
}

func property(tc *zizek.TestCase) error {
	return zizek.Run(&Counter{}, zizek.Machine[Counter]{
		Rules: []zizek.Rule[Counter]{
			{Name: "increment", Apply: increment},
			{Name: "decrement", Apply: decrement},
		},
		Invariants: []zizek.Rule[Counter]{
			{Name: "nonnegative", Apply: nonnegative},
		},
	}, tc)
}

err := zizek.Check(property, zizek.Config{Examples: 500})
```

## Learn more

- [GUIDE.md](GUIDE.md) walks through every feature with runnable examples:
  configuration and the error verdicts, the example database, keyed draws,
  coverage assertions, pools, and more.
- [`examples/`](examples) holds runnable examples; each is listed in the
  guide.
- [dev.md](dev.md) covers the core/shell split, verification, and the
  upstream example ports.
- [CHANGELOG.md](CHANGELOG.md) records each release.

## Functionality available now

- Sequential property exploration with deterministic seeds.
- Recorded integer, boolean, byte, IEEE-754 float, and Unicode text choices.
- Uniform `int64` generation for narrow ranges, and a per-case swarm profile
  that clusters wide operands around endpoints, interesting values, or diffuse
  fixed-width stress constants.
- Generic, typed generators through `zizek.Draw`, with `zizek.MustDraw` for
  bodies that have no error return.
- Assumption-based rejection with configurable rejection limits.
- Failure capture from returned errors or panics.
- Variable-length `zizek.Lists` whose elements shrink by deletion, plus
  `zizek.Sets`, `zizek.Maps`, and `zizek.UniqueLists` for distinct elements.
- Bounded `zizek.Recursive` generators for trees and other recursively-defined
  values, with subtree-aware shrinking.
- Per-choice shrinking plus element-span and rule-span deletion.
- Cloned streams: `tc.CloneStream()` hands back a child test case drawing from a
  stream of its own, framed inside the parent's blob and shrunk by the same
  schedule as any other stream.
- Portable failure blobs with exact replay, choice punning, and overrun
  detection, with punned draws counted and reported on the failure.
- An example database that stores counterexamples between runs and replays them
  before generating — in a layout the Python and Ruby packages share.
- Manual state machines with an initial invariant check, invariants after every
  successful rule, rejected-rule handling, notes, and a default budget of 50
  steps (rejected rules spend an attempt, not a step).
- Reusable and consumed `Pool[T]` selections addressed by generated stable identifiers.
- `zizek.Test` for running a property inside a Go test, alongside
  `zizek.Check` for the verdict as an error value.
- A complete ledger example that finds the deliberate overdraft bug and
  shrinks it to two account creations plus one transfer of `1`.
- Runnable ports of Hypothesis's quickstart and Hegel's stack, minimum-stack,
  multiple-origin, and Die Hard examples.

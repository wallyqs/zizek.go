# Changelog

All notable changes to the Go package are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[semantic versioning](https://semver.org/spec/v2.0.0.html) — while the version
is `0.x`, the API may change in any release.

## [0.1.1] - 2026-10-28

First release of the Go package.

### Added

- Sequential property exploration with deterministic seeds.
- Recorded integer, boolean, byte, IEEE-754 float, and Unicode text choices,
  with uniform `int64` generation for narrow ranges and a per-case swarm
  profile that clusters wide operands around endpoints, interesting values, or
  diffuse fixed-width stress constants.
- Generic, typed generators through `zizek.Draw`, with `zizek.MustDraw` for
  bodies that have no error return.
- Variable-length `zizek.Lists` whose elements shrink by deletion, plus
  `zizek.Sets`, `zizek.Maps`, and `zizek.UniqueLists` for distinct elements.
- Bounded `zizek.Recursive` generators for trees and other recursively-defined
  values, with subtree-aware shrinking.
- Per-choice shrinking plus element-span and rule-span deletion.
- Portable failure blobs with exact replay, choice punning, and overrun
  detection, with punned draws counted and reported on the failure.
- Coverage assertions that fail a completed run for a condition it never
  produced.
- Cloned streams: `tc.CloneStream()` hands back a child test case drawing from
  a stream of its own, framed inside the parent's blob and shrunk by the same
  schedule.
- An example database that stores counterexamples between runs and replays
  them before generating, in a layout the Python and Ruby packages share.
- State machines with an initial invariant check, invariants after every
  successful rule, rejected-rule handling, notes, and a default budget of 50
  steps (rejected rules spend an attempt, not a step), plus reusable and
  consumed `Pool[T]` selections.
- Assumption-based rejection with configurable limits, and failure capture
  from returned errors or panics.
- Runnable examples, including a ledger acceptance example and ports of
  Hypothesis's quickstart and Hegel's stack, minimum-stack, multiple-origin,
  and Die Hard examples.

[0.1.0]: https://github.com/wallyqs/zizek.go/releases/tag/v0.1.0


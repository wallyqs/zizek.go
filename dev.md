# Zizek for Go: development notes

How the package is laid out and checked, for working on it rather than using
it. The [README](README.md) covers installation and the basics, and
[GUIDE.md](GUIDE.md) every feature.

## Core and shell

Everything is one package, `zizek`, at the module root.

- `zizek_gen.go` is generated from ZizekIR and owns deterministic choice
  indexing, boundary candidates, structural rewrites, generation and
  mutation scheduling, stateful step and pool-position rules, serialization,
  replay, and the shrink search's decisions: which request to make next, the
  candidate schedule, the cursor arithmetic, and the accept rule. It is pure —
  plain data in, plain data out, with no package state, randomness, I/O,
  errors, or panics; it imports only `math` and `math/bits`. Do not edit it.
- `shell.go` owns every effect: random exploration, the clock, invoking user
  code, notes, and Go error and panic mapping — including deriving the opaque
  bug-origin token (a panic's site, or a returned error's unwrapped type
  chain) that the generated accept rule compares. It also drives the search
  and shrink loops, holding their cursors and schedules, and asks the core for
  each decision. That loop is written again in every language package; the
  shared conformance tables are what keep the packages in agreement.
- `generators.go` is the typed generator facade: `Draw`, `MustDraw`, and the
  immutable generator values.
- `stateful.go` owns host values and invokes user rules and invariants, asking
  the generated core for the step and pool decisions.
- `database.go` is the example database: the `Database` interface, the
  directory and in-memory backends, and the portable digest rule.
- `testing.go` is the `testing` integration, `Test` and the `T` alias.
- `examples/` holds the runnable feature and upstream-port examples;
  `examples/ledger` is the end-to-end stateful acceptance example.
- `zizek_test.go` and `hegel_examples_test.go` are the handwritten suite and
  the GoDoc examples. The `*_gen_test.go` files are generated from the shared
  conformance declarations; do not edit them either.

## Verify

From the package root:

```bash
go build ./...
go vet ./...
go test ./...
go run ./examples/ledger
```

`go test ./...` also runs each example's own test, which pins its shrunk
outcome and blob. The shell and database conformance tests additionally read
shared tables that live in the Zizek source tree and are not shipped with
this package; where the tables are absent, those tests skip and say so. The
Zizek source tree also checks the generated core against golden vectors from
an independent Rust reference.

To work on the package from an application, point the application's `go.mod`
at a local checkout:

```go
require github.com/wallyqs/zizek.go v0.0.0

replace github.com/wallyqs/zizek.go => ../zizek.go
```

## Example database layout

What is stored is the **raw choice stream**, not the base64 blob, in
`<root>/<key digest>/<stream digest>` with both digests being the FNV-1a hash
`ir/keys.zir` already computes for keyed draws; `zizek.DatabaseDigest` exposes
it. Each key directory also holds a `.name` file with the property's name, for
a human. That is what makes the directory portable: the Python and Ruby
packages write the same layout, so a counterexample found by one replays in
the others. Sharing needs a `DatabaseKey` both sides agree on; left empty, the
key is derived from the property's own function name (or, under `zizek.Test`,
the test's name), which is stable across builds and meaningless to another
language.

An entry that no longer parses is deleted along with ones that no longer
reproduce. Replays count as attempts but not toward `Examples`, the coverage
verdict, or the health checks, all of which are statements about what the run
explored. A `Reproduce` run touches the database not at all. Every operation
is best-effort, which is why none of the three `Database` methods returns an
error: an unwritable directory never fails a test that would otherwise pass.

[`examples/crosslang_database`](examples/crosslang_database) is the Go half
of a cross-language check: given a directory another package wrote and the
key it used, it replays the stored stream with `Examples: 1` and prints the
blob, which must match the one the other packages print.

## Cloned stream framing

A clone takes one choice position in its parent and is framed there as
`[5][u32 child count][children]`, so the child's draws neither shift nor are
shifted by the parent's. Clones nest at most 100 deep, which is both Hegel's
`MAX_CLONE_DEPTH` and the wire format's own limit.

## Upstream example ports

The package carries runnable ports of the examples used to compare Zizek with
Hypothesis and Hegel:

- [`examples/hypothesis_quickstart`](examples/hypothesis_quickstart) ports
  Hypothesis's introductory bounded-integer property. It shrinks
  `value >= 50` to exactly `50` and replays the portable blob.
- [`examples/min_stack`](examples/min_stack) ports Hegel's deliberately broken
  minimum stack. It draws unbounded integers and shrinks the duplicate-minimum
  bug to `push(0), push(0), pop`.
- [`examples/stack`](examples/stack) ports Hegel's passing stack model, with a
  rejected-rule precondition.
- [`examples/multi`](examples/multi) ports Hegel's two-assertion example and
  verifies that shrinking preserves the failure origin selected during
  discovery.
- [`examples/die_hard`](examples/die_hard) ports Hegel's water-jug state
  machine. It finds the classic solution and exercises generation-phase span
  mutation.
- [`examples/ledger`](examples/ledger) is the Hegel ledger port. It shrinks
  the overdraft bug to two account creations and one transfer of `1`, and
  replays a ledger fixture blob that every other language package replays
  too. The trace, with the invariant checks left out, reads:

  ```text
  - step 1: create_account
  - create account "0"
  - step 2: create_account
  - create account "1"
  - step 3: transfer
  - transfer 1 from "1" to "0"
  ```

Run them from the package root:

```bash
go run ./examples/hypothesis_quickstart
go run ./examples/min_stack
go run ./examples/stack
go run ./examples/multi
go run ./examples/die_hard
go run ./examples/ledger
```

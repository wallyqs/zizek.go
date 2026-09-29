// Port of Hegel's Die Hard water-jug state machine. The "failure" is a
// solution: generation finds operations leaving exactly four litres in the
// five-litre jug, then structural shrinking removes every unnecessary step.
package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

const legacyZizekDieHardFailure = "AAwAAAABAAAKAQAAAAEBAAAKAQAAAAUBAAAKAQAAAAIBAAAKAQAAAAUBAAAKAQAAAAEBAAAKAQAAAAU="
const portableDieHardFailure = "AXicbYo5CgAwDMPkQpeu/f9fc0AIgXixLfQBiSciWX6L3J5npdM1FrQAhg=="
const hegelZlibDieHardFailure = "AXicbYoxCgBACMPSg1tc/f9fdRBFcCgNIQ4ILEed1OYPvtPuNgAVzACA"

type dieHard struct {
	small int
	big   int
}

func (state *dieHard) fillSmall(_ *zizek.TestCase) error {
	state.small = 3
	return nil
}

func (state *dieHard) fillBig(_ *zizek.TestCase) error {
	state.big = 5
	return nil
}

func (state *dieHard) emptySmall(_ *zizek.TestCase) error {
	state.small = 0
	return nil
}

func (state *dieHard) emptyBig(_ *zizek.TestCase) error {
	state.big = 0
	return nil
}

func (state *dieHard) pourSmallIntoBig(_ *zizek.TestCase) error {
	previousBig := state.big
	state.big = min(5, state.big+state.small)
	state.small -= state.big - previousBig
	return nil
}

func (state *dieHard) pourBigIntoSmall(_ *zizek.TestCase) error {
	previousSmall := state.small
	state.small = min(3, state.small+state.big)
	state.big -= state.small - previousSmall
	return nil
}

func (state *dieHard) physicsOfJugs(_ *zizek.TestCase) error {
	if state.small < 0 || state.small > 3 {
		return fmt.Errorf("small jug escaped its capacity: %d", state.small)
	}
	if state.big < 0 || state.big > 5 {
		return fmt.Errorf("big jug escaped its capacity: %d", state.big)
	}
	return nil
}

func (state *dieHard) dieHardProblemNotSolved(tc *zizek.TestCase) error {
	tc.Note(fmt.Sprintf("small / big = %d / %d", state.small, state.big))
	if state.big == 4 {
		return errors.New("the five-litre jug contains four litres")
	}
	return nil
}

func property(tc *zizek.TestCase) error {
	state := dieHard{}
	machine := zizek.Machine[dieHard]{
		Rules: []zizek.Rule[dieHard]{
			{Name: "fill_small", Apply: (*dieHard).fillSmall},
			{Name: "fill_big", Apply: (*dieHard).fillBig},
			{Name: "empty_small", Apply: (*dieHard).emptySmall},
			{Name: "empty_big", Apply: (*dieHard).emptyBig},
			{Name: "pour_small_into_big", Apply: (*dieHard).pourSmallIntoBig},
			{Name: "pour_big_into_small", Apply: (*dieHard).pourBigIntoSmall},
		},
		Invariants: []zizek.Rule[dieHard]{
			{Name: "physics_of_jugs", Apply: (*dieHard).physicsOfJugs},
			{Name: "die_hard_problem_not_solved", Apply: (*dieHard).dieHardProblemNotSolved},
		},
	}
	return zizek.Run(&state, machine, tc)
}

func findAndReplaySolution() (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{Examples: 2_000, Seed: 0})
	var found *zizek.Failure
	if !errors.As(err, &found) {
		return nil, fmt.Errorf("four-litre solution was not found: %w", err)
	}
	if found.Blob != portableDieHardFailure {
		return nil, fmt.Errorf("blob = %q, want shared Die Hard fixture", found.Blob)
	}
	if found.Cause == nil || found.Cause.Error() != "the five-litre jug contains four litres" {
		return nil, fmt.Errorf("unexpected failure cause: %v", found.Cause)
	}

	err = zizek.Check(property, zizek.Config{Reproduce: found.Blob})
	var replayed *zizek.Failure
	if !errors.As(err, &replayed) {
		return nil, fmt.Errorf("shrunk solution did not replay: %w", err)
	}
	for _, blob := range []string{legacyZizekDieHardFailure, hegelZlibDieHardFailure} {
		err = zizek.Check(property, zizek.Config{Reproduce: blob})
		var compatible *zizek.Failure
		if !errors.As(err, &compatible) {
			return nil, fmt.Errorf("compatible blob did not replay: %w", err)
		}
	}
	return replayed, nil
}

func main() {
	failure, err := findAndReplaySolution()
	if err != nil {
		panic(err)
	}
	fmt.Println(failure)
}

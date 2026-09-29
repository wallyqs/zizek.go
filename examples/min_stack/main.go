// Port of Hegel's deliberately broken minimum-stack state machine. Duplicate
// minima are not cached, so push(0), push(0), pop leaves a stale minimum.
package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

const portableMinStackFailure = "AXicdYixCQAACIB0jf6/tyJoy0UxASVkWHnhd9oFC6sAWQ=="

type minStack struct {
	stack    []int64
	minimums []int64
}

func (stack *minStack) minimum() (int64, bool) {
	if len(stack.minimums) == 0 {
		return 0, false
	}
	return stack.minimums[len(stack.minimums)-1], true
}

func (stack *minStack) push(value int64) {
	stack.stack = append(stack.stack, value)
	minimum, present := stack.minimum()
	if !present || value < minimum {
		stack.minimums = append(stack.minimums, value)
	}
}

func (stack *minStack) pop() (int64, bool) {
	if len(stack.stack) == 0 {
		return 0, false
	}
	last := len(stack.stack) - 1
	value := stack.stack[last]
	stack.stack = stack.stack[:last]
	if minimum, present := stack.minimum(); present && value == minimum {
		stack.minimums = stack.minimums[:len(stack.minimums)-1]
	}
	return value, true
}

func (stack *minStack) referenceMinimum() (int64, bool) {
	if len(stack.stack) == 0 {
		return 0, false
	}
	minimum := stack.stack[0]
	for _, value := range stack.stack[1:] {
		minimum = min(minimum, value)
	}
	return minimum, true
}

type minStackTest struct {
	stack minStack
}

func (state *minStackTest) push(tc *zizek.TestCase) error {
	value, err := zizek.Draw(tc, zizek.Integers())
	if err != nil {
		return err
	}
	tc.Note(fmt.Sprintf("push %d", value))
	state.stack.push(value)
	return nil
}

func (state *minStackTest) pop(tc *zizek.TestCase) error {
	value, present := state.stack.pop()
	if present {
		tc.Note(fmt.Sprintf("pop %d", value))
	} else {
		tc.Note("pop nothing")
	}
	return nil
}

func (state *minStackTest) minimumsAgree(_ *zizek.TestCase) error {
	got, gotPresent := state.stack.minimum()
	want, wantPresent := state.stack.referenceMinimum()
	if got != want || gotPresent != wantPresent {
		return fmt.Errorf("cached minimum (%d, %t), expected (%d, %t)", got, gotPresent, want, wantPresent)
	}
	return nil
}

func property(tc *zizek.TestCase) error {
	state := minStackTest{}
	machine := zizek.Machine[minStackTest]{
		Rules: []zizek.Rule[minStackTest]{
			{Name: "push", Apply: (*minStackTest).push},
			{Name: "pop", Apply: (*minStackTest).pop},
		},
		Invariants: []zizek.Rule[minStackTest]{
			{Name: "minimums_agree", Apply: (*minStackTest).minimumsAgree},
		},
	}
	return zizek.Run(&state, machine, tc)
}

func findAndReplayBug() (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{Examples: 2_000, Seed: 1})
	var found *zizek.Failure
	if !errors.As(err, &found) {
		return nil, fmt.Errorf("duplicate-minimum bug was not found: %w", err)
	}
	if found.Blob != portableMinStackFailure {
		return nil, fmt.Errorf("blob = %q, want portable min-stack fixture", found.Blob)
	}

	err = zizek.Check(property, zizek.Config{Reproduce: found.Blob})
	var replayed *zizek.Failure
	if !errors.As(err, &replayed) {
		return nil, fmt.Errorf("min-stack failure did not replay: %w", err)
	}
	return replayed, nil
}

func main() {
	failure, err := findAndReplayBug()
	if err != nil {
		panic(err)
	}
	fmt.Println(failure)
}

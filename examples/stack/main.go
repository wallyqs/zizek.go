// Port of Hegel's passing stack state machine, including a rule whose
// assumption may reject an operation without rejecting the whole example.
package main

import (
	"fmt"

	"github.com/wallyqs/zizek.go"
)

type integerStack struct {
	values []int64
}

func (stack *integerStack) push(tc *zizek.TestCase) error {
	value, err := zizek.Draw(tc, zizek.Integers())
	if err != nil {
		return err
	}
	stack.values = append(stack.values, value)
	return nil
}

func (stack *integerStack) pop(_ *zizek.TestCase) error {
	if len(stack.values) != 0 {
		stack.values = stack.values[:len(stack.values)-1]
	}
	return nil
}

func (stack *integerStack) popPush(tc *zizek.TestCase) error {
	value, err := zizek.Draw(tc, zizek.Integers())
	if err != nil {
		return err
	}
	initial := append([]int64(nil), stack.values...)
	stack.values = append(stack.values, value)
	got := stack.values[len(stack.values)-1]
	stack.values = stack.values[:len(stack.values)-1]
	if got != value || !slicesEqual(stack.values, initial) {
		return fmt.Errorf("push then pop changed the stack")
	}
	return nil
}

func (stack *integerStack) pushPop(tc *zizek.TestCase) error {
	initial := append([]int64(nil), stack.values...)
	tc.Assume(len(stack.values) != 0)
	last := len(stack.values) - 1
	value := stack.values[last]
	stack.values = stack.values[:last]
	stack.values = append(stack.values, value)
	if !slicesEqual(stack.values, initial) {
		return fmt.Errorf("pop then push changed the stack")
	}
	return nil
}

func slicesEqual(left, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func property(tc *zizek.TestCase) error {
	state := integerStack{}
	machine := zizek.Machine[integerStack]{Rules: []zizek.Rule[integerStack]{
		{Name: "push", Apply: (*integerStack).push},
		{Name: "pop", Apply: (*integerStack).pop},
		{Name: "pop_push", Apply: (*integerStack).popPush},
		{Name: "push_pop", Apply: (*integerStack).pushPop},
	}}
	return zizek.Run(&state, machine, tc)
}

func verifyStackModel() error {
	return zizek.Check(property, zizek.Config{Examples: 100, Seed: 1})
}

func main() {
	if err := verifyStackModel(); err != nil {
		panic(err)
	}
	fmt.Println("stack model passed")
}

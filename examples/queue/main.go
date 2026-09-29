// Model-based stateful example: a bounded ring queue checked against a plain
// slice model.
//
// Demonstrates:
//   - manual state machines (zizek.Machine, zizek.Rule, zizek.Run);
//   - assumptions inside rules (pop is rejected while the queue is empty);
//   - invariants checked after every successful rule;
//   - panic capture — the bug eventually reads outside the backing array; and
//   - structural shrinking of whole rule spans plus blob replay.
package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

const capacity = 2

// The shrunk failure blob is identical in Go, Python, and Ruby.
const portableQueueFailure = "AXicjcnBCQBACAPB3e8VcP2XqiL4E8xnQvIB5UmlcYrbkp7ONAAiJQCa"

// RingQueue is a fixed-capacity FIFO queue over a circular buffer. Pop wraps
// the head index one slot too late — a deliberate off-by-one that reads
// outside the backing array on the third pop.
type RingQueue struct {
	data       [capacity]int64
	head, tail int
	size       int
}

func (q *RingQueue) Push(value int64) bool {
	if q.size == capacity {
		return false
	}
	q.data[q.tail] = value
	q.tail = (q.tail + 1) % capacity
	q.size++
	return true
}

func (q *RingQueue) Pop() int64 {
	value := q.data[q.head]
	q.head = (q.head + 1) % (capacity + 1) // BUG: should wrap modulo capacity
	q.size--
	return value
}

type queueTest struct {
	queue RingQueue
	model []int64
}

func (state *queueTest) push(tc *zizek.TestCase) error {
	value, err := zizek.Draw(tc, zizek.Integers().MinValue(0).MaxValue(99))
	if err != nil {
		return err
	}
	if state.queue.Push(value) {
		state.model = append(state.model, value)
		tc.Note(fmt.Sprintf("push %d", value))
	} else {
		tc.Note(fmt.Sprintf("push %d refused: queue full", value))
	}
	return nil
}

func (state *queueTest) pop(tc *zizek.TestCase) error {
	tc.Assume(len(state.model) != 0)
	want := state.model[0]
	state.model = state.model[1:]
	got := state.queue.Pop()
	tc.Note(fmt.Sprintf("pop -> %d", got))
	if got != want {
		return fmt.Errorf("pop returned %d, expected %d", got, want)
	}
	return nil
}

func (state *queueTest) sizesAgree(_ *zizek.TestCase) error {
	if state.queue.size != len(state.model) {
		return fmt.Errorf(
			"queue reports %d items, model holds %d",
			state.queue.size,
			len(state.model),
		)
	}
	return nil
}

func property(tc *zizek.TestCase) error {
	state := queueTest{}
	machine := zizek.Machine[queueTest]{
		Rules: []zizek.Rule[queueTest]{
			{Name: "push", Apply: (*queueTest).push},
			{Name: "pop", Apply: (*queueTest).pop},
		},
		Invariants: []zizek.Rule[queueTest]{
			{Name: "sizes_agree", Apply: (*queueTest).sizesAgree},
		},
	}
	return zizek.Run(&state, machine, tc)
}

func findAndReplayBug() (*zizek.Failure, error) {
	err := zizek.Check(property, zizek.Config{Examples: 2_000, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("queue bug was not found: %w", err)
	}
	if err := zizek.Check(
		property,
		zizek.Config{Reproduce: failure.Blob},
	); !errors.As(err, &failure) {
		return nil, fmt.Errorf("queue failure did not replay: %w", err)
	}
	return failure, nil
}

func main() {
	failure, err := findAndReplayBug()
	if err != nil {
		panic(err)
	}
	fmt.Println(failure)
}

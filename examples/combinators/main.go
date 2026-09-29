// Composing generators: the blob stays portable, the shrinker stays sharp.
//
// Demonstrates:
//   - zizek.Composite assembling a struct from a generator per field;
//   - zizek.SampledFrom, zizek.Map, zizek.Filter, and zizek.OneOf layered on
//     top of the scalar generators;
//   - a Composite whose later draw depends on an earlier one; and
//   - shrinking straight through the composition: combinators add no hidden
//     choices, so the minimal counterexample is minimal in every field at
//     once.
//
// Both scenarios are deterministic: fixed seeds, printed shrunk failures.
package main

import (
	"errors"
	"fmt"

	"github.com/wallyqs/zizek.go"
)

// Request is the value under test: four fields, four generators.
type Request struct {
	Verb    string
	Path    string
	Body    *string // nil means "no body"
	Retries int64
}

// mustAlphabet keeps the package-level generators readable. Alphabet only
// rejects codepoints that cannot be alphabet members, so a literal here is a
// programming error rather than a property failure.
func mustAlphabet(characters string) []byte {
	alphabet, err := zizek.Alphabet(characters)
	if err != nil {
		panic(err)
	}
	return alphabet
}

var (
	// SampledFrom records one integer index, shrinking towards the first
	// element — so "GET" is what a failure reports unless it needs otherwise.
	verbs = zizek.SampledFrom([]string{"GET", "PUT", "DELETE"})

	// Map records nothing of its own: the path shrinks exactly as the text it
	// is built from does.
	paths = zizek.Map(
		zizek.Text().MinSize(1).MaxSize(8).Alphabet(mustAlphabet("abc")),
		func(name string) string { return "/" + name },
	)

	// OneOf records one integer index and shrinks towards the first branch,
	// which is what makes "no body" the simpler of the two.
	bodies = zizek.OneOf(
		zizek.Just[*string](nil),
		zizek.Map(
			zizek.Text().MinSize(1).MaxSize(6).Alphabet(mustAlphabet("xyz")),
			func(text string) *string { return &text },
		),
	)

	// Filter redraws a bounded number of times and then rejects the example,
	// so an odd draw is discarded rather than failed.
	retries = zizek.Filter(
		zizek.Integers().MinValue(0).MaxValue(9),
		func(value int64) bool { return value%2 == 0 },
	)
)

// requests assembles the four fields. Go has no `builds`, so the natural
// spelling is Composite: it draws as it goes and returns the struct.
func requests() zizek.Generator[Request] {
	return zizek.Composite(func(tc *zizek.TestCase) (Request, error) {
		verb, err := zizek.Draw(tc, verbs)
		if err != nil {
			return Request{}, err
		}
		path, err := zizek.Draw(tc, paths)
		if err != nil {
			return Request{}, err
		}
		body, err := zizek.Draw(tc, bodies)
		if err != nil {
			return Request{}, err
		}
		count, err := zizek.Draw(tc, retries)
		if err != nil {
			return Request{}, err
		}
		return Request{Verb: verb, Path: path, Body: body, Retries: count}, nil
	})
}

// deleteWithABody finds the router bug: a DELETE is forwarded with its body
// instead of being rejected.
//
// The shrunk request is minimal in every field the failure does not need —
// shortest path, smallest retries — while Verb stays "DELETE" and Body stays
// present, because the failure does need those.
func deleteWithABody() (*zizek.Failure, error) {
	property := func(tc *zizek.TestCase) error {
		request, err := zizek.Draw(tc, requests())
		if err != nil {
			return err
		}
		body := "none"
		if request.Body != nil {
			body = fmt.Sprintf("%q", *request.Body)
		}
		tc.Note(fmt.Sprintf(
			"%s %s body=%s retries=%d",
			request.Verb, request.Path, body, request.Retries,
		))
		if request.Verb == "DELETE" && request.Body != nil {
			return errors.New("refusing to forward a DELETE with a body")
		}
		return nil
	}

	err := zizek.Check(property, zizek.Config{Examples: 500, Seed: 2})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("the DELETE-with-body bug should be discovered: %w", err)
	}
	return failure, nil
}

// meetings is a generator whose second draw depends on its first: a meeting
// may not end before it starts. Composite hands the test case straight to the
// body, so it can draw, assume, and note exactly like a property.
func meetings() zizek.Generator[[2]int64] {
	return zizek.Composite(func(tc *zizek.TestCase) ([2]int64, error) {
		starts, err := zizek.Draw(tc, zizek.Integers().MinValue(0).MaxValue(23))
		if err != nil {
			return [2]int64{}, err
		}
		ends, err := zizek.Draw(tc, zizek.Integers().MinValue(starts).MaxValue(23))
		if err != nil {
			return [2]int64{}, err
		}
		return [2]int64{starts, ends}, nil
	})
}

// meetingAcrossNoon finds the scheduler double-booking the noon slot. The
// failure needs a meeting spanning hour 12, so the shrunk example is the
// earliest start with the earliest end that still covers it.
func meetingAcrossNoon() (*zizek.Failure, error) {
	property := func(tc *zizek.TestCase) error {
		meeting, err := zizek.Draw(tc, meetings())
		if err != nil {
			return err
		}
		tc.Note(fmt.Sprintf("meeting %d:00-%d:59", meeting[0], meeting[1]))
		if meeting[0] <= 12 && meeting[1] >= 12 {
			return errors.New("the noon slot was double-booked")
		}
		return nil
	}

	err := zizek.Check(property, zizek.Config{Examples: 300, Seed: 1})
	var failure *zizek.Failure
	if !errors.As(err, &failure) {
		return nil, fmt.Errorf("the noon double-booking should be discovered: %w", err)
	}
	return failure, nil
}

func main() {
	request, err := deleteWithABody()
	if err != nil {
		panic(err)
	}
	fmt.Println("== 1. Composite + SampledFrom + Map + Filter + OneOf ==")
	fmt.Println(request)

	meeting, err := meetingAcrossNoon()
	if err != nil {
		panic(err)
	}
	fmt.Println("\n== 2. Composite: dependent draws ==")
	fmt.Println(meeting)
}

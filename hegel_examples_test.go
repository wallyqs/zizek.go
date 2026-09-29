package zizek_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/wallyqs/zizek.go"
)

// ExampleTest is the Zizek form of Hegel's basic testing.T integration.
func ExampleTest() {
	t := &testing.T{} // in real code, use the *testing.T passed to your test
	zizek.Test(t, func(zt *zizek.T) {
		n := zizek.MustDraw(zt, zizek.Integers().MinValue(0).MaxValue(200))
		// This intentionally fails and shrinks to n=50.
		zt.Always("n is below 50", n < 50)
	})
}

// ExampleFilter shows a generator that keeps only values satisfying a
// predicate. Exhausting its bounded retry allowance rejects that example.
func ExampleFilter() {
	t := &testing.T{}
	zizek.Test(t, func(zt *zizek.T) {
		evenIntegers := zizek.Filter(
			zizek.Integers(),
			func(value int64) bool { return value%2 == 0 },
		)
		n := zizek.MustDraw(zt, evenIntegers)
		zt.Always("filtered integers are even", n%2 == 0)
	})
}

// ExampleTest_assume demonstrates rejecting undefined inputs without marking
// them as failures.
func ExampleTest_assume() {
	t := &testing.T{}
	zizek.Test(t, func(zt *zizek.T) {
		numerator := zizek.MustDraw(zt, zizek.Integers().MinValue(-1000).MaxValue(1000))
		denominator := zizek.MustDraw(zt, zizek.Integers().MinValue(-1000).MaxValue(1000))
		zt.Assume(denominator != 0)

		quotient, remainder := numerator/denominator, numerator%denominator
		zt.Always(
			"integer division reconstructs its numerator",
			numerator == quotient*denominator+remainder,
		)
	})
}

// ExampleMap derives a new value without recording an extra choice.
func ExampleMap() {
	t := &testing.T{}
	zizek.Test(t, func(zt *zizek.T) {
		decimal := zizek.MustDraw(zt, zizek.Map(
			zizek.Integers().MinValue(0).MaxValue(100),
			func(value int64) string { return fmt.Sprintf("%d", value) },
		))
		for _, character := range decimal {
			zt.Always("decimal text contains only digits", character >= '0' && character <= '9')
		}
	})
}

// ExampleDraw_dependentGeneration uses earlier draws to constrain later ones.
func ExampleDraw_dependentGeneration() {
	t := &testing.T{}
	zizek.Test(t, func(zt *zizek.T) {
		size := zizek.MustDraw(zt, zizek.Integers().MinValue(1).MaxValue(10))
		values := zizek.MustDraw(zt, zizek.Lists(zizek.Integers()).
			MinSize(int(size)).
			MaxSize(int(size)))
		index := zizek.MustDraw(zt, zizek.Integers().MinValue(0).MaxValue(size-1))
		zt.Always("generated index is in range", index < int64(len(values)))
	})
}

// ExampleFlatMap packages dependent generation as a reusable generator.
func ExampleFlatMap() {
	fixedSizeLists := zizek.FlatMap(
		zizek.Integers().MinValue(1).MaxValue(5),
		func(size int64) zizek.Generator[[]int64] {
			return zizek.Lists(zizek.Integers().MinValue(math.MinInt64).MaxValue(math.MaxInt64)).
				MinSize(int(size)).
				MaxSize(int(size))
		},
	)

	t := &testing.T{}
	zizek.Test(t, func(zt *zizek.T) {
		values := zizek.MustDraw(zt, fixedSizeLists)
		zt.Always("flat-mapped list has the promised size", len(values) >= 1 && len(values) <= 5)
	})
}

type exampleNode struct {
	value       int64
	left, right *exampleNode
}

// ExampleRecursive adapts Hegel's recursive binary-tree example to Zizek's
// bounded Recursive generator. Depth and leaf limits are enforced by the
// generator, and every subtree is a shrinkable structural span.
func ExampleRecursive() {
	leaves := zizek.Map(
		zizek.Integers().MinValue(0).MaxValue(100),
		func(value int64) *exampleNode { return &exampleNode{value: value} },
	)
	trees := zizek.Recursive(leaves, func(subtrees zizek.Generator[*exampleNode]) zizek.Generator[*exampleNode] {
		return zizek.Composite(func(tc *zizek.TestCase) (*exampleNode, error) {
			value := zizek.MustDraw(tc, zizek.Integers().MinValue(0).MaxValue(100))
			left := zizek.MustDraw(tc, subtrees)
			right := zizek.MustDraw(tc, subtrees)
			return &exampleNode{value: value, left: left, right: right}, nil
		})
	}).MaxDepth(5).MaxLeaves(16)

	t := &testing.T{}
	zizek.Test(t, func(zt *zizek.T) {
		_ = zizek.MustDraw(zt, trees)
	})
}

// ExampleTest_noteAndTarget attaches failure context and steers generation
// toward larger operands without adding anything to the replay blob.
func ExampleTest_noteAndTarget() {
	t := &testing.T{}
	zizek.Test(t, func(zt *zizek.T) {
		x := zizek.MustDraw(zt, zizek.Integers().MinValue(0).MaxValue(10_000))
		y := zizek.MustDraw(zt, zizek.Integers().MinValue(0).MaxValue(10_000))
		zt.Note(fmt.Sprintf("trying x=%d, y=%d", x, y))
		zt.TargetLabelled(float64(x+y), "largest sum")
		zt.Always("addition is commutative", x+y == y+x)
	}, zizek.Config{Examples: 500})
}

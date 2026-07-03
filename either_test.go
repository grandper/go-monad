package monad_test

import (
	"fmt"
	"testing"

	monad "github.com/grandper/go-monad"

	"github.com/stretchr/testify/require"
)

const (
	testEitherLeftValue   = "error"
	testEitherRightValue  = 42
	testEitherRightValue2 = 100
)

func TestEither(t *testing.T) {
	t.Run("Left", func(t *testing.T) {
		e := monad.Left[string, int](testEitherLeftValue)
		require.True(t, e.IsLeft())
		require.False(t, e.IsRight())
		v, ok := e.LeftValue()
		require.True(t, ok)
		require.Equal(t, testEitherLeftValue, v)
		_, ok = e.RightValue()
		require.False(t, ok)
		require.Equal(t, fmt.Sprintf("Left(%v)", testEitherLeftValue), e.String())
	})

	t.Run("Right", func(t *testing.T) {
		e := monad.Right[string, int](testEitherRightValue)
		require.True(t, e.IsRight())
		require.False(t, e.IsLeft())
		v, ok := e.RightValue()
		require.True(t, ok)
		require.Equal(t, testEitherRightValue, v)
		_, ok = e.LeftValue()
		require.False(t, ok)
		require.Equal(t, fmt.Sprintf("Right(%v)", testEitherRightValue), e.String())
	})

	t.Run("MapEither", func(t *testing.T) {
		e := monad.Right[string, int](testEitherRightValue)
		mapped := monad.MapEither(e, func(v int) string { return fmt.Sprintf("val:%d", v) })
		v, ok := mapped.RightValue()
		require.True(t, ok)
		require.Equal(t, "val:42", v)

		left := monad.Left[string, int](testEitherLeftValue)
		mappedLeft := monad.MapEither(left, func(v int) string { return fmt.Sprintf("val:%d", v) })
		lv, ok := mappedLeft.LeftValue()
		require.True(t, ok)
		require.Equal(t, testEitherLeftValue, lv)
	})

	t.Run("FlatMapEither", func(t *testing.T) {
		e := monad.Right[string, int](testEitherRightValue)
		fm := monad.FlatMapEither(e, func(v int) monad.Either[string, int] {
			return monad.Right[string, int](v * 2)
		})
		v, ok := fm.RightValue()
		require.True(t, ok)
		require.Equal(t, testEitherRightValue*2, v)

		left := monad.Left[string, int](testEitherLeftValue)
		fmLeft := monad.FlatMapEither(left, func(v int) monad.Either[string, int] {
			return monad.Right[string, int](v * 2)
		})
		lv, ok := fmLeft.LeftValue()
		require.True(t, ok)
		require.Equal(t, testEitherLeftValue, lv)
	})

	t.Run("FoldEither", func(t *testing.T) {
		e := monad.Right[string, int](testEitherRightValue)
		result := monad.FoldEither(
			e,
			func(l string) string { return "fail:" + l },
			func(r int) string { return fmt.Sprintf("ok:%d", r) },
		)
		require.Equal(t, "ok:42", result)

		left := monad.Left[string, int](testEitherLeftValue)
		resultLeft := monad.FoldEither(
			left,
			func(l string) string { return "fail:" + l },
			func(r int) string { return fmt.Sprintf("ok:%d", r) },
		)
		require.Equal(t, "fail:"+testEitherLeftValue, resultLeft)
	})

	t.Run("Method syntax", func(t *testing.T) {
		e := monad.Right[string, int](testEitherRightValue)
		mapped := e.Map(func(v int) int { return v + 1 })
		fm := mapped.FlatMap(func(v int) monad.Either[string, int] { return monad.Right[string, int](v * 2) })
		result := fm.Fold(
			func(l string) string { return "fail:" + l },
			func(r int) string { return fmt.Sprintf("ok:%d", r) },
		)
		require.Equal(t, "ok:86", result)
	})
}

func ExampleLeft() {
	e := monad.Left[string, int]("error message")
	fmt.Println(e.IsLeft())
	// Output:
	// true
}

func ExampleRight() {
	e := monad.Right[string, int](42)
	fmt.Println(e.IsRight())
	// Output:
	// true
}

func ExampleEither_Fold() {
	describe := func(e monad.Either[string, int]) string {
		return e.Fold(
			func(s string) string { return "left: " + s },
			func(n int) string { return fmt.Sprintf("right: %d", n) },
		)
	}

	fmt.Println(describe(monad.Right[string, int](42)))
	fmt.Println(describe(monad.Left[string, int]("oops")))
	// Output:
	// right: 42
	// left: oops
}

func ExampleEither_Map() {
	e := monad.Right[string, int](21).Map(func(n int) int { return n * 2 })
	val, ok := e.RightValue()
	fmt.Println(ok, val)
	// Output:
	// true 42
}

// TestEitherZeroValue pins down the behavior of an Either that was never
// assigned, which is what a struct field or a slice element starts out as.
// Every operation must treat it as Left(zero L) instead of panicking.
func TestEitherZeroValue(t *testing.T) {
	var e monad.Either[string, int]

	t.Run("is a Left holding the zero value of L", func(t *testing.T) {
		require.True(t, e.IsLeft())
		require.False(t, e.IsRight())

		v, ok := e.LeftValue()
		require.True(t, ok)
		require.Empty(t, v)
	})

	t.Run("String reports the Left", func(t *testing.T) {
		require.Equal(t, "Left()", e.String())
	})

	t.Run("Map passes the Left through", func(t *testing.T) {
		mapped := e.Map(func(n int) int { return n * 2 })
		require.True(t, mapped.IsLeft())
	})

	t.Run("FlatMap passes the Left through", func(t *testing.T) {
		fm := e.FlatMap(func(n int) monad.Either[string, int] {
			return monad.Right[string](n)
		})
		require.True(t, fm.IsLeft())
	})

	t.Run("Fold takes the left branch", func(t *testing.T) {
		got := e.Fold(
			func(string) string { return "left" },
			func(int) string { return "right" },
		)
		require.Equal(t, "left", got)
	})

	t.Run("standalone functions agree", func(t *testing.T) {
		require.True(t, monad.MapEither(e, func(n int) int { return n }).IsLeft())
		require.True(t, monad.FlatMapEither(e, func(n int) monad.Either[string, int] {
			return monad.Right[string](n)
		}).IsLeft())
		require.Equal(t, "left", monad.FoldEither(e,
			func(string) string { return "left" },
			func(int) string { return "right" },
		))
	})
}

func TestEitherSwap(t *testing.T) {
	t.Run("turns a Right into a Left", func(t *testing.T) {
		swapped := monad.Right[string, int](testEitherRightValue).Swap()

		require.True(t, swapped.IsLeft())
		v, ok := swapped.LeftValue()
		require.True(t, ok)
		require.Equal(t, testEitherRightValue, v)
	})

	t.Run("turns a Left into a Right", func(t *testing.T) {
		swapped := monad.Left[string, int](testEitherLeftValue).Swap()

		require.True(t, swapped.IsRight())
		v, ok := swapped.RightValue()
		require.True(t, ok)
		require.Equal(t, testEitherLeftValue, v)
	})

	t.Run("is its own inverse", func(t *testing.T) {
		e := monad.Right[string, int](testEitherRightValue)
		require.Equal(t, e, e.Swap().Swap())
	})
}

func TestEitherMapLeft(t *testing.T) {
	t.Run("transforms a Left", func(t *testing.T) {
		mapped := monad.Left[string, int](testEitherLeftValue).
			MapLeft(func(s string) int { return len(s) })

		require.True(t, mapped.IsLeft())
		v, ok := mapped.LeftValue()
		require.True(t, ok)
		require.Equal(t, len(testEitherLeftValue), v)
	})

	t.Run("leaves a Right untouched", func(t *testing.T) {
		mapped := monad.Right[string, int](testEitherRightValue).
			MapLeft(func(s string) int { return len(s) })

		require.True(t, mapped.IsRight())
		v, ok := mapped.RightValue()
		require.True(t, ok)
		require.Equal(t, testEitherRightValue, v)
	})

	t.Run("standalone function agrees", func(t *testing.T) {
		mapped := monad.MapLeftEither(
			monad.Left[string, int](testEitherLeftValue),
			func(s string) int { return len(s) },
		)

		v, ok := mapped.LeftValue()
		require.True(t, ok)
		require.Equal(t, len(testEitherLeftValue), v)
	})
}

func TestEitherToOption(t *testing.T) {
	t.Run("a Right becomes Some", func(t *testing.T) {
		option := monad.Right[string, int](testEitherRightValue).ToOption()

		require.True(t, option.IsPresent())
		require.Equal(t, testEitherRightValue, option.OrElse(0))
	})

	t.Run("a Left becomes None", func(t *testing.T) {
		option := monad.Left[string, int](testEitherLeftValue).ToOption()

		require.True(t, option.IsEmpty())
	})
}

// TestEitherMonadLaws verifies the three monad laws for Either. Because Either
// is Right-biased, Right is the unit and a Left must short-circuit each law.
func TestEitherMonadLaws(t *testing.T) {
	unit := monad.Right[string, int]
	f := func(n int) monad.Either[string, int] { return unit(n * 2) }
	g := func(n int) monad.Either[string, int] { return unit(n + 1) }

	t.Run("left identity", func(t *testing.T) {
		require.Equal(t, f(testEitherRightValue), unit(testEitherRightValue).FlatMap(f))
	})

	t.Run("right identity", func(t *testing.T) {
		m := unit(testEitherRightValue)
		require.Equal(t, m, m.FlatMap(unit))

		left := monad.Left[string, int](testEitherLeftValue)
		require.Equal(t, left, left.FlatMap(unit))
	})

	t.Run("associativity", func(t *testing.T) {
		for _, m := range []monad.Either[string, int]{
			unit(testEitherRightValue),
			monad.Left[string, int](testEitherLeftValue),
		} {
			require.Equal(t,
				m.FlatMap(f).FlatMap(g),
				m.FlatMap(func(n int) monad.Either[string, int] { return f(n).FlatMap(g) }),
			)
		}
	})
}

func TestMapLeftEitherOnRight(t *testing.T) {
	// The standalone form must pass a Right through untouched, mirroring the
	// method. This is the branch the method test cannot reach.
	mapped := monad.MapLeftEither(
		monad.Right[string, int](testEitherRightValue),
		func(s string) int { return len(s) },
	)

	require.True(t, mapped.IsRight())
	v, ok := mapped.RightValue()
	require.True(t, ok)
	require.Equal(t, testEitherRightValue, v)
}

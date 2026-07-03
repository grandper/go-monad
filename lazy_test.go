package monad_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	monad "github.com/grandper/go-monad"

	"github.com/stretchr/testify/require"
)

const (
	testLazyIntValue    = 42
	testLazyDoubledInt  = 84
	testLazyStringValue = "hello"
)

func TestLazy(t *testing.T) {
	t.Run("Defer", func(t *testing.T) {
		t.Run("does not evaluate until accessed", func(t *testing.T) {
			var evaluated atomic.Bool
			_ = monad.Defer(func() int {
				evaluated.Store(true)
				return testLazyIntValue
			})

			require.False(t, evaluated.Load())
		})
	})

	t.Run("Evaluate", func(t *testing.T) {
		t.Run("returns the computed value", func(t *testing.T) {
			lazy := monad.Defer(func() int {
				return testLazyIntValue
			})

			require.Equal(t, testLazyIntValue, lazy.Evaluate())
		})

		t.Run("evaluates only once", func(t *testing.T) {
			var counter atomic.Int32
			lazy := monad.Defer(func() int {
				counter.Add(1)
				return testLazyIntValue
			})

			first := lazy.Evaluate()
			second := lazy.Evaluate()

			require.Equal(t, first, second)
			require.Equal(t, int32(1), counter.Load())
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Run("returns representation after evaluation", func(t *testing.T) {
			lazy := monad.Defer(func() int {
				return testLazyIntValue
			})
			lazy.Evaluate()

			require.Equal(t, "Lazy(42)", lazy.String())
		})
	})

	t.Run("Map", func(t *testing.T) {
		t.Run("transforms value lazily", func(t *testing.T) {
			lazy := monad.Defer(func() int {
				return testLazyIntValue
			})

			mapped := lazy.Map(func(v int) int {
				return v * 2
			})

			require.Equal(t, testLazyDoubledInt, mapped.Evaluate())
		})

		t.Run("does not evaluate original until mapped is evaluated", func(t *testing.T) {
			var evaluated atomic.Bool
			lazy := monad.Defer(func() int {
				evaluated.Store(true)
				return testLazyIntValue
			})

			_ = lazy.Map(func(v int) int {
				return v * 2
			})

			require.False(t, evaluated.Load())
		})
	})

	t.Run("FlatMap", func(t *testing.T) {
		t.Run("transforms value lazily", func(t *testing.T) {
			lazy := monad.Defer(func() int {
				return testLazyIntValue
			})

			flatMapped := lazy.FlatMap(func(_ int) *monad.Lazy[string] {
				return monad.Defer(func() string {
					return testLazyStringValue
				})
			})

			require.Equal(t, testLazyStringValue, flatMapped.Evaluate())
		})

		t.Run("does not evaluate original until flatmapped is evaluated", func(t *testing.T) {
			var evaluated atomic.Bool
			lazy := monad.Defer(func() int {
				evaluated.Store(true)
				return testLazyIntValue
			})

			_ = lazy.FlatMap(func(_ int) *monad.Lazy[string] {
				return monad.Defer(func() string {
					return testLazyStringValue
				})
			})

			require.False(t, evaluated.Load())
		})
	})

	t.Run("MapLazy", func(t *testing.T) {
		t.Run("transforms value lazily", func(t *testing.T) {
			lazy := monad.Defer(func() int {
				return testLazyIntValue
			})

			mapped := monad.MapLazy(lazy, func(v int) int {
				return v * 2
			})

			require.Equal(t, testLazyDoubledInt, mapped.Evaluate())
		})
	})

	t.Run("FlatMapLazy", func(t *testing.T) {
		t.Run("transforms value lazily", func(t *testing.T) {
			lazy := monad.Defer(func() int {
				return testLazyIntValue
			})

			flatMapped := monad.FlatMapLazy(lazy, func(_ int) *monad.Lazy[string] {
				return monad.Defer(func() string {
					return testLazyStringValue
				})
			})

			require.Equal(t, testLazyStringValue, flatMapped.Evaluate())
		})
	})
}

func ExampleDefer() {
	calls := 0
	lazy := monad.Defer(func() int {
		calls++
		return 42
	})

	fmt.Println(calls)           // not yet evaluated
	fmt.Println(lazy.Evaluate()) // evaluated once
	fmt.Println(lazy.Evaluate()) // cached, not re-evaluated
	fmt.Println(calls)
	// Output:
	// 0
	// 42
	// 42
	// 1
}

func ExampleLazy_Map() {
	lazy := monad.Defer(func() int { return 21 }).
		Map(func(n int) int { return n * 2 })
	fmt.Println(lazy.Evaluate())
	// Output:
	// 42
}

// TestLazyStringBeforeEvaluation guards the invariant that String never
// triggers the computation, and never reports a value the Lazy does not yet
// hold.
func TestLazyStringBeforeEvaluation(t *testing.T) {
	var calls atomic.Int32
	lazy := monad.Defer(func() int {
		calls.Add(1)
		return testLazyIntValue
	})

	require.Equal(t, "Lazy(pending)", lazy.String())
	require.Zero(t, calls.Load(), "String must not trigger the computation")

	lazy.Evaluate()
	require.Equal(t, "Lazy(42)", lazy.String())
}

// TestLazyStringIsRaceFree exercises String concurrently with Evaluate. It is
// meaningful under -race, where an unsynchronized read of the cached value
// would be reported.
func TestLazyStringIsRaceFree(t *testing.T) {
	lazy := monad.Defer(func() int { return testLazyIntValue })

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() { defer wg.Done(); _ = lazy.Evaluate() }()
		go func() { defer wg.Done(); _ = lazy.String() }()
	}
	wg.Wait()

	require.Equal(t, testLazyIntValue, lazy.Evaluate())
}

func TestLazyUninitialized(t *testing.T) {
	t.Run("Evaluate panics with a clear message", func(t *testing.T) {
		var lazy monad.Lazy[int]

		require.PanicsWithValue(t, "monad: Evaluate called on an uninitialized Lazy", func() {
			_ = lazy.Evaluate()
		})
	})

	t.Run("String reports pending rather than panicking", func(t *testing.T) {
		var lazy monad.Lazy[int]

		require.Equal(t, "Lazy(pending)", lazy.String())
	})
}

// TestLazyMonadLaws verifies the three monad laws for Lazy. A Lazy is compared
// by the value it produces, since two distinct Lazy pointers are equivalent
// exactly when they evaluate to the same result.
func TestLazyMonadLaws(t *testing.T) {
	unit := func(n int) *monad.Lazy[int] { return monad.Defer(func() int { return n }) }
	f := func(n int) *monad.Lazy[int] { return unit(n * 2) }
	g := func(n int) *monad.Lazy[int] { return unit(n + 1) }

	t.Run("left identity", func(t *testing.T) {
		require.Equal(t,
			f(testLazyIntValue).Evaluate(),
			unit(testLazyIntValue).FlatMap(f).Evaluate(),
		)
	})

	t.Run("right identity", func(t *testing.T) {
		require.Equal(t,
			unit(testLazyIntValue).Evaluate(),
			unit(testLazyIntValue).FlatMap(unit).Evaluate(),
		)
	})

	t.Run("associativity", func(t *testing.T) {
		require.Equal(t,
			unit(testLazyIntValue).FlatMap(f).FlatMap(g).Evaluate(),
			unit(testLazyIntValue).FlatMap(func(n int) *monad.Lazy[int] {
				return f(n).FlatMap(g)
			}).Evaluate(),
		)
	})
}

// TestLazyPanicIsRemembered covers a panic inside the computation. A sync.Once
// is spent even when the function it guarded panicked, so without memoizing the
// panic the first caller would see it and every later caller would silently
// receive the zero value of T as though the work had succeeded.
func TestLazyPanicIsRemembered(t *testing.T) {
	t.Run("every caller sees the panic, and it runs once", func(t *testing.T) {
		var runs atomic.Int32
		lazy := monad.Defer(func() int {
			runs.Add(1)
			panic("boom")
		})

		for range 3 {
			require.PanicsWithValue(t, "boom", func() { _ = lazy.Evaluate() })
		}
		require.Equal(t, int32(1), runs.Load())
	})

	t.Run("concurrent callers all see it", func(t *testing.T) {
		lazy := monad.Defer(func() int { panic("boom") })

		var wg sync.WaitGroup
		panicked := make([]bool, 8)
		for i := range panicked {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { panicked[i] = recover() != nil }()
				_ = lazy.Evaluate()
			}()
		}
		wg.Wait()

		for i, p := range panicked {
			require.True(t, p, "goroutine %d silently received a value", i)
		}
	})

	t.Run("String reports the failure rather than a value", func(t *testing.T) {
		lazy := monad.Defer(func() int { panic("boom") })
		require.Panics(t, func() { _ = lazy.Evaluate() })

		require.Equal(t, "Lazy(failed)", lazy.String())
	})
}

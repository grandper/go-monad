package monad_test

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	monad "github.com/grandper/go-monad"

	"github.com/stretchr/testify/require"
)

const (
	testPromiseIntValue       = 42
	testPromiseStringValue    = "hello"
	testPromiseFallbackInt    = -1
	testPromiseDoubledInt     = 84
	testPromiseRecoveredValue = 99
	testPromisePendingString  = "Promise[pending]"
)

var errTestPromise = errors.New("async computation failed")

func TestPromise(t *testing.T) {
	t.Run("NewPromise", func(t *testing.T) {
		t.Run("resolves successfully", func(t *testing.T) {
			promise := monad.NewPromise(func() monad.Result[int] {
				return monad.Success(testPromiseIntValue)
			})

			result := promise.Await()

			require.True(t, result.IsSuccess())
			require.Equal(t, testPromiseIntValue, result.OrElse(0))
		})

		t.Run("resolves with failure", func(t *testing.T) {
			promise := monad.NewPromise(func() monad.Result[int] {
				return monad.Failure[int](errTestPromise)
			})

			result := promise.Await()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestPromise)
		})

		t.Run("caches result across multiple awaits", func(t *testing.T) {
			var counter atomic.Int32
			promise := monad.NewPromise(func() monad.Result[int] {
				counter.Add(1)
				return monad.Success(testPromiseIntValue)
			})

			first := promise.Await()
			second := promise.Await()

			require.Equal(t, first.OrElse(0), second.OrElse(0))
			require.Equal(t, int32(1), counter.Load())
		})
	})

	t.Run("Resolve", func(t *testing.T) {
		t.Run("creates an already-resolved promise", func(t *testing.T) {
			promise := monad.Resolve(testPromiseIntValue)

			result := promise.Await()

			require.True(t, result.IsSuccess())
			require.Equal(t, testPromiseIntValue, result.OrElse(0))
		})
	})

	t.Run("Reject", func(t *testing.T) {
		t.Run("creates an already-failed promise", func(t *testing.T) {
			promise := monad.Reject[int](errTestPromise)

			result := promise.Await()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestPromise)
		})
	})

	t.Run("Then", func(t *testing.T) {
		t.Run("calls consumer on success", func(t *testing.T) {
			var captured atomic.Int32
			promise := monad.Resolve(testPromiseIntValue).Then(func(v int) {
				captured.Store(int32(v))
			})

			promise.Await()

			require.Equal(t, int32(testPromiseIntValue), captured.Load())
		})

		t.Run("does not call consumer on failure", func(t *testing.T) {
			var called atomic.Bool
			promise := monad.Reject[int](errTestPromise).Then(func(_ int) {
				called.Store(true)
			})

			promise.Await()

			require.False(t, called.Load())
		})
	})

	t.Run("Catch", func(t *testing.T) {
		t.Run("calls consumer on failure", func(t *testing.T) {
			var captured atomic.Value
			promise := monad.Reject[int](errTestPromise).Catch(func(err error) {
				captured.Store(err)
			})

			promise.Await()

			require.ErrorIs(t, captured.Load().(error), errTestPromise)
		})

		t.Run("does not call consumer on success", func(t *testing.T) {
			var called atomic.Bool
			promise := monad.Resolve(testPromiseIntValue).Catch(func(_ error) {
				called.Store(true)
			})

			promise.Await()

			require.False(t, called.Load())
		})
	})

	t.Run("Recover", func(t *testing.T) {
		t.Run("recovers from failure", func(t *testing.T) {
			promise := monad.Reject[int](errTestPromise).Recover(func(_ error) int {
				return testPromiseRecoveredValue
			})

			result := promise.Await()

			require.True(t, result.IsSuccess())
			require.Equal(t, testPromiseRecoveredValue, result.OrElse(0))
		})

		t.Run("does not change successful promise", func(t *testing.T) {
			promise := monad.Resolve(testPromiseIntValue).Recover(func(_ error) int {
				return testPromiseRecoveredValue
			})

			result := promise.Await()

			require.True(t, result.IsSuccess())
			require.Equal(t, testPromiseIntValue, result.OrElse(0))
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Run("returns pending representation", func(t *testing.T) {
			promise := monad.Resolve(testPromiseIntValue)

			require.Equal(t, testPromisePendingString, promise.String())
		})
	})

	t.Run("Map", func(t *testing.T) {
		t.Run("transforms value on success", func(t *testing.T) {
			promise := monad.Resolve(testPromiseIntValue).Map(func(v int) int {
				return v * 2
			})

			result := promise.Await()

			require.True(t, result.IsSuccess())
			require.Equal(t, testPromiseDoubledInt, result.OrElse(0))
		})

		t.Run("propagates failure", func(t *testing.T) {
			promise := monad.Reject[int](errTestPromise).Map(func(v int) int {
				return v * 2
			})

			result := promise.Await()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestPromise)
		})
	})

	t.Run("FlatMap", func(t *testing.T) {
		t.Run("transforms value on success", func(t *testing.T) {
			promise := monad.Resolve(testPromiseIntValue).FlatMap(func(_ int) monad.Promise[string] {
				return monad.Resolve(testPromiseStringValue)
			})

			result := promise.Await()

			require.True(t, result.IsSuccess())
			require.Equal(t, testPromiseStringValue, result.OrElse(""))
		})

		t.Run("propagates outer failure", func(t *testing.T) {
			promise := monad.Reject[int](errTestPromise).FlatMap(func(_ int) monad.Promise[string] {
				return monad.Resolve(testPromiseStringValue)
			})

			result := promise.Await()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestPromise)
		})

		t.Run("propagates inner failure", func(t *testing.T) {
			innerError := errors.New("inner promise failed")
			promise := monad.Resolve(testPromiseIntValue).FlatMap(func(_ int) monad.Promise[string] {
				return monad.Reject[string](innerError)
			})

			result := promise.Await()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), innerError)
		})
	})

	t.Run("MapPromise", func(t *testing.T) {
		t.Run("transforms value on success", func(t *testing.T) {
			promise := monad.MapPromise(monad.Resolve(testPromiseIntValue), func(v int) int {
				return v * 2
			})

			result := promise.Await()

			require.True(t, result.IsSuccess())
			require.Equal(t, testPromiseDoubledInt, result.OrElse(0))
		})

		t.Run("propagates failure", func(t *testing.T) {
			promise := monad.MapPromise(monad.Reject[int](errTestPromise), func(v int) int {
				return v * 2
			})

			result := promise.Await()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestPromise)
		})
	})

	t.Run("FlatMapPromise", func(t *testing.T) {
		t.Run("transforms value on success", func(t *testing.T) {
			promise := monad.FlatMapPromise(monad.Resolve(testPromiseIntValue), func(_ int) monad.Promise[string] {
				return monad.Resolve(testPromiseStringValue)
			})

			result := promise.Await()

			require.True(t, result.IsSuccess())
			require.Equal(t, testPromiseStringValue, result.OrElse(""))
		})

		t.Run("propagates failure", func(t *testing.T) {
			promise := monad.FlatMapPromise(monad.Reject[int](errTestPromise), func(_ int) monad.Promise[string] {
				return monad.Resolve(testPromiseStringValue)
			})

			result := promise.Await()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestPromise)
		})
	})
}

func ExampleResolve() {
	p := monad.Resolve(42)
	r := p.Await()
	fmt.Println(r.IsSuccess())
	fmt.Println(r.OrElse(0))
	// Output:
	// true
	// 42
}

func ExampleReject() {
	p := monad.Reject[int](errors.New("failed"))
	r := p.Await()
	fmt.Println(r.IsSuccess())
	fmt.Println(r.OrElse(-1))
	// Output:
	// false
	// -1
}

func ExamplePromise_Map() {
	p := monad.Resolve(21).Map(func(n int) int { return n * 2 })
	fmt.Println(p.Await().OrElse(0))
	// Output:
	// 42
}

// TestPromiseZeroValue covers a Promise that was never assigned a computation.
// Await returns a Result, so the misuse surfaces as a Failure rather than a
// panic.
func TestPromiseZeroValue(t *testing.T) {
	var p monad.Promise[int]

	t.Run("Await fails with ErrUninitialized", func(t *testing.T) {
		r := p.Await()

		require.True(t, r.IsFailure())
		require.ErrorIs(t, r.Error(), monad.ErrUninitialized)
	})

	t.Run("the failure propagates through a chain", func(t *testing.T) {
		r := p.Map(func(n int) int { return n * 2 }).Await()

		require.True(t, r.IsFailure())
		require.ErrorIs(t, r.Error(), monad.ErrUninitialized)
	})

	t.Run("Recover can rescue it", func(t *testing.T) {
		r := p.Recover(func(error) int { return testPromiseRecoveredValue }).Await()

		require.True(t, r.IsSuccess())
		require.Equal(t, testPromiseRecoveredValue, r.OrElse(0))
	})
}

func TestPromiseRecoverWith(t *testing.T) {
	t.Run("replaces a failure with the fallback promise", func(t *testing.T) {
		p := monad.Reject[int](errTestPromise).
			RecoverWith(func(error) monad.Promise[int] {
				return monad.Resolve(testPromiseRecoveredValue)
			})

		require.Equal(t, testPromiseRecoveredValue, p.Await().OrElse(0))
	})

	t.Run("leaves a success untouched and skips the fallback", func(t *testing.T) {
		var called atomic.Bool
		p := monad.Resolve(testPromiseIntValue).
			RecoverWith(func(error) monad.Promise[int] {
				called.Store(true)
				return monad.Resolve(0)
			})

		require.Equal(t, testPromiseIntValue, p.Await().OrElse(0))
		require.False(t, called.Load())
	})

	t.Run("propagates a failing fallback", func(t *testing.T) {
		p := monad.Reject[int](errTestPromise).
			RecoverWith(func(err error) monad.Promise[int] { return monad.Reject[int](err) })

		require.True(t, p.Await().IsFailure())
	})
}

// TestPromiseMonadLaws verifies the three monad laws for Promise, comparing
// promises by the Result they resolve to.
func TestPromiseMonadLaws(t *testing.T) {
	f := func(n int) monad.Promise[int] { return monad.Resolve(n * 2) }
	g := func(n int) monad.Promise[int] { return monad.Resolve(n + 1) }

	t.Run("left identity", func(t *testing.T) {
		require.Equal(t,
			f(testPromiseIntValue).Await(),
			monad.Resolve(testPromiseIntValue).FlatMap(f).Await(),
		)
	})

	t.Run("right identity", func(t *testing.T) {
		m := monad.Resolve(testPromiseIntValue)
		require.Equal(t, m.Await(), m.FlatMap(monad.Resolve).Await())

		rejected := monad.Reject[int](errTestPromise)
		require.Equal(t, rejected.Await(), rejected.FlatMap(monad.Resolve).Await())
	})

	t.Run("associativity", func(t *testing.T) {
		for _, m := range []monad.Promise[int]{
			monad.Resolve(testPromiseIntValue),
			monad.Reject[int](errTestPromise),
		} {
			require.Equal(t,
				m.FlatMap(f).FlatMap(g).Await(),
				m.FlatMap(func(n int) monad.Promise[int] { return f(n).FlatMap(g) }).Await(),
			)
		}
	})
}

// TestPromisePanicBecomesFailure covers a panic inside the computation. It runs
// on the Promise's own goroutine, where a recover in the awaiting code cannot
// reach it — unrecovered, it would take the whole process down.
func TestPromisePanicBecomesFailure(t *testing.T) {
	t.Run("a panic in the computation", func(t *testing.T) {
		r := monad.NewPromise(func() monad.Result[int] { panic("boom") }).Await()

		require.True(t, r.IsFailure())
		require.ErrorIs(t, r.Error(), monad.ErrPanic)
		require.Contains(t, r.Error().Error(), "boom")
	})

	t.Run("a panic inside Map", func(t *testing.T) {
		r := monad.Resolve(1).Map(func(int) int { panic("boom in Map") }).Await()

		require.True(t, r.IsFailure())
		require.ErrorIs(t, r.Error(), monad.ErrPanic)
	})

	t.Run("a panic inside FlatMap", func(t *testing.T) {
		r := monad.Resolve(1).
			FlatMap(func(int) monad.Promise[int] { panic("boom in FlatMap") }).
			Await()

		require.True(t, r.IsFailure())
		require.ErrorIs(t, r.Error(), monad.ErrPanic)
	})

	t.Run("repeated awaits return the failure instead of deadlocking", func(t *testing.T) {
		p := monad.NewPromise(func() monad.Result[int] { panic("boom") })

		for range 3 {
			require.True(t, p.Await().IsFailure())
		}
	})

	t.Run("the failure can be recovered like any other", func(t *testing.T) {
		r := monad.NewPromise(func() monad.Result[int] { panic("boom") }).
			Recover(func(error) int { return testPromiseRecoveredValue }).
			Await()

		require.Equal(t, testPromiseRecoveredValue, r.OrElse(0))
	})
}

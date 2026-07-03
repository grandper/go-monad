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
	testIOIntValue       = 42
	testIOStringValue    = "hello"
	testIODoubledInt     = 84
	testIORecoveredValue = 99
	testIOSuspendedText  = "IO[suspended]"
)

var errTestIO = errors.New("io effect failed")

func TestIO(t *testing.T) {
	t.Run("NewIO", func(t *testing.T) {
		t.Run("runs successful effect", func(t *testing.T) {
			io := monad.NewIO(func() monad.Result[int] {
				return monad.Success(testIOIntValue)
			})

			result := io.Run()

			require.True(t, result.IsSuccess())
			require.Equal(t, testIOIntValue, result.OrElse(0))
		})

		t.Run("runs failing effect", func(t *testing.T) {
			io := monad.NewIO(func() monad.Result[int] {
				return monad.Failure[int](errTestIO)
			})

			result := io.Run()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestIO)
		})

		t.Run("re-executes effect on each run", func(t *testing.T) {
			var counter atomic.Int32
			io := monad.NewIO(func() monad.Result[int] {
				counter.Add(1)
				return monad.Success(testIOIntValue)
			})

			io.Run()
			io.Run()

			require.Equal(t, int32(2), counter.Load())
		})
	})

	t.Run("PureIO", func(t *testing.T) {
		t.Run("wraps value without side effects", func(t *testing.T) {
			io := monad.PureIO(testIOIntValue)

			result := io.Run()

			require.True(t, result.IsSuccess())
			require.Equal(t, testIOIntValue, result.OrElse(0))
		})
	})

	t.Run("FailIO", func(t *testing.T) {
		t.Run("wraps error without side effects", func(t *testing.T) {
			io := monad.FailIO[int](errTestIO)

			result := io.Run()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestIO)
		})
	})

	t.Run("Recover", func(t *testing.T) {
		t.Run("recovers from failure", func(t *testing.T) {
			io := monad.FailIO[int](errTestIO).Recover(func(_ error) int {
				return testIORecoveredValue
			})

			result := io.Run()

			require.True(t, result.IsSuccess())
			require.Equal(t, testIORecoveredValue, result.OrElse(0))
		})

		t.Run("does not change successful IO", func(t *testing.T) {
			io := monad.PureIO(testIOIntValue).Recover(func(_ error) int {
				return testIORecoveredValue
			})

			result := io.Run()

			require.True(t, result.IsSuccess())
			require.Equal(t, testIOIntValue, result.OrElse(0))
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Run("returns suspended representation", func(t *testing.T) {
			io := monad.PureIO(testIOIntValue)

			require.Equal(t, testIOSuspendedText, io.String())
		})
	})

	t.Run("Map", func(t *testing.T) {
		t.Run("transforms value on success", func(t *testing.T) {
			io := monad.PureIO(testIOIntValue).Map(func(v int) int {
				return v * 2
			})

			result := io.Run()

			require.True(t, result.IsSuccess())
			require.Equal(t, testIODoubledInt, result.OrElse(0))
		})

		t.Run("propagates failure", func(t *testing.T) {
			io := monad.FailIO[int](errTestIO).Map(func(v int) int {
				return v * 2
			})

			result := io.Run()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestIO)
		})
	})

	t.Run("FlatMap", func(t *testing.T) {
		t.Run("transforms value on success", func(t *testing.T) {
			io := monad.PureIO(testIOIntValue).FlatMap(func(_ int) monad.IO[string] {
				return monad.PureIO(testIOStringValue)
			})

			result := io.Run()

			require.True(t, result.IsSuccess())
			require.Equal(t, testIOStringValue, result.OrElse(""))
		})

		t.Run("propagates outer failure", func(t *testing.T) {
			io := monad.FailIO[int](errTestIO).FlatMap(func(_ int) monad.IO[string] {
				return monad.PureIO(testIOStringValue)
			})

			result := io.Run()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestIO)
		})

		t.Run("propagates inner failure", func(t *testing.T) {
			innerError := errors.New("inner io failed")
			io := monad.PureIO(testIOIntValue).FlatMap(func(_ int) monad.IO[string] {
				return monad.FailIO[string](innerError)
			})

			result := io.Run()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), innerError)
		})
	})

	t.Run("MapIO", func(t *testing.T) {
		t.Run("transforms value on success", func(t *testing.T) {
			io := monad.MapIO(monad.PureIO(testIOIntValue), func(v int) int {
				return v * 2
			})

			result := io.Run()

			require.True(t, result.IsSuccess())
			require.Equal(t, testIODoubledInt, result.OrElse(0))
		})

		t.Run("propagates failure", func(t *testing.T) {
			io := monad.MapIO(monad.FailIO[int](errTestIO), func(v int) int {
				return v * 2
			})

			result := io.Run()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestIO)
		})
	})

	t.Run("FlatMapIO", func(t *testing.T) {
		t.Run("transforms value on success", func(t *testing.T) {
			io := monad.FlatMapIO(monad.PureIO(testIOIntValue), func(_ int) monad.IO[string] {
				return monad.PureIO(testIOStringValue)
			})

			result := io.Run()

			require.True(t, result.IsSuccess())
			require.Equal(t, testIOStringValue, result.OrElse(""))
		})

		t.Run("propagates failure", func(t *testing.T) {
			io := monad.FlatMapIO(monad.FailIO[int](errTestIO), func(_ int) monad.IO[string] {
				return monad.PureIO(testIOStringValue)
			})

			result := io.Run()

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestIO)
		})
	})
}

func ExamplePureIO() {
	io := monad.PureIO(42)
	r := io.Run()
	fmt.Println(r.IsSuccess())
	fmt.Println(r.OrElse(0))
	// Output:
	// true
	// 42
}

func ExampleFailIO() {
	io := monad.FailIO[int](errors.New("effect failed"))
	r := io.Run()
	fmt.Println(r.IsSuccess())
	// Output:
	// false
}

func ExampleIO_Map() {
	io := monad.PureIO(21).Map(func(n int) int { return n * 2 })
	fmt.Println(io.Run().OrElse(0))
	// Output:
	// 42
}

func ExampleIO_Run() {
	counter := 0
	io := monad.NewIO(func() monad.Result[int] {
		counter++
		return monad.Success(counter)
	})

	fmt.Println(io.Run().OrElse(0))
	fmt.Println(io.Run().OrElse(0)) // each Run re-executes the effect
	// Output:
	// 1
	// 2
}

// TestIOZeroValue covers an IO that was never assigned an effect. Since Run
// already returns a Result, the misuse is reported through that channel rather
// than as a panic.
func TestIOZeroValue(t *testing.T) {
	var io monad.IO[int]

	t.Run("Run fails with ErrUninitialized", func(t *testing.T) {
		r := io.Run()

		require.True(t, r.IsFailure())
		require.ErrorIs(t, r.Error(), monad.ErrUninitialized)
	})

	t.Run("the failure propagates through a chain", func(t *testing.T) {
		r := io.Map(func(n int) int { return n * 2 }).Run()

		require.True(t, r.IsFailure())
		require.ErrorIs(t, r.Error(), monad.ErrUninitialized)
	})

	t.Run("Recover can rescue it", func(t *testing.T) {
		r := io.Recover(func(error) int { return testIORecoveredValue }).Run()

		require.True(t, r.IsSuccess())
		require.Equal(t, testIORecoveredValue, r.OrElse(0))
	})
}

func TestIORecoverWith(t *testing.T) {
	t.Run("replaces a failure with the fallback effect", func(t *testing.T) {
		io := monad.FailIO[int](errTestIO).
			RecoverWith(func(error) monad.IO[int] {
				return monad.PureIO(testIORecoveredValue)
			})

		require.Equal(t, testIORecoveredValue, io.Run().OrElse(0))
	})

	t.Run("leaves a success untouched and skips the fallback", func(t *testing.T) {
		called := false
		io := monad.PureIO(testIOIntValue).
			RecoverWith(func(error) monad.IO[int] {
				called = true
				return monad.PureIO(0)
			})

		require.Equal(t, testIOIntValue, io.Run().OrElse(0))
		require.False(t, called)
	})

	t.Run("propagates a failing fallback", func(t *testing.T) {
		io := monad.FailIO[int](errTestIO).
			RecoverWith(func(err error) monad.IO[int] { return monad.FailIO[int](err) })

		require.True(t, io.Run().IsFailure())
	})
}

// TestIOMonadLaws verifies the three monad laws for IO, comparing actions by
// the Result they produce when run.
func TestIOMonadLaws(t *testing.T) {
	f := func(n int) monad.IO[int] { return monad.PureIO(n * 2) }
	g := func(n int) monad.IO[int] { return monad.PureIO(n + 1) }

	t.Run("left identity", func(t *testing.T) {
		require.Equal(t,
			f(testIOIntValue).Run(),
			monad.PureIO(testIOIntValue).FlatMap(f).Run(),
		)
	})

	t.Run("right identity", func(t *testing.T) {
		m := monad.PureIO(testIOIntValue)
		require.Equal(t, m.Run(), m.FlatMap(monad.PureIO).Run())

		failed := monad.FailIO[int](errTestIO)
		require.Equal(t, failed.Run(), failed.FlatMap(monad.PureIO).Run())
	})

	t.Run("associativity", func(t *testing.T) {
		for _, m := range []monad.IO[int]{
			monad.PureIO(testIOIntValue),
			monad.FailIO[int](errTestIO),
		} {
			require.Equal(t,
				m.FlatMap(f).FlatMap(g).Run(),
				m.FlatMap(func(n int) monad.IO[int] { return f(n).FlatMap(g) }).Run(),
			)
		}
	})
}

package monad_test

import (
	"errors"
	"fmt"
	"testing"

	monad "github.com/grandper/go-monad"

	"github.com/stretchr/testify/require"
)

const (
	testResultIntValue       = 42
	testResultStringValue    = "hello"
	testResultFallbackInt    = -1
	testResultDoubledInt     = 84
	testResultSuccessString  = "Success(42)"
	testResultFoldSuccess    = "value: 42"
	testResultFoldFailure    = "error: something went wrong"
	testResultFoldFormat     = "value: %d"
	testResultFoldErrFormat  = "error: %s"
	testResultRecoveredValue = 99
)

var (
	errTestResult       = errors.New("something went wrong")
	errTestResultFilter = errors.New("value did not match filter")
)

func TestResult(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Run("is successful", func(t *testing.T) {
			result := monad.Success(testResultIntValue)

			require.True(t, result.IsSuccess())
		})

		t.Run("is not a failure", func(t *testing.T) {
			result := monad.Success(testResultIntValue)

			require.False(t, result.IsFailure())
		})

		t.Run("has no error", func(t *testing.T) {
			result := monad.Success(testResultIntValue)

			require.NoError(t, result.Error())
		})

		t.Run("returns string representation", func(t *testing.T) {
			result := monad.Success(testResultIntValue)

			require.Equal(t, testResultSuccessString, result.String())
		})
	})

	t.Run("Failure", func(t *testing.T) {
		t.Run("is not successful", func(t *testing.T) {
			result := monad.Failure[int](errTestResult)

			require.False(t, result.IsSuccess())
		})

		t.Run("is a failure", func(t *testing.T) {
			result := monad.Failure[int](errTestResult)

			require.True(t, result.IsFailure())
		})

		t.Run("contains the error", func(t *testing.T) {
			result := monad.Failure[int](errTestResult)

			require.ErrorIs(t, result.Error(), errTestResult)
		})

		t.Run("returns string representation", func(t *testing.T) {
			result := monad.Failure[int](errTestResult)

			require.Equal(t, fmt.Sprintf("Failure(%v)", errTestResult), result.String())
		})
	})

	t.Run("Filter", func(t *testing.T) {
		t.Run("keeps value when predicate is satisfied", func(t *testing.T) {
			result := monad.Success(testResultIntValue).Filter(func(v int) bool {
				return v > 0
			}, errTestResultFilter)

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultIntValue, result.OrElse(0))
		})

		t.Run("returns Failure when predicate is not satisfied", func(t *testing.T) {
			result := monad.Success(testResultIntValue).Filter(func(v int) bool {
				return v < 0
			}, errTestResultFilter)

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestResultFilter)
		})

		t.Run("returns original Failure when already failed", func(t *testing.T) {
			result := monad.Failure[int](errTestResult).Filter(func(v int) bool {
				return v > 0
			}, errTestResultFilter)

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestResult)
		})
	})

	t.Run("Recover", func(t *testing.T) {
		t.Run("recovers from failure", func(t *testing.T) {
			result := monad.Failure[int](errTestResult).Recover(func(_ error) int {
				return testResultRecoveredValue
			})

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultRecoveredValue, result.OrElse(0))
		})

		t.Run("does not change successful result", func(t *testing.T) {
			result := monad.Success(testResultIntValue).Recover(func(_ error) int {
				return testResultRecoveredValue
			})

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultIntValue, result.OrElse(0))
		})
	})

	t.Run("RecoverWith", func(t *testing.T) {
		t.Run("recovers from failure with a new Result", func(t *testing.T) {
			result := monad.Failure[int](errTestResult).RecoverWith(func(_ error) monad.Result[int] {
				return monad.Success(testResultRecoveredValue)
			})

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultRecoveredValue, result.OrElse(0))
		})

		t.Run("can recover into another failure", func(t *testing.T) {
			secondError := errors.New("recovery also failed")
			result := monad.Failure[int](errTestResult).RecoverWith(func(_ error) monad.Result[int] {
				return monad.Failure[int](secondError)
			})

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), secondError)
		})

		t.Run("does not change successful result", func(t *testing.T) {
			result := monad.Success(testResultIntValue).RecoverWith(func(_ error) monad.Result[int] {
				return monad.Success(testResultRecoveredValue)
			})

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultIntValue, result.OrElse(0))
		})
	})

	t.Run("OrElse", func(t *testing.T) {
		t.Run("returns value when successful", func(t *testing.T) {
			result := monad.Success(testResultIntValue).OrElse(testResultFallbackInt)

			require.Equal(t, testResultIntValue, result)
		})

		t.Run("returns fallback when failed", func(t *testing.T) {
			result := monad.Failure[int](errTestResult).OrElse(testResultFallbackInt)

			require.Equal(t, testResultFallbackInt, result)
		})
	})

	t.Run("OrElseGet", func(t *testing.T) {
		t.Run("returns value when successful", func(t *testing.T) {
			result := monad.Success(testResultIntValue).OrElseGet(func(_ error) int {
				return testResultFallbackInt
			})

			require.Equal(t, testResultIntValue, result)
		})

		t.Run("returns supplier result when failed", func(t *testing.T) {
			result := monad.Failure[int](errTestResult).OrElseGet(func(_ error) int {
				return testResultFallbackInt
			})

			require.Equal(t, testResultFallbackInt, result)
		})
	})

	t.Run("IfSuccess", func(t *testing.T) {
		t.Run("calls consumer when successful", func(t *testing.T) {
			var captured int
			monad.Success(testResultIntValue).IfSuccess(func(v int) {
				captured = v
			})

			require.Equal(t, testResultIntValue, captured)
		})

		t.Run("does not call consumer when failed", func(t *testing.T) {
			called := false
			monad.Failure[int](errTestResult).IfSuccess(func(_ int) {
				called = true
			})

			require.False(t, called)
		})
	})

	t.Run("IfFailure", func(t *testing.T) {
		t.Run("calls consumer when failed", func(t *testing.T) {
			var captured error
			monad.Failure[int](errTestResult).IfFailure(func(err error) {
				captured = err
			})

			require.ErrorIs(t, captured, errTestResult)
		})

		t.Run("does not call consumer when successful", func(t *testing.T) {
			called := false
			monad.Success(testResultIntValue).IfFailure(func(_ error) {
				called = true
			})

			require.False(t, called)
		})
	})

	t.Run("ToOption", func(t *testing.T) {
		t.Run("converts Success to Some", func(t *testing.T) {
			option := monad.Success(testResultIntValue).ToOption()

			require.True(t, option.IsPresent())
			require.Equal(t, testResultIntValue, option.OrElse(0))
		})

		t.Run("converts Failure to None", func(t *testing.T) {
			option := monad.Failure[int](errTestResult).ToOption()

			require.True(t, option.IsEmpty())
		})
	})

	t.Run("Map", func(t *testing.T) {
		t.Run("transforms value when successful", func(t *testing.T) {
			result := monad.Success(testResultIntValue).Map(func(v int) int {
				return v * 2
			})

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultDoubledInt, result.OrElse(0))
		})

		t.Run("returns original Failure when failed", func(t *testing.T) {
			result := monad.Failure[int](errTestResult).Map(func(v int) int {
				return v * 2
			})

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestResult)
		})
	})

	t.Run("FlatMap", func(t *testing.T) {
		t.Run("transforms value when successful", func(t *testing.T) {
			result := monad.Success(testResultIntValue).FlatMap(func(_ int) monad.Result[string] {
				return monad.Success(testResultStringValue)
			})

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultStringValue, result.OrElse(""))
		})

		t.Run("returns original Failure when failed", func(t *testing.T) {
			result := monad.Failure[int](errTestResult).FlatMap(func(_ int) monad.Result[string] {
				return monad.Success(testResultStringValue)
			})

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestResult)
		})

		t.Run("returns Failure when function returns Failure", func(t *testing.T) {
			flatMapError := errors.New("flatmap failed")
			result := monad.Success(testResultIntValue).FlatMap(func(_ int) monad.Result[string] {
				return monad.Failure[string](flatMapError)
			})

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), flatMapError)
		})
	})

	t.Run("Fold", func(t *testing.T) {
		t.Run("applies onSuccess when successful", func(t *testing.T) {
			result := monad.Success(testResultIntValue).Fold(
				func(err error) string { return fmt.Sprintf(testResultFoldErrFormat, err.Error()) },
				func(v int) string { return fmt.Sprintf(testResultFoldFormat, v) },
			)

			require.Equal(t, testResultFoldSuccess, result)
		})

		t.Run("applies onFailure when failed", func(t *testing.T) {
			result := monad.Failure[int](errTestResult).Fold(
				func(err error) string { return fmt.Sprintf(testResultFoldErrFormat, err.Error()) },
				func(v int) string { return fmt.Sprintf(testResultFoldFormat, v) },
			)

			require.Equal(t, testResultFoldFailure, result)
		})
	})

	t.Run("MapResult", func(t *testing.T) {
		t.Run("transforms value when successful", func(t *testing.T) {
			result := monad.MapResult(monad.Success(testResultIntValue), func(v int) int {
				return v * 2
			})

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultDoubledInt, result.OrElse(0))
		})

		t.Run("returns original Failure when failed", func(t *testing.T) {
			result := monad.MapResult(monad.Failure[int](errTestResult), func(v int) int {
				return v * 2
			})

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestResult)
		})
	})

	t.Run("FlatMapResult", func(t *testing.T) {
		t.Run("transforms value when successful", func(t *testing.T) {
			result := monad.FlatMapResult(monad.Success(testResultIntValue), func(_ int) monad.Result[string] {
				return monad.Success(testResultStringValue)
			})

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultStringValue, result.OrElse(""))
		})

		t.Run("returns original Failure when failed", func(t *testing.T) {
			result := monad.FlatMapResult(monad.Failure[int](errTestResult), func(_ int) monad.Result[string] {
				return monad.Success(testResultStringValue)
			})

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestResult)
		})
	})

	t.Run("FoldResult", func(t *testing.T) {
		t.Run("applies onSuccess when successful", func(t *testing.T) {
			result := monad.FoldResult(
				monad.Success(testResultIntValue),
				func(err error) string { return fmt.Sprintf(testResultFoldErrFormat, err.Error()) },
				func(v int) string { return fmt.Sprintf(testResultFoldFormat, v) },
			)

			require.Equal(t, testResultFoldSuccess, result)
		})

		t.Run("applies onFailure when failed", func(t *testing.T) {
			result := monad.FoldResult(
				monad.Failure[int](errTestResult),
				func(err error) string { return fmt.Sprintf(testResultFoldErrFormat, err.Error()) },
				func(v int) string { return fmt.Sprintf(testResultFoldFormat, v) },
			)

			require.Equal(t, testResultFoldFailure, result)
		})
	})

	t.Run("ApplyResult", func(t *testing.T) {
		t.Run("applies function to value when both successful", func(t *testing.T) {
			double := func(v int) int { return v * 2 }
			result := monad.ApplyResult(monad.Success(double), monad.Success(testResultIntValue))

			require.True(t, result.IsSuccess())
			require.Equal(t, testResultDoubledInt, result.OrElse(0))
		})

		t.Run("returns Failure when function failed", func(t *testing.T) {
			result := monad.ApplyResult(
				monad.Failure[func(int) int](errTestResult),
				monad.Success(testResultIntValue),
			)

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestResult)
		})

		t.Run("returns Failure when value failed", func(t *testing.T) {
			double := func(v int) int { return v * 2 }
			result := monad.ApplyResult(monad.Success(double), monad.Failure[int](errTestResult))

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestResult)
		})
	})
}

func ExampleSuccess() {
	r := monad.Success(42)
	fmt.Println(r.IsSuccess())
	fmt.Println(r.OrElse(0))
	// Output:
	// true
	// 42
}

func ExampleFailure() {
	r := monad.Failure[int](errors.New("something went wrong"))
	fmt.Println(r.IsSuccess())
	fmt.Println(r.OrElse(-1))
	// Output:
	// false
	// -1
}

func ExampleResult_Map() {
	r := monad.Success(21).Map(func(n int) int { return n * 2 })
	fmt.Println(r.OrElse(0))
	// Output:
	// 42
}

func ExampleResult_FlatMap() {
	divide := func(n int) monad.Result[int] {
		if n == 0 {
			return monad.Failure[int](errors.New("division by zero"))
		}
		return monad.Success(100 / n)
	}

	fmt.Println(monad.Success(5).FlatMap(divide).OrElse(-1))
	fmt.Println(monad.Success(0).FlatMap(divide).OrElse(-1))
	// Output:
	// 20
	// -1
}

func ExampleResult_Fold() {
	ok := monad.Success("done")
	fail := monad.Failure[string](errors.New("oops"))

	format := func(r monad.Result[string]) string {
		return r.Fold(
			func(err error) string { return "error: " + err.Error() },
			func(s string) string { return "ok: " + s },
		)
	}

	fmt.Println(format(ok))
	fmt.Println(format(fail))
	// Output:
	// ok: done
	// error: oops
}

func ExampleResult_ToOption() {
	opt := monad.Success(42).ToOption()
	fmt.Println(opt.IsPresent())
	fmt.Println(opt.OrElse(0))
	// Output:
	// true
	// 42
}

func ExampleFoldResult() {
	r := monad.Success(7)
	out := monad.FoldResult(r,
		func(_ error) string { return "failed" },
		func(n int) string { return fmt.Sprintf("got %d", n) },
	)
	fmt.Println(out)
	// Output:
	// got 7
}

func ExampleMapResult() {
	r := monad.MapResult(monad.Success(10), func(n int) string {
		return fmt.Sprintf("n=%d", n)
	})
	fmt.Println(r.OrElse("none"))
	// Output:
	// n=10
}

// TestResultMonadLaws verifies the three monad laws for Result, including the
// failure case, where every law must short-circuit identically.
func TestResultMonadLaws(t *testing.T) {
	f := func(n int) monad.Result[int] { return monad.Success(n * 2) }
	g := func(n int) monad.Result[int] { return monad.Success(n + 1) }

	t.Run("left identity", func(t *testing.T) {
		require.Equal(t, f(testResultIntValue), monad.Success(testResultIntValue).FlatMap(f))
	})

	t.Run("right identity", func(t *testing.T) {
		m := monad.Success(testResultIntValue)
		require.Equal(t, m, m.FlatMap(monad.Success))

		failed := monad.Failure[int](errTestResult)
		require.Equal(t, failed, failed.FlatMap(monad.Success))
	})

	t.Run("associativity", func(t *testing.T) {
		for _, m := range []monad.Result[int]{
			monad.Success(testResultIntValue),
			monad.Failure[int](errTestResult),
		} {
			require.Equal(t,
				m.FlatMap(f).FlatMap(g),
				m.FlatMap(func(n int) monad.Result[int] { return f(n).FlatMap(g) }),
			)
		}
	})
}

// TestResultFailureAlwaysHasError pins the invariant that IsFailure and
// Error() != nil can never disagree. A failure carrying a nil error would be
// read as success by any caller bridging back to Go's (T, error) convention.
func TestResultFailureAlwaysHasError(t *testing.T) {
	var zero monad.Result[int]

	cases := map[string]monad.Result[int]{
		"zero value":     zero,
		"Failure(nil)":   monad.Failure[int](nil),
		"Filter(p, nil)": monad.Success(1).Filter(func(int) bool { return false }, nil),
		"ToResult(nil)":  monad.None[int]().ToResult(nil),
	}

	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			require.True(t, r.IsFailure())
			require.Error(t, r.Error(), "a failure must never report a nil error")
			require.ErrorIs(t, r.Error(), monad.ErrUninitialized)
		})
	}

	t.Run("a real error is preserved untouched", func(t *testing.T) {
		r := monad.Failure[int](errTestResult)
		require.ErrorIs(t, r.Error(), errTestResult)
		require.NotErrorIs(t, r.Error(), monad.ErrUninitialized)
	})

	t.Run("a success still reports nil", func(t *testing.T) {
		require.NoError(t, monad.Success(1).Error())
	})

	t.Run("every failure-path callback receives it", func(t *testing.T) {
		require.ErrorIs(t, zero.Fold(
			func(err error) error { return err },
			func(int) error { return nil },
		), monad.ErrUninitialized)

		zero.IfFailure(func(err error) {
			require.ErrorIs(t, err, monad.ErrUninitialized)
		})

		require.Equal(t, -1, zero.OrElseGet(func(err error) int {
			require.ErrorIs(t, err, monad.ErrUninitialized)
			return -1
		}))

		zero.Recover(func(err error) int {
			require.ErrorIs(t, err, monad.ErrUninitialized)
			return 0
		})

		zero.RecoverWith(func(err error) monad.Result[int] {
			require.ErrorIs(t, err, monad.ErrUninitialized)
			return monad.Success(0)
		})
	})

	t.Run("it survives Map and FlatMap propagation", func(t *testing.T) {
		mapped := zero.Map(func(n int) int { return n })
		require.ErrorIs(t, mapped.Error(), monad.ErrUninitialized)

		chained := zero.FlatMap(monad.Success)
		require.ErrorIs(t, chained.Error(), monad.ErrUninitialized)
	})
}

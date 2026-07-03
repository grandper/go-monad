package monad

import (
	"errors"
	"fmt"
)

// ErrUninitialized stands in wherever a monad has no error of its own to
// report but must report one. It is what [IO.Run] and [Promise.Await] fail
// with when called on a zero value that carries no computation, and what
// [Result.Error] returns for a failure built without an error — the zero
// [Result], or one constructed with an explicit nil such as
// Failure[T](nil), Filter(p, nil), or Option.ToResult(nil).
var ErrUninitialized = errors.New("monad: computation is uninitialized")

// ErrPanic wraps a value recovered from a panic inside a [Promise]
// computation. A Promise runs its work on another goroutine, where a panic
// would otherwise take the whole process down and be unrecoverable by the
// awaiting code; it is turned into a Failure carrying this error instead.
var ErrPanic = errors.New("monad: computation panicked")

// Result represents the outcome of a computation that can either succeed with a
// value of type T (created with [Success]) or fail with an error (created with
// [Failure]).
//
// Result is a functor and a monad: it supports [Result.Map], [Result.FlatMap],
// and [Result.Fold] operations. Standalone generic functions [MapResult],
// [FlatMapResult], and [FoldResult] are also provided.
type Result[T any] struct {
	value   T
	err     error
	success bool
}

// Success creates a Result representing a successful computation.
func Success[T any](value T) Result[T] {
	return Result[T]{value: value, success: true}
}

// Failure creates a Result representing a failed computation.
func Failure[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// IsSuccess returns true if the Result represents a successful outcome.
func (r Result[T]) IsSuccess() bool {
	return r.success
}

// IsFailure returns true if the Result represents a failed outcome.
func (r Result[T]) IsFailure() bool {
	return !r.success
}

// Filter returns the Result unchanged when it is successful and the value
// satisfies the predicate. Returns Failure with the given error otherwise.
func (r Result[T]) Filter(predicate func(T) bool, err error) Result[T] {
	if !r.success {
		return r
	}
	if predicate(r.value) {
		return r
	}
	return Failure[T](err)
}

// Recover transforms a failed Result into a successful one using the recovery
// function. Successful Results are returned unchanged.
func (r Result[T]) Recover(f func(error) T) Result[T] {
	if r.success {
		return r
	}
	return Success(f(r.failureError()))
}

// RecoverWith transforms a failed Result using a function that returns a new
// Result. Successful Results are returned unchanged.
func (r Result[T]) RecoverWith(f func(error) Result[T]) Result[T] {
	if r.success {
		return r
	}
	return f(r.failureError())
}

// OrElse returns the contained value when successful, otherwise the fallback.
func (r Result[T]) OrElse(fallback T) T {
	if !r.success {
		return fallback
	}
	return r.value
}

// OrElseGet returns the contained value when successful, otherwise the result
// of the supplier function.
func (r Result[T]) OrElseGet(supplier func(error) T) T {
	if !r.success {
		return supplier(r.failureError())
	}
	return r.value
}

// IfSuccess calls the consumer function with the value when the Result is
// successful.
func (r Result[T]) IfSuccess(consumer func(T)) {
	if r.success {
		consumer(r.value)
	}
}

// IfFailure calls the consumer function with the error when the Result is
// a failure.
func (r Result[T]) IfFailure(consumer func(error)) {
	if !r.success {
		consumer(r.failureError())
	}
}

// ToOption converts the Result to an Option, discarding the error information.
// A successful Result becomes Some; a failed Result becomes None.
func (r Result[T]) ToOption() Option[T] {
	if !r.success {
		return None[T]()
	}
	return Some(r.value)
}

// Error returns the error of a failed Result, or nil when it is successful.
//
// A failed Result always reports a non-nil error, so `IsFailure()` and
// `Error() != nil` can never disagree. That matters at the boundary where a
// Result is turned back into Go's (T, error) convention: a failure whose error
// was nil would be read by the caller as success. A failure built without an
// error reports [ErrUninitialized] instead.
func (r Result[T]) Error() error {
	if r.success {
		return nil
	}
	return r.failureError()
}

// String returns a human-readable representation of the Result.
func (r Result[T]) String() string {
	if !r.success {
		return fmt.Sprintf("Failure(%v)", r.failureError())
	}
	return fmt.Sprintf("Success(%v)", r.value)
}

// ---------------------------------------------------------------------------
// Methods
// ---------------------------------------------------------------------------

// Map transforms the contained value using f when the Result is successful.
// Returns the original Failure otherwise.
func (r Result[T]) Map[B any](f func(T) B) Result[B] {
	if !r.success {
		return Failure[B](r.failureError())
	}
	return Success(f(r.value))
}

// FlatMap transforms the contained value using a function that itself returns
// a Result. Returns the original Failure when the Result is already failed.
func (r Result[T]) FlatMap[B any](f func(T) Result[B]) Result[B] {
	if !r.success {
		return Failure[B](r.failureError())
	}
	return f(r.value)
}

// Fold reduces the Result to a single value by applying onFailure when failed
// or onSuccess when successful.
func (r Result[T]) Fold[B any](onFailure func(error) B, onSuccess func(T) B) B {
	if !r.success {
		return onFailure(r.failureError())
	}
	return onSuccess(r.value)
}

// failureError returns the error to hand to callers on the failed path, never
// nil. Every operation that exposes the error of a failure goes through it.
func (r Result[T]) failureError() error {
	if r.err != nil {
		return r.err
	}
	return ErrUninitialized
}

// ---------------------------------------------------------------------------
// Standalone generic functions
// ---------------------------------------------------------------------------

// MapResult applies f to the value inside r when successful.
// Returns the original Failure otherwise.
func MapResult[A, B any](r Result[A], f func(A) B) Result[B] {
	if !r.success {
		return Failure[B](r.failureError())
	}
	return Success(f(r.value))
}

// FlatMapResult applies f to the value inside r, where f returns a Result.
// Returns the original Failure when r is already failed.
func FlatMapResult[A, B any](r Result[A], f func(A) Result[B]) Result[B] {
	if !r.success {
		return Failure[B](r.failureError())
	}
	return f(r.value)
}

// FoldResult reduces r to a single value using the provided functions.
func FoldResult[A, B any](r Result[A], onFailure func(error) B, onSuccess func(A) B) B {
	if !r.success {
		return onFailure(r.failureError())
	}
	return onSuccess(r.value)
}

// ApplyResult applies a function wrapped in a Result to a value wrapped in a
// Result. Returns Failure when either is failed.
func ApplyResult[A, B any](resF Result[func(A) B], resA Result[A]) Result[B] {
	if !resF.success {
		return Failure[B](resF.err)
	}
	if !resA.success {
		return Failure[B](resA.err)
	}
	return Success(resF.value(resA.value))
}

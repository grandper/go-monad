package monad

import (
	"fmt"
	"sync"
)

// Promise represents an asynchronous computation that will eventually produce a
// [Result]. The computation is started immediately upon creation and runs in a
// separate goroutine. The outcome is cached so that multiple calls to
// [Promise.Await] return the same result without re-executing the computation.
//
// Promise is a functor and a monad: it supports [Promise.Map] and
// [Promise.FlatMap]. Standalone generic functions [MapPromise] and
// [FlatMapPromise] are also provided.
type Promise[T any] struct {
	await func() Result[T]
}

// NewPromise creates a Promise that immediately starts the given computation in
// a new goroutine. The result is cached after the first evaluation.
//
// A panic inside the computation is recovered and delivered as a Failure
// wrapping [ErrPanic]. It has to be: the computation runs on its own
// goroutine, so a panic there could not be recovered by the code awaiting the
// Promise and would terminate the program instead.
func NewPromise[T any](computation func() Result[T]) Promise[T] {
	done := make(chan Result[T], 1)
	go func() {
		// The recovery must still deliver a value. Returning without sending
		// would leave every awaiter blocked forever, trading a crash for a
		// deadlock.
		defer func() {
			if r := recover(); r != nil {
				done <- Failure[T](fmt.Errorf("%w: %v", ErrPanic, r))
			}
		}()
		done <- computation()
	}()

	var result *Result[T]
	var once sync.Once

	return Promise[T]{
		await: func() Result[T] {
			once.Do(func() {
				r := <-done
				result = &r
			})
			return *result
		},
	}
}

// Resolve creates a Promise that is already successfully resolved with the
// given value.
func Resolve[T any](value T) Promise[T] {
	r := Success(value)
	return Promise[T]{
		await: func() Result[T] { return r },
	}
}

// Reject creates a Promise that is already resolved with a failure.
func Reject[T any](err error) Promise[T] {
	r := Failure[T](err)
	return Promise[T]{
		await: func() Result[T] { return r },
	}
}

// Await blocks until the Promise resolves and returns the Result. The outcome
// is cached, so awaiting the same Promise more than once returns the same
// Result without re-running the computation, and it is safe to await from
// several goroutines.
//
// Awaiting the zero value of Promise yields a Failure wrapping
// [ErrUninitialized] rather than panicking.
func (p Promise[T]) Await() Result[T] {
	if p.await == nil {
		return Failure[T](ErrUninitialized)
	}
	return p.await()
}

// Then calls the consumer with the value once the Promise resolves
// successfully, and returns a Promise carrying that same Result so further
// calls can be chained. A failed Promise passes through untouched.
func (p Promise[T]) Then(consumer func(T)) Promise[T] {
	return NewPromise(func() Result[T] {
		r := p.Await()
		r.IfSuccess(consumer)
		return r
	})
}

// Catch calls the consumer with the error once the Promise resolves with a
// failure, and returns a Promise carrying that same Result so further calls
// can be chained. A successful Promise passes through untouched.
func (p Promise[T]) Catch(consumer func(error)) Promise[T] {
	return NewPromise(func() Result[T] {
		r := p.Await()
		r.IfFailure(consumer)
		return r
	})
}

// Recover transforms a failed Promise into a successful one using the recovery
// function. Successful Promises are returned unchanged.
func (p Promise[T]) Recover(f func(error) T) Promise[T] {
	return NewPromise(func() Result[T] {
		return p.Await().Recover(f)
	})
}

// RecoverWith transforms a failed Promise using a function that returns a new
// Promise, which allows the fallback to be asynchronous in its own right. A
// successful Promise is returned unchanged.
func (p Promise[T]) RecoverWith(f func(error) Promise[T]) Promise[T] {
	return NewPromise(func() Result[T] {
		r := p.Await()
		if r.IsSuccess() {
			return r
		}
		return f(r.Error()).Await()
	})
}

// String returns a human-readable representation of the Promise.
func (p Promise[T]) String() string {
	return "Promise[pending]"
}

// ---------------------------------------------------------------------------
// Methods
// ---------------------------------------------------------------------------

// Map transforms the resolved value using f. The transformation runs
// asynchronously in a new goroutine.
func (p Promise[T]) Map[B any](f func(T) B) Promise[B] {
	return NewPromise(func() Result[B] {
		return MapResult(p.Await(), f)
	})
}

// FlatMap transforms the resolved value using a function that itself returns a
// Promise. The inner Promise is awaited before returning.
func (p Promise[T]) FlatMap[B any](f func(T) Promise[B]) Promise[B] {
	return NewPromise(func() Result[B] {
		r := p.Await()
		if r.IsFailure() {
			return Failure[B](r.Error())
		}
		return f(r.value).Await()
	})
}

// ---------------------------------------------------------------------------
// Standalone generic functions
// ---------------------------------------------------------------------------

// MapPromise transforms the resolved value of p using f. Returns a new
// Promise whose computation runs asynchronously.
func MapPromise[A, B any](p Promise[A], f func(A) B) Promise[B] {
	return NewPromise(func() Result[B] {
		return MapResult(p.Await(), f)
	})
}

// FlatMapPromise transforms the resolved value of p using a function that
// returns a Promise. The inner Promise is awaited before returning.
func FlatMapPromise[A, B any](p Promise[A], f func(A) Promise[B]) Promise[B] {
	return NewPromise(func() Result[B] {
		r := p.Await()
		if r.IsFailure() {
			return Failure[B](r.Error())
		}
		return f(r.value).Await()
	})
}

package monad

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Lazy represents a deferred computation that is evaluated at most once. The
// computation does not start until [Lazy.Evaluate] is called, and later calls
// return the cached result instead of re-executing it. That memoization is
// what separates Lazy from [IO], whose effect re-runs on every call to
// [IO.Run].
//
// A Lazy is used through a pointer because it caches its result in place;
// copying one after evaluation would duplicate the cache.
//
// Lazy is a functor and a monad: it supports [Lazy.Map] and [Lazy.FlatMap].
// Standalone generic functions [MapLazy] and [FlatMapLazy] are also provided.
type Lazy[T any] struct {
	computation func() T
	value       T
	once        sync.Once
	evaluated   atomic.Bool
	panicValue  any
	panicked    atomic.Bool
}

// Defer creates a Lazy value that runs the given computation on first access.
func Defer[T any](computation func() T) *Lazy[T] {
	return &Lazy[T]{computation: computation}
}

// Evaluate runs the computation if it has not run yet and returns the result.
// It is safe for concurrent use: the first caller runs the computation while
// the others block until it completes, and all of them observe the same value.
//
// If the computation panics, the panic is remembered and re-raised for every
// later call. The computation still runs at most once — a sync.Once is spent
// even when the function it guarded panicked — so without this the first
// caller would see the panic and everyone after it would silently receive the
// zero value of T as though the work had succeeded.
//
// Evaluate also panics if the Lazy was built with a nil computation, which
// includes the zero value of Lazy. Unlike [IO] and [Promise], Lazy has no error
// channel through which it could report that misuse.
func (l *Lazy[T]) Evaluate() T {
	l.once.Do(func() {
		if l.computation == nil {
			l.fail("monad: Evaluate called on an uninitialized Lazy")
			return
		}
		defer func() {
			if r := recover(); r != nil {
				l.fail(r)
			}
		}()
		l.value = l.computation()
		l.evaluated.Store(true)
	})

	if l.panicked.Load() {
		panic(l.panicValue)
	}
	return l.value
}

// String returns a human-readable representation of the Lazy value. It never
// triggers the computation: an unevaluated Lazy renders as "Lazy(pending)",
// and one whose computation panicked as "Lazy(failed)", rather than reporting
// a value that was never produced.
func (l *Lazy[T]) String() string {
	if l.panicked.Load() {
		return "Lazy(failed)"
	}
	if !l.evaluated.Load() {
		return "Lazy(pending)"
	}
	return fmt.Sprintf("Lazy(%v)", l.value)
}

// ---------------------------------------------------------------------------
// Methods
// ---------------------------------------------------------------------------

// Map returns a new Lazy that applies f to the result of this one. Neither
// computation runs until the returned Lazy is evaluated.
func (l *Lazy[T]) Map[B any](f func(T) B) *Lazy[B] {
	return Defer(func() B {
		return f(l.Evaluate())
	})
}

// FlatMap returns a new Lazy that applies f to the result of this one and
// evaluates the Lazy that f returns. Nothing runs until the returned Lazy is
// evaluated.
func (l *Lazy[T]) FlatMap[B any](f func(T) *Lazy[B]) *Lazy[B] {
	return Defer(func() B {
		return f(l.Evaluate()).Evaluate()
	})
}

// fail records why the computation could not produce a value. The atomic store
// is what lets a later caller read panicValue safely, exactly as evaluated
// guards value.
func (l *Lazy[T]) fail(reason any) {
	l.panicValue = reason
	l.panicked.Store(true)
}

// ---------------------------------------------------------------------------
// Standalone generic functions
// ---------------------------------------------------------------------------

// MapLazy returns a new Lazy that applies f to the result of l when evaluated.
func MapLazy[A, B any](l *Lazy[A], f func(A) B) *Lazy[B] {
	return Defer(func() B {
		return f(l.Evaluate())
	})
}

// FlatMapLazy returns a new Lazy that applies f to the result of l and
// evaluates the Lazy that f returns.
func FlatMapLazy[A, B any](l *Lazy[A], f func(A) *Lazy[B]) *Lazy[B] {
	return Defer(func() B {
		return f(l.Evaluate()).Evaluate()
	})
}

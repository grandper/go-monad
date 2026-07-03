// Package monad provides a small family of monadic types for Go, for composing
// computations that may be absent, may fail, may be deferred, or may run
// asynchronously — without threading those conditions through every call site
// by hand.
//
// Each type wraps a value together with a context: whether it is there, whether
// it succeeded, when it runs. Map and FlatMap let you keep working with the
// value while the context is carried along for you, so the check happens once,
// where you unwrap, instead of at every step in between.
//
// # Monadic types
//
//   - [Option] — a value that may be absent (Some or None)
//   - [Result] — a computation that either succeeded or failed with an error
//   - [Either] — a value of one of two types, Left or Right
//   - [Lazy] — a deferred computation, evaluated at most once and cached
//   - [Promise] — an asynchronous computation, started on creation
//   - [IO] — a side-effecting computation, re-run on every call to Run
//
// # Two calling styles
//
// This library uses type parameters on methods (Go 1.27+), so the monadic
// operations are available directly as methods and read left to right:
//
//	monad.Some("42").
//		FlatMap(parse).
//		Filter(func(n int) bool { return n > 0 }).
//		OrElse(0)
//
// Every operation that changes the contained type also exists as a standalone
// generic function — [MapOption], [FlatMapResult], [FoldEither], and so on —
// which works on any Go version supporting generics and suits a more
// function-oriented style.
//
// # Beyond the monad operations
//
// [Option] implements JSON and YAML (un)marshaling: a present value encodes as
// the value itself and an empty Option as null. Because it implements IsZero,
// the "omitzero" (JSON) and "omitempty" (YAML) struct tags drop absent fields
// entirely. Escape hatches back to plain Go are [Option.OrElseError],
// [Option.OrElsePanic], and [Option.ToResult].
//
// The package also provides typed constructors that read optional configuration
// from environment variables, such as [GetIntFromEnv],
// [MustGetDurationFromEnv], or [GetStringSliceFromEnv]. They are built on the
// generic [OptionFromEnv] and [OptionSliceFromEnv], which accept any parser.
//
// # Zero values
//
// Every type in this package is usable before it is assigned, so a monad
// embedded in a struct never panics on first touch. The zero [Option] is None,
// the zero [Result] is a failure, and the zero [Either] is a Left holding the
// zero value of L. The three deferred types carry no computation when unset:
// [IO.Run] and [Promise.Await] report that as a failure wrapping
// [ErrUninitialized], while [Lazy.Evaluate], which has no error channel,
// panics with a message naming the mistake.
//
// A failed [Result] always carries a non-nil error, reporting
// [ErrUninitialized] when it was built without one, so [Result.IsFailure] and a
// non-nil [Result.Error] can never disagree.
//
// A [Promise] runs on its own goroutine, where a panic could not be recovered
// by the awaiting code and would end the process, so a panic in its
// computation becomes a failure wrapping [ErrPanic]. [Lazy] remembers a panic
// and re-raises it for every later caller, since its sync.Once is spent either
// way and the alternative is silently handing out a zero value.
//
// # Monad laws
//
// Every monadic type here satisfies the three monad laws, which is what makes
// refactoring a chain safe:
//
//  1. Left identity:  Unit(a).FlatMap(f)       ≡  f(a)
//  2. Right identity: m.FlatMap(Unit)          ≡  m
//  3. Associativity:  m.FlatMap(f).FlatMap(g)  ≡  m.FlatMap(x -> f(x).FlatMap(g))
//
// where Unit is the type's constructor ([Some], [Success], [Right], [Defer],
// [Resolve], [PureIO]) and FlatMap is the bind operation. The package tests
// verify all three for each type.
package monad

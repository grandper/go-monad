package monad

// IO represents a side-effecting computation. The effect is not executed until
// [IO.Run] is called, giving the caller full control over when side effects
// occur. Each call to Run re-executes the effect, which is what separates IO
// from [Lazy]: Lazy caches its first result, IO deliberately does not.
//
// IO is a functor and a monad: it supports [IO.Map] and [IO.FlatMap].
// Standalone generic functions [MapIO] and [FlatMapIO] are also provided.
type IO[T any] struct {
	effect func() Result[T]
}

// NewIO creates an IO action from the given side-effecting function.
func NewIO[T any](effect func() Result[T]) IO[T] {
	return IO[T]{effect: effect}
}

// PureIO creates an IO action that immediately succeeds with the given value
// without performing any side effects.
func PureIO[T any](value T) IO[T] {
	return IO[T]{
		effect: func() Result[T] { return Success(value) },
	}
}

// FailIO creates an IO action that immediately fails with the given error
// without performing any side effects.
func FailIO[T any](err error) IO[T] {
	return IO[T]{
		effect: func() Result[T] { return Failure[T](err) },
	}
}

// Run executes the side-effecting computation and returns the Result. Each
// call re-executes the effect.
//
// Running the zero value of IO yields a Failure wrapping [ErrUninitialized]
// rather than panicking, so a chain built on an unassigned IO fails the way
// any other failing effect would.
func (io IO[T]) Run() Result[T] {
	if io.effect == nil {
		return Failure[T](ErrUninitialized)
	}
	return io.effect()
}

// Recover transforms a failed IO into a successful one using the recovery
// function. If the IO succeeds, the recovery function is not called.
func (io IO[T]) Recover(f func(error) T) IO[T] {
	return NewIO(func() Result[T] {
		return io.Run().Recover(f)
	})
}

// RecoverWith transforms a failed IO using a function that returns a new IO,
// which allows the fallback to perform its own side effects. A successful IO
// is returned unchanged and the fallback is never built.
func (io IO[T]) RecoverWith(f func(error) IO[T]) IO[T] {
	return NewIO(func() Result[T] {
		r := io.Run()
		if r.IsSuccess() {
			return r
		}
		return f(r.Error()).Run()
	})
}

// String returns a human-readable representation of the IO action. An IO
// describes an effect that has not run, so there is no value to show.
func (io IO[T]) String() string {
	return "IO[suspended]"
}

// ---------------------------------------------------------------------------
// Methods
// ---------------------------------------------------------------------------

// Map creates a new IO that, when run, applies f to the result of this IO.
func (io IO[T]) Map[B any](f func(T) B) IO[B] {
	return NewIO(func() Result[B] {
		return MapResult(io.Run(), f)
	})
}

// FlatMap creates a new IO that, when run, applies f to the result of this IO
// and runs the resulting IO.
func (io IO[T]) FlatMap[B any](f func(T) IO[B]) IO[B] {
	return NewIO(func() Result[B] {
		r := io.Run()
		if r.IsFailure() {
			return Failure[B](r.Error())
		}
		return f(r.value).Run()
	})
}

// ---------------------------------------------------------------------------
// Standalone generic functions
// ---------------------------------------------------------------------------

// MapIO creates a new IO that applies f to the result of io when run.
func MapIO[A, B any](io IO[A], f func(A) B) IO[B] {
	return NewIO(func() Result[B] {
		return MapResult(io.Run(), f)
	})
}

// FlatMapIO creates a new IO that applies f to the result of io (where f
// returns an IO) and runs the inner IO.
func FlatMapIO[A, B any](io IO[A], f func(A) IO[B]) IO[B] {
	return NewIO(func() Result[B] {
		r := io.Run()
		if r.IsFailure() {
			return Failure[B](r.Error())
		}
		return f(r.value).Run()
	})
}

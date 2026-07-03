package monad

import "fmt"

// Either represents a value of one of two possible types (a disjoint union).
// An Either[L, R] holds either a Left (a value of type L) or a Right (a value
// of type R), and never both.
//
// By convention the Right side carries the value a computation is working
// toward and the Left side carries the alternative — an error, a validation
// message, or simply the other branch of a choice. The mapping operations
// therefore act on the Right side and pass a Left through untouched, which is
// what makes Either chainable. Use [Either.MapLeft] to transform the Left side
// and [Either.Swap] to exchange the two.
//
// The zero value is Left with the zero value of L, so an Either embedded in a
// struct is safe to use before it is assigned.
//
// Either is a functor and a monad: it supports [Either.Map], [Either.FlatMap],
// and [Either.Fold]. Standalone generic functions [MapEither], [FlatMapEither],
// and [FoldEither] are also provided.
type Either[L any, R any] struct {
	left    L
	right   R
	isRight bool
}

// Left creates an Either holding a value of type L.
func Left[L any, R any](value L) Either[L, R] {
	return Either[L, R]{left: value}
}

// Right creates an Either holding a value of type R.
func Right[L any, R any](value R) Either[L, R] {
	return Either[L, R]{right: value, isRight: true}
}

// IsLeft returns true if the Either holds a Left value.
func (e Either[L, R]) IsLeft() bool {
	return !e.isRight
}

// IsRight returns true if the Either holds a Right value.
func (e Either[L, R]) IsRight() bool {
	return e.isRight
}

// LeftValue returns the left value and true when this is a Left, or the zero
// value of L and false otherwise.
func (e Either[L, R]) LeftValue() (L, bool) {
	if e.isRight {
		var zero L
		return zero, false
	}
	return e.left, true
}

// RightValue returns the right value and true when this is a Right, or the
// zero value of R and false otherwise.
func (e Either[L, R]) RightValue() (R, bool) {
	if !e.isRight {
		var zero R
		return zero, false
	}
	return e.right, true
}

// ToOption converts the Either into an Option, keeping the Right value and
// discarding the Left. A Right becomes Some; a Left becomes None.
func (e Either[L, R]) ToOption() Option[R] {
	if !e.isRight {
		return None[R]()
	}
	return Some(e.right)
}

// Swap exchanges the two sides, turning a Left into a Right and a Right into a
// Left. It lets the Right-biased operations work on what is currently the Left
// side.
func (e Either[L, R]) Swap() Either[R, L] {
	if e.isRight {
		return Left[R, L](e.right)
	}
	return Right[R, L](e.left)
}

// String returns a human-readable representation of the Either.
func (e Either[L, R]) String() string {
	if e.isRight {
		return fmt.Sprintf("Right(%v)", e.right)
	}
	return fmt.Sprintf("Left(%v)", e.left)
}

// ---------------------------------------------------------------------------
// Methods
// ---------------------------------------------------------------------------

// Map transforms the right value using f. A Left is returned unchanged.
func (e Either[L, R]) Map[B any](f func(R) B) Either[L, B] {
	if !e.isRight {
		return Left[L, B](e.left)
	}
	return Right[L, B](f(e.right))
}

// MapLeft transforms the left value using f. A Right is returned unchanged.
func (e Either[L, R]) MapLeft[B any](f func(L) B) Either[B, R] {
	if e.isRight {
		return Right[B, R](e.right)
	}
	return Left[B, R](f(e.left))
}

// FlatMap transforms the right value using a function that itself returns an
// Either. A Left is returned unchanged, which is what lets a chain of FlatMap
// calls short-circuit on the first Left.
func (e Either[L, R]) FlatMap[B any](f func(R) Either[L, B]) Either[L, B] {
	if !e.isRight {
		return Left[L, B](e.left)
	}
	return f(e.right)
}

// Fold reduces the Either to a single value by applying ifLeft to a Left or
// ifRight to a Right. It is the way out of an Either when both sides must
// produce the same type.
func (e Either[L, R]) Fold[B any](ifLeft func(L) B, ifRight func(R) B) B {
	if !e.isRight {
		return ifLeft(e.left)
	}
	return ifRight(e.right)
}

// ---------------------------------------------------------------------------
// Standalone generic functions
// ---------------------------------------------------------------------------

// MapEither transforms the right value of e using f. A Left is returned
// unchanged.
func MapEither[L, R, B any](e Either[L, R], f func(R) B) Either[L, B] {
	if !e.isRight {
		return Left[L, B](e.left)
	}
	return Right[L, B](f(e.right))
}

// MapLeftEither transforms the left value of e using f. A Right is returned
// unchanged.
func MapLeftEither[L, R, B any](e Either[L, R], f func(L) B) Either[B, R] {
	if e.isRight {
		return Right[B, R](e.right)
	}
	return Left[B, R](f(e.left))
}

// FlatMapEither transforms the right value of e using a function that itself
// returns an Either. A Left is returned unchanged.
func FlatMapEither[L, R, B any](e Either[L, R], f func(R) Either[L, B]) Either[L, B] {
	if !e.isRight {
		return Left[L, B](e.left)
	}
	return f(e.right)
}

// FoldEither reduces e to a single value by applying ifLeft to a Left or
// ifRight to a Right.
func FoldEither[L, R, B any](e Either[L, R], ifLeft func(L) B, ifRight func(R) B) B {
	if !e.isRight {
		return ifLeft(e.left)
	}
	return ifRight(e.right)
}

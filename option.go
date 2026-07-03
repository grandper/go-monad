package monad

import (
	"encoding/json"
	"fmt"
)

// jsonNull is the JSON literal used for an absent Option.
const jsonNull = "null"

// Option represents an optional value. An Option[T] either contains a value
// (created with [Some]) or is empty (created with [None]).
//
// Option is a functor and a monad: it supports [Option.Map], [Option.FlatMap],
// and [Option.Fold] operations. Standalone generic functions [MapOption],
// [FlatMapOption], and [FoldOption] are also provided.
type Option[T any] struct {
	value   T
	present bool
}

// Some creates an Option containing the given value.
func Some[T any](value T) Option[T] {
	return Option[T]{value: value, present: true}
}

// None creates an empty Option.
func None[T any]() Option[T] {
	return Option[T]{}
}

// IsPresent returns true if the Option contains a value.
func (o Option[T]) IsPresent() bool {
	return o.present
}

// IsEmpty returns true if the Option does not contain a value.
func (o Option[T]) IsEmpty() bool {
	return !o.present
}

// Filter returns the Option unchanged if it contains a value satisfying the
// predicate. Otherwise it returns None.
func (o Option[T]) Filter(predicate func(T) bool) Option[T] {
	if !o.present {
		return o
	}
	if predicate(o.value) {
		return o
	}
	return None[T]()
}

// OrElse returns the contained value when present, otherwise the fallback.
func (o Option[T]) OrElse(fallback T) T {
	if !o.present {
		return fallback
	}
	return o.value
}

// OrElseGet returns the contained value when present, otherwise the result of
// the supplier function.
func (o Option[T]) OrElseGet(supplier func() T) T {
	if !o.present {
		return supplier()
	}
	return o.value
}

// IfPresent calls the consumer with the contained value when present, and does
// nothing when empty. It is the side-effecting counterpart to [Option.Map]: use
// it for logging or metrics, where there is no new value to carry forward.
func (o Option[T]) IfPresent(consumer func(T)) {
	if o.present {
		consumer(o.value)
	}
}

// IfEmpty calls f when the Option is empty, and does nothing when a value is
// present. It mirrors [Option.IfPresent] for the absent case.
func (o Option[T]) IfEmpty(f func()) {
	if !o.present {
		f()
	}
}

// OrElseError returns the contained value when present. Otherwise it returns the
// zero value of T and the provided error.
func (o Option[T]) OrElseError(err error) (T, error) {
	if !o.present {
		var zero T
		return zero, err
	}
	return o.value, nil
}

// OrElsePanic returns the contained value when present and panics with msg
// otherwise. It is the escape hatch for cases where absence is a programmer
// error rather than a condition to handle; prefer [Option.OrElse] or
// [Option.OrElseError] everywhere else.
func (o Option[T]) OrElsePanic(msg string) T {
	if !o.present {
		panic(msg)
	}
	return o.value
}

// ToResult converts the Option into a Result: a present value becomes a
// Success, an empty Option becomes a Failure holding err.
func (o Option[T]) ToResult(err error) Result[T] {
	if !o.present {
		return Failure[T](err)
	}
	return Success(o.value)
}

// IsZero reports whether the Option is empty. It is the encoder-facing alias of
// [Option.IsEmpty]: encoding/json looks up this exact method name for the
// "omitzero" struct-tag option, and gopkg.in/yaml.v3 for "omitempty" via
// yaml.IsZeroer. Removing it does not fall back to IsEmpty — the encoders drop
// to reflection instead, which silently omits present zero values such as
// Some(0). Keep it even though it duplicates IsEmpty semantically.
func (o Option[T]) IsZero() bool {
	return o.IsEmpty()
}

// String returns a human-readable representation of the Option.
func (o Option[T]) String() string {
	if !o.present {
		return "None"
	}
	return fmt.Sprintf("Some(%v)", o.value)
}

// ---------------------------------------------------------------------------
// Methods
// ---------------------------------------------------------------------------

// Map transforms the contained value using f. Returns None when the Option is
// empty.
func (o Option[T]) Map[B any](f func(T) B) Option[B] {
	if !o.present {
		return None[B]()
	}
	return Some(f(o.value))
}

// FlatMap transforms the contained value using a function that itself returns
// an Option. Returns None when the Option is empty.
func (o Option[T]) FlatMap[B any](f func(T) Option[B]) Option[B] {
	if !o.present {
		return None[B]()
	}
	return f(o.value)
}

// Fold reduces the Option to a single value by applying ifNone when empty or
// ifSome when a value is present.
func (o Option[T]) Fold[B any](ifNone func() B, ifSome func(T) B) B {
	if !o.present {
		return ifNone()
	}
	return ifSome(o.value)
}

// ---------------------------------------------------------------------------
// Standalone generic functions
// ---------------------------------------------------------------------------

// MapOption applies f to the value inside o, returning a new Option.
// Returns None when o is empty.
func MapOption[A, B any](o Option[A], f func(A) B) Option[B] {
	if !o.present {
		return None[B]()
	}
	return Some(f(o.value))
}

// FlatMapOption applies f to the value inside o, where f returns an Option.
// Returns None when o is empty.
func FlatMapOption[A, B any](o Option[A], f func(A) Option[B]) Option[B] {
	if !o.present {
		return None[B]()
	}
	return f(o.value)
}

// FoldOption reduces o to a single value using the provided functions.
func FoldOption[A, B any](o Option[A], ifNone func() B, ifSome func(A) B) B {
	if !o.present {
		return ifNone()
	}
	return ifSome(o.value)
}

// ApplyOption applies a function wrapped in an Option to a value wrapped in an
// Option. Returns None when either is empty.
func ApplyOption[A, B any](optF Option[func(A) B], optA Option[A]) Option[B] {
	if !optF.present {
		return None[B]()
	}
	if !optA.present {
		return None[B]()
	}
	return Some(optF.value(optA.value))
}

// MarshalJSON implements [json.Marshaler]. A present Option marshals as its
// contained value; an empty Option marshals as null.
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.present {
		return []byte(jsonNull), nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON implements [json.Unmarshaler]. JSON null produces an empty
// Option; any other value is unmarshaled into T and produces a present Option.
// A key that is absent from the document never reaches this method, so the
// field keeps its zero value, which is None.
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if string(data) == jsonNull {
		*o = None[T]()
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*o = Some(value)
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface without importing a YAML
// package. A present Option marshals as its contained value; an empty Option
// marshals as null.
func (o Option[T]) MarshalYAML() (any, error) {
	if !o.present {
		return nil, nil //nolint:nilnil // nil is the YAML null value
	}
	return o.value, nil
}

// UnmarshalYAML implements the legacy yaml.Unmarshaler interface without
// importing a YAML package. It is only invoked for non-null nodes:
// gopkg.in/yaml.v3 resolves explicit nulls and absent keys itself, leaving the
// target at its zero value, which is None. Decoding an explicit null into an
// Option that already holds a value therefore leaves that value in place.
func (o *Option[T]) UnmarshalYAML(unmarshal func(any) error) error {
	var value T
	if err := unmarshal(&value); err != nil {
		return err
	}
	*o = Some(value)
	return nil
}

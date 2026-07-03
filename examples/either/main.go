// Command either demonstrates monad.Either, a value that is one of two types.
//
// Result fixes the failure side to error. Either does not: the left side can
// be any type at all. That matters when what went wrong is structured data —
// an HTTP status, a validation report — or when neither side is a failure and
// you simply have a choice between two shapes.
package main

import (
	"fmt"
	"strings"

	monad "github.com/grandper/go-monad"
)

// Rejection is the kind of left value Either exists for: an error would flatten
// this into a string and the caller would have to parse it back out.
type Rejection struct {
	Field  string
	Reason string
	Code   int
}

func main() {
	fmt.Println("=== Creating and Inspecting ===")
	creating()

	fmt.Println("\n=== Right Bias: Map and FlatMap ===")
	rightBias()

	fmt.Println("\n=== Fold: Both Sides, One Type ===")
	folding()

	fmt.Println("\n=== MapLeft: Transforming the Other Side ===")
	mappingLeft()

	fmt.Println("\n=== Swap: Changing Which Side Is Biased ===")
	swapping()

	fmt.Println("\n=== ToOption: Discarding the Left ===")
	toOption()

	fmt.Println("\n=== A Choice, Not a Failure ===")
	notAFailure()

	fmt.Println("\n=== The Zero Value ===")
	zeroValue()
}

func creating() {
	// Both type arguments are required: only one side is present, so the other
	// cannot be inferred from the argument.
	accepted := monad.Right[Rejection, string]("alice@example.com")
	rejected := monad.Left[Rejection, string](Rejection{
		Field: "email", Reason: "missing @", Code: 422,
	})

	fmt.Printf("Right: %v (right: %v)\n", accepted, accepted.IsRight())
	fmt.Printf("Left:  %v (left: %v)\n", rejected, rejected.IsLeft())

	// LeftValue and RightValue use the comma-ok form, so asking for the side
	// that is not there is not a mistake you can make silently.
	if r, ok := rejected.LeftValue(); ok {
		fmt.Printf("  rejected %s: %s (code %d)\n", r.Field, r.Reason, r.Code)
	}
	if _, ok := rejected.RightValue(); !ok {
		fmt.Println("  there is no right value to read")
	}
}

func validate(email string) monad.Either[Rejection, string] {
	if !strings.Contains(email, "@") {
		return monad.Left[Rejection, string](Rejection{
			Field: "email", Reason: "missing @", Code: 422,
		})
	}
	return monad.Right[Rejection, string](email)
}

func rightBias() {
	// By convention the Right side carries the value being worked toward, so
	// Map and FlatMap act on it and a Left passes through untouched. That
	// asymmetry is what lets Either be chained the way Result is.
	normalize := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

	fmt.Println(validate(" Alice@Example.com ").Map(normalize))
	fmt.Println(validate("not-an-email").Map(normalize))

	// FlatMap requires the same left type throughout, which is what keeps a
	// chain's failure vocabulary consistent from end to end.
	domain := func(email string) monad.Either[Rejection, string] {
		parts := strings.SplitN(email, "@", 2)
		if len(parts) != 2 || parts[1] == "" {
			return monad.Left[Rejection, string](Rejection{
				Field: "email", Reason: "no domain", Code: 422,
			})
		}
		return monad.Right[Rejection, string](parts[1])
	}
	fmt.Println(validate("alice@example.com").Map(normalize).FlatMap(domain))
}

func folding() {
	// Fold is the usual way out: it handles both sides and makes them agree on
	// a single type. Here an Either becomes an HTTP-ish response.
	respond := func(e monad.Either[Rejection, string]) string {
		return e.Fold(
			func(r Rejection) string { return fmt.Sprintf("%d %s: %s", r.Code, r.Field, r.Reason) },
			func(email string) string { return "200 OK " + email },
		)
	}

	fmt.Println(" ", respond(validate("alice@example.com")))
	fmt.Println(" ", respond(validate("nope")))
}

func mappingLeft() {
	// Map only reaches the Right. MapLeft is its mirror, and it is how a left
	// side gets normalized — here a structured Rejection becomes the string a
	// logger wants — without disturbing the happy path.
	toMessage := func(r Rejection) string {
		return fmt.Sprintf("%s is invalid (%s)", r.Field, r.Reason)
	}

	fmt.Println(validate("nope").MapLeft(toMessage))
	fmt.Println(validate("alice@example.com").MapLeft(toMessage))
}

func swapping() {
	// Swap exchanges the two sides. It lets the right-biased operations work
	// on what is currently the left, which saves writing mirrored versions of
	// every combinator.
	rejected := validate("nope")

	codes := rejected.Swap().Map(func(r Rejection) int { return r.Code })
	fmt.Printf("code via Swap+Map: %v\n", codes)

	// Swapping twice is the identity, so it is safe to use mid-chain.
	fmt.Printf("swap twice == original: %v\n", rejected.Swap().Swap() == rejected)
}

func toOption() {
	// When the left side has done its job and only presence matters, ToOption
	// drops it. Anything the Left carried is gone after this, so fold or log
	// it first if it still matters.
	fmt.Printf("Right -> %v\n", validate("alice@example.com").ToOption())
	fmt.Printf("Left  -> %v\n", validate("nope").ToOption())
}

func notAFailure() {
	// Neither side has to be an error. Either[Cached, Fresh] models a value
	// that came from one of two places, where both are perfectly good — the
	// type records the provenance the caller may want to act on.
	type Cached struct{ Age int }
	type Fresh struct{ Bytes int }

	fromCache := monad.Left[Cached, Fresh](Cached{Age: 12})
	fromOrigin := monad.Right[Cached, Fresh](Fresh{Bytes: 4096})

	describe := func(e monad.Either[Cached, Fresh]) string {
		return e.Fold(
			func(c Cached) string { return fmt.Sprintf("cache hit, %ds old", c.Age) },
			func(f Fresh) string { return fmt.Sprintf("fetched %d bytes", f.Bytes) },
		)
	}
	fmt.Println(" ", describe(fromCache))
	fmt.Println(" ", describe(fromOrigin))
}

func zeroValue() {
	// The zero value is a Left holding the zero value of L. Left is the
	// non-success side by convention, so an unassigned Either behaves like a
	// rejection rather than panicking or silently claiming success.
	var e monad.Either[Rejection, string]

	fmt.Printf("unassigned: %v (left: %v)\n", e, e.IsLeft())
	fmt.Printf("Map is a no-op: %v\n", e.Map(strings.ToUpper))
}

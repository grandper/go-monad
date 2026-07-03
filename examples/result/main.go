// Command result demonstrates monad.Result, which models a computation that
// either produced a value or failed with an error.
//
// Go's (T, error) pair is a Result that has been taken apart. Keeping the two
// halves together means a failure can travel through several steps as one
// value, and be inspected once at the end, instead of being re-checked between
// every call.
package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	monad "github.com/grandper/go-monad"
)

var (
	errEmptyInput = errors.New("input is empty")
	errOutOfRange = errors.New("port out of range")
	errNoDatabase = errors.New("database unreachable")
)

func main() {
	fmt.Println("=== Creating and Inspecting ===")
	creating()

	fmt.Println("\n=== Map and FlatMap: Building a Pipeline ===")
	pipeline()

	fmt.Println("\n=== Filter: Failing a Valid-Looking Value ===")
	filtering()

	fmt.Println("\n=== Fold: Collapsing Both Outcomes ===")
	folding()

	fmt.Println("\n=== Recovering From Failure ===")
	recovering()

	fmt.Println("\n=== Side Effects on Each Branch ===")
	sideEffects()

	fmt.Println("\n=== Accumulating Independent Results ===")
	combining()

	fmt.Println("\n=== Crossing Back to Option and error ===")
	bridging()
}

func creating() {
	ok := monad.Success(42)
	bad := monad.Failure[int](errEmptyInput)

	// Failure needs the type argument: the error says nothing about T.
	fmt.Printf("Success: %v (success: %v)\n", ok, ok.IsSuccess())
	fmt.Printf("Failure: %v (failure: %v)\n", bad, bad.IsFailure())

	// Error returns nil on the success path, which makes the familiar
	// `if err != nil` still available when you want it.
	fmt.Printf("errors: %v / %v\n", ok.Error(), bad.Error())
}

// parsePort reports failure by returning a Failure rather than a second value,
// which is what lets it be dropped into a FlatMap chain.
func parsePort(s string) monad.Result[int] {
	if strings.TrimSpace(s) == "" {
		return monad.Failure[int](errEmptyInput)
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		// Wrapping keeps the original cause reachable through errors.Is/As.
		return monad.Failure[int](fmt.Errorf("parsing %q: %w", s, err))
	}
	return monad.Success(n)
}

func pipeline() {
	// Each step assumes the previous one succeeded. The first failure
	// short-circuits everything after it and arrives at the end unchanged, so
	// the error message still describes where it actually came from.
	format := func(input string) monad.Result[string] {
		return parsePort(input).
			Filter(func(p int) bool { return p >= 1 && p <= 65535 }, errOutOfRange).
			Map(func(p int) string { return fmt.Sprintf(":%d", p) })
	}

	for _, input := range []string{"8080", "99999", "http", ""} {
		fmt.Printf("  %-8q -> %v\n", input, format(input))
	}
}

func filtering() {
	// Filter differs from Option's: a rejected value needs a reason, so the
	// predicate is paired with the error to fail with.
	positive := func(n int) bool { return n > 0 }

	fmt.Println(monad.Success(5).Filter(positive, errOutOfRange))
	fmt.Println(monad.Success(-5).Filter(positive, errOutOfRange))

	// An already-failed Result keeps its original error; Filter never replaces
	// a cause that was established earlier.
	fmt.Println(monad.Failure[int](errEmptyInput).Filter(positive, errOutOfRange))
}

func folding() {
	// Fold forces both branches to be handled and to agree on a type. It is
	// the natural last step of a pipeline, where a value and an error have to
	// become one thing: a response, a log line, an exit code.
	render := func(r monad.Result[int]) string {
		return r.Fold(
			func(err error) string { return "error: " + err.Error() },
			func(port int) string { return fmt.Sprintf("ready on :%d", port) },
		)
	}

	fmt.Println(" ", render(parsePort("8080")))
	fmt.Println(" ", render(parsePort("nope")))
}

func recovering() {
	// Recover supplies a value for the failed case, turning a Result that
	// might be a failure into one that is certainly a success.
	fmt.Printf("Recover:     %v\n",
		monad.Failure[int](errNoDatabase).Recover(func(error) int { return -1 }))

	// RecoverWith returns a whole Result, so the fallback is allowed to fail
	// in turn — the right tool when the recovery is itself a real attempt
	// rather than a constant.
	fromCache := func(err error) monad.Result[int] {
		fmt.Printf("  primary failed (%v), consulting the cache\n", err)
		return monad.Success(7)
	}
	fmt.Printf("RecoverWith: %v\n",
		monad.Failure[int](errNoDatabase).RecoverWith(fromCache))

	// Neither runs on a success, so the expensive fallback path costs nothing
	// when it is not needed.
	fmt.Printf("untouched:   %v\n",
		monad.Success(42).RecoverWith(func(error) monad.Result[int] {
			panic("never reached")
		}))
}

func sideEffects() {
	// IfSuccess and IfFailure observe an outcome without unwrapping it. They
	// return nothing, so they terminate a chain rather than extend it.
	monad.Success(42).IfSuccess(func(v int) { fmt.Println("  served:", v) })
	monad.Failure[int](errNoDatabase).IfFailure(func(err error) { fmt.Println("  alert:", err) })
}

func combining() {
	// FlatMap sequences dependent steps. ApplyResult combines independent
	// ones: both are evaluated, and the first failure decides the outcome.
	// This is the shape to reach for when two values are needed at once.
	makeAddr := func(host string) func(int) string {
		return func(port int) string { return fmt.Sprintf("%s:%d", host, port) }
	}

	host := monad.Success("localhost")
	port := parsePort("8080")
	fmt.Printf("both ok:   %v\n", monad.ApplyResult(monad.MapResult(host, makeAddr), port))

	bad := parsePort("nope")
	fmt.Printf("one fails: %v\n", monad.ApplyResult(monad.MapResult(host, makeAddr), bad))
}

func bridging() {
	// ToOption discards the error and keeps only whether there was a value.
	// Use it when the caller has no use for the cause — but log or fold the
	// error first if it matters, because it is gone afterwards.
	fmt.Printf("ToOption: %v / %v\n",
		monad.Success(42).ToOption(),
		monad.Failure[int](errNoDatabase).ToOption())

	// Returning to the (T, error) convention at an API boundary takes OrElse
	// for the value and Error for the cause.
	r := parsePort("nope")
	fmt.Printf("as (T, error): %d, %v\n", r.OrElse(0), r.Error())
}

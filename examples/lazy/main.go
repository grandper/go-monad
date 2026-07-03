// Command lazy demonstrates monad.Lazy, a computation that is deferred until
// its value is asked for and then remembered.
//
// Two properties define it, and both matter: the work does not happen until
// something needs the result, and it happens at most once no matter how many
// callers ask. That combination is what separates Lazy from IO, whose effect
// deliberately re-runs on every call.
package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	monad "github.com/grandper/go-monad"
)

var errConnRefused = errors.New("connection refused")

func main() {
	fmt.Println("=== Nothing Runs Until You Ask ===")
	deferred()

	fmt.Println("\n=== Evaluated At Most Once ===")
	memoized()

	fmt.Println("\n=== Map and FlatMap Stay Lazy ===")
	transforming()

	fmt.Println("\n=== A Shared Prefix Is Computed Once ===")
	sharing()

	fmt.Println("\n=== Concurrent Access ===")
	concurrent()

	fmt.Println("\n=== Inspecting Without Forcing ===")
	inspecting()

	fmt.Println("\n=== Deferring Something That Can Fail ===")
	fallible()
}

func deferred() {
	// Defer only records the computation. Building a Lazy is free, which is
	// what makes it reasonable to construct one for a value you may not end
	// up needing.
	lazy := monad.Defer(func() string {
		fmt.Println("  ...expensive work running now")
		return "result"
	})

	fmt.Println("  Lazy built, nothing has run yet")
	fmt.Println("  value:", lazy.Evaluate())
}

func memoized() {
	// The cache is the point. A Lazy passed to five callers does its work for
	// the first one and hands the other four the stored answer, without any
	// of them coordinating.
	var runs atomic.Int32
	config := monad.Defer(func() string {
		runs.Add(1)
		return "loaded"
	})

	for range 5 {
		_ = config.Evaluate()
	}
	fmt.Printf("  five calls to Evaluate, %d actual run(s)\n", runs.Load())
}

func transforming() {
	// Map and FlatMap return new Lazy values and run nothing. A whole chain
	// can be assembled and then discarded without ever paying for it.
	var runs atomic.Int32
	base := monad.Defer(func() int {
		runs.Add(1)
		return 21
	})

	doubled := base.Map(func(n int) int { return n * 2 })
	labelled := doubled.Map(func(n int) string { return fmt.Sprintf("value=%d", n) })
	fmt.Printf("  chain built, runs so far: %d\n", runs.Load())

	fmt.Printf("  evaluating: %s (runs: %d)\n", labelled.Evaluate(), runs.Load())

	// FlatMap is for when the next step is itself deferred — the decision of
	// *which* computation to run is postponed along with the work.
	chained := base.FlatMap(func(n int) *monad.Lazy[string] {
		return monad.Defer(func() string { return strings.Repeat("*", n/7) })
	})
	fmt.Printf("  FlatMap: %q\n", chained.Evaluate())
}

func sharing() {
	// Each Lazy in a chain caches independently, so a shared prefix is
	// computed once however many derived values are built on top of it. This
	// is the property that makes Lazy useful for dependency graphs.
	var runs atomic.Int32
	base := monad.Defer(func() int {
		runs.Add(1)
		return 10
	})

	plusOne := base.Map(func(n int) int { return n + 1 })
	timesTwo := base.Map(func(n int) int { return n * 2 })

	fmt.Printf("  %d and %d, base ran %d time(s)\n",
		plusOne.Evaluate(), timesTwo.Evaluate(), runs.Load())
}

func concurrent() {
	// Evaluate is safe to call from several goroutines. The first caller runs
	// the computation while the rest block, and every one of them observes the
	// same value — so a Lazy can be shared without a mutex of your own.
	var runs atomic.Int32
	shared := monad.Defer(func() int {
		runs.Add(1)
		time.Sleep(10 * time.Millisecond)
		return 42
	})

	var wg sync.WaitGroup
	results := make([]int, 8)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = shared.Evaluate()
		}()
	}
	wg.Wait()

	fmt.Printf("  8 goroutines, %d run(s), all agreed on %d: %v\n",
		runs.Load(), results[0], allEqual(results))
}

func allEqual(xs []int) bool {
	for _, x := range xs {
		if x != xs[0] {
			return false
		}
	}
	return true
}

func inspecting() {
	// String deliberately does not force evaluation: printing a value for
	// debugging should never be what triggers expensive work. An unevaluated
	// Lazy says so rather than reporting a zero it does not have.
	lazy := monad.Defer(func() int { return 42 })

	fmt.Printf("  before: %s\n", lazy)
	lazy.Evaluate()
	fmt.Printf("  after:  %s\n", lazy)
}

func fallible() {
	// Lazy has no failure channel of its own. When the deferred work can fail,
	// defer a Result: the memoization then covers the error case too, so a
	// failing computation is not retried on every access.
	var attempts atomic.Int32
	risky := monad.Defer(func() monad.Result[int] {
		attempts.Add(1)
		return monad.Failure[int](errConnRefused)
	})

	first := risky.Evaluate()
	second := risky.Evaluate()
	fmt.Printf("  %v (attempts: %d, cached: %v)\n",
		first, attempts.Load(), errors.Is(second.Error(), errConnRefused))
}

// Command promise demonstrates monad.Promise, an asynchronous computation whose
// outcome is a Result.
//
// A Promise starts running the moment it is created and caches what it
// produces. The value it gives you is a Result, so a goroutine that failed is
// not a special case to handle separately — it is the same failure that flows
// through Map, FlatMap, and Recover like any other.
package main

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	monad "github.com/grandper/go-monad"
)

var errUnreachable = errors.New("service unreachable")

func main() {
	fmt.Println("=== Work Starts Immediately ===")
	eager()

	fmt.Println("\n=== Await Caches the Outcome ===")
	caching()

	fmt.Println("\n=== Map and FlatMap Compose Async Steps ===")
	composing()

	fmt.Println("\n=== Failure Flows Through ===")
	failing()

	fmt.Println("\n=== Then and Catch: Observing a Settlement ===")
	observing()

	fmt.Println("\n=== Recovering ===")
	recovering()

	fmt.Println("\n=== Running Work in Parallel ===")
	fanOut()

	fmt.Println("\n=== Cancellation Is Yours to Supply ===")
	cancellation()
}

// fetch stands in for a slow call. It returns a Result, which is the contract
// NewPromise expects: report failure in the value, not by panicking.
func fetch(name string, delay time.Duration) monad.Promise[string] {
	return monad.NewPromise(func() monad.Result[string] {
		time.Sleep(delay)
		if name == "" {
			return monad.Failure[string](errUnreachable)
		}
		return monad.Success("data from " + name)
	})
}

func eager() {
	// Creating the Promise starts the goroutine. By the time the sleep below
	// finishes, the work has already been done — which is the difference
	// between Promise and the two lazy types.
	start := time.Now()
	p := fetch("alpha", 50*time.Millisecond)

	time.Sleep(60 * time.Millisecond)
	fmt.Printf("  %v after %v of waiting\n", p.Await(), time.Since(start).Round(10*time.Millisecond))
}

func caching() {
	// Await blocks the first time and returns the stored Result afterwards, so
	// several parts of a program can await the same Promise without any of
	// them re-triggering the work.
	var runs atomic.Int32
	p := monad.NewPromise(func() monad.Result[int] {
		runs.Add(1)
		return monad.Success(42)
	})

	for range 3 {
		_ = p.Await()
	}
	fmt.Printf("  three awaits, %d run(s)\n", runs.Load())
}

func composing() {
	// Map and FlatMap do not block: each returns a new Promise whose goroutine
	// waits on the previous one. The pipeline is described now and resolved
	// when you finally await it.
	pipeline := fetch("beta", 20*time.Millisecond).
		Map(func(s string) int { return len(s) }).
		FlatMap(func(n int) monad.Promise[string] {
			return monad.NewPromise(func() monad.Result[string] {
				return monad.Success(fmt.Sprintf("%d bytes", n))
			})
		})

	fmt.Println(" ", pipeline.Await())
}

func failing() {
	// A failure short-circuits the rest of the chain. The transformations are
	// never called, and the original error arrives intact at the end.
	result := fetch("", 10*time.Millisecond).
		Map(func(string) int { panic("never reached") }).
		Await()

	fmt.Printf("  %v (is unreachable: %v)\n", result, errors.Is(result.Error(), errUnreachable))
}

func observing() {
	// Then and Catch run a side effect on the matching branch and hand back a
	// new Promise carrying the same Result. They are for logging and metrics;
	// because the Promise they return is the one that runs the effect, it has
	// to be awaited for anything to happen.
	fetch("gamma", 10*time.Millisecond).
		Then(func(s string) { fmt.Println("  ok:", s) }).
		Catch(func(err error) { fmt.Println("  failed:", err) }).
		Await()

	fetch("", 10*time.Millisecond).
		Then(func(s string) { fmt.Println("  ok:", s) }).
		Catch(func(err error) { fmt.Println("  failed:", err) }).
		Await()
}

func recovering() {
	// Recover substitutes a value for a failure.
	fmt.Printf("  Recover:     %v\n",
		fetch("", 10*time.Millisecond).
			Recover(func(error) string { return "default" }).
			Await())

	// RecoverWith substitutes another Promise, so the fallback is itself
	// asynchronous — a retry against a second host rather than a constant.
	fmt.Printf("  RecoverWith: %v\n",
		fetch("", 10*time.Millisecond).
			RecoverWith(func(err error) monad.Promise[string] {
				fmt.Printf("    primary failed (%v), trying the backup\n", err)
				return fetch("backup", 10*time.Millisecond)
			}).
			Await())
}

func fanOut() {
	// Because each Promise is already running, creating several and awaiting
	// them afterwards runs them concurrently. The total time is the slowest
	// one, not the sum — and no channel or WaitGroup appears in this code.
	start := time.Now()
	promises := []monad.Promise[string]{
		fetch("one", 40*time.Millisecond),
		fetch("two", 40*time.Millisecond),
		fetch("three", 40*time.Millisecond),
	}

	for _, p := range promises {
		fmt.Println("  ", p.Await())
	}
	fmt.Printf("  three 40ms calls took %v\n", time.Since(start).Round(10*time.Millisecond))
}

func cancellation() {
	// Promise has no built-in timeout or cancellation: the computation is an
	// ordinary function, so the way to bound it is the way you always would —
	// close over a context and report a Failure when it is done.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	p := monad.NewPromise(func() monad.Result[string] {
		select {
		case <-time.After(200 * time.Millisecond):
			return monad.Success("finished in time")
		case <-ctx.Done():
			return monad.Failure[string](fmt.Errorf("giving up: %w", ctx.Err()))
		}
	})

	fmt.Println(" ", p.Await())
}

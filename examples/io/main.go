// Command io demonstrates monad.IO, a description of a side effect that has not
// happened yet.
//
// Building an IO performs no work; it records what would be done. That turns an
// effect into an ordinary value you can pass around, store, and compose, with
// one place — the call to Run — where the effects actually occur. Unlike Lazy,
// an IO re-runs every time, which is what makes it right for effects rather
// than for cached values.
package main

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	monad "github.com/grandper/go-monad"
)

var errMissingFile = errors.New("no such file")

func main() {
	fmt.Println("=== Describing an Effect Without Running It ===")
	describing()

	fmt.Println("\n=== Every Run Re-executes ===")
	repeating()

	fmt.Println("\n=== Composing a Program ===")
	composing()

	fmt.Println("\n=== Failure Short-Circuits ===")
	failing()

	fmt.Println("\n=== Recovering ===")
	recovering()

	fmt.Println("\n=== IO Compared With Lazy ===")
	versusLazy()

	fmt.Println("\n=== The Zero Value ===")
	zeroValue()
}

// fakeFS stands in for the outside world so the example stays self-contained.
func fakeFS() map[string]string {
	return map[string]string{
		"/etc/app.conf": "mode=production\nworkers=4",
		"./app.conf":    "mode=development\nworkers=1",
	}
}

// readFile returns a description of a read, not the contents of one. Nothing
// touches the "filesystem" until the returned IO is run.
func readFile(path string) monad.IO[string] {
	return monad.NewIO(func() monad.Result[string] {
		fmt.Printf("    (reading %s)\n", path)
		body, ok := fakeFS()[path]
		if !ok {
			return monad.Failure[string](fmt.Errorf("%w: %s", errMissingFile, path))
		}
		return monad.Success(body)
	})
}

func describing() {
	// Constructing the IO prints nothing: the effect inside has not been
	// invoked. This is what lets an effect be built in one place and run in
	// another, under a policy the builder does not need to know about.
	action := readFile("/etc/app.conf")
	fmt.Println("  IO built; nothing has been read")
	fmt.Printf("  %s\n", action)

	fmt.Println("  now running it:")
	fmt.Printf("  first line: %q\n", firstLine(action.Run().OrElse("")))
}

func repeating() {
	// Each Run performs the effect again. That is deliberate: an effect whose
	// result is cached is no longer an effect, and reading a file twice should
	// be able to observe a change between the reads.
	var reads atomic.Int32
	counter := monad.NewIO(func() monad.Result[int] {
		return monad.Success(int(reads.Add(1)))
	})

	fmt.Printf("  three runs: %v, %v, %v\n",
		counter.Run().OrElse(0), counter.Run().OrElse(0), counter.Run().OrElse(0))
}

func composing() {
	// Map and FlatMap build a bigger description out of smaller ones. The
	// whole pipeline is still inert; the effects happen in order at Run, and
	// only then.
	program := readFile("/etc/app.conf").
		Map(strings.TrimSpace).
		FlatMap(func(body string) monad.IO[int] {
			return monad.NewIO(func() monad.Result[int] {
				return monad.Success(len(strings.Split(body, "\n")))
			})
		})

	fmt.Println("  program assembled, still nothing has run")
	fmt.Printf("  running: %v line(s)\n", program.Run().OrElse(0))
}

func failing() {
	// A failing step stops the chain. The steps after it are never run, so
	// their effects never happen either — the failure prevents the work rather
	// than being discovered after it.
	result := readFile("/nope.conf").
		Map(func(string) string { panic("never reached") }).
		Run()

	fmt.Printf("  %v (missing file: %v)\n",
		result, errors.Is(result.Error(), errMissingFile))
}

func recovering() {
	// Recover supplies a plain value for the failed case.
	fmt.Printf("  Recover:     %q\n",
		readFile("/nope.conf").Recover(func(error) string { return "mode=default" }).Run().OrElse(""))

	// RecoverWith supplies another IO, so the fallback can perform its own
	// effects — the usual shape for "try this file, then that one". The
	// fallback is only built if the first attempt actually fails.
	config := readFile("/nope.conf").
		RecoverWith(func(error) monad.IO[string] {
			return readFile("./app.conf")
		})
	fmt.Printf("  RecoverWith: %q\n", firstLine(config.Run().OrElse("")))
}

func versusLazy() {
	// The same computation as an IO and as a Lazy. Both defer the work; only
	// Lazy remembers it. Choosing between them is choosing whether repeating
	// the work is correct.
	var ioRuns, lazyRuns atomic.Int32

	effect := monad.NewIO(func() monad.Result[int] {
		return monad.Success(int(ioRuns.Add(1)))
	})
	cached := monad.Defer(func() int { return int(lazyRuns.Add(1)) })

	for range 3 {
		_ = effect.Run()
		_ = cached.Evaluate()
	}
	fmt.Printf("  after 3 rounds — IO ran %d time(s), Lazy ran %d time(s)\n",
		ioRuns.Load(), lazyRuns.Load())
}

func zeroValue() {
	// An IO with no effect has nothing to perform. Because Run already returns
	// a Result, that is reported as a failure rather than a panic, and it
	// flows through a chain like any other error — Recover can even rescue it.
	var unset monad.IO[int]

	r := unset.Run()
	fmt.Printf("  %v (uninitialized: %v)\n", r, errors.Is(r.Error(), monad.ErrUninitialized))
	fmt.Printf("  recovered: %v\n", unset.Recover(func(error) int { return -1 }).Run())
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

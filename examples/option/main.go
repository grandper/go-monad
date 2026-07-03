// Command option demonstrates monad.Option, which models a value that may be
// absent.
//
// The idea Option exists to serve: absence should be visible in the type, not
// discovered at runtime. A *User can be nil and nothing in the signature warns
// you; an Option[User] cannot be read without deciding what the empty case
// means. Everything below is a variation on that one theme.
package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	monad "github.com/grandper/go-monad"
)

var errNoValue = errors.New("no value present")

func main() {
	fmt.Println("=== Creating and Inspecting ===")
	creating()

	fmt.Println("\n=== Map: Working Inside the Box ===")
	mapping()

	fmt.Println("\n=== FlatMap: Chaining Operations That May Fail ===")
	flatMapping()

	fmt.Println("\n=== Filter: Turning a Value Into an Absence ===")
	filtering()

	fmt.Println("\n=== Fold: Handling Both Cases at Once ===")
	folding()

	fmt.Println("\n=== Getting the Value Out ===")
	unwrapping()

	fmt.Println("\n=== Side Effects Without Unwrapping ===")
	sideEffects()

	fmt.Println("\n=== Combining Independent Options ===")
	combining()

	fmt.Println("\n=== Crossing Into Result and error ===")
	bridging()

	fmt.Println("\n=== The Zero Value ===")
	zeroValue()
}

func creating() {
	present := monad.Some(42)
	absent := monad.None[int]()

	// None needs an explicit type argument: there is no value to infer it from.
	fmt.Printf("Some: %v (present: %v)\n", present, present.IsPresent())
	fmt.Printf("None: %v (empty: %v)\n", absent, absent.IsEmpty())

	// Some(0) is a *present* zero. This is the distinction a bare int cannot
	// draw, and the reason "0 means unset" conventions eventually break.
	zero := monad.Some(0)
	fmt.Printf("Some(0) is present: %v, but == None: %v\n",
		zero.IsPresent(), zero == absent)
}

func mapping() {
	// Map applies a function to the value if there is one. The empty case is
	// not something you handle here — it simply skips the function, so a
	// transformation never has to ask whether it has anything to work on.
	fmt.Println(monad.Some(21).Map(func(n int) int { return n * 2 }))
	fmt.Println(monad.None[int]().Map(func(n int) int { return n * 2 }))

	// The type inside the box can change. Option[int] becomes Option[string]
	// while the presence or absence rides along untouched.
	label := monad.Some(42).Map(func(n int) string { return "n=" + strconv.Itoa(n) })
	fmt.Println(label)

	// Map never calls the function on an empty Option, so an expensive or
	// panicking transformation is safe to write without a nil guard.
	monad.None[int]().Map(func(_ int) int { panic("never reached") })
	fmt.Println("Map on None did not call the function")
}

// parsePort is the shape that composes: a function from a plain value to an
// Option, reporting failure by returning None rather than by a second value.
func parsePort(s string) monad.Option[int] {
	n, err := strconv.Atoi(s)
	if err != nil {
		return monad.None[int]()
	}
	return monad.Some(n)
}

func flatMapping() {
	// Map with a function that itself returns an Option would give you
	// Option[Option[int]]. FlatMap is Map plus one unwrapping, which is what
	// keeps a chain flat however many fallible steps it has.
	fmt.Println(monad.Some("8080").FlatMap(parsePort))
	fmt.Println(monad.Some("http").FlatMap(parsePort))

	// Once any step yields None the rest are skipped. The absence propagates
	// on its own; no step in the middle checks for it.
	result := monad.Some(" 443 ").
		Map(strings.TrimSpace).
		FlatMap(parsePort).
		Map(func(p int) string { return fmt.Sprintf(":%d", p) })
	fmt.Println(result)
}

func filtering() {
	// Filter is the one operation that can *create* an absence from a present
	// value. It turns a validation rule into a value of the same type, so the
	// rule composes with everything else instead of needing its own if.
	inRange := func(p int) bool { return p >= 1 && p <= 65535 }

	fmt.Println(monad.Some(8080).Filter(inRange))
	fmt.Println(monad.Some(99999).Filter(inRange))
	fmt.Println(monad.None[int]().Filter(inRange))
}

func folding() {
	// Fold is the way out when both cases must produce the same type. Passing
	// a function for each case makes it impossible to forget the empty one —
	// the compiler asks for it.
	describe := func(o monad.Option[int]) string {
		return o.Fold(
			func() string { return "no port configured" },
			func(p int) string { return fmt.Sprintf("listening on :%d", p) },
		)
	}

	fmt.Println(describe(monad.Some(8080)))
	fmt.Println(describe(monad.None[int]()))
}

func unwrapping() {
	// OrElse supplies a default for the empty case.
	fmt.Printf("OrElse:      %d\n", monad.None[int]().OrElse(8080))

	// OrElseGet defers the fallback to a function, so an expensive default is
	// only computed when it is actually needed.
	fmt.Printf("OrElseGet:   %d\n", monad.None[int]().OrElseGet(func() int {
		fmt.Print("  (computing the fallback) ")
		return 3000
	}))

	// OrElseError converts to Go's (T, error) convention at the boundary of
	// the library, where callers expect an error rather than an Option.
	value, err := monad.None[int]().OrElseError(errNoValue)
	fmt.Printf("OrElseError: %d, %v\n", value, err)

	// OrElsePanic is for cases where absence is a programmer error rather than
	// a condition to handle. The message is yours, so the panic names the
	// invariant that was violated.
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("OrElsePanic: recovered from %q\n", r)
		}
	}()
	_ = monad.None[int]().OrElsePanic("port must be configured by now")
}

func sideEffects() {
	// IfPresent and IfEmpty run a function for its effect and return nothing,
	// so they end a chain. They exist for logging and metrics, where there is
	// no new value to carry forward.
	monad.Some("alice").IfPresent(func(name string) {
		fmt.Println("  welcome back,", name)
	})
	monad.None[string]().IfEmpty(func() {
		fmt.Println("  no session found, showing the login page")
	})
}

func combining() {
	// Map handles one Option. When a function needs two or more, ApplyOption
	// combines them: the result is present only if every input is, and the
	// first absence wins without any nesting of checks.
	add := func(a int) func(int) int {
		return func(b int) int { return a + b }
	}

	both := monad.ApplyOption(monad.Some(add(2)), monad.Some(3))
	fmt.Printf("Some(2) + Some(3) = %v\n", both)

	missing := monad.ApplyOption(monad.Some(add(2)), monad.None[int]())
	fmt.Printf("Some(2) + None    = %v\n", missing)
}

func bridging() {
	// An Option says whether a value is there; a Result says why it is not.
	// ToResult upgrades one to the other by supplying the missing reason.
	found := monad.Some(42).ToResult(errNoValue)
	absent := monad.None[int]().ToResult(errNoValue)
	fmt.Printf("ToResult: %v / %v\n", found, absent)

	// The standalone functions are equivalent to the methods and exist for
	// callers who prefer a function-oriented style, or an older Go release.
	fmt.Printf("MapOption: %v\n", monad.MapOption(monad.Some(21), func(n int) int { return n * 2 }))
}

func zeroValue() {
	// The zero value of an Option is None, which is what makes it safe to
	// embed in a struct: reading a field nobody assigned yields an absence
	// rather than a panic.
	type settings struct {
		Port monad.Option[int]
	}

	var s settings
	fmt.Printf("unassigned field: %v -> OrElse gives %d\n", s.Port, s.Port.OrElse(8080))
}

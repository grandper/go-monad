# go-monad

[![Go Reference](https://pkg.go.dev/badge/github.com/grandper/go-monad.svg)](https://pkg.go.dev/github.com/grandper/go-monad)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.27+](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go)](https://go.dev/doc/go1.27)

`go-monad` is a package that brings *monads* to Golang: a small set of generic
container types — `Option`, `Result`, `Either`, `Promise`, `Lazy`, and `IO` —
that let you chain computations together without sprinkling `if err != nil`
and `if x == nil` checks between every step.

**Main features:**

- Six monadic types: `Option[T]`, `Result[T]`, `Either[L, R]`, `Promise[T]`,
  `Lazy[T]`, and `IO[T]`.
- Every type supports `Map` and `FlatMap`; the synchronous ones also support
  `Fold`, and the ones that can fail support `Filter` and `Recover`.
- `Apply` (applicative functor) for `Option` and `Result`; `Either` adds
  `MapLeft`, `Swap`, and `ToOption` for working with its left side.
- Two equivalent styles: fluent method chaining (`opt.Map(f).FlatMap(g)`, made
  possible by generic methods in Go 1.27) and standalone generic functions
  (`MapOption(opt, f)`).
- `Option` reads and writes JSON and YAML out of the box: a present value is
  encoded as the value, an empty one as `null`.
- Typed constructors that read optional settings from environment variables
  (`GetIntFromEnv`, `MustGetDurationFromEnv`, `GetStringSliceFromEnv`, ...),
  built on a generic `OptionFromEnv` that accepts any parser.
- No dependencies outside the standard library (`testify` and `yaml.v3` are only
  used by the tests).
- Every type satisfies the three monad laws (left identity, right identity,
  associativity), so pipelines can be refactored freely — and the test suite
  verifies all three for all six types.
- Every type has a usable zero value, so a monad embedded in a struct is safe
  to read before it is assigned.

## Installation

```bash
go get github.com/grandper/go-monad
```

The package requires **Go 1.27 or later**: the method-chaining API relies on
type parameters on methods, which were introduced in that release.

Throughout this document the package is imported under the alias `monad`:
```go
import monad "github.com/grandper/go-monad"
```

## What is a monad?
The word sounds intimidating, but the idea is simple: a monad is a **box**
that holds a value *and some context* about it — "there may be no value",
"the computation may have failed", "the value is not available yet", "the
value has not been computed yet", "computing the value has side effects".
The box gives you a handful of operations to work *on the value inside* without
having to open the box, check the context, and close it again at every step.

### The old way: checking every step
Imagine you need to read a port number from a string, make sure it is valid,
and turn it into an address. Idiomatic Go looks like this:
```go
func address(raw string) (string, error) {
	port, err := strconv.Atoi(raw)
	if err != nil {
		return "", fmt.Errorf("invalid port: %w", err)
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("port out of range: %d", port)
	}
	return fmt.Sprintf(":%d", port), nil
}
```
Every step has to check the outcome of the previous one before doing its own
work. Three steps, two checks, and the actual logic — parse, validate, format —
is buried between them.

### The monadic way
With a `Result`, the checks disappear into the box. Each step only describes
what to do *when there is a value*; a failure at any point simply flows
through to the end:
```go
func address(raw string) monad.Result[string] {
	return parsePort(raw).
		Filter(func(p int) bool { return p >= 1 && p <= 65535 }, errors.New("port out of range")).
		Map(func(p int) string { return fmt.Sprintf(":%d", p) })
}
```
The rest of this document explains each of the boxes the library provides,
the operations they share, and when to reach for which one.

## Core Concepts
All six types are built on the same three ideas from functional programming —
*functor*, *applicative*, and *monad*. They are stacked: each one adds one
operation on top of the previous. You do not need to know the theory to use the
library, but knowing the vocabulary makes the API predictable: once you know
what `Map` does on `Option`, you know what it does on `Promise`.

### Functor: `Map`
A functor is any box that supports **`Map`**: apply a plain function to the
value inside, and get a new box holding the result. The context is preserved —
an empty `Option` stays empty, a failed `Result` stays failed, a `Promise` stays
asynchronous.
```
Map : F[A] → (A → B) → F[B]
```
In Go terms, `Map` takes a `func(A) B` and turns an `F[A]` into an `F[B]`:
```go
length := monad.Some("hello").Map(func(s string) int { return len(s) })
// length is Option[int] holding Some(5)

nothing := monad.None[string]().Map(func(s string) int { return len(s) })
// the function is never called; nothing is None
```
All six types implement `Map`.

### Monad: `FlatMap`
A monad is a functor that also supports **`FlatMap`** (called `bind`, `>>=`,
`andThen`, or `then` in other languages). The difference with `Map` is the
function you pass: it returns a *box* rather than a plain value.
```
FlatMap : M[A] → (A → M[B]) → M[B]
```
Why does that matter? Suppose you have a function that can itself fail to
produce a value, such as parsing an integer:
```go
parse := func(s string) monad.Option[int] {
	n, err := strconv.Atoi(s)
	if err != nil {
		return monad.None[int]()
	}
	return monad.Some(n)
}
```
If you use `Map` with it, you end up with a box inside a box:
```go
nested := monad.Some("42").Map(parse) // Option[Option[int]] — awkward to use
```
`FlatMap` applies the function *and flattens* the result into a single box:
```go
flat := monad.Some("42").FlatMap(parse) // Option[int] holding Some(42)
```
This is what makes chaining possible: every step in a pipeline can be a
function that returns a box, and the pipeline stays flat. If any step returns
an empty/failed box, the following steps are skipped and the empty/failed box
is what comes out at the end.

Rule of thumb: use `Map` when your function returns a plain value, and
`FlatMap` when it returns a box of the same kind. All six types implement
`FlatMap`.

### Applicative: `Apply`
An applicative sits between functor and monad. It supports **`Apply`**: apply
a *function that is itself inside a box* to a value inside a box.
```
Apply : F[A → B] → F[A] → F[B]
```
When do you end up with a function inside a box? When you `Map` a function of
two arguments (written in curried form) over its first argument: you get back a
box holding "the function waiting for its second argument". `Apply` then feeds
it the second argument, which lives in another box:
```go
add := func(a int) func(int) int {
	return func(b int) int { return a + b }
}

sum := monad.ApplyOption(monad.Some(1).Map(add), monad.Some(2))
// sum is Some(3)

none := monad.ApplyOption(monad.Some(1).Map(add), monad.None[int]())
// none is None — one of the two inputs is empty
```
The practical difference with `FlatMap`: `FlatMap` expresses computations that
*depend* on each other (the second needs the result of the first), whereas
`Apply` combines computations that are *independent* (both inputs exist on
their own, you only need them together). `ApplyOption` and `ApplyResult` are
provided as standalone functions.

### Leaving the box: `Fold` and friends
At some point you need to get a plain value back out — to print it, return it
from an HTTP handler, or hand it to code that does not know about monads.
**`Fold`** is the universal way to do it: you provide one function per possible
state of the box, both returning the same type `B`, and `Fold` calls the one
that matches.
```
Fold : M[A] → (context → B) → (A → B) → B
```
```go
message := monad.Some(42).Fold(
	func() string { return "no value" },
	func(n int) string { return fmt.Sprintf("value is %d", n) },
)
// message is "value is 42"
```
`Fold` is the only operation that forces you to handle *every* case, which is
what makes it safe. For the common cases there are shortcuts — `OrElse`,
`OrElseGet`, `IfPresent`/`IfSuccess`, `Await`, `Evaluate`, `Run` — described
with each type below.

### Filter and Recover
Two more operations show up on the types that have a notion of failure:
- **`Filter`** turns a successful box into an empty/failed one when the value
  does not satisfy a predicate. It is the way to add a validation step to a
  pipeline without leaving it.
- **`Recover`** is the mirror image of `Map`: instead of transforming the
  success side, it transforms the *failure* side back into a success. It is how
  you supply a fallback in the middle of a pipeline instead of at the end.

Both are explained in detail with `Option`, `Result`, `Promise`, and `IO`.

### Monad laws
Every type in this library satisfies the three monad laws, where `Unit` is the
constructor that puts a plain value in the box (`Some`, `Success`, `Right`,
`Resolve`, `Defer`, `PureIO`):

| Law            | Statement                                                       |
|----------------|-----------------------------------------------------------------|
| Left identity  | `Unit(a).FlatMap(f)` ≡ `f(a)`                                   |
| Right identity | `m.FlatMap(Unit)` ≡ `m`                                         |
| Associativity  | `m.FlatMap(f).FlatMap(g)` ≡ `m.FlatMap(x => f(x).FlatMap(g))`   |

In practice the laws guarantee that wrapping a value and immediately unwrapping
it changes nothing, and that it does not matter how you group the steps of a
pipeline — you can refactor a long chain into helper functions freely.

This is not just an aspiration: the test suite checks all three laws for all six
types, on both the success and the failure path, in the `Test<Type>MonadLaws`
functions.

### Zero values
Go gives every variable a zero value whether you ask for one or not: a struct
field you have not assigned, a slice element `append` just grew into, a map
lookup that missed. A type that panics in that state is a type you cannot embed
safely, so every monad in this library is usable before it is initialized.

The three synchronous types have a natural interpretation of "nothing here yet",
and they use it:

| Zero value    | Behaves as        | Rationale                                  |
|---------------|-------------------|--------------------------------------------|
| `Option[T]`   | `None`            | No value has been put in the box           |
| `Result[T]`   | `Failure(ErrUninitialized)` | No computation has succeeded      |
| `Either[L,R]` | `Left(zero L)`    | Left is the non-success side by convention  |

```go
type Config struct {
	Port    monad.Option[int]
	Fallback monad.Either[string, int]
}

var cfg Config
cfg.Port.OrElse(8080)      // 8080 — the zero Option is None
cfg.Fallback.IsLeft()      // true — the zero Either is a Left
```

The three deferred types are different: `Lazy`, `Promise`, and `IO` each wrap a
*function*, and a zero value has no function to call. There is no sensible
result to invent, so they report the mistake instead — each through whatever
channel it already has. `IO.Run` and `Promise.Await` return a `Result`, so they
fail with an error you can test for:
```go
var io monad.IO[int]

r := io.Run()
errors.Is(r.Error(), monad.ErrUninitialized) // true
```
`ErrUninitialized` is also what a `Result` reports when it is a failure with no
error of its own — the zero value, or one built with an explicit `nil` such as
`Failure[T](nil)`, `Filter(p, nil)`, or `Option.ToResult(nil)`. The guarantee is
that `IsFailure()` and `Error() != nil` never disagree, so bridging back to
`(T, error)` is always safe.
That failure flows through a chain like any other, so `Recover` can rescue it
and nothing panics along the way. `Lazy.Evaluate` returns a bare `T` with no
room for an error, so it panics with a message naming the mistake rather than
returning a zero value that would silently look like a real result.

The rule of thumb: if a zero monad reaches your code, something upstream forgot
to build it. The library makes that discoverable instead of fatal.

### Two styles: methods and standalone functions
Every operation that changes the type inside the box (`Map`, `FlatMap`, `Fold`)
exists in two forms that behave exactly the same:

- a **method**, which reads left to right and chains naturally:
  ```go
  value := monad.Some(42).
  	Map(func(v int) string { return strconv.Itoa(v) }).
  	FlatMap(parsePositiveInt).
  	OrElse(0)
  ```
- a **standalone function**, named after the type (`MapOption`, `FlatMapResult`,
  `FoldEither`, `MapPromise`, `FlatMapLazy`, `MapIO`, ...), which is handy when
  you want to pass the operation around as a value or prefer a more
  function-oriented style:
  ```go
  opt := monad.Some(42)
  text := monad.MapOption(opt, func(v int) string { return strconv.Itoa(v) })
  parsed := monad.FlatMapOption(text, parsePositiveInt)
  value := parsed.OrElse(0)
  ```

Operations that keep the same type (`Filter`, `Recover`, `OrElse`, `Await`,
`Run`, ...) only exist as methods, since they do not need extra type
parameters. Pick whichever style reads best; mixing them is fine.

## The Types
In this section we go through each type in turn: what it represents, how to
create one, how to transform it, and how to get a value back out. A short
"which one should I use" guide closes the section.

### Option
`Option[T]` represents a value that **may be absent**. It is either `Some(value)`
or `None`. Use it wherever you would otherwise return a pointer that may be
`nil`, a `(value, ok bool)` pair, or a sentinel like `-1` or `""`.

#### Creating an Option
```go
name := monad.Some("Alice")   // Option[string] holding a value
empty := monad.None[string]() // Option[string] holding nothing
```
`None` needs an explicit type argument, since there is no value to infer it
from. The zero value of `Option[T]` is `None`, so an `Option` field in a struct
starts out empty.

#### Inspecting an Option
```go
name.IsPresent() // true
name.IsEmpty()   // false
name.String()    // "Some(Alice)"; None prints "None"
```
`IsPresent` and `IsEmpty` only tell you *whether* there is a value. To use the
value, reach for `Map`, `FlatMap`, `Fold`, or one of the `OrElse` helpers
below — they take care of the empty case for you.

#### Transforming with `Map`
`Map` applies a function to the value when there is one, and does nothing
otherwise. The function you pass never has to worry about the empty case:
```go
upper := monad.Some("alice").Map(strings.ToUpper)     // Some(ALICE)
still := monad.None[string]().Map(strings.ToUpper)     // None — ToUpper is not called
length := monad.Some("alice").Map(func(s string) int { // Some(5), an Option[int]
	return len(s)
})
```
The result can be a different type than the input: `Map` on an `Option[string]`
with a `func(string) int` gives an `Option[int]`.

#### Chaining with `FlatMap`
`FlatMap` is for functions that themselves return an `Option`, such as a lookup
that may find nothing:
```go
users := map[string]string{"alice": "alice@example.com"}

lookupEmail := func(user string) monad.Option[string] {
	email, ok := users[user]
	if !ok {
		return monad.None[string]()
	}
	return monad.Some(email)
}

domainOf := func(email string) monad.Option[string] {
	_, domain, ok := strings.Cut(email, "@")
	if !ok {
		return monad.None[string]()
	}
	return monad.Some(domain)
}

domain := monad.Some("alice").FlatMap(lookupEmail).FlatMap(domainOf) // Some(example.com)
missing := monad.Some("bob").FlatMap(lookupEmail).FlatMap(domainOf)  // None — domainOf is never called
```
Each step gets the value produced by the previous one, and the first `None`
short-circuits the rest of the chain.

#### Narrowing with `Filter`
`Filter` keeps the value only if it satisfies a predicate; otherwise the
`Option` becomes `None`. It lets you put a validation rule in the middle of a
chain:
```go
isEven := func(n int) bool { return n%2 == 0 }

monad.Some(4).Filter(isEven)          // Some(4)
monad.Some(3).Filter(isEven)          // None
monad.None[int]().Filter(isEven)      // None — the predicate is not called
```

#### Reducing with `Fold`
`Fold` takes two functions — one for `None`, one for `Some` — and returns
whatever the matching one returns. Both must return the same type:
```go
describe := func(o monad.Option[string]) string {
	return o.Fold(
		func() string { return "nobody" },
		func(name string) string { return "hello, " + name },
	)
}

describe(monad.Some("Alice"))   // "hello, Alice"
describe(monad.None[string]()) // "nobody"
```

#### Getting the value out
When all you need is a default for the empty case, the `OrElse` family is
shorter than `Fold`:
```go
monad.Some("Alice").OrElse("stranger")    // "Alice"
monad.None[string]().OrElse("stranger")   // "stranger"
```
`OrElseGet` takes a function instead of a value, so the fallback is only
computed when it is needed — useful when it is expensive or has side effects:
```go
monad.None[string]().OrElseGet(func() string {
	return loadDefaultNameFromConfig()
})
```
`IfPresent` runs a function for its side effect when there is a value, and
does nothing otherwise. `IfEmpty` is its mirror and runs when there is none —
it takes no argument, because an empty `Option` has nothing to hand over:
```go
monad.Some("Alice").IfPresent(func(name string) {
	fmt.Println("welcome back,", name)
})

cached.IfEmpty(func() {
	metrics.CacheMiss.Inc()
})
```
Neither returns anything, so they end a chain rather than continue it. When you
want to keep going, use `Map` or `Fold` instead.
At the boundary with code that speaks `(T, error)`, `OrElseError` hands the value
over and turns the empty case into the error you give it — so the pair lines up
with Go's convention and can be returned as is:
```go
func (repo Repository) FindByEmail(email string) (User, error) {
	return repo.lookup(email).OrElseError(ErrUserNotFound)
}
```
When absence is a programming error rather than a normal outcome — a value you
have just validated, a lookup in a table you populated yourself — `OrElsePanic`
returns the value or panics with the message you supply. Reach for it
sparingly; in doubt, prefer `OrElse` or `OrElseError`:
```go
port := monad.Some(8080).OrElsePanic("port must be configured") // 8080
```

#### Serialization
`Option` implements `MarshalJSON`/`UnmarshalJSON` and `MarshalYAML`/
`UnmarshalYAML`, so it satisfies `json.Marshaler`/`json.Unmarshaler` and the
equivalent
YAML interfaces, so it can be used directly as a struct field in configuration
files and API payloads. The encoding is deliberately transparent: a present
`Option` is written exactly as its value would be, an empty one as `null`.
```go
type Profile struct {
	Name     monad.Option[string] `json:"name"`
	Nickname monad.Option[string] `json:"nickname"`
}

data, _ := json.Marshal(Profile{Name: monad.Some("Ada")})
// {"name":"Ada","nickname":null}
```
Decoding goes the other way. A key holding a value gives `Some`, a key set to
`null` gives `None`, and a key that is missing altogether also gives `None`,
because the zero value of an `Option` is `None`:
```go
var profile Profile
_ = json.Unmarshal([]byte(`{"name": "Ada"}`), &profile)
profile.Name     // Some(Ada)
profile.Nickname // None
```
If you would rather not emit `null` for absent fields, use the `omitzero`
struct-tag option (Go 1.24+). `Option` implements `IsZero`, so `omitzero` skips
empty Options and keeps present ones — including a present zero value such as
`Some(0)` or `Some("")`. Note that `omitempty` does *not* do this for JSON: an
`Option` is a struct, which `omitempty` never considers empty.
```go
type Profile struct {
	Name     monad.Option[string] `json:"name"`
	Nickname monad.Option[string] `json:"nickname,omitzero"`
}

data, _ := json.Marshal(Profile{Name: monad.Some("Ada")})
// {"name":"Ada"}
```
YAML works the same way with `gopkg.in/yaml.v3`, which recognizes the marshaling
methods by their shape. The library itself does not import a YAML package, so
nothing is pulled into your build unless you use YAML yourself. With YAML, the
`omitempty` tag option is the one that honors `IsZero`:
```go
type Config struct {
	Host monad.Option[string] `yaml:"host"`
	Port monad.Option[int]    `yaml:"port,omitempty"`
}

data, _ := yaml.Marshal(Config{Host: monad.Some("localhost")})
// host: localhost
```
Two limitations are worth knowing. First, `yaml.v3` resolves `null` nodes
itself and does not call custom unmarshalers for them, so decoding an explicit
`null` into an `Option` that *already holds a value* leaves that value in place;
decoding into a fresh struct behaves as expected. Second, because an empty
`Option` and a `null` are indistinguishable on the wire, a nested
`Some(None[int]())` is written as `null` and read back as an outer `None`.

#### Combining independent Options with `ApplyOption`
When you need two (or more) optional values *together*, and neither depends on
the other, `ApplyOption` combines them without nesting `FlatMap` calls. Write
the combining function in curried form, `Map` it over the first `Option`, then
`ApplyOption` the result to the second:
```go
fullName := func(first string) func(string) string {
	return func(last string) string { return first + " " + last }
}

first := monad.Some("Ada")
last := monad.Some("Lovelace")

name := monad.ApplyOption(first.Map(fullName), last) // Some(Ada Lovelace)

noLast := monad.ApplyOption(first.Map(fullName), monad.None[string]()) // None
```
If either input is `None`, the result is `None` and the combining function is
never called.

#### Standalone functions
`MapOption`, `FlatMapOption`, and `FoldOption` mirror the methods:
```go
greeting := monad.MapOption(monad.Some("Alice"), func(n string) string { return "Hi " + n })
domain := monad.FlatMapOption(monad.Some("alice"), lookupEmail)
text := monad.FoldOption(monad.None[int](),
	func() string { return "none" },
	func(n int) string { return strconv.Itoa(n) },
)
```

### Result
`Result[T]` represents the outcome of a computation that **can fail**. It is
either `Success(value)` or `Failure(err)`. It is the monadic counterpart of
Go's `(T, error)` return pair, and the type you will use most often.

#### Creating a Result
```go
ok := monad.Success(42)                          // Result[int] holding a value
ko := monad.Failure[int](errors.New("boom"))     // Result[int] holding an error
```
`Failure` needs an explicit type argument for the same reason as `None`. Note
that the zero value of `Result[T]` is a `Failure` with a `nil` error — always
construct a `Result` with `Success` or `Failure`.

#### Inspecting a Result
```go
ok.IsSuccess() // true
ok.IsFailure() // false
ko.Error()     // the error; nil on a Success
ok.String()    // "Success(42)"; a failure prints "Failure(boom)"
```

#### Transforming with `Map`
`Map` applies a function to the value of a `Success` and leaves a `Failure`
untouched, error included:
```go
doubled := monad.Success(21).Map(func(n int) int { return n * 2 }) // Success(42)

failed := monad.Failure[int](errors.New("boom")).
	Map(func(n int) int { return n * 2 }) // Failure(boom) — the function is not called
```

#### Chaining with `FlatMap`
`FlatMap` is for steps that can fail on their own. Each step returns a
`Result`; the first `Failure` stops the chain and becomes the final outcome:
```go
parse := func(s string) monad.Result[int] {
	n, err := strconv.Atoi(s)
	if err != nil {
		return monad.Failure[int](fmt.Errorf("parse %q: %w", s, err))
	}
	return monad.Success(n)
}

divide := func(n int) monad.Result[int] {
	if n == 0 {
		return monad.Failure[int](errors.New("division by zero"))
	}
	return monad.Success(100 / n)
}

monad.Success("5").FlatMap(parse).FlatMap(divide)   // Success(20)
monad.Success("0").FlatMap(parse).FlatMap(divide)   // Failure(division by zero)
monad.Success("x").FlatMap(parse).FlatMap(divide)   // Failure(parse "x": ...) — divide is never called
```
Compare this to the equivalent hand-written code, where each of the two steps
needs its own `if err != nil { return ... }`.

#### Narrowing with `Filter`
`Filter` on a `Result` also takes the error to use when the predicate is not
satisfied — a `Failure` needs a reason:
```go
positive := func(n int) bool { return n > 0 }
errNotPositive := errors.New("must be positive")

monad.Success(5).Filter(positive, errNotPositive)   // Success(5)
monad.Success(-1).Filter(positive, errNotPositive)  // Failure(must be positive)
```
An existing `Failure` passes through unchanged; the predicate is not called.

#### Reducing with `Fold`
`Fold` takes a function for the failure (receiving the error) and one for the
success (receiving the value):
```go
status := monad.Success("saved").Fold(
	func(err error) string { return "error: " + err.Error() },
	func(msg string) string { return "ok: " + msg },
)
// status is "ok: saved"
```

#### Recovering from failures: `Recover` and `RecoverWith`
`Recover` is the mirror image of `Map`: it works on the *failure* side. When
the `Result` is a `Failure`, the recovery function receives the error and
returns a plain value that becomes a `Success`. A `Success` is left untouched
and the function is not called:
```go
count := monad.Failure[int](errors.New("cache miss")).
	Recover(func(err error) int { return 0 }) // Success(0)
```
`RecoverWith` is its `FlatMap` counterpart: the recovery function returns a
`Result`, so the recovery itself may fail — for example when trying a second
source:
```go
readFromCache := func() monad.Result[string] {
	return monad.Failure[string](errors.New("cache miss"))
}
readFromDatabase := func(err error) monad.Result[string] {
	return monad.Success("value from db") // or a new Failure
}

value := readFromCache().RecoverWith(readFromDatabase) // Success(value from db)
```
Because `Recover` and `RecoverWith` return a `Result`, you can keep chaining
after them. That is the difference with `OrElse`: `OrElse` ends the chain and
gives you the raw value.

#### Getting the value out
```go
monad.Success(42).OrElse(0)                              // 42
monad.Failure[int](errors.New("boom")).OrElse(0)         // 0

monad.Failure[int](errors.New("boom")).OrElseGet(func(err error) int {
	log.Println("falling back:", err)
	return -1
}) // -1
```
Unlike its `Option` counterpart, `OrElseGet` receives the error, so you can log
it or derive the fallback from it.

`IfSuccess` and `IfFailure` run a side effect for one of the two cases:
```go
result.IfSuccess(func(v int) { fmt.Println("got", v) })
result.IfFailure(func(err error) { fmt.Println("failed:", err) })
```

#### Converting to an Option: `ToOption`
When the caller only cares whether there is a value and not *why* there is
none, `ToOption` drops the error:
```go
monad.Success(42).ToOption()                       // Some(42)
monad.Failure[int](errors.New("boom")).ToOption()  // None
```

#### Combining independent Results with `ApplyResult`
`ApplyResult` works like `ApplyOption`. It is convenient to build a value out of
several independently validated fields:
```go
type User struct {
	Name string
	Age  int
}

mkUser := func(name string) func(int) User {
	return func(age int) User { return User{Name: name, Age: age} }
}

validName := monad.Success("Ada")
validAge := monad.Success(36)
badAge := monad.Failure[int](errors.New("age must be positive"))

monad.ApplyResult(validName.Map(mkUser), validAge) // Success({Ada 36})
monad.ApplyResult(validName.Map(mkUser), badAge)   // Failure(age must be positive)
```
When both inputs are failures, the error of the *function* side (the first
argument) wins.

#### Standalone functions
```go
doubled := monad.MapResult(monad.Success(21), func(n int) int { return n * 2 })
parsed := monad.FlatMapResult(monad.Success("42"), parse)
text := monad.FoldResult(parsed,
	func(err error) string { return "failed: " + err.Error() },
	func(n int) string { return fmt.Sprintf("got %d", n) },
)
```

### Either
`Either[L, R]` represents a value that is **one of two types**: a `Left`
holding an `L` or a `Right` holding a `R`. It is the most general of the
synchronous types — `Option` is an `Either` whose left side carries nothing,
`Result` is an `Either` whose left side is an `error`.

By convention, and in this library, `Either` is **right-biased**: `Map`,
`FlatMap`, and the success path work on the `Right` value ("right" as in
"correct"), while `Left` passes through unchanged. Reach for `Either` when:
- the failure side is not an `error` but a structured type (an HTTP status, a
  validation report, a domain-specific rejection), or
- neither side is really a failure — for example `Either[CachedPage, FreshPage]`.

#### Creating an Either
```go
found := monad.Right[NotFound, string]("hello")           // Either[NotFound, string]
missing := monad.Left[NotFound, string](NotFound{ID: 7}) // Either[NotFound, string]
```
Both constructors need both type arguments, since only one of them can be
inferred from the argument. The zero value is a `Left` holding the zero value
of `L`, which means an `Either` sitting in a struct field is safe to read
before anything has assigned it — see [Zero values](#zero-values).

#### Inspecting an Either
```go
found.IsRight()  // true
found.IsLeft()   // false
found.String()   // "Right(hello)"; a Left prints "Left(...)"
```
`LeftValue` and `RightValue` return the side's value and a boolean telling
whether that side is present, in the familiar comma-ok style:
```go
if page, ok := found.RightValue(); ok {
	fmt.Println(page)
}
if problem, ok := missing.LeftValue(); ok {
	fmt.Println("not found:", problem.ID)
}
```

#### `Map`, `FlatMap`, and `Fold`
`Map` transforms the `Right` value; the type of the `Left` side never changes:
```go
shout := found.Map(strings.ToUpper) // Right(HELLO)
same := missing.Map(strings.ToUpper) // Left({7}) — ToUpper is not called
```
`FlatMap` chains functions that return an `Either` with the *same* left type:
```go
nonEmpty := func(s string) monad.Either[NotFound, string] {
	if s == "" {
		return monad.Left[NotFound, string](NotFound{ID: 0})
	}
	return monad.Right[NotFound, string](s)
}

found.FlatMap(nonEmpty) // Right(hello)
```
`Fold` handles both sides and is the usual way to turn an `Either` into a
plain value:
```go
message := missing.Fold(
	func(nf NotFound) string { return fmt.Sprintf("no page with id %d", nf.ID) },
	func(page string) string { return "page: " + page },
)
// message is "no page with id 7"
```
The standalone forms `MapEither`, `FlatMapEither`, and `FoldEither` take the
`Either` as their first argument.

#### Working on the left side: `MapLeft` and `Swap`
Right-bias is convenient until the value you need to change is on the left.
`MapLeft` is the mirror of `Map`: it transforms the `Left` and passes a `Right`
through untouched. It is how you normalize an error side without disturbing the
happy path:
```go
missing.MapLeft(func(nf NotFound) string {
	return fmt.Sprintf("no page with id %d", nf.ID)
})
// Either[string, string] holding Left(no page with id 7)
```
`Swap` exchanges the two sides outright, turning a `Left` into a `Right` and a
`Right` into a `Left`. It lets the right-biased operations work on what is
currently the left side, and swapping twice returns the original:
```go
missing.Swap().Map(func(nf NotFound) int { return nf.ID }) // Right(7)
```
The standalone form of `MapLeft` is `MapLeftEither`. `Swap` needs no standalone
version, since it introduces no new type parameter.

#### Converting to an Option: `ToOption`
When the left side has served its purpose and you only care whether a `Right`
is there, `ToOption` discards it:
```go
found.ToOption()   // Some(hello)
missing.ToOption() // None
```
This is the `Either` counterpart of `Result.ToOption`, and the same warning
applies: whatever the `Left` was carrying is gone afterwards, so fold or log it
first if it matters.

### Promise
`Promise[T]` represents an **asynchronous computation** that will eventually
produce a `Result[T]`. It is the library's way to run work in a goroutine and
compose what happens next without touching channels or `sync.WaitGroup`.

#### Creating a Promise
`NewPromise` starts the computation **immediately** in its own goroutine and
returns without waiting:
```go
users := monad.NewPromise(func() monad.Result[[]string] {
	time.Sleep(100 * time.Millisecond) // pretend this is a network call
	return monad.Success([]string{"alice", "bob"})
})
```
`Resolve` and `Reject` create promises that are already settled — useful as
starting points, as return values when there is nothing to compute, and in
tests:
```go
ready := monad.Resolve(42)                       // already Success(42)
broken := monad.Reject[int](errors.New("boom")) // already Failure(boom)
```

#### Waiting for the outcome: `Await`
`Await` blocks until the computation finishes and returns its `Result`. The
outcome is **cached**: calling `Await` again returns the same `Result` without
re-running anything, and it is safe to call from several goroutines:
```go
names := users.Await().OrElse(nil)
```
From here you have a plain `Result` and every operation of the previous
section is available.

#### Transforming with `Map` and `FlatMap`
`Map` and `FlatMap` do not wait. They return a *new* `Promise` whose
computation waits for the original one and then applies the function. Failures
propagate: if the original promise fails, the function is not called and the
new promise fails with the same error.
```go
count := users.Map(func(names []string) int { return len(names) })
```
`FlatMap` is for a follow-up step that is itself asynchronous:
```go
fetchProfile := func(name string) monad.Promise[string] {
	return monad.NewPromise(func() monad.Result[string] {
		return monad.Success("profile of " + name)
	})
}

firstProfile := users.
	Map(func(names []string) string { return names[0] }).
	FlatMap(fetchProfile)

fmt.Println(firstProfile.Await().OrElse("unavailable")) // profile of alice
```
Because every `NewPromise` starts right away, promises created side by side
run **in parallel**. Combine them with `FlatMap` and `Map` once you need both
outcomes:
```go
users := monad.NewPromise(fetchUsers)   // runs now
orders := monad.NewPromise(fetchOrders) // runs now, concurrently with users

report := users.FlatMap(func(u []string) monad.Promise[string] {
	return orders.Map(func(o []string) string {
		return fmt.Sprintf("%d users, %d orders", len(u), len(o))
	})
})

fmt.Println(report.Await().OrElse("report unavailable"))
```

#### Side effects on settlement: `Then` and `Catch`
`Then` runs a function with the value once the promise succeeds; `Catch` runs
a function with the error once it fails. Both return a new `Promise` with the
same outcome, so they can be inserted anywhere in a chain, for logging or
metrics:
```go
report.
	Then(func(text string) { log.Println("report ready:", text) }).
	Catch(func(err error) { log.Println("report failed:", err) })
```
Like `Map`, they do not block. The consumer still runs — the promise they
return starts its goroutine immediately, whether or not anyone awaits it — but
you have no guarantee about *when*. `Await` the returned promise at the point
where the effect must have finished, and to keep the program alive long enough
for it to happen at all.

#### Recovering with `Recover`
`Recover` turns a failed promise into a successful one, exactly like
`Result.Recover`, and leaves a successful promise untouched:
```go
safe := monad.Reject[int](errors.New("timeout")).
	Recover(func(err error) int { return -1 })

fmt.Println(safe.Await().OrElse(0)) // -1
```
`RecoverWith` is the version for when the fallback is itself asynchronous — a
retry against a second host, a slower backup service. It hands you the error
and expects a new promise in return, so the fallback runs concurrently just as
the original did:
```go
withBackup := fetchFrom(primary).
	RecoverWith(func(err error) monad.Promise[Report] {
		log.Println("primary failed, trying backup:", err)
		return fetchFrom(backup)
	})
```
A successful promise is returned untouched and the fallback is never built.

#### Things to know
- `String()` always returns `"Promise[pending]"`; it does not wait for or
  reveal the outcome.
- Each `Map`, `FlatMap`, `Then`, `Catch`, `Recover`, and `RecoverWith` returns
  a *new* promise and spawns one goroutine that blocks on the previous one.
  This is cheap in Go, but a promise whose computation never returns will leak
  that goroutine.
- `Await` caches the outcome, so awaiting the same promise repeatedly — or from
  several goroutines at once — returns the same `Result` without re-running the
  computation.
- A panic inside the computation — or inside any function you pass to `Map`,
  `FlatMap`, `Then`, `Catch`, or `Recover` — is recovered and delivered as a
  `Failure` wrapping `ErrPanic`. It has to be: that code runs on another
  goroutine, where a `recover` around your `Await` could not reach it and the
  panic would end the process.
- There is no cancellation or timeout built in. Your computation is a plain
  function, so hand it a `context.Context` from the enclosing scope and return
  a `Failure` when the context is done.
- `MapPromise` and `FlatMapPromise` are the standalone equivalents of the two
  methods.

### Lazy
`Lazy[T]` represents a **deferred computation evaluated at most once**. Nothing
runs until you ask for the value; when you do, the result is computed, cached,
and returned as-is on every subsequent request. It is a memoized thunk, and it
is safe to share between goroutines.

#### Creating a Lazy
`Defer` wraps a function and returns a `*Lazy[T]` (a pointer, because the
value carries its cache):
```go
config := monad.Defer(func() Config {
	fmt.Println("loading configuration...") // printed at most once
	return loadConfig()
})
```

#### Evaluating
`Evaluate` triggers the computation the first time and returns the cached value
afterwards. Concurrent callers block until the first evaluation completes and
then all see the same value:
```go
calls := 0
answer := monad.Defer(func() int {
	calls++
	return 42
})

answer.Evaluate() // 42 — computed now
answer.Evaluate() // 42 — cached
fmt.Println(calls) // 1
```

#### Transforming with `Map` and `FlatMap`
`Map` and `FlatMap` build a new `Lazy` *without evaluating anything*. The
derived value is computed the first time *it* is evaluated, which in turn
evaluates its source:
```go
base := monad.Defer(func() int {
	fmt.Println("computing base")
	return 10
})
doubled := base.Map(func(n int) int {
	fmt.Println("doubling")
	return n * 2
})
// nothing printed yet

fmt.Println(doubled.Evaluate()) // prints "computing base", "doubling", then 20
fmt.Println(doubled.Evaluate()) // prints 20 only
```
`FlatMap` is for a step that returns another `Lazy` — for instance choosing a
second deferred computation based on the first value:
```go
settings := base.FlatMap(func(n int) *monad.Lazy[string] {
	if n > 5 {
		return monad.Defer(func() string { return "large" })
	}
	return monad.Defer(func() string { return "small" })
})
```
Every `Lazy` in a chain caches its own result, so a shared prefix (`base`
above) is computed once no matter how many derived values use it.

#### Things to know
- `String()` never triggers evaluation. Before the first `Evaluate` it returns
  `"Lazy(pending)"`; afterwards, `"Lazy(<value>)"`. It is safe to call
  concurrently with `Evaluate`.
- `Evaluate` is safe for concurrent use: the first caller runs the computation
  while the others block, and every caller observes the same value.
- If the computation panics, the panic is remembered and re-raised for every
  later call, and `String` reports `"Lazy(failed)"`. A `sync.Once` is spent even
  when the function it guarded panicked, so without this only the first caller
  would learn of the failure and the rest would receive the zero value of `T` as
  though the work had succeeded.
- `Lazy` has no notion of failure. If the computation can fail, defer a
  `Result`: `monad.Defer(func() monad.Result[T] { ... })`.
- `MapLazy` and `FlatMapLazy` are the standalone equivalents.

### IO
`IO[T]` represents a **side-effecting computation** — reading a file, calling
an API, writing to a database. Creating an `IO` does *not* run it: it is a
description of an effect that produces a `Result[T]`. You build a program by
composing descriptions, and run it, as many times as you like, with `Run`.

This separation is what makes `IO` useful:
- the composition is pure and easy to test — swap the leaves for `PureIO`
  values and check the pipeline without touching the real world;
- the caller decides *when* and *how many times* the effects happen.

#### Creating an IO
`NewIO` wraps a function that performs the effect and reports its outcome as a
`Result`:
```go
readConfig := monad.NewIO(func() monad.Result[string] {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return monad.Failure[string](err)
	}
	return monad.Success(string(data))
})
```
`PureIO` and `FailIO` create effect-free `IO` values that succeed or fail
immediately — the equivalent of `Resolve` and `Reject`:
```go
fallback := monad.PureIO("{}")
missing := monad.FailIO[string](errors.New("no config"))
```

#### Running an IO
`Run` executes the effect and returns its `Result`. Each call **re-executes**
the effect; nothing is cached:
```go
counter := 0
tick := monad.NewIO(func() monad.Result[int] {
	counter++
	return monad.Success(counter)
})

tick.Run().OrElse(0) // 1
tick.Run().OrElse(0) // 2
```
This is the essential difference with `Lazy` (runs once, then cached) and
`Promise` (runs immediately, once).

#### Composing with `Map` and `FlatMap`
`Map` transforms the outcome of an effect; `FlatMap` sequences a second effect
after the first. Neither runs anything until `Run` is called on the final
value, and a `Failure` anywhere short-circuits the rest:
```go
parseConfig := func(raw string) Config { /* ... */ }

connect := func(cfg Config) monad.IO[*sql.DB] {
	return monad.NewIO(func() monad.Result[*sql.DB] {
		db, err := sql.Open(cfg.Driver, cfg.DSN)
		if err != nil {
			return monad.Failure[*sql.DB](err)
		}
		return monad.Success(db)
	})
}

program := readConfig.
	Map(parseConfig).
	FlatMap(connect)

// nothing has happened yet; this is where the effects run:
db := program.Run()
```

#### Recovering with `Recover`
`Recover` wraps an `IO` so that, when it is run and fails, the recovery
function supplies a value instead:
```go
raw := readConfig.Recover(func(err error) string {
	log.Println("using default configuration:", err)
	return "{}"
})
```
The recovery is part of the description: it is evaluated on every `Run`, not
just once.

`RecoverWith` takes a function returning another `IO`, which lets the fallback
perform side effects of its own — the point of reaching for it over `Recover`:
```go
config := readFile("/etc/app.conf").
	RecoverWith(func(error) monad.IO[[]byte] {
		return readFile("./app.conf")
	})
```
Because both sides are descriptions, nothing touches the filesystem until
`Run`, and the fallback is only built if the first read actually fails.

#### Things to know
- `String()` always returns `"IO[suspended]"`.
- `MapIO` and `FlatMapIO` are the standalone equivalents.
- Combine `IO` with `Lazy` to get a memoized effect
  (`monad.Defer(func() monad.Result[T] { return io.Run() })`), or with
  `Promise` to run it in the background
  (`monad.NewPromise(io.Run)`).

### Which type should I use?

| You need to represent…                                   | Use            | Get the value with        |
|-----------------------------------------------------------|----------------|---------------------------|
| A value that may be absent                                | `Option[T]`    | `OrElse`, `Fold`          |
| A computation that may fail with an `error`               | `Result[T]`    | `OrElse`, `Fold`          |
| One of two arbitrary types, or a typed failure            | `Either[L, R]` | `Fold`, `RightValue`      |
| Work already running in the background                    | `Promise[T]`   | `Await`                   |
| An expensive value computed once, on first use            | `Lazy[T]`      | `Evaluate`                |
| A side effect you want to describe now and run later      | `IO[T]`        | `Run`                     |

A quick comparison of the three "computation" types:

|                          | `Promise`                | `Lazy`                    | `IO`                       |
|--------------------------|--------------------------|---------------------------|----------------------------|
| When does it run?        | Immediately, on creation | On first `Evaluate`       | On every `Run`             |
| How many times?          | Once                     | Once (cached)             | Every call                 |
| Where?                   | A new goroutine          | The calling goroutine     | The calling goroutine      |
| Can it fail?             | Yes (`Result`)           | No (wrap a `Result`)      | Yes (`Result`)             |

## Usage
Now that we have seen the six types and the operations they share, this section
covers the patterns that come up when using them in real code.

#### Bridging with idiomatic Go
Most of the Go ecosystem speaks `(T, error)`. The library does not ship a
converter, but the one you will want is three lines long — define it once in
your codebase:
```go
func FromError[T any](value T, err error) monad.Result[T] {
	if err != nil {
		return monad.Failure[T](err)
	}
	return monad.Success(value)
}

port := FromError(strconv.Atoi("8080")) // Result[int]
```
Going the other way, turn a `Result` back into a `(T, error)` pair at the
boundary of your monadic code with `OrElse` and `Error`. `Error` returns `nil`
on a `Success` and never returns `nil` on a `Failure`, so the two values always
line up with Go's convention — a failure cannot be mistaken for a success by a
caller that only checks the error:
```go
func (repo Repository) Find(id int) (User, error) {
	r := findUser(id)
	return r.OrElse(User{}), r.Error()
}
```
(`Fold` cannot be used here: a single type parameter `B` cannot stand for the
two return values of `(User, error)`.) For an `Option`, `OrElseError` does the same
job in one call: `opt.OrElseError(ErrNotFound)`.

#### Converting between types
- `Result` → `Option`: `r.ToOption()`.
- `Option` → `Result`: give the missing value a reason with `ToResult`:
  `opt.ToResult(errors.New("not found"))`.
- `Either` → `Result`: `Fold` with `Failure` on the left and `Success` on the
  right; `Result` → `Either`: `Fold` with `Left` and `Right`.
- `Promise` → `Result`: `p.Await()`; `IO` → `Result`: `io.Run()`;
  `Result` → `Promise`: `Resolve`/`Reject`; `Result` → `IO`: `PureIO`/`FailIO`.

#### Reading configuration from environment variables
Environment variables are the textbook case for an optional value: a setting is
either present or it is not, and when it is present it still has to be parsed.
The library ships constructors that do both in one call. For every supported
type there are four functions, named after the type:

- `Get<Type>FromEnv(name)` returns `(Option[Type], error)`. An unset variable
  gives `None` and no error — that is the normal "not configured" case. A
  variable that is set but cannot be parsed gives `None` *and* an error.
- `MustGet<Type>FromEnv(name)` is the same but panics instead of returning the
  error, for settings you would rather fail fast on at startup.
- `Get<Type>SliceFromEnv(name)` and `MustGet<Type>SliceFromEnv(name)` read a
  comma-separated list into `Option[[]Type]`.

```go
port := monad.MustGetIntFromEnv("PORT").OrElse(8080)
timeout := monad.MustGetDurationFromEnv("HTTP_TIMEOUT").OrElse(30 * time.Second)

origins, err := monad.GetStringSliceFromEnv("CORS_ORIGINS")
if err != nil {
	return fmt.Errorf("CORS_ORIGINS: %w", err)
}
origins.IfPresent(enableCORS)
```
The supported types are `Bool`, `Duration`, `Time`, `Float32`, `Float64`,
`Int`, `Int8`, `Int16`, `Int32`, `Int64`, `Uint`, `Uint8`, `Uint16`, `Uint32`,
`Uint64`, `String`, and `URL` (`net/url`, stored by value). A few details are
worth knowing:

- Integers are parsed in base 0, so `PORT=0x1F`, `0o17`, and `0b101` are
  accepted alongside decimal, and values that do not fit the width (`200` for
  an `int8`) are errors.
- Booleans accept what `strconv.ParseBool` accepts: `1`, `t`, `true`, `True`,
  `TRUE` and their negatives. `yes`/`no` are errors.
- The `Time` functions take the layout first, as `time.Parse` does:
  `monad.GetTimeFromEnv(time.RFC3339, "STARTED_AT")`. A layout containing a
  comma (`time.RFC1123`) needs the double-quoting described below when read as
  a slice.
- URLs must be absolute: a scheme is required, so `example.com` and `42` are
  parse errors rather than being accepted as relative references.
- `GetStringFromEnv` returns an `Option[string]` and no error, since a string
  needs no parsing, and there is consequently no `MustGetStringFromEnv`. An
  empty variable (`NAME=`) yields `Some("")`, not `None`.
- Slices are read as CSV. Items are trimmed, and a variable that is empty or
  contains only whitespace yields a present, empty slice.
- A single matching pair of quotes wrapping the *whole* value is treated as
  shell noise and removed, so `HOSTS="a.example.com, b.example.com"` gives
  `Some([a.example.com b.example.com])`. The pair is only removed when the text
  between the quotes does not contain that character again, so `"a","b"` is
  still read as two items rather than one wrapper.
- Each item may carry its own quotes (`'1',"2",` `` `3` ``), and CSV quoting
  protects an embedded comma: `"1,2",3` is the two items `1,2` and `3`. Quotes
  inside a word are left alone, so `O'Brien` survives intact. To keep a comma
  inside a value that is a single item, quote it twice: `'"Mon, 02 Jan 2006"'`.
- A newline separates items just as a comma does, so a multi-line variable is
  read in full rather than truncated at its first line.

When the type you need is not in the list, use the generic `OptionFromEnv` and
`OptionSliceFromEnv` with your own parser — the typed functions are thin
wrappers over them:
```go
parseIP := func(s string) (net.IP, error) {
	if ip := net.ParseIP(s); ip != nil {
		return ip, nil
	}
	return nil, fmt.Errorf("invalid IP address %q", s)
}

bind, err := monad.OptionFromEnv("BIND_ADDR", parseIP)   // Option[net.IP]
peers, err := monad.OptionSliceFromEnv("PEERS", parseIP) // Option[[]net.IP]
```
Any `func(string) (T, error)` works as a parser, so existing functions such as
`strconv.Atoi`, `time.ParseDuration`, or `uuid.Parse` can be passed directly.

#### Validation pipelines
The most common use of `Result` is a sequence of checks where any one may
reject the input. Write each check as a `func(T) Result[T]` and chain them
with `FlatMap`, or use `Filter` for checks that are a plain predicate:
```go
validateEmail := func(email string) monad.Result[string] {
	if !strings.Contains(email, "@") {
		return monad.Failure[string](fmt.Errorf("invalid email: %s", email))
	}
	return monad.Success(email)
}

register := func(email string, age int) monad.Result[User] {
	return validateEmail(email).
		FlatMap(func(email string) monad.Result[User] {
			return monad.Success(age).
				Filter(func(a int) bool { return a >= 18 }, errors.New("must be an adult")).
				Map(func(a int) User { return User{Email: email, Age: a} })
		})
}

register("ada@example.com", 36) // Success({ada@example.com 36})
register("ada", 36)             // Failure(invalid email: ada)
register("ada@example.com", 12) // Failure(must be an adult)
```
Nesting a `FlatMap` inside another is how a later step gets access to the
values of *all* the earlier steps (here `email` and `a`). When the checks are
independent, `ApplyResult` (see the [`Result`](#result) section) keeps the code
flat.

#### Fan-out with Promise
Start every independent request with `NewPromise`, then combine them. The total
wait is the slowest request, not the sum:
```go
type Dashboard struct {
	Users  int
	Orders int
}

dashboard := func() monad.Promise[Dashboard] {
	users := monad.NewPromise(countUsers)   // starts now
	orders := monad.NewPromise(countOrders) // starts now

	return users.FlatMap(func(u int) monad.Promise[Dashboard] {
		return orders.Map(func(o int) Dashboard { return Dashboard{Users: u, Orders: o} })
	})
}

dashboard().
	Catch(func(err error) { log.Println("dashboard failed:", err) }).
	Await().
	IfSuccess(render)
```

#### Describing a program with IO
Build the whole program as an `IO`, keep every `Run` at the edge (`main`, an
HTTP handler, a test), and inject fakes by replacing the leaves:
```go
func Program(read monad.IO[string], write func(string) monad.IO[int]) monad.IO[int] {
	return read.
		Map(strings.ToUpper).
		FlatMap(write)
}

// production
n := Program(readFile("in.txt"), writeFile("out.txt")).Run()

// test
written := ""
fakeWrite := func(s string) monad.IO[int] {
	written = s
	return monad.PureIO(len(s))
}
Program(monad.PureIO("hello"), fakeWrite).Run() // written == "HELLO"
```

#### Memoizing with Lazy
Declare package-level values that are expensive to build but not always needed,
and let the first user pay the cost:
```go
var templates = monad.Defer(func() *template.Template {
	return template.Must(template.ParseGlob("templates/*.html"))
})

func handler(w http.ResponseWriter, r *http.Request) {
	templates.Evaluate().ExecuteTemplate(w, "index.html", nil)
}
```

## Coming from go-optional
`go-monad` is a superset of
[`go-optional`](https://github.com/grandper/go-optional): everything
`Optional[T]` offered is available on `Option[T]`, with the same JSON/YAML
behavior and the same environment-variable constructors. Migrating is a matter
of renaming:

| go-optional                                   | go-monad                                       |
|-----------------------------------------------|------------------------------------------------|
| `optional.Optional[T]`                        | `monad.Option[T]`                              |
| `optional.New(v)` / `optional.Empty[T]()`     | `monad.Some(v)` / `monad.None[T]()`            |
| `o.HasValue()`                                | `o.IsPresent()`                                |
| `o.ValueOr(x)`                                | `o.OrElse(x)`                                  |
| `o.ValueOrError(err)`                         | `o.OrElseError(err)`                               |
| `o.ValueOrPanic(msg)`                         | `o.OrElsePanic(msg)`                           |
| `optional.GetUIntFromEnv`, `GetUInt8FromEnv`… | `monad.GetUintFromEnv`, `GetUint8FromEnv`…     |
| every other `Get…FromEnv` / `MustGet…FromEnv` | same name, same behavior                       |

On top of that you gain `ToResult`, `IsZero` (for `omitzero`/`omitempty`), the
generic `OptionFromEnv`/`OptionSliceFromEnv`, and of course `Map`, `FlatMap`,
`Filter`, and `Fold`.

## Design Decisions
This section describes the choices behind the library's shape, for readers who
want to understand *why* the API looks the way it does.

### Value semantics vs pointer semantics
- **`Option`, `Result`, and `Either`** are small immutable structs passed by
  value. Copying one is cheap and cannot lead to shared mutable state.
- **`Lazy`** is returned as a pointer (`*Lazy[T]`) because it carries a
  `sync.Once` and a cached value: copying it would break the "at most once"
  guarantee.
- **`Promise`** is a value that captures shared state (the result channel and a
  `sync.Once`) in a closure, so it behaves like a reference while presenting a
  value API. Copies of a `Promise` share the same outcome.
- **`IO`** is a value: it is a pure description of an effect and holds no state.

### Standalone functions next to methods
Go 1.27 allows methods to declare their own type parameters, which is what
makes `opt.Map(f)` return an `Option[B]` for a `func(T) B`. Every such method
also exists as a standalone generic function (`MapOption`, `FlatMapResult`,
`FoldEither`, ...). They are useful when you need to pass the operation as a
value, and they keep the library approachable for readers coming from earlier
Go versions.

### Early returns
All operations are written with early returns rather than `if`/`else` chains,
so the happy path always reads last and unindented:
```go
func (o Option[T]) OrElse(fallback T) T {
	if !o.present {
		return fallback
	}
	return o.value
}
```

### No shared `Monad` interface
You may wonder why there is no `Monad[T]` interface that all six types
implement. Expressing it needs *higher-kinded types* — the ability to abstract
over the container itself (`Monad[F[_], T]`), not just over the element type —
which Go does not have. Each type therefore implements the same pattern
independently; the shared vocabulary is a convention that the tests enforce. If
Go gains higher-kinded types, a common interface can be added without breaking
the existing API.

## Examples
The [`examples`](examples/README.md) directory contains one runnable program per
monadic type, plus two covering the features built on `Option`:

- [`option`](examples/option/main.go) — absence as a value: `Map`, `FlatMap`,
  `Filter`, `Fold`, the `OrElse` family, `ApplyOption`, and why `Some(0)` is
  not `None`.
- [`result`](examples/result/main.go) — failure as a value: short-circuiting
  pipelines, `Recover`/`RecoverWith`, `ApplyResult`, and the way back to
  `(T, error)`.
- [`either`](examples/either/main.go) — a structured left side, right-bias,
  `MapLeft`, `Swap`, and an `Either` where neither side is a failure.
- [`lazy`](examples/lazy/main.go) — deferral and memoization, a shared prefix
  computed once, and concurrent access.
- [`promise`](examples/promise/main.go) — work already running, `Then`/`Catch`,
  parallel fan-out, and supplying your own cancellation.
- [`io`](examples/io/main.go) — describing an effect, re-running it, fallback
  effects with `RecoverWith`, and a comparison with `Lazy`.
- [`environment`](examples/environment/main.go) — configuration from environment
  variables, including custom parsers.
- [`serialization`](examples/serialization/main.go) — JSON and YAML, keeping
  absent, null, and zero distinguishable.

Run one with:
```bash
go run ./examples/option
```

## Testing
```bash
go test ./...
```
The test suite exercises both the method and the standalone API of every type
and includes runnable `Example` functions whose output is checked. Run it with
`-race` when working on `Promise` or `Lazy`:
```bash
go test -race ./...
```

## Common Issues

| Issue                                   | Cause                                                       | Fix                                                                            |
|-----------------------------------------|-------------------------------------------------------------|--------------------------------------------------------------------------------|
| `cannot infer T`                        | Constructors with no value to infer from                    | Add the type argument: `None[int]()`, `Failure[int](err)`, `Reject[int](err)`, `FailIO[int](err)`, `Left[string, int](v)` |
| `cannot use generic method`             | Go older than 1.27                                          | Upgrade to Go 1.27 or use the standalone functions                             |
| `Option[Option[int]]` where `Option[int]` was expected | `Map` used with a function that returns an `Option` | Use `FlatMap`                                                       |
| A zero `Either` takes the left branch   | The zero value is `Left(zero L)` by design                  | Construct with `Left`/`Right` when you need a specific side                     |
| A `Promise` side effect seems not to run | It runs on its own goroutine, which the program may exit before reaching | `Await` the promise `Then`/`Catch` returned, to wait for it |
| `Lazy.String()` returns `Lazy(pending)` | `String` never triggers the computation                     | Call `Evaluate` first if you want the value                                    |
| `IO.Run`/`Promise.Await` fails with `ErrUninitialized` | The value is a zero `IO`/`Promise` with no computation | Build it with `NewIO`/`PureIO` or `NewPromise`/`Resolve`      |
| `Evaluate called on an uninitialized Lazy` | A zero `Lazy`, or `Defer(nil)`                           | Build it with `Defer(f)` where `f` is non-nil                                  |
| A `Promise` fails with `ErrPanic`       | The computation, or a function passed to `Map`/`Then`/..., panicked | Fix the panic; it is reported rather than crashing the process        |
| A list item loses a comma               | The value was quoted only once, so the outer pair was stripped | Quote it twice: `'"a,b"'`                                            |
| `OrElsePanic` panics                        | The `Option` is `None`                                      | Use `OrElse`, `OrElseError`, or check `IsPresent` first                            |
| `MustGet…FromEnv` panics at startup     | The variable is set but cannot be parsed                    | Fix the value, or use the `Get…FromEnv` form and handle the error              |
| `omitempty` still writes `null` in JSON | `omitempty` never omits a struct                            | Use `omitzero` (JSON) — `omitempty` is the right option for YAML               |

## License
Licensed under MIT License.

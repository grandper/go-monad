# Examples

Runnable programs demonstrating `go-monad`. There is one example per monadic
type, plus two that cover the features built on top of `Option`.

Each program is self-contained and prints what it does, so the quickest way to
read one is to run it and follow along in the source.

## Running

```bash
go run ./examples/option
```

Or from inside the directory:

```bash
cd examples/option
go run main.go
```

To run all of them in sequence:

```bash
for d in ./examples/*/; do echo "--- $d"; go run "$d"; done
```

## One per type

Read them in this order if you are new to the library: `option` and `result`
introduce the vocabulary that every other type reuses.

### `option/` — a value that may be absent
Creating and inspecting an `Option`, `Map`, `FlatMap`, `Filter`, `Fold`, the
whole `OrElse` family, `IfPresent`/`IfEmpty`, combining independent Options with
`ApplyOption`, converting to a `Result`, and why the zero value is `None`.
The running theme is that `Some(0)` and `None` are different things.

### `result/` — a computation that may fail
`Success` and `Failure`, building a pipeline where the first error
short-circuits the rest, `Filter` with a reason, `Fold`, `Recover` and
`RecoverWith`, `IfSuccess`/`IfFailure`, `ApplyResult` for independent values,
and how to return to Go's `(T, error)` convention at an API boundary.

### `either/` — one of two types
Why `Either` exists when `Result` already handles failure: a left side that is
structured data rather than an `error`. Covers right-bias, `Fold`, `MapLeft`,
`Swap`, `ToOption`, an `Either` where neither side is a failure, and the zero
value.

### `lazy/` — computed once, on demand
Deferral and memoization: nothing runs until `Evaluate`, and then only once.
Shows a shared prefix being computed a single time, concurrent access from
several goroutines, why `String` refuses to force evaluation, and how to defer
something that can fail by deferring a `Result`.

### `promise/` — work already running
Starting work on creation, `Await` caching the outcome, composing async steps
with `Map`/`FlatMap`, `Then`/`Catch` for observation, `Recover`/`RecoverWith`,
running several calls in parallel, and supplying your own cancellation with a
`context.Context`.

### `io/` — an effect described now, run later
Building a description that performs no work, re-running it on every `Run`,
composing a small program, short-circuiting on failure, `Recover` and
`RecoverWith` for fallback effects, a side-by-side comparison with `Lazy`, and
what a zero `IO` does.

## Features built on `Option`

### `environment/` — configuration from environment variables
The three states of a setting — absent, valid, malformed — and why they are not
two. Defaults, `Get` versus `MustGet`, comma-separated lists, the parsing rules
worth knowing (base-0 integers, `time.Parse` layouts, empty strings), custom
parsers through `OptionFromEnv`, and assembling a config struct that still
remembers which settings were explicitly set.

### `serialization/` — JSON and YAML
Keeping *absent*, *null*, and *zero* distinguishable across a wire boundary in
both directions. Covers the `omitzero` (JSON) and `omitempty` (YAML) tags, what
each one does to an empty `Option`, and how decoding errors are reported.

## Related: documentation examples

These programs are tutorials meant to be read and run. Separately, the library
ships Go [example functions](https://go.dev/blog/examples) — `ExampleSome`,
`ExampleOption_Map`, and so on — which live beside the code they document in
`option_test.go`, `result_test.go`, `either_test.go`, `lazy_test.go`,
`promise_test.go`, `io_test.go`, and `env_test.go`. Those are compiled and
their output verified by `go test`, and they appear on pkg.go.dev attached to
the symbol each one documents.

## Note on Go 1.27

These examples use the method-chaining syntax, which requires Go 1.27+. On an
earlier version, use the standalone functions (`MapOption`, `FlatMapResult`,
`FoldEither`, ...) described in the main README.

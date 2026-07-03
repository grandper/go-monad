package monad_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	monad "github.com/grandper/go-monad"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	testOptionIntValue     = 42
	testOptionStringValue  = "hello"
	testOptionFallbackInt  = -1
	testOptionDoubledInt   = 84
	testOptionNoneString   = "None"
	testOptionSomeString   = "Some(42)"
	testOptionFoldNone     = "empty"
	testOptionFoldSome     = "value: 42"
	testOptionFoldFormat   = "value: %d"
	testOptionPanicMessage = "option is empty"

	testCodecName    = "alice"
	testCodecAge     = 3
	testCodecPrior   = 7
	testCodecJSONInt = "3"
	testCodecJSONStr = `"hi"`
	testCodecNull    = "null"
	testCodecYAMLInt = "3\n"
	testCodecYAMLNul = "null\n"
)

var errTestOption = errors.New("option is empty")

func TestOption(t *testing.T) {
	t.Run("Some", func(t *testing.T) {
		t.Run("is present", func(t *testing.T) {
			option := monad.Some(testOptionIntValue)

			require.True(t, option.IsPresent())
		})

		t.Run("is not empty", func(t *testing.T) {
			option := monad.Some(testOptionIntValue)

			require.False(t, option.IsEmpty())
		})

		t.Run("returns string representation", func(t *testing.T) {
			option := monad.Some(testOptionIntValue)

			require.Equal(t, testOptionSomeString, option.String())
		})
	})

	t.Run("None", func(t *testing.T) {
		t.Run("is not present", func(t *testing.T) {
			option := monad.None[int]()

			require.False(t, option.IsPresent())
		})

		t.Run("is empty", func(t *testing.T) {
			option := monad.None[int]()

			require.True(t, option.IsEmpty())
		})

		t.Run("returns string representation", func(t *testing.T) {
			option := monad.None[int]()

			require.Equal(t, testOptionNoneString, option.String())
		})
	})

	t.Run("Filter", func(t *testing.T) {
		t.Run("keeps value when predicate is satisfied", func(t *testing.T) {
			option := monad.Some(testOptionIntValue).Filter(func(v int) bool {
				return v > 0
			})

			require.True(t, option.IsPresent())
			require.Equal(t, testOptionIntValue, option.OrElse(0))
		})

		t.Run("returns None when predicate is not satisfied", func(t *testing.T) {
			option := monad.Some(testOptionIntValue).Filter(func(v int) bool {
				return v < 0
			})

			require.True(t, option.IsEmpty())
		})

		t.Run("returns None when Option is empty", func(t *testing.T) {
			option := monad.None[int]().Filter(func(v int) bool {
				return v > 0
			})

			require.True(t, option.IsEmpty())
		})
	})

	t.Run("OrElse", func(t *testing.T) {
		t.Run("returns value when present", func(t *testing.T) {
			result := monad.Some(testOptionIntValue).OrElse(testOptionFallbackInt)

			require.Equal(t, testOptionIntValue, result)
		})

		t.Run("returns fallback when empty", func(t *testing.T) {
			result := monad.None[int]().OrElse(testOptionFallbackInt)

			require.Equal(t, testOptionFallbackInt, result)
		})
	})

	t.Run("OrElseGet", func(t *testing.T) {
		t.Run("returns value when present", func(t *testing.T) {
			result := monad.Some(testOptionIntValue).OrElseGet(func() int {
				return testOptionFallbackInt
			})

			require.Equal(t, testOptionIntValue, result)
		})

		t.Run("returns supplier result when empty", func(t *testing.T) {
			result := monad.None[int]().OrElseGet(func() int {
				return testOptionFallbackInt
			})

			require.Equal(t, testOptionFallbackInt, result)
		})
	})

	t.Run("IfPresent", func(t *testing.T) {
		t.Run("calls consumer when present", func(t *testing.T) {
			var captured int
			monad.Some(testOptionIntValue).IfPresent(func(v int) {
				captured = v
			})

			require.Equal(t, testOptionIntValue, captured)
		})

		t.Run("does not call consumer when empty", func(t *testing.T) {
			called := false
			monad.None[int]().IfPresent(func(_ int) {
				called = true
			})

			require.False(t, called)
		})
	})

	t.Run("Map", func(t *testing.T) {
		t.Run("transforms value when present", func(t *testing.T) {
			option := monad.Some(testOptionIntValue).Map(func(v int) int {
				return v * 2
			})

			require.True(t, option.IsPresent())
			require.Equal(t, testOptionDoubledInt, option.OrElse(0))
		})

		t.Run("returns None when empty", func(t *testing.T) {
			option := monad.None[int]().Map(func(v int) int {
				return v * 2
			})

			require.True(t, option.IsEmpty())
		})
	})

	t.Run("FlatMap", func(t *testing.T) {
		t.Run("transforms value when present", func(t *testing.T) {
			option := monad.Some(testOptionIntValue).FlatMap(func(_ int) monad.Option[string] {
				return monad.Some(testOptionStringValue)
			})

			require.True(t, option.IsPresent())
			require.Equal(t, testOptionStringValue, option.OrElse(""))
		})

		t.Run("returns None when original is empty", func(t *testing.T) {
			option := monad.None[int]().FlatMap(func(_ int) monad.Option[string] {
				return monad.Some(testOptionStringValue)
			})

			require.True(t, option.IsEmpty())
		})

		t.Run("returns None when function returns None", func(t *testing.T) {
			option := monad.Some(testOptionIntValue).FlatMap(func(_ int) monad.Option[string] {
				return monad.None[string]()
			})

			require.True(t, option.IsEmpty())
		})
	})

	t.Run("Fold", func(t *testing.T) {
		t.Run("applies ifSome when present", func(t *testing.T) {
			result := monad.Some(testOptionIntValue).Fold(
				func() string { return testOptionFoldNone },
				func(v int) string { return fmt.Sprintf(testOptionFoldFormat, v) },
			)

			require.Equal(t, testOptionFoldSome, result)
		})

		t.Run("applies ifNone when empty", func(t *testing.T) {
			result := monad.None[int]().Fold(
				func() string { return testOptionFoldNone },
				func(v int) string { return fmt.Sprintf(testOptionFoldFormat, v) },
			)

			require.Equal(t, testOptionFoldNone, result)
		})
	})

	t.Run("MapOption", func(t *testing.T) {
		t.Run("transforms value when present", func(t *testing.T) {
			option := monad.MapOption(monad.Some(testOptionIntValue), func(v int) int {
				return v * 2
			})

			require.True(t, option.IsPresent())
			require.Equal(t, testOptionDoubledInt, option.OrElse(0))
		})

		t.Run("returns None when empty", func(t *testing.T) {
			option := monad.MapOption(monad.None[int](), func(v int) int {
				return v * 2
			})

			require.True(t, option.IsEmpty())
		})
	})

	t.Run("FlatMapOption", func(t *testing.T) {
		t.Run("transforms value when present", func(t *testing.T) {
			option := monad.FlatMapOption(monad.Some(testOptionIntValue), func(_ int) monad.Option[string] {
				return monad.Some(testOptionStringValue)
			})

			require.True(t, option.IsPresent())
			require.Equal(t, testOptionStringValue, option.OrElse(""))
		})

		t.Run("returns None when empty", func(t *testing.T) {
			option := monad.FlatMapOption(monad.None[int](), func(_ int) monad.Option[string] {
				return monad.Some(testOptionStringValue)
			})

			require.True(t, option.IsEmpty())
		})
	})

	t.Run("FoldOption", func(t *testing.T) {
		t.Run("applies ifSome when present", func(t *testing.T) {
			result := monad.FoldOption(
				monad.Some(testOptionIntValue),
				func() string { return testOptionFoldNone },
				func(v int) string { return fmt.Sprintf(testOptionFoldFormat, v) },
			)

			require.Equal(t, testOptionFoldSome, result)
		})

		t.Run("applies ifNone when empty", func(t *testing.T) {
			result := monad.FoldOption(
				monad.None[int](),
				func() string { return testOptionFoldNone },
				func(v int) string { return fmt.Sprintf(testOptionFoldFormat, v) },
			)

			require.Equal(t, testOptionFoldNone, result)
		})
	})

	t.Run("OrElseError", func(t *testing.T) {
		t.Run("returns value and nil error when present", func(t *testing.T) {
			value, err := monad.Some(testOptionIntValue).OrElseError(errTestOption)

			require.NoError(t, err)
			require.Equal(t, testOptionIntValue, value)
		})

		t.Run("returns zero value and the error when empty", func(t *testing.T) {
			value, err := monad.None[int]().OrElseError(errTestOption)

			require.ErrorIs(t, err, errTestOption)
			require.Equal(t, 0, value)
		})

		t.Run("returns the error exactly as given", func(t *testing.T) {
			_, err := monad.None[int]().OrElseError(nil)

			require.NoError(t, err)
		})
	})

	t.Run("OrElsePanic", func(t *testing.T) {
		t.Run("returns value when present", func(t *testing.T) {
			require.Equal(t, testOptionIntValue,
				monad.Some(testOptionIntValue).OrElsePanic(testOptionPanicMessage))
		})

		t.Run("panics with the provided message when empty", func(t *testing.T) {
			require.PanicsWithValue(t, testOptionPanicMessage, func() {
				monad.None[int]().OrElsePanic(testOptionPanicMessage)
			})
		})

		t.Run("does not panic for a present zero value", func(t *testing.T) {
			require.Equal(t, 0, monad.Some(0).OrElsePanic(testOptionPanicMessage))
		})
	})

	t.Run("ToResult", func(t *testing.T) {
		t.Run("returns Success when present", func(t *testing.T) {
			result := monad.Some(testOptionIntValue).ToResult(errTestOption)

			require.True(t, result.IsSuccess())
			require.Equal(t, testOptionIntValue, result.OrElse(0))
		})

		t.Run("returns Failure holding the error when empty", func(t *testing.T) {
			result := monad.None[int]().ToResult(errTestOption)

			require.True(t, result.IsFailure())
			require.ErrorIs(t, result.Error(), errTestOption)
		})
	})

	t.Run("IsZero", func(t *testing.T) {
		t.Run("is true when empty", func(t *testing.T) {
			require.True(t, monad.None[int]().IsZero())
		})

		t.Run("is false for a present zero value", func(t *testing.T) {
			require.False(t, monad.Some(0).IsZero())
			require.False(t, monad.Some("").IsZero())
		})

		t.Run("is false when present", func(t *testing.T) {
			require.False(t, monad.Some(testOptionStringValue).IsZero())
		})
	})

	t.Run("ApplyOption", func(t *testing.T) {
		t.Run("applies function to value when both present", func(t *testing.T) {
			double := func(v int) int { return v * 2 }
			option := monad.ApplyOption(monad.Some(double), monad.Some(testOptionIntValue))

			require.True(t, option.IsPresent())
			require.Equal(t, testOptionDoubledInt, option.OrElse(0))
		})

		t.Run("returns None when function is absent", func(t *testing.T) {
			option := monad.ApplyOption(monad.None[func(int) int](), monad.Some(testOptionIntValue))

			require.True(t, option.IsEmpty())
		})

		t.Run("returns None when value is absent", func(t *testing.T) {
			double := func(v int) int { return v * 2 }
			option := monad.ApplyOption(monad.Some(double), monad.None[int]())

			require.True(t, option.IsEmpty())
		})
	})
}

type codecStruct struct {
	Name  monad.Option[string] `json:"name"            yaml:"name"`
	Age   monad.Option[int]    `json:"age,omitzero"    yaml:"age,omitempty"`
	Email monad.Option[string] `json:"email,omitempty" yaml:"email"`
}

func TestOptionJSON(t *testing.T) {
	t.Run("Marshal", func(t *testing.T) {
		t.Run("present value marshals as the value", func(t *testing.T) {
			data, err := json.Marshal(monad.Some(testCodecAge))

			require.NoError(t, err)
			require.JSONEq(t, testCodecJSONInt, string(data))

			data, err = json.Marshal(monad.Some("hi"))

			require.NoError(t, err)
			require.JSONEq(t, testCodecJSONStr, string(data))
		})

		t.Run("empty marshals as null", func(t *testing.T) {
			data, err := json.Marshal(monad.None[int]())

			require.NoError(t, err)
			require.Equal(t, testCodecNull, string(data))
		})

		t.Run("nested options collapse", func(t *testing.T) {
			data, err := json.Marshal(monad.Some(monad.Some(testCodecAge)))

			require.NoError(t, err)
			require.JSONEq(t, testCodecJSONInt, string(data))

			data, err = json.Marshal(monad.Some(monad.None[int]()))

			require.NoError(t, err)
			require.Equal(t, testCodecNull, string(data))
		})

		t.Run("omitzero drops empty fields but omitempty does not", func(t *testing.T) {
			data, err := json.Marshal(codecStruct{Name: monad.Some(testCodecName)})

			require.NoError(t, err)
			require.JSONEq(t, `{"name":"alice","email":null}`, string(data))
		})

		t.Run("omitzero keeps a present zero value", func(t *testing.T) {
			data, err := json.Marshal(codecStruct{Name: monad.Some(testCodecName), Age: monad.Some(0)})

			require.NoError(t, err)
			require.JSONEq(t, `{"name":"alice","age":0,"email":null}`, string(data))
		})
	})

	t.Run("Unmarshal", func(t *testing.T) {
		t.Run("null yields None", func(t *testing.T) {
			var option monad.Option[int]

			require.NoError(t, json.Unmarshal([]byte(testCodecNull), &option))
			require.True(t, option.IsEmpty())
		})

		t.Run("null resets a populated Option", func(t *testing.T) {
			option := monad.Some(testCodecPrior)

			require.NoError(t, json.Unmarshal([]byte(testCodecNull), &option))
			require.True(t, option.IsEmpty())
		})

		t.Run("value yields Some", func(t *testing.T) {
			var option monad.Option[int]

			require.NoError(t, json.Unmarshal([]byte(testCodecJSONInt), &option))
			require.Equal(t, monad.Some(testCodecAge), option)
		})

		t.Run("nested options decode", func(t *testing.T) {
			var option monad.Option[monad.Option[int]]

			require.NoError(t, json.Unmarshal([]byte(testCodecJSONInt), &option))
			require.Equal(t, monad.Some(monad.Some(testCodecAge)), option)
		})

		t.Run("wrong type returns an error and leaves the target unchanged", func(t *testing.T) {
			option := monad.Some(testCodecPrior)

			require.Error(t, json.Unmarshal([]byte(testCodecJSONStr), &option))
			require.Equal(t, monad.Some(testCodecPrior), option)
		})

		t.Run("absent key yields None", func(t *testing.T) {
			var value codecStruct

			require.NoError(t, json.Unmarshal([]byte(`{}`), &value))
			require.True(t, value.Name.IsEmpty())
			require.True(t, value.Age.IsEmpty())
		})

		t.Run("null key yields None", func(t *testing.T) {
			var value codecStruct

			require.NoError(t, json.Unmarshal([]byte(`{"name": null}`), &value))
			require.True(t, value.Name.IsEmpty())
		})

		t.Run("struct fields round-trip", func(t *testing.T) {
			var value codecStruct

			require.NoError(t, json.Unmarshal([]byte(`{"name": "alice", "age": 3}`), &value))
			require.Equal(t, monad.Some(testCodecName), value.Name)
			require.Equal(t, monad.Some(testCodecAge), value.Age)
			require.True(t, value.Email.IsEmpty())
		})
	})
}

func TestOptionYAML(t *testing.T) {
	t.Run("Marshal", func(t *testing.T) {
		t.Run("present value marshals as the value", func(t *testing.T) {
			data, err := yaml.Marshal(monad.Some(testCodecAge))

			require.NoError(t, err)
			require.YAMLEq(t, testCodecYAMLInt, string(data))
		})

		t.Run("empty marshals as null", func(t *testing.T) {
			data, err := yaml.Marshal(monad.None[int]())

			require.NoError(t, err)
			require.YAMLEq(t, testCodecYAMLNul, string(data))
		})

		t.Run("omitempty drops empty fields", func(t *testing.T) {
			data, err := yaml.Marshal(codecStruct{Name: monad.Some(testCodecName)})

			require.NoError(t, err)
			require.Equal(t, "name: alice\nemail: null\n", string(data))
		})

		t.Run("omitempty keeps a present zero value", func(t *testing.T) {
			data, err := yaml.Marshal(codecStruct{Name: monad.Some(testCodecName), Age: monad.Some(0)})

			require.NoError(t, err)
			require.Equal(t, "name: alice\nage: 0\nemail: null\n", string(data))
		})
	})

	t.Run("Unmarshal", func(t *testing.T) {
		t.Run("value yields Some", func(t *testing.T) {
			var option monad.Option[string]

			require.NoError(t, yaml.Unmarshal([]byte("hello\n"), &option))
			require.Equal(t, monad.Some("hello"), option)
		})

		t.Run("null yields None on a fresh target", func(t *testing.T) {
			var option monad.Option[int]

			require.NoError(t, yaml.Unmarshal([]byte(testCodecYAMLNul), &option))
			require.True(t, option.IsEmpty())
		})

		t.Run("null leaves a populated Option unchanged (yaml.v3 limitation)", func(t *testing.T) {
			// gopkg.in/yaml.v3 resolves null nodes itself and does not zero
			// struct kinds, so custom unmarshalers never see them. This test
			// documents that behavior so a change in the library is noticed.
			option := monad.Some(testCodecPrior)

			require.NoError(t, yaml.Unmarshal([]byte(testCodecYAMLNul), &option))
			require.Equal(t, monad.Some(testCodecPrior), option)
		})

		t.Run("wrong type returns an error", func(t *testing.T) {
			var option monad.Option[int]

			require.Error(t, yaml.Unmarshal([]byte("hello\n"), &option))
		})

		t.Run("empty document yields None", func(t *testing.T) {
			var value codecStruct

			require.NoError(t, yaml.Unmarshal([]byte(""), &value))
			require.True(t, value.Name.IsEmpty())
		})

		t.Run("absent key yields None", func(t *testing.T) {
			var value codecStruct

			require.NoError(t, yaml.Unmarshal([]byte("foo: bar\n"), &value))
			require.True(t, value.Name.IsEmpty())
		})

		t.Run("key without value yields None", func(t *testing.T) {
			var value codecStruct

			require.NoError(t, yaml.Unmarshal([]byte("age:\n"), &value))
			require.True(t, value.Age.IsEmpty())
		})

		t.Run("struct fields round-trip", func(t *testing.T) {
			var value codecStruct

			require.NoError(t, yaml.Unmarshal([]byte("name: alice\nage: 3\n"), &value))
			require.Equal(t, monad.Some(testCodecName), value.Name)
			require.Equal(t, monad.Some(testCodecAge), value.Age)
			require.True(t, value.Email.IsEmpty())
		})
	})
}

func ExampleSome() {
	opt := monad.Some(42)
	fmt.Println(opt.IsPresent())
	fmt.Println(opt.OrElse(0))
	// Output:
	// true
	// 42
}

func ExampleNone() {
	opt := monad.None[int]()
	fmt.Println(opt.IsPresent())
	fmt.Println(opt.OrElse(0))
	// Output:
	// false
	// 0
}

func ExampleOption_Map() {
	opt := monad.Some(21).Map(func(n int) int { return n * 2 })
	fmt.Println(opt.OrElse(0))
	// Output:
	// 42
}

func ExampleOption_FlatMap() {
	parse := func(s string) monad.Option[int] {
		n, err := strconv.Atoi(s)
		if err != nil {
			return monad.None[int]()
		}
		return monad.Some(n)
	}

	fmt.Println(monad.Some("42").FlatMap(parse).OrElse(0))
	fmt.Println(monad.Some("oops").FlatMap(parse).OrElse(-1))
	// Output:
	// 42
	// -1
}

func ExampleOption_Fold() {
	present := monad.Some("hello")
	absent := monad.None[string]()

	describe := func(o monad.Option[string]) string {
		return o.Fold(
			func() string { return "nothing" },
			func(s string) string { return "got: " + s },
		)
	}

	fmt.Println(describe(present))
	fmt.Println(describe(absent))
	// Output:
	// got: hello
	// nothing
}

func ExampleOption_Filter() {
	even := monad.Some(4).Filter(func(n int) bool { return n%2 == 0 })
	odd := monad.Some(3).Filter(func(n int) bool { return n%2 == 0 })

	fmt.Println(even.OrElse(-1))
	fmt.Println(odd.OrElse(-1))
	// Output:
	// 4
	// -1
}

func ExampleMapOption() {
	result := monad.MapOption(monad.Some(5), func(n int) string {
		return fmt.Sprintf("value=%d", n)
	})
	fmt.Println(result.OrElse("none"))
	// Output:
	// value=5
}

func ExampleOption_OrElseError() {
	value, err := monad.None[int]().OrElseError(errors.New("no value"))
	fmt.Println(value, err)
	value, err = monad.Some(42).OrElseError(errors.New("no value"))
	fmt.Println(value, err)
	// Output:
	// 0 no value
	// 42 <nil>
}

func ExampleOption_OrElsePanic() {
	fmt.Println(monad.Some("hello").OrElsePanic("value must be present"))
	// Output:
	// hello
}

func ExampleOption_ToResult() {
	res := monad.None[int]().ToResult(errors.New("not found"))
	fmt.Println(res)
	// Output:
	// Failure(not found)
}

func ExampleOption_MarshalJSON() {
	type user struct {
		Name  monad.Option[string] `json:"name"`
		Email monad.Option[string] `json:"email,omitzero"`
	}
	data, _ := json.Marshal(user{Name: monad.Some("Alice")})
	fmt.Println(string(data))

	var decoded user
	_ = json.Unmarshal([]byte(`{"name": null, "email": "alice@example.com"}`), &decoded)
	fmt.Println(decoded.Name, decoded.Email)
	// Output:
	// {"name":"Alice"}
	// None Some(alice@example.com)
}

func TestOptionIfEmpty(t *testing.T) {
	t.Run("runs the function when empty", func(t *testing.T) {
		called := false
		monad.None[int]().IfEmpty(func() { called = true })

		require.True(t, called)
	})

	t.Run("does nothing when a value is present", func(t *testing.T) {
		called := false
		monad.Some(testOptionIntValue).IfEmpty(func() { called = true })

		require.False(t, called)
	})
}

// TestOptionMonadLaws verifies the three monad laws for Option. They are what
// guarantee that restructuring a chain — inlining a step, extracting one,
// regrouping two FlatMaps — cannot change its meaning.
func TestOptionMonadLaws(t *testing.T) {
	f := func(n int) monad.Option[int] { return monad.Some(n * 2) }
	g := func(n int) monad.Option[int] { return monad.Some(n + 1) }

	t.Run("left identity", func(t *testing.T) {
		require.Equal(t, f(testOptionIntValue), monad.Some(testOptionIntValue).FlatMap(f))
	})

	t.Run("right identity", func(t *testing.T) {
		m := monad.Some(testOptionIntValue)
		require.Equal(t, m, m.FlatMap(monad.Some))

		empty := monad.None[int]()
		require.Equal(t, empty, empty.FlatMap(monad.Some))
	})

	t.Run("associativity", func(t *testing.T) {
		for _, m := range []monad.Option[int]{monad.Some(testOptionIntValue), monad.None[int]()} {
			require.Equal(t,
				m.FlatMap(f).FlatMap(g),
				m.FlatMap(func(n int) monad.Option[int] { return f(n).FlatMap(g) }),
			)
		}
	})
}

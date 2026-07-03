package monad_test

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	monad "github.com/grandper/go-monad"

	"github.com/stretchr/testify/require"
)

const (
	testEnvName    = "GO_MONAD_TEST_VARIABLE"
	testEnvUnset   = "GO_MONAD_TEST_UNSET_VARIABLE"
	testEnvInvalid = ":-("
)

var errTestEnvParse = errors.New("cannot parse")

// ---------------------------------------------------------------------------
// Generic helpers
// ---------------------------------------------------------------------------

func assertUnsetFromEnv[T any](t *testing.T, get func(string) (monad.Option[T], error)) {
	t.Helper()

	t.Run("yields None when the variable is unset", func(t *testing.T) {
		option, err := get(testEnvUnset)

		require.NoError(t, err)
		require.True(t, option.IsEmpty())
	})
}

func assertFromEnv[T any](t *testing.T, raw string, want T, get func(string) (monad.Option[T], error)) {
	t.Helper()

	t.Run(fmt.Sprintf("parses %q", raw), func(t *testing.T) {
		t.Setenv(testEnvName, raw)

		option, err := get(testEnvName)

		require.NoError(t, err)
		require.Equal(t, monad.Some(want), option)
	})
}

func assertMustFromEnv[T any](t *testing.T, raw string, want T, get func(string) monad.Option[T]) {
	t.Helper()

	t.Run(fmt.Sprintf("Must parses %q", raw), func(t *testing.T) {
		t.Setenv(testEnvName, raw)

		require.Equal(t, monad.Some(want), get(testEnvName))
	})

	t.Run("Must yields None when the variable is unset", func(t *testing.T) {
		require.True(t, get(testEnvUnset).IsEmpty())
	})
}

func assertInvalidFromEnv[T any](t *testing.T, raw string, get func(string) (monad.Option[T], error)) {
	t.Helper()

	t.Run(fmt.Sprintf("fails to parse %q", raw), func(t *testing.T) {
		t.Setenv(testEnvName, raw)

		option, err := get(testEnvName)

		require.Error(t, err)
		require.True(t, option.IsEmpty())
	})
}

func assertMustPanics[T any](t *testing.T, raw string, get func(string) monad.Option[T]) {
	t.Helper()

	t.Run(fmt.Sprintf("Must panics on %q", raw), func(t *testing.T) {
		t.Setenv(testEnvName, raw)

		require.Panics(t, func() {
			get(testEnvName)
		})
	})
}

// assertScalar runs the standard scenario for a scalar constructor pair.
func assertScalar[T any](
	t *testing.T,
	raw string,
	want T,
	get func(string) (monad.Option[T], error),
	must func(string) monad.Option[T],
) {
	t.Helper()

	assertUnsetFromEnv(t, get)
	assertFromEnv(t, raw, want, get)
	assertMustFromEnv(t, raw, want, must)
	assertInvalidFromEnv(t, testEnvInvalid, get)
	assertMustPanics(t, testEnvInvalid, must)
}

// ---------------------------------------------------------------------------
// Generic core
// ---------------------------------------------------------------------------

func parseIP(s string) (net.IP, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, errTestEnvParse
	}
	return ip, nil
}

func TestOptionFromEnv(t *testing.T) {
	get := func(name string) (monad.Option[net.IP], error) { return monad.OptionFromEnv(name, parseIP) }

	assertUnsetFromEnv(t, get)
	assertFromEnv(t, "127.0.0.1", net.ParseIP("127.0.0.1"), get)

	t.Run("returns the parse error unwrapped", func(t *testing.T) {
		t.Setenv(testEnvName, testEnvInvalid)

		option, err := get(testEnvName)

		require.ErrorIs(t, err, errTestEnvParse)
		require.True(t, option.IsEmpty())
	})
}

func TestOptionSliceFromEnv(t *testing.T) {
	get := func(name string) (monad.Option[[]net.IP], error) { return monad.OptionSliceFromEnv(name, parseIP) }

	assertUnsetFromEnv(t, get)
	assertFromEnv(t, "127.0.0.1, ::1", []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}, get)

	t.Run("returns the parse error unwrapped", func(t *testing.T) {
		t.Setenv(testEnvName, "127.0.0.1,"+testEnvInvalid)

		option, err := get(testEnvName)

		require.ErrorIs(t, err, errTestEnvParse)
		require.True(t, option.IsEmpty())
	})
}

// ---------------------------------------------------------------------------
// Typed constructors
// ---------------------------------------------------------------------------

func TestGetBoolFromEnv(t *testing.T) {
	assertScalar(t, "true", true, monad.GetBoolFromEnv, monad.MustGetBoolFromEnv)
	assertFromEnv(t, "1", true, monad.GetBoolFromEnv)
	assertFromEnv(t, "False", false, monad.GetBoolFromEnv)
	assertInvalidFromEnv(t, "yes", monad.GetBoolFromEnv)
}

func TestGetBoolSliceFromEnv(t *testing.T) {
	assertScalar(t, "true, false", []bool{true, false}, monad.GetBoolSliceFromEnv, monad.MustGetBoolSliceFromEnv)
}

func TestGetDurationFromEnv(t *testing.T) {
	assertScalar(t, "4h3m", 4*time.Hour+3*time.Minute, monad.GetDurationFromEnv, monad.MustGetDurationFromEnv)
	assertInvalidFromEnv(t, "90", monad.GetDurationFromEnv)
}

func TestGetDurationSliceFromEnv(t *testing.T) {
	assertScalar(
		t,
		"3m,4h3m",
		[]time.Duration{3 * time.Minute, 4*time.Hour + 3*time.Minute},
		monad.GetDurationSliceFromEnv,
		monad.MustGetDurationSliceFromEnv,
	)
}

func TestGetTimeFromEnv(t *testing.T) {
	get := func(name string) (monad.Option[time.Time], error) { return monad.GetTimeFromEnv(time.DateTime, name) }
	must := func(name string) monad.Option[time.Time] { return monad.MustGetTimeFromEnv(time.DateTime, name) }
	want := time.Date(2007, time.February, 3, 16, 5, 8, 0, time.UTC)

	assertScalar(t, "2007-02-03 16:05:08", want, get, must)
	assertInvalidFromEnv(t, "2007-02-03T16:05:08Z", get)
}

func TestGetTimeSliceFromEnv(t *testing.T) {
	get := func(name string) (monad.Option[[]time.Time], error) {
		return monad.GetTimeSliceFromEnv(time.DateTime, name)
	}
	must := func(name string) monad.Option[[]time.Time] { return monad.MustGetTimeSliceFromEnv(time.DateTime, name) }
	want := []time.Time{
		time.Date(2007, time.February, 3, 16, 5, 8, 0, time.UTC),
		time.Date(2020, time.April, 5, 10, 2, 1, 0, time.UTC),
	}

	assertScalar(t, "2007-02-03 16:05:08,2020-04-05 10:02:01", want, get, must)
}

func TestGetFloat32FromEnv(t *testing.T) {
	assertScalar(t, "42.42", float32(42.42), monad.GetFloat32FromEnv, monad.MustGetFloat32FromEnv)
}

func TestGetFloat32SliceFromEnv(t *testing.T) {
	assertScalar(
		t,
		"42.42,43.43",
		[]float32{42.42, 43.43},
		monad.GetFloat32SliceFromEnv,
		monad.MustGetFloat32SliceFromEnv,
	)
}

func TestGetFloat64FromEnv(t *testing.T) {
	assertScalar(t, "42.42", 42.42, monad.GetFloat64FromEnv, monad.MustGetFloat64FromEnv)
}

func TestGetFloat64SliceFromEnv(t *testing.T) {
	assertScalar(
		t,
		"42.42,43.43",
		[]float64{42.42, 43.43},
		monad.GetFloat64SliceFromEnv,
		monad.MustGetFloat64SliceFromEnv,
	)
}

func TestGetIntFromEnv(t *testing.T) {
	assertScalar(t, "42", 42, monad.GetIntFromEnv, monad.MustGetIntFromEnv)
	assertFromEnv(t, "-7", -7, monad.GetIntFromEnv)
	assertFromEnv(t, "0x1F", 31, monad.GetIntFromEnv)
	assertFromEnv(t, "0o17", 15, monad.GetIntFromEnv)
	assertFromEnv(t, "0b101", 5, monad.GetIntFromEnv)
	assertInvalidFromEnv(t, "4.2", monad.GetIntFromEnv)
}

func TestGetIntSliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []int{42, 43}, monad.GetIntSliceFromEnv, monad.MustGetIntSliceFromEnv)

	t.Run("trims whitespace around items", func(t *testing.T) {
		t.Setenv(testEnvName, " 1 , 2 ,3")

		option, err := monad.GetIntSliceFromEnv(testEnvName)

		require.NoError(t, err)
		require.Equal(t, monad.Some([]int{1, 2, 3}), option)
	})

	t.Run("strips quotes", func(t *testing.T) {
		t.Setenv(testEnvName, `'1',"2",`+"`3`")

		option, err := monad.GetIntSliceFromEnv(testEnvName)

		require.NoError(t, err)
		require.Equal(t, monad.Some([]int{1, 2, 3}), option)
	})

	t.Run("yields a present empty slice for an empty variable", func(t *testing.T) {
		t.Setenv(testEnvName, "")

		option, err := monad.GetIntSliceFromEnv(testEnvName)

		require.NoError(t, err)
		require.True(t, option.IsPresent())
		require.Empty(t, option.OrElsePanic("expected a value"))
	})

	t.Run("fails when one item is invalid", func(t *testing.T) {
		t.Setenv(testEnvName, "1,x,3")

		option, err := monad.GetIntSliceFromEnv(testEnvName)

		require.Error(t, err)
		require.True(t, option.IsEmpty())
	})
}

func TestGetInt8FromEnv(t *testing.T) {
	assertScalar(t, "42", int8(42), monad.GetInt8FromEnv, monad.MustGetInt8FromEnv)
	assertInvalidFromEnv(t, "200", monad.GetInt8FromEnv)
}

func TestGetInt8SliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []int8{42, 43}, monad.GetInt8SliceFromEnv, monad.MustGetInt8SliceFromEnv)
}

func TestGetInt16FromEnv(t *testing.T) {
	assertScalar(t, "42", int16(42), monad.GetInt16FromEnv, monad.MustGetInt16FromEnv)
	assertInvalidFromEnv(t, "40000", monad.GetInt16FromEnv)
}

func TestGetInt16SliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []int16{42, 43}, monad.GetInt16SliceFromEnv, monad.MustGetInt16SliceFromEnv)
}

func TestGetInt32FromEnv(t *testing.T) {
	assertScalar(t, "42", int32(42), monad.GetInt32FromEnv, monad.MustGetInt32FromEnv)
	assertInvalidFromEnv(t, "3000000000", monad.GetInt32FromEnv)
}

func TestGetInt32SliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []int32{42, 43}, monad.GetInt32SliceFromEnv, monad.MustGetInt32SliceFromEnv)
}

func TestGetInt64FromEnv(t *testing.T) {
	assertScalar(t, "42", int64(42), monad.GetInt64FromEnv, monad.MustGetInt64FromEnv)
}

func TestGetInt64SliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []int64{42, 43}, monad.GetInt64SliceFromEnv, monad.MustGetInt64SliceFromEnv)
}

func TestGetUintFromEnv(t *testing.T) {
	assertScalar(t, "42", uint(42), monad.GetUintFromEnv, monad.MustGetUintFromEnv)
	assertFromEnv(t, "0b101", uint(5), monad.GetUintFromEnv)
	assertInvalidFromEnv(t, "-1", monad.GetUintFromEnv)
}

func TestGetUintSliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []uint{42, 43}, monad.GetUintSliceFromEnv, monad.MustGetUintSliceFromEnv)
}

func TestGetUint8FromEnv(t *testing.T) {
	assertScalar(t, "42", uint8(42), monad.GetUint8FromEnv, monad.MustGetUint8FromEnv)
	assertInvalidFromEnv(t, "300", monad.GetUint8FromEnv)
}

func TestGetUint8SliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []uint8{42, 43}, monad.GetUint8SliceFromEnv, monad.MustGetUint8SliceFromEnv)
}

func TestGetUint16FromEnv(t *testing.T) {
	assertScalar(t, "42", uint16(42), monad.GetUint16FromEnv, monad.MustGetUint16FromEnv)
	assertInvalidFromEnv(t, "70000", monad.GetUint16FromEnv)
}

func TestGetUint16SliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []uint16{42, 43}, monad.GetUint16SliceFromEnv, monad.MustGetUint16SliceFromEnv)
}

func TestGetUint32FromEnv(t *testing.T) {
	assertScalar(t, "42", uint32(42), monad.GetUint32FromEnv, monad.MustGetUint32FromEnv)
	assertInvalidFromEnv(t, "5000000000", monad.GetUint32FromEnv)
}

func TestGetUint32SliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []uint32{42, 43}, monad.GetUint32SliceFromEnv, monad.MustGetUint32SliceFromEnv)
}

func TestGetUint64FromEnv(t *testing.T) {
	assertScalar(t, "42", uint64(42), monad.GetUint64FromEnv, monad.MustGetUint64FromEnv)
}

func TestGetUint64SliceFromEnv(t *testing.T) {
	assertScalar(t, "42,43", []uint64{42, 43}, monad.GetUint64SliceFromEnv, monad.MustGetUint64SliceFromEnv)
}

func TestGetStringFromEnv(t *testing.T) {
	t.Run("yields None when the variable is unset", func(t *testing.T) {
		require.True(t, monad.GetStringFromEnv(testEnvUnset).IsEmpty())
	})

	t.Run("yields Some when the variable is set", func(t *testing.T) {
		t.Setenv(testEnvName, "hello")

		require.Equal(t, monad.Some("hello"), monad.GetStringFromEnv(testEnvName))
	})

	t.Run("yields Some of an empty string when the variable is empty", func(t *testing.T) {
		t.Setenv(testEnvName, "")

		require.Equal(t, monad.Some(""), monad.GetStringFromEnv(testEnvName))
	})
}

func TestGetStringSliceFromEnv(t *testing.T) {
	assertUnsetFromEnv(t, monad.GetStringSliceFromEnv)
	assertFromEnv(t, "a, b", []string{"a", "b"}, monad.GetStringSliceFromEnv)
	assertFromEnv(t, `'a', "b"`, []string{"a", "b"}, monad.GetStringSliceFromEnv)
	assertFromEnv(t, "a,,b", []string{"a", "", "b"}, monad.GetStringSliceFromEnv)
	assertFromEnv(t, "", []string{}, monad.GetStringSliceFromEnv)
	assertMustFromEnv(t, "a, b", []string{"a", "b"}, monad.MustGetStringSliceFromEnv)
}

func TestGetURLFromEnv(t *testing.T) {
	want, err := url.Parse("https://example.com/path?q=1")
	require.NoError(t, err)

	assertScalar(t, want.String(), *want, monad.GetURLFromEnv, monad.MustGetURLFromEnv)
	assertInvalidFromEnv(t, "://bad", monad.GetURLFromEnv)
}

func TestGetURLSliceFromEnv(t *testing.T) {
	first, err := url.Parse("https://example.com")
	require.NoError(t, err)
	second, err := url.Parse("https://example.org/x")
	require.NoError(t, err)

	assertScalar(
		t,
		first.String()+", "+second.String(),
		[]url.URL{*first, *second},
		monad.GetURLSliceFromEnv,
		monad.MustGetURLSliceFromEnv,
	)
	assertInvalidFromEnv(t, "https://example.com,://bad", monad.GetURLSliceFromEnv)
}

func ExampleGetIntFromEnv() {
	_ = os.Setenv("EXAMPLE_PORT", "8080")
	defer func() { _ = os.Unsetenv("EXAMPLE_PORT") }()

	port, err := monad.GetIntFromEnv("EXAMPLE_PORT")
	fmt.Println(port, err)

	missing, err := monad.GetIntFromEnv("EXAMPLE_MISSING")
	fmt.Println(missing, err)
	// Output:
	// Some(8080) <nil>
	// None <nil>
}

func ExampleOptionFromEnv() {
	_ = os.Setenv("EXAMPLE_IP", "127.0.0.1")
	defer func() { _ = os.Unsetenv("EXAMPLE_IP") }()

	parse := func(s string) (net.IP, error) {
		ip := net.ParseIP(s)
		if ip == nil {
			return nil, errors.New("invalid IP: " + s)
		}
		return ip, nil
	}
	ip, err := monad.OptionFromEnv("EXAMPLE_IP", parse)
	fmt.Println(ip, err)
	// Output:
	// Some(127.0.0.1) <nil>
}

// TestSliceQuoting covers how a comma-separated variable is split. The rules
// have to serve two masters: shell and .env files that wrap a whole value in
// quotes, and items that legitimately contain a comma or an apostrophe.
func TestSliceQuoting(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"plain list", `a,b`, []string{"a", "b"}},
		{"whole value wrapped, as a shell would", `"a.example.com, b.example.com"`,
			[]string{"a.example.com", "b.example.com"}},
		{"per-item quotes of every flavor", `'1',"2",` + "`3`", []string{"1", "2", "3"}},
		{"CSV quoting protects an embedded comma", `"1,2",3`, []string{"1,2", "3"}},
		{"two quoted items are not one wrapped value", `"A","B"`, []string{"A", "B"}},
		{"a single wrapped item stays one item", `"only one"`, []string{"only one"}},
		{"an apostrophe inside a word survives", `O'Brien,Smith`, []string{"O'Brien", "Smith"}},
		{"an inner quote is not stripped", `say "hi",bye`, []string{`say "hi"`, "bye"}},
		{"a newline separates items like a comma", "a,b\nc,d", []string{"a", "b", "c", "d"}},
		{"surrounding whitespace is trimmed", ` a , b `, []string{"a", "b"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(testEnvName, tc.raw)

			option, err := monad.GetStringSliceFromEnv(testEnvName)

			require.NoError(t, err)
			require.Equal(t, tc.want, option.OrElse(nil))
		})
	}
}

// TestSliceBlankValues pins that every way of spelling "no content" gives the
// same answer. A single space used to read as one empty item, which then failed
// to parse for every type except string.
func TestSliceBlankValues(t *testing.T) {
	for _, raw := range []string{"", " ", "\n", "   \n  "} {
		t.Run(fmt.Sprintf("%q", raw), func(t *testing.T) {
			t.Setenv(testEnvName, raw)

			option, err := monad.GetIntSliceFromEnv(testEnvName)

			require.NoError(t, err)
			require.True(t, option.IsPresent())
			require.Empty(t, option.OrElse(nil))
		})
	}
}

// TestSliceMultilineReportsErrors guards against silently truncating a value at
// its first line, which hid any bad item on the lines after it.
func TestSliceMultilineReportsErrors(t *testing.T) {
	t.Setenv(testEnvName, "1,2\n3,x")

	option, err := monad.GetIntSliceFromEnv(testEnvName)

	require.Error(t, err, "an unparsable item on a later line must be reported")
	require.True(t, option.IsEmpty())
}

// TestURLRequiresAbsolute covers the contract every other type follows: a set
// but unparsable variable yields None and an error. url.Parse alone accepts
// almost anything as a relative reference, so URL used to never fail.
func TestURLRequiresAbsolute(t *testing.T) {
	t.Run("rejects values that are not URLs", func(t *testing.T) {
		for _, raw := range []string{"", " ", "42", "not a url", "example.com"} {
			t.Run(fmt.Sprintf("%q", raw), func(t *testing.T) {
				t.Setenv(testEnvName, raw)

				option, err := monad.GetURLFromEnv(testEnvName)

				require.Error(t, err)
				require.True(t, option.IsEmpty())
			})
		}
	})

	t.Run("accepts absolute URLs of several shapes", func(t *testing.T) {
		for _, raw := range []string{
			"https://example.com/path?q=1",
			"mailto:someone@example.com",
			"file:///tmp/x",
			"postgres://user@host:5432/db",
		} {
			t.Run(raw, func(t *testing.T) {
				t.Setenv(testEnvName, raw)

				option, err := monad.GetURLFromEnv(testEnvName)

				require.NoError(t, err)
				require.True(t, option.IsPresent())
			})
		}
	})
}

// TestIntUsesPlatformWidth guards the bit size used for int and uint. Parsing
// at 64 bits and converting would wrap silently on a 32-bit build, turning an
// out-of-range value into a plausible one with no error.
func TestIntUsesPlatformWidth(t *testing.T) {
	t.Run("accepts the largest value an int can hold", func(t *testing.T) {
		t.Setenv(testEnvName, strconv.Itoa(math.MaxInt))

		option, err := monad.GetIntFromEnv(testEnvName)

		require.NoError(t, err)
		require.Equal(t, math.MaxInt, option.OrElse(0))
	})

	t.Run("rejects one past it rather than wrapping", func(t *testing.T) {
		t.Setenv(testEnvName, "1"+strconv.FormatUint(math.MaxUint64, 10))

		option, err := monad.GetIntFromEnv(testEnvName)

		require.Error(t, err)
		require.True(t, option.IsEmpty())
	})

	t.Run("uint still rejects a negative value", func(t *testing.T) {
		t.Setenv(testEnvName, "-1")

		_, err := monad.GetUintFromEnv(testEnvName)

		require.Error(t, err)
	})
}

package monad

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// OptionFromEnv reads the environment variable name and parses it with parse.
// An unset variable yields None and no error; a set but unparsable variable
// yields None and the parse error; otherwise Some of the parsed value.
func OptionFromEnv[T any](name string, parse func(string) (T, error)) (Option[T], error) {
	raw, found := os.LookupEnv(name)
	if !found {
		return None[T](), nil
	}
	value, err := parse(raw)
	if err != nil {
		return None[T](), err
	}
	return Some(value), nil
}

// OptionSliceFromEnv reads the environment variable name as a comma-separated
// list and parses each item with parse. An unset variable yields None and no
// error; a malformed list or a parse failure on any item yields None and the
// error. An empty variable yields Some of an empty slice.
func OptionSliceFromEnv[T any](name string, parse func(string) (T, error)) (Option[[]T], error) {
	raw, found := os.LookupEnv(name)
	if !found {
		return None[[]T](), nil
	}
	items, err := readSlice(raw)
	if err != nil {
		return None[[]T](), err
	}
	out := make([]T, 0, len(items))
	for _, item := range items {
		value, parseErr := parse(item)
		if parseErr != nil {
			return None[[]T](), parseErr
		}
		out = append(out, value)
	}
	return Some(out), nil
}

// GetBoolFromEnv reads the environment variable name as a boolean. An
// unset variable yields None; a set but unparsable one yields None and an
// error.
func GetBoolFromEnv(name string) (Option[bool], error) {
	return OptionFromEnv(name, strconv.ParseBool)
}

// MustGetBoolFromEnv is like [GetBoolFromEnv] but panics on a parse error.
func MustGetBoolFromEnv(name string) Option[bool] {
	return mustFromEnv(name, GetBoolFromEnv)
}

// GetBoolSliceFromEnv reads the environment variable name as a
// comma-separated list of booleans. An unset variable yields None; a parse
// failure yields None and an error.
func GetBoolSliceFromEnv(name string) (Option[[]bool], error) {
	return OptionSliceFromEnv(name, strconv.ParseBool)
}

// MustGetBoolSliceFromEnv is like [GetBoolSliceFromEnv] but panics on a
// parse error.
func MustGetBoolSliceFromEnv(name string) Option[[]bool] {
	return mustFromEnv(name, GetBoolSliceFromEnv)
}

// GetDurationFromEnv reads the environment variable name as a duration. An
// unset variable yields None; a set but unparsable one yields None and an
// error.
func GetDurationFromEnv(name string) (Option[time.Duration], error) {
	return OptionFromEnv(name, time.ParseDuration)
}

// MustGetDurationFromEnv is like [GetDurationFromEnv] but panics on a parse error.
func MustGetDurationFromEnv(name string) Option[time.Duration] {
	return mustFromEnv(name, GetDurationFromEnv)
}

// GetDurationSliceFromEnv reads the environment variable name as a
// comma-separated list of durations. An unset variable yields None; a parse
// failure yields None and an error.
func GetDurationSliceFromEnv(name string) (Option[[]time.Duration], error) {
	return OptionSliceFromEnv(name, time.ParseDuration)
}

// MustGetDurationSliceFromEnv is like [GetDurationSliceFromEnv] but panics on a
// parse error.
func MustGetDurationSliceFromEnv(name string) Option[[]time.Duration] {
	return mustFromEnv(name, GetDurationSliceFromEnv)
}

func parseTime(layout string) func(string) (time.Time, error) {
	return func(s string) (time.Time, error) {
		return time.Parse(layout, s)
	}
}

// GetTimeFromEnv reads the environment variable name as a time formatted
// according to layout (see [time.Parse]). An unset variable yields None; a set
// but unparsable one yields None and an error.
func GetTimeFromEnv(layout, name string) (Option[time.Time], error) {
	return OptionFromEnv(name, parseTime(layout))
}

// MustGetTimeFromEnv is like [GetTimeFromEnv] but panics on a parse error.
func MustGetTimeFromEnv(layout, name string) Option[time.Time] {
	value, err := GetTimeFromEnv(layout, name)
	if err != nil {
		panic(err)
	}
	return value
}

// GetTimeSliceFromEnv reads the environment variable name as a comma-separated
// list of times formatted according to layout (see [time.Parse]). An unset
// variable yields None; a parse failure yields None and an error.
func GetTimeSliceFromEnv(layout, name string) (Option[[]time.Time], error) {
	return OptionSliceFromEnv(name, parseTime(layout))
}

// MustGetTimeSliceFromEnv is like [GetTimeSliceFromEnv] but panics on a parse
// error.
func MustGetTimeSliceFromEnv(layout, name string) Option[[]time.Time] {
	value, err := GetTimeSliceFromEnv(layout, name)
	if err != nil {
		panic(err)
	}
	return value
}

// parseFloat32 parses s as a 32-bit float.
func parseFloat32(s string) (float32, error) {
	value, err := strconv.ParseFloat(s, 32)
	return float32(value), err
}

// GetFloat32FromEnv reads the environment variable name as a 32-bit float. An
// unset variable yields None; a set but unparsable one yields None and an
// error.
func GetFloat32FromEnv(name string) (Option[float32], error) {
	return OptionFromEnv(name, parseFloat32)
}

// MustGetFloat32FromEnv is like [GetFloat32FromEnv] but panics on a parse error.
func MustGetFloat32FromEnv(name string) Option[float32] {
	return mustFromEnv(name, GetFloat32FromEnv)
}

// GetFloat32SliceFromEnv reads the environment variable name as a
// comma-separated list of 32-bit floats. An unset variable yields None; a parse
// failure yields None and an error.
func GetFloat32SliceFromEnv(name string) (Option[[]float32], error) {
	return OptionSliceFromEnv(name, parseFloat32)
}

// MustGetFloat32SliceFromEnv is like [GetFloat32SliceFromEnv] but panics on a
// parse error.
func MustGetFloat32SliceFromEnv(name string) Option[[]float32] {
	return mustFromEnv(name, GetFloat32SliceFromEnv)
}

// parseFloat64 parses s as a 64-bit float.
func parseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// GetFloat64FromEnv reads the environment variable name as a 64-bit float. An
// unset variable yields None; a set but unparsable one yields None and an
// error.
func GetFloat64FromEnv(name string) (Option[float64], error) {
	return OptionFromEnv(name, parseFloat64)
}

// MustGetFloat64FromEnv is like [GetFloat64FromEnv] but panics on a parse error.
func MustGetFloat64FromEnv(name string) Option[float64] {
	return mustFromEnv(name, GetFloat64FromEnv)
}

// GetFloat64SliceFromEnv reads the environment variable name as a
// comma-separated list of 64-bit floats. An unset variable yields None; a parse
// failure yields None and an error.
func GetFloat64SliceFromEnv(name string) (Option[[]float64], error) {
	return OptionSliceFromEnv(name, parseFloat64)
}

// MustGetFloat64SliceFromEnv is like [GetFloat64SliceFromEnv] but panics on a
// parse error.
func MustGetFloat64SliceFromEnv(name string) Option[[]float64] {
	return mustFromEnv(name, GetFloat64SliceFromEnv)
}

// parseInt parses s as an int.
func parseInt(s string) (int, error) {
	// bitSize 0 means "the width of an int on this platform". Parsing at 64
	// and converting would silently wrap on a 32-bit build, turning an
	// out-of-range value into a plausible one with no error.
	value, err := strconv.ParseInt(s, 0, 0)
	return int(value), err
}

// GetIntFromEnv reads the environment variable name as an int. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetIntFromEnv(name string) (Option[int], error) {
	return OptionFromEnv(name, parseInt)
}

// MustGetIntFromEnv is like [GetIntFromEnv] but panics on a parse error.
func MustGetIntFromEnv(name string) Option[int] {
	return mustFromEnv(name, GetIntFromEnv)
}

// GetIntSliceFromEnv reads the environment variable name as a
// comma-separated list of ints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetIntSliceFromEnv(name string) (Option[[]int], error) {
	return OptionSliceFromEnv(name, parseInt)
}

// MustGetIntSliceFromEnv is like [GetIntSliceFromEnv] but panics on a
// parse error.
func MustGetIntSliceFromEnv(name string) Option[[]int] {
	return mustFromEnv(name, GetIntSliceFromEnv)
}

// parseInt8 parses s as an 8-bit int.
func parseInt8(s string) (int8, error) {
	value, err := strconv.ParseInt(s, 0, 8)
	return int8(value), err
}

// GetInt8FromEnv reads the environment variable name as an 8-bit int. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetInt8FromEnv(name string) (Option[int8], error) {
	return OptionFromEnv(name, parseInt8)
}

// MustGetInt8FromEnv is like [GetInt8FromEnv] but panics on a parse error.
func MustGetInt8FromEnv(name string) Option[int8] {
	return mustFromEnv(name, GetInt8FromEnv)
}

// GetInt8SliceFromEnv reads the environment variable name as a
// comma-separated list of 8-bit ints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetInt8SliceFromEnv(name string) (Option[[]int8], error) {
	return OptionSliceFromEnv(name, parseInt8)
}

// MustGetInt8SliceFromEnv is like [GetInt8SliceFromEnv] but panics on a
// parse error.
func MustGetInt8SliceFromEnv(name string) Option[[]int8] {
	return mustFromEnv(name, GetInt8SliceFromEnv)
}

// parseInt16 parses s as a 16-bit int.
func parseInt16(s string) (int16, error) {
	value, err := strconv.ParseInt(s, 0, 16)
	return int16(value), err
}

// GetInt16FromEnv reads the environment variable name as a 16-bit int. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetInt16FromEnv(name string) (Option[int16], error) {
	return OptionFromEnv(name, parseInt16)
}

// MustGetInt16FromEnv is like [GetInt16FromEnv] but panics on a parse error.
func MustGetInt16FromEnv(name string) Option[int16] {
	return mustFromEnv(name, GetInt16FromEnv)
}

// GetInt16SliceFromEnv reads the environment variable name as a
// comma-separated list of 16-bit ints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetInt16SliceFromEnv(name string) (Option[[]int16], error) {
	return OptionSliceFromEnv(name, parseInt16)
}

// MustGetInt16SliceFromEnv is like [GetInt16SliceFromEnv] but panics on a
// parse error.
func MustGetInt16SliceFromEnv(name string) Option[[]int16] {
	return mustFromEnv(name, GetInt16SliceFromEnv)
}

// parseInt32 parses s as a 32-bit int.
func parseInt32(s string) (int32, error) {
	value, err := strconv.ParseInt(s, 0, 32)
	return int32(value), err
}

// GetInt32FromEnv reads the environment variable name as a 32-bit int. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetInt32FromEnv(name string) (Option[int32], error) {
	return OptionFromEnv(name, parseInt32)
}

// MustGetInt32FromEnv is like [GetInt32FromEnv] but panics on a parse error.
func MustGetInt32FromEnv(name string) Option[int32] {
	return mustFromEnv(name, GetInt32FromEnv)
}

// GetInt32SliceFromEnv reads the environment variable name as a
// comma-separated list of 32-bit ints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetInt32SliceFromEnv(name string) (Option[[]int32], error) {
	return OptionSliceFromEnv(name, parseInt32)
}

// MustGetInt32SliceFromEnv is like [GetInt32SliceFromEnv] but panics on a
// parse error.
func MustGetInt32SliceFromEnv(name string) Option[[]int32] {
	return mustFromEnv(name, GetInt32SliceFromEnv)
}

// parseInt64 parses s as a 64-bit int.
func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 0, 64)
}

// GetInt64FromEnv reads the environment variable name as a 64-bit int. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetInt64FromEnv(name string) (Option[int64], error) {
	return OptionFromEnv(name, parseInt64)
}

// MustGetInt64FromEnv is like [GetInt64FromEnv] but panics on a parse error.
func MustGetInt64FromEnv(name string) Option[int64] {
	return mustFromEnv(name, GetInt64FromEnv)
}

// GetInt64SliceFromEnv reads the environment variable name as a
// comma-separated list of 64-bit ints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetInt64SliceFromEnv(name string) (Option[[]int64], error) {
	return OptionSliceFromEnv(name, parseInt64)
}

// MustGetInt64SliceFromEnv is like [GetInt64SliceFromEnv] but panics on a
// parse error.
func MustGetInt64SliceFromEnv(name string) Option[[]int64] {
	return mustFromEnv(name, GetInt64SliceFromEnv)
}

// parseUint parses s as an uint.
func parseUint(s string) (uint, error) {
	// See parseInt: bitSize 0 tracks the platform width of a uint.
	value, err := strconv.ParseUint(s, 0, 0)
	return uint(value), err
}

// GetUintFromEnv reads the environment variable name as an uint. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUintFromEnv(name string) (Option[uint], error) {
	return OptionFromEnv(name, parseUint)
}

// MustGetUintFromEnv is like [GetUintFromEnv] but panics on a parse error.
func MustGetUintFromEnv(name string) Option[uint] {
	return mustFromEnv(name, GetUintFromEnv)
}

// GetUintSliceFromEnv reads the environment variable name as a
// comma-separated list of uints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUintSliceFromEnv(name string) (Option[[]uint], error) {
	return OptionSliceFromEnv(name, parseUint)
}

// MustGetUintSliceFromEnv is like [GetUintSliceFromEnv] but panics on a
// parse error.
func MustGetUintSliceFromEnv(name string) Option[[]uint] {
	return mustFromEnv(name, GetUintSliceFromEnv)
}

// parseUint8 parses s as an 8-bit uint.
func parseUint8(s string) (uint8, error) {
	value, err := strconv.ParseUint(s, 0, 8)
	return uint8(value), err
}

// GetUint8FromEnv reads the environment variable name as an 8-bit uint. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUint8FromEnv(name string) (Option[uint8], error) {
	return OptionFromEnv(name, parseUint8)
}

// MustGetUint8FromEnv is like [GetUint8FromEnv] but panics on a parse error.
func MustGetUint8FromEnv(name string) Option[uint8] {
	return mustFromEnv(name, GetUint8FromEnv)
}

// GetUint8SliceFromEnv reads the environment variable name as a
// comma-separated list of 8-bit uints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUint8SliceFromEnv(name string) (Option[[]uint8], error) {
	return OptionSliceFromEnv(name, parseUint8)
}

// MustGetUint8SliceFromEnv is like [GetUint8SliceFromEnv] but panics on a
// parse error.
func MustGetUint8SliceFromEnv(name string) Option[[]uint8] {
	return mustFromEnv(name, GetUint8SliceFromEnv)
}

// parseUint16 parses s as a 16-bit uint.
func parseUint16(s string) (uint16, error) {
	value, err := strconv.ParseUint(s, 0, 16)
	return uint16(value), err
}

// GetUint16FromEnv reads the environment variable name as a 16-bit uint. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUint16FromEnv(name string) (Option[uint16], error) {
	return OptionFromEnv(name, parseUint16)
}

// MustGetUint16FromEnv is like [GetUint16FromEnv] but panics on a parse error.
func MustGetUint16FromEnv(name string) Option[uint16] {
	return mustFromEnv(name, GetUint16FromEnv)
}

// GetUint16SliceFromEnv reads the environment variable name as a
// comma-separated list of 16-bit uints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUint16SliceFromEnv(name string) (Option[[]uint16], error) {
	return OptionSliceFromEnv(name, parseUint16)
}

// MustGetUint16SliceFromEnv is like [GetUint16SliceFromEnv] but panics on a
// parse error.
func MustGetUint16SliceFromEnv(name string) Option[[]uint16] {
	return mustFromEnv(name, GetUint16SliceFromEnv)
}

// parseUint32 parses s as a 32-bit uint.
func parseUint32(s string) (uint32, error) {
	value, err := strconv.ParseUint(s, 0, 32)
	return uint32(value), err
}

// GetUint32FromEnv reads the environment variable name as a 32-bit uint. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUint32FromEnv(name string) (Option[uint32], error) {
	return OptionFromEnv(name, parseUint32)
}

// MustGetUint32FromEnv is like [GetUint32FromEnv] but panics on a parse error.
func MustGetUint32FromEnv(name string) Option[uint32] {
	return mustFromEnv(name, GetUint32FromEnv)
}

// GetUint32SliceFromEnv reads the environment variable name as a
// comma-separated list of 32-bit uints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUint32SliceFromEnv(name string) (Option[[]uint32], error) {
	return OptionSliceFromEnv(name, parseUint32)
}

// MustGetUint32SliceFromEnv is like [GetUint32SliceFromEnv] but panics on a
// parse error.
func MustGetUint32SliceFromEnv(name string) Option[[]uint32] {
	return mustFromEnv(name, GetUint32SliceFromEnv)
}

// parseUint64 parses s as a 64-bit uint.
func parseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 0, 64)
}

// GetUint64FromEnv reads the environment variable name as a 64-bit uint. An
// unset variable yields None; a set but unparsable one yields None and an
// error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUint64FromEnv(name string) (Option[uint64], error) {
	return OptionFromEnv(name, parseUint64)
}

// MustGetUint64FromEnv is like [GetUint64FromEnv] but panics on a parse error.
func MustGetUint64FromEnv(name string) Option[uint64] {
	return mustFromEnv(name, GetUint64FromEnv)
}

// GetUint64SliceFromEnv reads the environment variable name as a
// comma-separated list of 64-bit uints. An unset variable yields None; a parse
// failure yields None and an error. Integers are parsed in base 0, so the 0x, 0o, and 0b prefixes are
// accepted.
func GetUint64SliceFromEnv(name string) (Option[[]uint64], error) {
	return OptionSliceFromEnv(name, parseUint64)
}

// MustGetUint64SliceFromEnv is like [GetUint64SliceFromEnv] but panics on a
// parse error.
func MustGetUint64SliceFromEnv(name string) Option[[]uint64] {
	return mustFromEnv(name, GetUint64SliceFromEnv)
}

// GetStringFromEnv reads the environment variable name as a string. An unset
// variable yields None; a set one yields Some, even when it is empty. A lookup
// cannot fail, so there is no error to report and no Must variant.
func GetStringFromEnv(name string) Option[string] {
	value, found := os.LookupEnv(name)
	if !found {
		return None[string]()
	}
	return Some(value)
}

// GetStringSliceFromEnv reads the environment variable name as a
// comma-separated list of strings. An unset variable yields None; a malformed
// list yields None and an error.
func GetStringSliceFromEnv(name string) (Option[[]string], error) {
	return OptionSliceFromEnv(name, func(s string) (string, error) { return s, nil })
}

// MustGetStringSliceFromEnv is like [GetStringSliceFromEnv] but panics on a
// parse error.
func MustGetStringSliceFromEnv(name string) Option[[]string] {
	return mustFromEnv(name, GetStringSliceFromEnv)
}

// parseURL parses s as a URL and returns it by value.
//
// url.Parse accepts almost any string as a relative reference, so on its own it
// would report success for "", " ", and "not a url" — leaving URL the one type
// whose "set but unparsable" case never fires. Requiring an absolute URL
// restores the contract every other type follows.
func parseURL(s string) (url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return url.URL{}, err
	}
	if u.Scheme == "" {
		return url.URL{}, fmt.Errorf("parse %q: missing URL scheme", s)
	}
	if u.Host == "" && u.Opaque == "" && u.Path == "" {
		return url.URL{}, fmt.Errorf("parse %q: missing URL host", s)
	}
	return *u, nil
}

// GetURLFromEnv reads the environment variable name as a URL. An
// unset variable yields None; a set but unparsable one yields None and an
// error.
func GetURLFromEnv(name string) (Option[url.URL], error) {
	return OptionFromEnv(name, parseURL)
}

// MustGetURLFromEnv is like [GetURLFromEnv] but panics on a parse error.
func MustGetURLFromEnv(name string) Option[url.URL] {
	return mustFromEnv(name, GetURLFromEnv)
}

// GetURLSliceFromEnv reads the environment variable name as a
// comma-separated list of URLs. An unset variable yields None; a parse
// failure yields None and an error.
func GetURLSliceFromEnv(name string) (Option[[]url.URL], error) {
	return OptionSliceFromEnv(name, parseURL)
}

// MustGetURLSliceFromEnv is like [GetURLSliceFromEnv] but panics on a
// parse error.
func MustGetURLSliceFromEnv(name string) Option[[]url.URL] {
	return mustFromEnv(name, GetURLSliceFromEnv)
}

func mustFromEnv[T any](name string, get func(string) (Option[T], error)) Option[T] {
	value, err := get(name)
	if err != nil {
		panic(err)
	}
	return value
}

// readSlice parses value as a comma-separated list.
//
// Shell and .env files routinely wrap a whole value in quotes, so one matching
// pair around the entire string is removed first and treated as noise. What
// remains is read as CSV, which is what allows an item to contain a comma
// (HOSTS='"a,b",c' is two items). Each item is then trimmed and has its own
// matching quote pair removed, so 'a',"b",`c` reads as three items.
//
// Stripping every quote character up front instead would be simpler but wrong:
// it makes CSV quoting inert, so no item could ever contain a comma, and it
// mangles ordinary data — O'Brien would become OBrien.
func readSlice(value string) ([]string, error) {
	value = stripSurroundingQuotes(value)

	// A blank value is an empty list, however it is spelled. Without this a
	// variable set to a single space would read as one empty item, which then
	// fails to parse for every type except string.
	if strings.TrimSpace(value) == "" {
		return []string{}, nil
	}

	items, err := readAsCSV(value)
	if err != nil {
		return []string{}, err
	}
	for i, item := range items {
		items[i] = stripSurroundingQuotes(strings.TrimSpace(item))
	}
	return items, nil
}

// stripSurroundingQuotes removes one matching pair of ", ' or ` wrapping s.
//
// The pair is only removed when the text between the quotes contains no
// further occurrence of that same character. Without that condition a
// perfectly good two-item list such as `"a","b"` would look like one quoted
// value, lose its outer quotes, and be re-split in the wrong places.
func stripSurroundingQuotes(s string) string {
	// A quoted value needs at least the two quote characters themselves.
	const minQuoted = 2

	if len(s) < minQuoted {
		return s
	}
	first, last := s[0], s[len(s)-1]
	if first != last {
		return s
	}
	if first != '"' && first != '\'' && first != '`' {
		return s
	}
	inner := s[1 : len(s)-1]
	if strings.IndexByte(inner, first) >= 0 {
		return s
	}
	return inner
}

// readAsCSV reads every record in value and flattens them into one list, so a
// newline separates items just as a comma does. Reading a single record would
// silently discard everything after the first line — including items that
// would not have parsed, hiding the error entirely.
func readAsCSV(value string) ([]string, error) {
	reader := csv.NewReader(strings.NewReader(value))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	out := []string{}
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, record...)
	}
}

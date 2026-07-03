// Command serialization demonstrates encoding and decoding Options with
// encoding/json and gopkg.in/yaml.v3.
//
// The problem an Option solves at a wire boundary: a JSON document can say a
// field is absent, is null, or holds a zero value, and those three often mean
// different things. Decoding into a plain int collapses all of them to 0.
// Option keeps them apart in both directions.
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	monad "github.com/grandper/go-monad"

	"gopkg.in/yaml.v3"
)

// User shows the three tag styles that matter. A plain tag always writes the
// field, using null when the Option is empty. "omitzero" (JSON, Go 1.24+) drops
// an empty Option because Option implements IsZero. For YAML the tag that
// consults IsZero is "omitempty" — note that "omitempty" does NOT do this for
// JSON, where an Option is a struct and structs are never considered empty.
type User struct {
	Name     monad.Option[string] `json:"name"              yaml:"name"`
	Nickname monad.Option[string] `json:"nickname,omitzero" yaml:"nickname,omitempty"`
	Age      monad.Option[int]    `json:"age,omitzero"      yaml:"age,omitempty"`
}

func main() {
	fmt.Println("=== A Present Value Encodes as Itself ===")
	encoding()

	fmt.Println("\n=== Absent, Null, and Zero Are Three Things ===")
	threeStates()

	fmt.Println("\n=== Choosing How Absence Appears ===")
	tags()

	fmt.Println("\n=== YAML ===")
	yamlRoundTrip()

	fmt.Println("\n=== Errors ===")
	errorsOnMismatch()
}

func encoding() {
	// An Option adds no wrapper of its own to the document: Some(x) encodes
	// exactly as x would, so the JSON stays the shape your API already
	// promises and Option remains an internal concern.
	data, _ := json.Marshal(User{
		Name:     monad.Some("Alice"),
		Nickname: monad.Some("Ali"),
		Age:      monad.Some(30),
	})
	fmt.Printf("  %s\n", data)

	bare, _ := json.Marshal(monad.Some(42))
	empty, _ := json.Marshal(monad.None[int]())
	fmt.Printf("  bare Some: %s, bare None: %s\n", bare, empty)
}

func threeStates() {
	// The case that motivates the type. Age 0 is a real answer; a missing age
	// is not an answer at all. Both survive the round trip distinguishable,
	// which is what a plain int field cannot manage.
	present, _ := json.Marshal(User{Name: monad.Some("Carol"), Age: monad.Some(0)})
	absent, _ := json.Marshal(User{Name: monad.Some("Carol")})
	fmt.Printf("  age present and zero: %s\n", present)
	fmt.Printf("  age absent:           %s\n", absent)

	// Decoding preserves the same distinction. An explicit null and a missing
	// key both give None; a present zero gives Some(0).
	var explicitNull, missingKey, zeroValue User
	_ = json.Unmarshal([]byte(`{"name":"Carol","age":null}`), &explicitNull)
	_ = json.Unmarshal([]byte(`{"name":"Carol"}`), &missingKey)
	_ = json.Unmarshal([]byte(`{"name":"Carol","age":0}`), &zeroValue)

	fmt.Printf("  null -> %v, missing -> %v, 0 -> %v\n",
		explicitNull.Age, missingKey.Age, zeroValue.Age)
}

func tags() {
	// With a plain tag the field is always written, so a consumer can tell
	// "we know there is no nickname" from "we did not send that field".
	// With omitzero it disappears entirely, which suits sparse payloads.
	sparse, _ := json.Marshal(User{Name: monad.Some("Bob")})
	fmt.Printf("  name is plain, nickname/age are omitzero: %s\n", sparse)

	// A key that never reaches UnmarshalJSON leaves the field at its zero
	// value — and the zero value of an Option is None, so absence decodes
	// correctly without any extra handling.
	var decoded User
	_ = json.Unmarshal([]byte(`{}`), &decoded)
	fmt.Printf("  empty document decodes to: %v / %v\n", decoded.Name, decoded.Age)
}

func yamlRoundTrip() {
	// YAML behaves the same way, with omitempty playing the role omitzero
	// plays in JSON.
	data, _ := yaml.Marshal(User{Name: monad.Some("Alice"), Age: monad.Some(30)})
	fmt.Printf("  encoded:\n%s", indent(string(data)))

	var back User
	if err := yaml.Unmarshal(data, &back); err != nil {
		fmt.Println("  decode failed:", err)
		return
	}
	fmt.Printf("  decoded: name=%v nickname=%v age=%v\n", back.Name, back.Nickname, back.Age)
}

func errorsOnMismatch() {
	// A type mismatch is reported as an ordinary decoding error rather than
	// being swallowed into None, so a malformed document is not silently
	// mistaken for an absent field.
	var age monad.Option[int]
	if err := json.Unmarshal([]byte(`"thirty"`), &age); err != nil {
		fmt.Printf("  wrong type: %v\n", err)
	}

	// The target is left untouched when decoding fails, so a previously
	// decoded value is not clobbered by a bad document.
	existing := monad.Some(30)
	_ = json.Unmarshal([]byte(`"thirty"`), &existing)
	fmt.Printf("  target after a failed decode: %v\n", existing)
}

// indent shifts a block of YAML over so it reads as nested output.
func indent(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		b.WriteString("    " + line + "\n")
	}
	return b.String()
}

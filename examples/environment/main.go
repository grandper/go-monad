// Command environment demonstrates reading configuration from environment
// variables as Options.
//
// A setting read from the environment has three possible states, not two: it
// can be absent, present and valid, or present and malformed. Returning
// (Option[T], error) keeps those separate — None with no error means "not
// configured", which is normal, while an error means "configured wrongly",
// which usually is not.
package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	monad "github.com/grandper/go-monad"
)

func main() {
	seed()
	defer clearEnv()

	fmt.Println("=== The Three States of a Setting ===")
	threeStates()

	fmt.Println("\n=== Supplying Defaults ===")
	defaults()

	fmt.Println("\n=== Get Versus MustGet ===")
	getVersusMust()

	fmt.Println("\n=== Lists ===")
	lists()

	fmt.Println("\n=== Parsing Rules Worth Knowing ===")
	parsingRules()

	fmt.Println("\n=== Any Type, With Your Own Parser ===")
	customParsers()

	fmt.Println("\n=== Assembling a Config ===")
	assembling()
}

// fixture is the environment this example reads. A real program would inherit
// these from its process environment instead of setting them.
func fixture() map[string]string {
	return map[string]string{
		"APP_PORT":     "8080",
		"APP_TIMEOUT":  "30s",
		"APP_DEBUG":    "true",
		"APP_HOSTS":    "alpha.example.com, beta.example.com",
		"APP_BIND_IP":  "127.0.0.1",
		"APP_HEX_PORT": "0x1F90",
		"APP_BROKEN":   "not-a-number",
	}
}

func seed() {
	for k, v := range fixture() {
		_ = os.Setenv(k, v)
	}
}

func clearEnv() {
	for k := range fixture() {
		_ = os.Unsetenv(k)
	}
}

func threeStates() {
	// Present and valid.
	port, err := monad.GetIntFromEnv("APP_PORT")
	fmt.Printf("  set and valid:   %v, err=%v\n", port, err)

	// Absent. This is not an error: most settings are optional, and the
	// absence is the signal to fall back to a default.
	missing, err := monad.GetIntFromEnv("APP_ABSENT")
	fmt.Printf("  not set:         %v, err=%v\n", missing, err)

	// Present but unparseable. The Option is None *and* there is an error,
	// so a caller that ignores the error still cannot mistake a broken value
	// for a configured one.
	broken, err := monad.GetIntFromEnv("APP_BROKEN")
	fmt.Printf("  set but invalid: %v, err=%v\n", broken, err)
}

func defaults() {
	// Because an absent variable is None, the whole OrElse family applies and
	// no separate "has this been set?" check is needed.
	port := monad.MustGetIntFromEnv("APP_ABSENT").OrElse(3000)
	timeout := monad.MustGetDurationFromEnv("APP_TIMEOUT").OrElse(15 * time.Second)

	fmt.Printf("  port=%d timeout=%v\n", port, timeout)

	// OrElseGet defers an expensive default until it is actually needed.
	host := monad.GetStringFromEnv("APP_HOSTNAME").OrElseGet(func() string {
		name, _ := os.Hostname()
		return name
	})
	fmt.Printf("  hostname fell back to the OS: %v\n", host != "")
}

func getVersusMust() {
	// The two forms differ only in how they report a malformed value. Get
	// hands you the error to deal with; MustGet panics, which is what you
	// want at startup where a bad setting should stop the program before it
	// begins serving traffic.
	if _, err := monad.GetIntFromEnv("APP_BROKEN"); err != nil {
		fmt.Println("  Get returned:", err)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("  MustGet panicked, as intended")
		}
	}()
	_ = monad.MustGetIntFromEnv("APP_BROKEN")
}

func lists() {
	// Slice variables are one comma-separated record. Items are trimmed, so
	// the readable spacing above survives.
	hosts, err := monad.GetStringSliceFromEnv("APP_HOSTS")
	fmt.Printf("  hosts: %v, err=%v\n", hosts, err)

	// An absent list is None rather than an empty slice, which preserves the
	// difference between "no list configured" and "configured as empty".
	absent, _ := monad.GetIntSliceFromEnv("APP_ABSENT_LIST")
	fmt.Printf("  absent list: %v\n", absent)
}

func parsingRules() {
	// Integers are parsed in base 0, so the notation of the value decides the
	// base — handy for ports and permission masks.
	hex, _ := monad.GetIntFromEnv("APP_HEX_PORT")
	fmt.Printf("  0x1F90 parsed as %v\n", hex)

	// Booleans accept what strconv.ParseBool accepts; yes/no are errors.
	debug, _ := monad.GetBoolFromEnv("APP_DEBUG")
	fmt.Printf("  debug: %v\n", debug)

	// Time takes the layout first, mirroring time.Parse.
	when, err := monad.GetTimeFromEnv(time.RFC3339, "APP_STARTED_AT")
	fmt.Printf("  timestamp: %v, err=%v\n", when, err)

	// Strings need no parsing, so GetStringFromEnv returns no error at all —
	// and an empty variable is Some(""), not None.
	name := monad.GetStringFromEnv("APP_NAME")
	fmt.Printf("  unset string: %v\n", name)
}

func customParsers() {
	// The typed constructors are thin wrappers over OptionFromEnv, which takes
	// any func(string) (T, error). Anything with that shape — strconv.Atoi,
	// uuid.Parse, your own validator — plugs straight in.
	parseIP := func(s string) (net.IP, error) {
		if ip := net.ParseIP(s); ip != nil {
			return ip, nil
		}
		return nil, errors.New("invalid IP address: " + s)
	}

	bind, err := monad.OptionFromEnv("APP_BIND_IP", parseIP)
	fmt.Printf("  bind: %v, err=%v\n", bind, err)

	peers, err := monad.OptionSliceFromEnv("APP_PEERS", parseIP)
	fmt.Printf("  peers: %v, err=%v\n", peers, err)
}

// Config keeps the Options rather than resolved values, so the code that uses
// it can still tell a configured setting from a defaulted one.
type Config struct {
	Port    monad.Option[int]
	Timeout monad.Option[time.Duration]
	Hosts   monad.Option[[]string]
}

func assembling() {
	// Gathering settings with the Get form lets every parse error be reported
	// together, rather than the program dying on the first one.
	var problems []error
	collect := func(name string, err error) {
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", name, err))
		}
	}

	port, err := monad.GetIntFromEnv("APP_PORT")
	collect("APP_PORT", err)
	timeout, err := monad.GetDurationFromEnv("APP_TIMEOUT")
	collect("APP_TIMEOUT", err)
	hosts, err := monad.GetStringSliceFromEnv("APP_HOSTS")
	collect("APP_HOSTS", err)

	if len(problems) > 0 {
		fmt.Println("  configuration errors:", errors.Join(problems...))
		return
	}

	cfg := Config{Port: port, Timeout: timeout, Hosts: hosts}
	fmt.Printf("  port=%d timeout=%v hosts=%d\n",
		cfg.Port.OrElse(8080),
		cfg.Timeout.OrElse(time.Minute),
		len(cfg.Hosts.OrElse(nil)))

	// Because the fields are Options, "was this set?" is still answerable
	// after the fact — something a struct of plain ints cannot tell you.
	cfg.Port.IfPresent(func(p int) { fmt.Printf("  port was explicitly set to %d\n", p) })
	cfg.Hosts.IfEmpty(func() { fmt.Println("  no hosts configured") })
}

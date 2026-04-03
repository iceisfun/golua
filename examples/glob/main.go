// Example: Glob matching from Go and Lua
//
// This example shows how to:
// - Use the glob package directly from Go
// - Use the built-in glob library from Lua (loaded by stdlib.Open)
// - Match single strings, word-by-word, and with named captures
//
// Run: go run ./examples/glob
package main

import (
	"fmt"
	"log"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/glob"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func main() {
	// --- Part 1: Using glob directly from Go ---

	fmt.Println("=== Go-Side Glob Matching ===")

	// Basic matching
	matched, _ := glob.Match("hel*", "hello")
	fmt.Printf("Match(%q, %q) = %v\n", "hel*", "hello", matched)

	// Case-insensitive
	matched, _ = glob.Match("HELLO", "hello")
	fmt.Printf("Match(%q, %q) = %v\n", "HELLO", "hello", matched)

	// Character classes
	matched, _ = glob.Match("v[12].*", "v2.0")
	fmt.Printf("Match(%q, %q) = %v\n", "v[12].*", "v2.0", matched)

	// Word matching
	matched, _ = glob.MatchWords("ORG* PEACH", "ORGANIC PEACH")
	fmt.Printf("MatchWords(%q, %q) = %v\n", "ORG* PEACH", "ORGANIC PEACH", matched)

	// Named captures
	ok, caps, _ := glob.MatchNamed("*/api/:version/:resource", "/service/api/v2/users")
	fmt.Printf("MatchNamed = %v, captures = %v\n", ok, caps)

	// Check for metacharacters
	fmt.Printf("HasPatternCharacters(%q) = %v\n", "hello", glob.HasPatternCharacters("hello"))
	fmt.Printf("HasPatternCharacters(%q) = %v\n", "h*llo", glob.HasPatternCharacters("h*llo"))

	// --- Part 2: Using glob from Lua ---
	// The glob library is loaded automatically by stdlib.Open(v).
	// No manual binding needed.

	fmt.Println("\n=== Lua-Side Glob Matching ===")

	v := vm.New()
	stdlib.Open(v)

	source := `
		-- Basic pattern matching
		print("--- Basic Matching ---")
		print("Match 'hel*' vs 'hello':", glob.match("hel*", "hello"))
		print("Match 'h?llo' vs 'hello':", glob.match("h?llo", "hello"))
		print("Match 'h[ae]llo' vs 'hello':", glob.match("h[ae]llo", "hello"))
		print("Match 'world' vs 'hello':", glob.match("world", "hello"))

		-- Case-insensitive matching
		print("\n--- Case Insensitive ---")
		print("Match 'HELLO' vs 'hello':", glob.match("HELLO", "hello"))
		print("Match 'hello' vs 'HELLO':", glob.match("hello", "HELLO"))

		-- Word-based matching
		print("\n--- Word Matching ---")
		local labels = {
			"ORGANIC PEACH",
			"ORGANIC WHITE PEACH",
			"CONVENTIONAL PEACH",
			"ORGANIC APPLE",
		}
		local pattern = "ORG* PEACH"
		print("Pattern: " .. pattern)
		for _, label in ipairs(labels) do
			local ok = glob.match_words(pattern, label)
			print("  " .. label .. " -> " .. tostring(ok))
		end

		-- Named captures
		print("\n--- Named Captures ---")
		local routes = {
			"/api/v1/users",
			"/api/v2/orders",
			"/api/v1/products",
			"/web/home",
		}
		local route_pattern = "/api/:version/:resource"
		print("Route pattern: " .. route_pattern)
		for _, route in ipairs(routes) do
			local ok, caps = glob.match_named(route_pattern, route)
			if ok then
				print("  " .. route .. " -> version=" .. caps.version .. ", resource=" .. caps.resource)
			else
				print("  " .. route .. " -> no match")
			end
		end

		-- Filtering a list using glob
		print("\n--- Filtering ---")
		local items = {"alpha-100", "alpha-200", "beta-100", "beta-300", "gamma-100"}
		print("Items matching 'alpha-*':")
		for _, item in ipairs(items) do
			if glob.match("alpha-*", item) then
				print("  " .. item)
			end
		end
		print("Items matching '*-100':")
		for _, item in ipairs(items) do
			if glob.match("*-100", item) then
				print("  " .. item)
			end
		end

		-- Pattern detection
		print("\n--- Pattern Detection ---")
		local inputs = {"hello", "h*llo", "config.json", "*.lua", "[test]"}
		for _, s in ipairs(inputs) do
			print("  " .. s .. " has patterns: " .. tostring(glob.has_pattern(s)))
		end
	`

	block, err := parser.Parse("demo", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("demo", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
}

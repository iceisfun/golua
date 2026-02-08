// Example: Glob matching from Go and Lua
//
// This example shows how to:
// - Use the glob package directly from Go
// - Expose glob matching to Lua as native functions
// - Use word-based matching for multi-word inputs
// - Use named captures to extract parts of a match
// - Load Lua scripts that drive glob matching logic
//
// Run: go run ./examples/glob
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/glob"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// FileProvider loads Lua source from a directory on disk.
type FileProvider struct {
	dir string
}

func (p *FileProvider) LoadChunk(name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
	path := filepath.Join(p.dir, name)
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot load %s: %w", name, err)
	}
	return source, "@" + name, nil
}

func (p *FileProvider) Capabilities() vm.LuaLoaderCaps {
	return vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true}
}

func main() {
	// Determine the directory containing our Lua files.
	dir := filepath.Join("examples", "glob")
	if _, err := os.Stat(filepath.Join(dir, "filter.lua")); err != nil {
		dir = "."
	}

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

	// --- Part 2: Expose glob to Lua ---

	fmt.Println("\n=== Lua-Side Glob Matching ===")

	v := vm.New()
	v.SetCodeProvider(&FileProvider{dir: dir})
	stdlib.Open(v)

	// glob.match(pattern, name) -> boolean
	globLib := vm.NewEmptyTable()

	globLib.SetString("match", vm.NewNativeFunc(func(v *vm.VM) int {
		pattern := v.Get(1).AsString()
		name := v.Get(2).AsString()
		matched, err := glob.Match(pattern, name)
		if err != nil {
			panic(fmt.Sprintf("bad pattern: %v", err))
		}
		v.Set(0, vm.NewBool(matched))
		return 1
	}))

	// glob.match_words(pattern, name) -> boolean
	globLib.SetString("match_words", vm.NewNativeFunc(func(v *vm.VM) int {
		pattern := v.Get(1).AsString()
		name := v.Get(2).AsString()
		matched, err := glob.MatchWords(pattern, name)
		if err != nil {
			panic(fmt.Sprintf("bad pattern: %v", err))
		}
		v.Set(0, vm.NewBool(matched))
		return 1
	}))

	// glob.match_named(pattern, text) -> boolean, table
	globLib.SetString("match_named", vm.NewNativeFunc(func(v *vm.VM) int {
		pattern := v.Get(1).AsString()
		text := v.Get(2).AsString()
		ok, caps, err := glob.MatchNamed(pattern, text)
		if err != nil {
			panic(fmt.Sprintf("bad pattern: %v", err))
		}
		v.Set(0, vm.NewBool(ok))
		t := vm.NewEmptyTable()
		for k, val := range caps {
			t.SetString(k, vm.NewString(val))
		}
		v.Set(1, vm.NewTable(t))
		return 2
	}))

	// glob.has_pattern(s) -> boolean
	globLib.SetString("has_pattern", vm.NewNativeFunc(func(v *vm.VM) int {
		s := v.Get(1).AsString()
		v.Set(0, vm.NewBool(glob.HasPatternCharacters(s)))
		return 1
	}))

	v.SetGlobal("glob", vm.NewTable(globLib))

	// Run the Lua demo
	source, err := os.ReadFile(filepath.Join(dir, "demo.lua"))
	if err != nil {
		log.Fatalf("Cannot read demo.lua: %v", err)
	}

	block, err := parser.Parse("demo", string(source))
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

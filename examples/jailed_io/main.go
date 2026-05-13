// Example: Sandboxed IO and OS with JailedIoProvider and DefaultOsProvider
//
// This example shows how to:
// - Use JailedIoProvider for read-only, directory-jailed file access
// - Use DefaultOsProvider for safe OS operations (clock, time, date, getenv)
// - Combine IO and OS providers in a single VM
// - Use filtered environment variable access
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/parser"
	"github.com/iceisfun/golua/v1/stdlib"
	"github.com/iceisfun/golua/v1/vm"
)

func main() {
	// Create a temporary directory with some files to read
	tmpDir, err := os.MkdirTemp("", "golua-io-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write test files
	os.WriteFile(filepath.Join(tmpDir, "greeting.txt"), []byte("Hello from a jailed file!\nThis is line 2.\nAnd line 3.\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "data.csv"), []byte("name,age\nAlice,30\nBob,25\n"), 0644)

	// Create providers
	ioProvider := vm.NewJailedIoProvider(tmpDir)
	osProvider := vm.NewDefaultOsProvider()

	fmt.Println("=== JailedIoProvider + DefaultOsProvider Example ===")
	fmt.Printf("Jail root: %s\n\n", tmpDir)

	source := `
		-- Read entire file
		print("--- Reading entire file ---")
		local f = io.open("greeting.txt", "r")
		local content = f:read("*a")
		print(content)
		f:close()

		-- Read file line by line with io.lines
		print("--- Using io.lines ---")
		for line in io.lines("data.csv") do
			print("  " .. line)
		end

		-- Check file type
		print("\n--- io.type demo ---")
		local f2 = io.open("greeting.txt", "r")
		print("Before close: io.type =", io.type(f2))
		f2:close()
		print("After close:  io.type =", io.type(f2))
		print("Non-file:     io.type =", io.type("hello"))

		-- OS functions
		print("\n--- OS functions ---")
		print("os.clock:", os.clock())
		print("os.time:", os.time())
		print("os.date:", os.date("%Y-%m-%d %H:%M:%S"))

		local t = os.date("*t")
		print("Date table:", t.year, t.month, t.day)

		local t1 = os.time()
		local t2 = t1 + 3600
		print("difftime:", os.difftime(t2, t1), "seconds")

		-- Write mode should be rejected
		print("\n--- Write rejection ---")
		local wf, werr = io.open("output.txt", "w")
		if not wf then
			print("Write correctly rejected:", werr)
		end
	`

	block, err := parser.Parse("example", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("example", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	v := vm.New()
	if err := v.SetIoProvider(ioProvider); err != nil {
		log.Fatalf("SetIoProvider: %v", err)
	}
	if err := v.SetOsProvider(osProvider); err != nil {
		log.Fatalf("SetOsProvider: %v", err)
	}
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	// Demonstrate filtered env provider
	fmt.Println("\n=== Filtered Environment Provider ===")

	filteredOsProvider := vm.NewFilteredOsProvider(func(name string) bool {
		// Only allow USER and HOME
		return name == "USER" || name == "HOME"
	})

	filteredSource := `
		local user = os.getenv("USER")
		print("USER:", user or "(not set)")

		local home = os.getenv("HOME")
		print("HOME:", home or "(not set)")

		local secret = os.getenv("SECRET_KEY")
		print("SECRET_KEY:", secret or "(filtered/not set)")
	`

	block2, _ := parser.Parse("filtered", filteredSource)
	proto2, _ := compiler.Compile("filtered", block2)

	v2 := vm.New()
	if err := v2.SetOsProvider(filteredOsProvider); err != nil {
		log.Fatalf("SetOsProvider: %v", err)
	}
	stdlib.Open(v2)
	v2.Run(proto2)

	fmt.Println("\n=== Complete ===")
}

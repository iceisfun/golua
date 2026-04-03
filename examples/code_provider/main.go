// Example: Sandboxed code loading with LuaCodeProvider
//
// This example shows how to:
// - Implement LuaCodeProvider for controlled code loading
// - Enable loadfile() and dofile() in a sandboxed way
// - Restrict which files Lua can access
// - Implement virtual file systems or database-backed code
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// InMemoryProvider loads code from an in-memory map.
// This is useful for:
// - Embedded scripts in a binary
// - Database-backed code storage
// - Virtual file systems
type InMemoryProvider struct {
	scripts map[string]string
}

func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		scripts: make(map[string]string),
	}
}

func (p *InMemoryProvider) Add(name, source string) {
	p.scripts[name] = source
}

func (p *InMemoryProvider) LoadChunk(ctx context.Context, name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
	// Log who is requesting what (useful for debugging/auditing)
	if caller != nil {
		fmt.Printf("[Provider] Loading '%s' (requested by: %s)\n", name, caller.ScriptName)
	} else {
		fmt.Printf("[Provider] Loading '%s'\n", name)
	}

	// Look up the script
	source, ok := p.scripts[name]
	if !ok {
		return nil, "", fmt.Errorf("script not found: %s", name)
	}

	// Return source, display name for stack traces, and no error
	return []byte(source), "@" + name, nil
}

func (p *InMemoryProvider) Capabilities(ctx context.Context) vm.LuaLoaderCaps {
	return vm.LuaLoaderCaps{
		AllowDofile:   true,
		AllowLoadfile: true,
	}
}

// RestrictedProvider only allows loading from an allowlist.
// This is useful for security-sensitive applications.
type RestrictedProvider struct {
	inner     vm.LuaCodeProvider
	allowlist map[string]bool
}

func NewRestrictedProvider(inner vm.LuaCodeProvider, allowed []string) *RestrictedProvider {
	allowlist := make(map[string]bool)
	for _, name := range allowed {
		allowlist[name] = true
	}
	return &RestrictedProvider{
		inner:     inner,
		allowlist: allowlist,
	}
}

func (p *RestrictedProvider) LoadChunk(ctx context.Context, name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
	// Check allowlist
	if !p.allowlist[name] {
		return nil, "", fmt.Errorf("access denied: %s", name)
	}
	return p.inner.LoadChunk(ctx, name, caller)
}

func (p *RestrictedProvider) Capabilities(ctx context.Context) vm.LuaLoaderCaps {
	return p.inner.Capabilities(ctx)
}

func main() {
	// Create an in-memory provider with some scripts
	provider := NewInMemoryProvider()

	provider.Add("utils.lua", `
		local M = {}
		function M.double(x)
			return x * 2
		end
		function M.greet(name)
			return "Hello, " .. name
		end
		return M
	`)

	provider.Add("config.lua", `
		return {
			debug = true,
			version = "1.0.0",
			max_connections = 100
		}
	`)

	provider.Add("main.lua", `
		-- Load modules using dofile
		local utils = dofile("utils.lua")
		local config = dofile("config.lua")

		print("Config version:", config.version)
		print("Debug mode:", config.debug)
		print("Double 21:", utils.double(21))
		print("Greeting:", utils.greet("World"))

		-- Reload utils to show caching doesn't happen
		local utils2 = dofile("utils.lua")
		print("Loaded again:", utils2.double(10))
	`)

	// Create VM with provider
	v := vm.New()
	if err := v.SetCodeProvider(provider); err != nil {
		log.Fatalf("SetCodeProvider: %v", err)
	}
	stdlib.Open(v)

	fmt.Println("=== Running with InMemoryProvider ===")
	fmt.Println()

	// Parse and run main.lua content directly
	// (In real usage, you might also load main.lua via the provider)
	source := provider.scripts["main.lua"]
	block, err := parser.Parse("main.lua", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("main.lua", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	// Now demonstrate restricted provider
	fmt.Println()
	fmt.Println("=== Running with RestrictedProvider ===")
	fmt.Println()

	// Only allow utils.lua, not config.lua
	restricted := NewRestrictedProvider(provider, []string{"utils.lua"})

	v2 := vm.New()
	if err := v2.SetCodeProvider(restricted); err != nil {
		log.Fatalf("SetCodeProvider: %v", err)
	}
	stdlib.Open(v2)

	restrictedSource := `
		-- This will work
		local utils = dofile("utils.lua")
		print("Utils loaded:", utils.double(5))

		-- This will fail (not in allowlist)
		local ok, err = pcall(function()
			dofile("config.lua")
		end)
		if not ok then
			print("Expected error:", err)
		end
	`

	block2, _ := parser.Parse("restricted.lua", restrictedSource)
	proto2, _ := compiler.Compile("restricted.lua", block2)
	v2.Run(proto2)

	fmt.Println("\n=== Complete ===")
}

// Helper to check if a path is within allowed directory
func isPathAllowed(path string, allowedDirs []string) bool {
	for _, dir := range allowedDirs {
		if strings.HasPrefix(path, dir) {
			return true
		}
	}
	return false
}

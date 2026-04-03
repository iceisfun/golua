// Example: Exposing a Go-backed object to Lua
//
// This example demonstrates the canonical pattern for giving Lua
// access to Go-owned state through an explicit adapter layer,
// paired with a Lua companion module that adds behavior.
//
// Run: go run ./examples/expose_object
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// FileProvider loads Lua source from a directory on disk.
type FileProvider struct {
	dir string
}

func (p *FileProvider) LoadChunk(ctx context.Context, name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
	path := filepath.Join(p.dir, name)
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot load %s: %w", name, err)
	}
	return source, "@" + name, nil
}

func (p *FileProvider) Capabilities(ctx context.Context) vm.LuaLoaderCaps {
	return vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true}
}

func main() {
	// Determine the directory containing enemy.lua.
	dir := filepath.Join("examples", "expose_object")
	if _, err := os.Stat(filepath.Join(dir, "enemy.lua")); err != nil {
		// Fallback: running from within the example directory.
		dir = "."
	}

	// --- Step 1: Create Go objects ---

	goblin := NewEnemy("Goblin", 50, 10, 5)
	dragon := NewEnemy("Dragon", 200, 0, 0)

	// --- Step 2: Convert to Lua tables via adapter ---

	goblinLua := EnemyToLua(goblin)
	dragonLua := EnemyToLua(dragon)

	// --- Step 3: Create VM and inject ---

	v := vm.New()
	if err := v.SetCodeProvider(&FileProvider{dir: dir}); err != nil {
		log.Fatalf("SetCodeProvider: %v", err)
	}
	stdlib.Open(v)

	v.SetGlobal("goblin_core", vm.NewTable(goblinLua))
	v.SetGlobal("dragon_core", vm.NewTable(dragonLua))

	// --- Step 4: Load companion module and exercise behavior ---

	source := `
		local Enemy = dofile("enemy.lua")

		-- Wrap the Go-provided tables
		local goblin = Enemy.wrap(goblin_core)
		local dragon = Enemy.wrap(dragon_core)

		print("=== Initial State ===")
		print(goblin:status())
		print(dragon:status())

		print("\n=== Combat Round 1 ===")
		-- Dragon attacks goblin for 35 damage
		goblin_core.take_damage(35)
		print("Dragon attacks Goblin for 35!")
		print(goblin:status())
		print("Should flee?", goblin:should_flee())
		print("Should heal?", goblin:should_heal())

		print("\n=== Goblin's Turn ===")
		-- Goblin AI decides what to do (target is at origin)
		local action = goblin:take_turn(0, 0)
		print("Action:", action)
		print(goblin:status())

		print("\n=== Combat Round 2 ===")
		-- Another hit finishes the goblin
		goblin_core.take_damage(30)
		print("Dragon attacks Goblin for 30!")
		print(goblin:status())
		print("Alive?", goblin:is_alive())

		print("\n=== Dragon Takes Damage ===")
		dragon_core.take_damage(180)
		print(dragon:status())
		print("Should flee?", dragon:should_flee())

		local action2 = dragon:take_turn(10, 10)
		print("Action:", action2)
		print(dragon:status())

		print("\n=== Done ===")
	`

	block, err := parser.Parse("main", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("main", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	// --- Step 5: Verify Go-side state reflects Lua mutations ---

	fmt.Println("\n=== Go-Side Verification ===")
	fmt.Printf("Goblin health (Go): %d (alive: %v)\n", goblin.Health(), goblin.IsAlive())
	fmt.Printf("Dragon health (Go): %d (alive: %v)\n", dragon.Health(), dragon.IsAlive())
	x, y := dragon.Position()
	fmt.Printf("Dragon position (Go): (%.1f, %.1f)\n", x, y)
}

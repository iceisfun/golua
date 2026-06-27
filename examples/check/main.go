// Example: check Lua source and print diagnostics as JSON.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/iceisfun/golua/ast"
	"github.com/iceisfun/golua/check"
)

func main() {
	// This source has several independent problems. The reference parser stops
	// at the first; check.Check recovers and reports each one, and tags every
	// diagnostic with a stable Code (e.g. "unexpected-symbol").
	source := `
local x = 42
print(x ==)
local y = 'unterminated
if true then
`

	result := check.Check("editor.lua", source)

	fmt.Printf("Parsed %d statement(s)\n", len(result.Block.Stmts))
	fmt.Printf("Has errors: %v\n", result.HasErrors())
	fmt.Printf("Diagnostics found: %d\n\n", len(result.Diagnostics))

	if len(result.Diagnostics) > 0 {
		fmt.Println("Diagnostics (JSON):")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result.Diagnostics)
	}

	fmt.Println("\nPartial AST:")
	fmt.Print(ast.DumpString(result.Block))
}

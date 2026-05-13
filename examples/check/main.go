// Example: check Lua source and print diagnostics as JSON.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/iceisfun/golua/v1/ast"
	"github.com/iceisfun/golua/v1/check"
)

func main() {
	source := `
local x = 42
print(x)
if true then
`

	result := check.Check("editor.lua", source)

	fmt.Printf("Parsed %d statement(s)\n", len(result.Block.Stmts))
	fmt.Printf("Has errors: %v\n\n", result.HasErrors())

	if len(result.Diagnostics) > 0 {
		fmt.Println("Diagnostics (JSON):")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result.Diagnostics)
	}

	fmt.Println("\nPartial AST:")
	fmt.Print(ast.DumpString(result.Block))
}

// Example: Source-level directive parsing.
//
// Shows how to extract @-prefixed metadata from a Lua source file's
// header without compiling or running the code. Directives are a
// golua-specific extension and are NOT part of the Lua language —
// see the README for the full non-standard-Lua notes.
package main

import (
	"fmt"
	"log"

	"github.com/iceisfun/golua/directives"
)

const source = `-- @tick 30s
-- @scope alias_expander
-- @disabled
-- @import shared/util
-- @import shared/log
-- this comment is ignored, but does not end the header

local function run() return 42 end
return run()
`

func main() {
	f, err := directives.Parse(source)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}

	if v, ok := f.Get("tick"); ok {
		fmt.Printf("tick = %q\n", v)
	}
	if v, ok := f.Get("scope"); ok {
		fmt.Printf("scope = %q\n", v)
	}
	if f.Has("disabled") {
		fmt.Println("script is disabled")
	}
	for _, p := range f.Lookup("import") {
		fmt.Printf("import: %s\n", p)
	}
}

package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// runForError compiles and runs source with stdlib loaded and returns the
// runtime error string (empty if none).
func runForError(t *testing.T, name, source string) string {
	t.Helper()
	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// "@" prefix marks a file-based chunk so errors render as "name:line:".
	proto, err := compiler.Compile("@"+name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	stdlib.Open(v)
	if _, err := v.Run(proto); err != nil {
		return err.Error()
	}
	return ""
}

// TestCloseErrorLineAttribution verifies that when a to-be-closed variable's
// __close metamethod errors at block exit, golua attributes the error to the
// same source line as reference Lua 5.5. For block scopes (do/if/while/for) Lua
// uses ls->lastline at leaveblock — the last statement inside the block. For
// repeat...until the close is attributed to the until-condition line; for a
// function body it is the closing 'end'. Each source clears mt.__close so the
// close attempt raises "attempt to call a nil value (metamethod 'close')".
func TestCloseErrorLineAttribution(t *testing.T) {
	const closeMsg = "metamethod 'close'"
	cases := []struct {
		name    string
		src     string
		wantLoc string
	}{
		{
			name:    "do block",
			src:     "local mt={__close=function() end}\ndo\n  local x <close> = setmetatable({},mt)\n  mt.__close=nil\nend\n",
			wantLoc: ":4:",
		},
		{
			name:    "if block",
			src:     "local mt={__close=function() end}\nif true then\n  local x <close> = setmetatable({},mt)\n  mt.__close=nil\nend\n",
			wantLoc: ":4:",
		},
		{
			name:    "while block",
			src:     "local mt={__close=function() end}\nlocal once=true\nwhile once do\n  once=false\n  local x <close> = setmetatable({},mt)\n  mt.__close=nil\nend\n",
			wantLoc: ":6:",
		},
		{
			name:    "numeric for",
			src:     "local mt={__close=function() end}\nfor i=1,1 do\n  local x <close> = setmetatable({},mt)\n  mt.__close=nil\nend\n",
			wantLoc: ":4:",
		},
		{
			name:    "generic for",
			src:     "local mt={__close=function() end}\nfor k in pairs({a=1}) do\n  local x <close> = setmetatable({},mt)\n  mt.__close=nil\nend\n",
			wantLoc: ":4:",
		},
		{
			name:    "repeat until true",
			src:     "local mt={__close=function() end}\nrepeat\n  local x <close> = setmetatable({},mt)\n  mt.__close=nil\nuntil true\n",
			wantLoc: ":5:",
		},
		{
			name:    "repeat general",
			src:     "local mt={__close=function() end}\nlocal n=0\nrepeat\n  local x <close> = setmetatable({},mt)\n  mt.__close=nil\n  n=n+1\nuntil n>0\n",
			wantLoc: ":7:",
		},
		{
			name:    "nested do in function body",
			src:     "local mt={__close=function() end}\nlocal function f()\n  do\n    local x <close> = setmetatable({},mt)\n    mt.__close=nil\n  end\nend\nf()\n",
			wantLoc: ":5:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runForError(t, "close.lua", tc.src)
			if got == "" {
				t.Fatalf("expected a __close error, got none")
			}
			if !strings.Contains(got, closeMsg) {
				t.Fatalf("expected %q in error, got: %s", closeMsg, got)
			}
			if !strings.Contains(got, "close.lua"+tc.wantLoc) {
				t.Errorf("expected error at %q, got: %s", "close.lua"+tc.wantLoc, got)
			}
		})
	}
}

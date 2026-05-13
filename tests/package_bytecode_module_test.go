package tests

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func TestRequireLoadsBytecodeModule(t *testing.T) {
	hexChunk, err := runGoLua(t, `
local dumped = string.dump(function() return 42 end, true)
local hex = {}
for i = 1, #dumped do
    hex[#hex + 1] = string.format("%02x", dumped:byte(i))
end
print(table.concat(hex))
`)
	if err != nil {
		t.Fatalf("failed to create bytecode chunk: %v", err)
	}

	chunk, err := hex.DecodeString(strings.TrimSpace(hexChunk))
	if err != nil {
		t.Fatalf("failed to decode chunk: %v", err)
	}

	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "m.lua")
	if err := os.WriteFile(modulePath, chunk, 0o644); err != nil {
		t.Fatalf("failed to write module: %v", err)
	}

	proto, err := compileLua("=require-bytecode", `
package.path = "?.lua"
local value, path = require("m")
print(value)
print(path:match("m%.lua$") ~= nil)
`)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	v := vm.New(vm.WithCaptureOutput(true))
	v.SetCodeProvider(vm.NewDirCodeProvider(tmpDir, vm.LuaLoaderCaps{
		AllowLoadfile: true,
		AllowDofile:   true,
	}))
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("require should load bytecode module: %v", err)
	}

	out := strings.Join(v.OutputLines(), "\n")
	if out != "42\ntrue" {
		t.Fatalf("unexpected output: %q", out)
	}
}

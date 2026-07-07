package golua_test

import (
	"sync"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// two INDEPENDENT VMs (no shared Lua state) must run race-free.
// VM-A toggles a table's weak mode via setmetatable(__mode) (writing the table's
// extra.weak pointer); VM-B runs collectgarbage(), whose process-global weak
// sweep reads that pointer. Run under -race to catch the regression.
func TestWeakModeToggleVsCollect(t *testing.T) {
	mustProto := func(name, src string) *compiler.Proto {
		block, err := parser.Parse("="+name, src)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		p, err := compiler.Compile("="+name, block)
		if err != nil {
			t.Fatalf("compile %s: %v", name, err)
		}
		return p
	}
	const n = 1500
	toggle := mustProto("toggle", `
local weak, strong = {__mode = "v"}, {}
local t = setmetatable({}, weak)
for r = 1, `+itoaReap(n)+` do
  setmetatable(t, (r % 2 == 0) and weak or strong)
  for i = 1, 50 do t[i] = {r + i} end
end
return 0`)
	collect := mustProto("collect", `for r = 1, `+itoaReap(n)+` do collectgarbage("collect") end return 0`)

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			v := vm.New()
			stdlib.Open(v)
			if _, err := v.Run(toggle); err != nil {
				t.Errorf("toggle run: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			v := vm.New()
			stdlib.Open(v)
			if _, err := v.Run(collect); err != nil {
				t.Errorf("collect run: %v", err)
			}
		}()
	}
	wg.Wait()
}

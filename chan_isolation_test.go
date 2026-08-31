package golua_test

// Ownership rules for the chan module's Go-side state. A single Go process
// routinely hosts many independent VMs, so nothing the chan module keeps may be
// shared between two VMs, and nothing it keeps may outlive the Lua values that
// reach it. These tests cover both: the handle metatable is per-VM (as it is in
// reference Lua, where a library's handle metatable lives in that lua_State's
// registry), and a channel whose handle Lua has dropped is ordinary garbage.

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// newChanVM builds an independent VM with the full stdlib and a chan provider.
func newChanVM(t *testing.T) *vm.VM {
	t.Helper()
	v := vm.New()
	stdlib.Open(v)
	stdlib.ProvideChan(v)
	return v
}

// compileChunk compiles source for reuse across several VMs.
func compileChunk(t *testing.T, name, source string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return proto
}

// TestChan_HandleMetatableIsPerVM checks that a script mutating the channel
// handle metatable changes only its own VM: a second VM must see a pristine
// metatable, with no injected fields and no injected __index.
func TestChan_HandleMetatableIsPerVM(t *testing.T) {
	a := newChanVM(t)
	if _, err := a.Run(compileChunk(t, "poke", `
		local mt = getmetatable(chan.make(1))
		mt.injected = "from the first VM"
		mt.__name = "not a channel"
		mt.__index = function(_, k) return "injected:" .. tostring(k) end
	`)); err != nil {
		t.Fatalf("first VM: %v", err)
	}

	b := newChanVM(t)
	if _, err := b.Run(compileChunk(t, "observe", `
		local c = chan.make(1)
		local mt = getmetatable(c)
		assert(rawget(mt, "injected") == nil,
			"handle metatable carries a field set by another VM")
		assert(rawget(mt, "__name") == "channel",
			"handle metatable __name was changed by another VM: " .. tostring(rawget(mt, "__name")))
		assert(c.no_such_field == nil,
			"missing field resolved through another VM's __index: " .. tostring(c.no_such_field))
	`)); err != nil {
		t.Fatalf("second VM: %v", err)
	}
}

// TestChan_HandleMetatableSharedWithinVM checks the other half of the scoping
// rule: every handle a VM tree makes, including one made inside a coroutine,
// shares that VM's one metatable.
func TestChan_HandleMetatableSharedWithinVM(t *testing.T) {
	source := `
		local a = chan.make(1)
		local b = chan.make(1)
		assert(getmetatable(a) == getmetatable(b),
			"two handles from one VM have different metatables")
		local co = coroutine.create(function()
			return getmetatable(chan.make(1))
		end)
		local ok, mt = coroutine.resume(co)
		assert(ok, mt)
		assert(mt == getmetatable(a),
			"a handle made in a coroutine has a different metatable")
	`
	if _, err := runLuaWithChan(t, source, "chan_meta_shared", nil, vm.Limits{}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestChan_ConcurrentHandleMetatableWrites runs several independent VMs at once,
// each hammering its own handle metatable. Under -race this catches the module
// handing every VM the same *vm.Table; without it, concurrent writes to one
// shared table are a fatal Go map error rather than a recoverable one.
func TestChan_ConcurrentHandleMetatableWrites(t *testing.T) {
	proto := compileChunk(t, "hammer", `
		local mt = getmetatable(chan.make(1))
		for i = 1, 2000 do mt["k" .. i] = i end
		local n = 0
		for _ in pairs(mt) do n = n + 1 end
		return n
	`)

	const goroutines = 4
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := newChanVM(t).Run(proto)
			if err != nil {
				errs <- err
				return
			}
			// 2000 injected keys plus __name; a metatable shared with another
			// VM would show that VM's keys too.
			if got, _ := res[0].ToInt(); got != 2001 {
				errs <- fmt.Errorf("metatable has %d entries, want 2001", got)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestChan_SelectRejectsPlainTable checks that chan.select resolves the channel
// through the handle it was given rather than through a process- or VM-wide
// side table: a bare table naming a channel id is not a handle.
func TestChan_SelectRejectsPlainTable(t *testing.T) {
	provider := vm.NewDefaultChanProvider()
	ch := provider.NewChannel(nil, 4)
	ch.TrySend(vm.NewString("payload"))

	v := vm.New()
	v.SetChanProvider(provider)
	stdlib.Open(v)
	// The host wraps the channel for its own use but never exposes the handle.
	_ = stdlib.WrapChannel(v, ch)

	if _, err := v.Run(compileChunk(t, "plain", `
		for id = 1, 8 do
			local ok, err = pcall(chan.select, {__chan_id = id}, 0.01)
			assert(not ok, "chan.select accepted a plain table for id " .. id)
			assert(type(err) == "string" and err:find("channel expected", 1, true),
				"unexpected error for id " .. id .. ": " .. tostring(err))
		end
	`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The channel still holds its value: nothing consumed it.
	if _, _, received := ch.TryRecv(); !received {
		t.Fatal("channel was drained by the rejected selects")
	}
}

// TestChan_DroppedChannelsAreReleased checks that the module keeps no table of
// every channel a VM ever made: once Lua drops a handle, the channel and
// anything still buffered in it are collectable like any other garbage.
func TestChan_DroppedChannelsAreReleased(t *testing.T) {
	const channels = 20000
	proto := compileChunk(t, "churn", fmt.Sprintf(`
		for i = 1, %d do
			local c = chan.make(4)
			c:try_send(("x"):rep(2000))
			c:try_send(("y"):rep(2000))
		end
		return true
	`, channels))

	v := newChanVM(t)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(v)

	retained := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	// Each channel carries two 2 KiB payloads, so retaining even a tenth of
	// them lands far above this bound while releasing them all stays near zero.
	const limit = 16 << 20
	if retained > limit {
		t.Fatalf("%d dropped channels retained %d KiB, want under %d KiB",
			channels, retained/1024, limit/1024)
	}
}

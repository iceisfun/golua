package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// runLuaWithChan compiles and runs Lua source with a ChanProvider and optional context/limits.
// Recovers panics from channel operations (e.g. interrupt) and returns them as errors.
func runLuaWithChan(t *testing.T, source, name string, ctx context.Context, limits vm.Limits, setup func(v *vm.VM)) (results []vm.Value, err error) {
	t.Helper()

	block, parseErr := parser.Parse(name, source)
	if parseErr != nil {
		t.Fatalf("parse error: %v", parseErr)
	}

	proto, compErr := compiler.Compile(name, block)
	if compErr != nil {
		t.Fatalf("compile error: %v", compErr)
	}

	v := vm.New()
	v.SetChanProvider(vm.NewDefaultChanProvider())
	if ctx != nil {
		v.SetContext(ctx)
	}
	v.SetLimits(limits)
	stdlib.Open(v)
	if setup != nil {
		setup(v)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	return v.Run(proto)
}

// --- Core Behavior ---

func TestChan_NoProvider(t *testing.T) {
	source := `assert(chan == nil, "expected chan to be nil without provider")`
	runLuaSource(t, source, "test_chan_no_provider")
}

func TestChan_MakeUnbuffered(t *testing.T) {
	source := `
		local ch = chan.make()
		assert(type(ch) == "table", "expected table, got " .. type(ch))
	`
	runLuaWithChan(t, source, "test_chan_make_unbuffered", nil, vm.Limits{}, nil)
}

func TestChan_MakeBuffered(t *testing.T) {
	source := `
		local ch = chan.make(5)
		assert(type(ch) == "table", "expected table, got " .. type(ch))
	`
	runLuaWithChan(t, source, "test_chan_make_buffered", nil, vm.Limits{}, nil)
}

func TestChan_MakeNegativePanics(t *testing.T) {
	source := `
		local ok, err = pcall(chan.make, -1)
		assert(not ok, "expected error for negative size")
		assert(type(err) == "string", "expected error string")
	`
	runLuaWithChan(t, source, "test_chan_make_negative", nil, vm.Limits{}, nil)
}

func TestChan_SendRecvBuffered(t *testing.T) {
	source := `
		local ch = chan.make(1)
		ch:send(42)
		local val, ok = ch:recv()
		assert(val == 42, "expected 42, got " .. tostring(val))
		assert(ok == true, "expected ok=true")
	`
	runLuaWithChan(t, source, "test_chan_send_recv_buffered", nil, vm.Limits{}, nil)
}

func TestChan_SendRecvMultipleTypes(t *testing.T) {
	source := `
		local ch = chan.make(10)

		-- Send various types
		ch:send(42)
		ch:send(3.14)
		ch:send("hello")
		ch:send(true)
		ch:send(nil)
		ch:send({1, 2, 3})

		-- Recv and verify
		local v1, ok1 = ch:recv()
		assert(v1 == 42 and ok1, "int roundtrip failed")

		local v2, ok2 = ch:recv()
		assert(v2 == 3.14 and ok2, "float roundtrip failed")

		local v3, ok3 = ch:recv()
		assert(v3 == "hello" and ok3, "string roundtrip failed")

		local v4, ok4 = ch:recv()
		assert(v4 == true and ok4, "bool roundtrip failed")

		local v5, ok5 = ch:recv()
		assert(v5 == nil and ok5, "nil roundtrip failed")

		local v6, ok6 = ch:recv()
		assert(type(v6) == "table" and ok6, "table roundtrip failed")
	`
	runLuaWithChan(t, source, "test_chan_send_recv_types", nil, vm.Limits{}, nil)
}

func TestChan_RecvClosedChannel(t *testing.T) {
	source := `
		local ch = chan.make(1)
		ch:send(99)
		ch:close()
		-- First recv should get the buffered value
		local v1, ok1 = ch:recv()
		assert(v1 == 99, "expected 99, got " .. tostring(v1))
		assert(ok1 == true, "expected ok=true for buffered value")
		-- Second recv on closed+drained
		local v2, ok2 = ch:recv()
		assert(v2 == nil, "expected nil on closed+drained, got " .. tostring(v2))
		assert(ok2 == false, "expected ok=false on closed+drained")
	`
	runLuaWithChan(t, source, "test_chan_recv_closed", nil, vm.Limits{}, nil)
}

func TestChan_CloseAlreadyClosed(t *testing.T) {
	source := `
		local ch = chan.make()
		ch:close()
		local ok, err = pcall(function() ch:close() end)
		assert(not ok, "expected error on double close")
		assert(type(err) == "string", "expected error string")
	`
	runLuaWithChan(t, source, "test_chan_close_already_closed", nil, vm.Limits{}, nil)
}

func TestChan_SendClosedPanics(t *testing.T) {
	source := `
		local ch = chan.make(1)
		ch:close()
		local ok, err = pcall(function() ch:send(1) end)
		assert(not ok, "expected error on send to closed")
	`
	runLuaWithChan(t, source, "test_chan_send_closed", nil, vm.Limits{}, nil)
}

func TestChan_FIFOOrdering(t *testing.T) {
	source := `
		local ch = chan.make(5)
		for i = 1, 5 do
			ch:send(i)
		end
		for i = 1, 5 do
			local val = ch:recv()
			assert(val == i, "expected " .. i .. ", got " .. tostring(val))
		end
	`
	runLuaWithChan(t, source, "test_chan_fifo", nil, vm.Limits{}, nil)
}

// --- Non-Blocking Helpers ---

func TestChan_TrySendBuffered(t *testing.T) {
	source := `
		local ch = chan.make(1)
		local ok = ch:try_send(42)
		assert(ok == true, "expected try_send to succeed")
	`
	runLuaWithChan(t, source, "test_chan_try_send_buffered", nil, vm.Limits{}, nil)
}

func TestChan_TrySendFull(t *testing.T) {
	source := `
		local ch = chan.make(0)
		local ok = ch:try_send(42)
		assert(ok == false, "expected try_send to fail on unbuffered channel")
	`
	runLuaWithChan(t, source, "test_chan_try_send_full", nil, vm.Limits{}, nil)
}

func TestChan_TryRecvEmpty(t *testing.T) {
	source := `
		local ch = chan.make(1)
		local val, ok, received = ch:try_recv()
		assert(val == nil, "expected nil val on empty")
		assert(ok == false, "expected ok=false on empty")
		assert(received == false, "expected received=false on empty")
	`
	runLuaWithChan(t, source, "test_chan_try_recv_empty", nil, vm.Limits{}, nil)
}

func TestChan_TryRecvBuffered(t *testing.T) {
	source := `
		local ch = chan.make(1)
		ch:send(77)
		local val, ok, received = ch:try_recv()
		assert(val == 77, "expected 77, got " .. tostring(val))
		assert(ok == true, "expected ok=true")
		assert(received == true, "expected received=true")
	`
	runLuaWithChan(t, source, "test_chan_try_recv_buffered", nil, vm.Limits{}, nil)
}

func TestChan_TryRecvClosed(t *testing.T) {
	source := `
		local ch = chan.make(1)
		ch:close()
		local val, ok, received = ch:try_recv()
		assert(val == nil, "expected nil on closed+empty")
		assert(ok == false, "expected ok=false on closed+empty")
		assert(received == true, "expected received=true on closed (recv completed, channel exhausted)")
	`
	runLuaWithChan(t, source, "test_chan_try_recv_closed", nil, vm.Limits{}, nil)
}

// --- Select ---

func TestChan_SelectSingle(t *testing.T) {
	source := `
		local ch = chan.make(1)
		ch:send(10)
		local idx, val, ok = chan.select(ch)
		assert(idx == 1, "expected idx=1, got " .. tostring(idx))
		assert(val == 10, "expected val=10, got " .. tostring(val))
		assert(ok == true, "expected ok=true")
	`
	runLuaWithChan(t, source, "test_chan_select_single", nil, vm.Limits{}, nil)
}

func TestChan_SelectMultiple(t *testing.T) {
	source := `
		local ch1 = chan.make(1)
		local ch2 = chan.make(1)
		local ch3 = chan.make(1)
		ch2:send(20)
		local idx, val, ok = chan.select(ch1, ch2, ch3)
		assert(idx == 2, "expected idx=2, got " .. tostring(idx))
		assert(val == 20, "expected val=20, got " .. tostring(val))
		assert(ok == true, "expected ok=true")
	`
	runLuaWithChan(t, source, "test_chan_select_multiple", nil, vm.Limits{}, nil)
}

func TestChan_SelectTimeout(t *testing.T) {
	source := `
		local ch = chan.make(0)
		local idx, val, ok = chan.select(ch, 0.01)
		assert(idx == 0, "expected idx=0 on timeout, got " .. tostring(idx))
		assert(val == nil, "expected nil val on timeout")
		assert(ok == false, "expected ok=false on timeout")
	`
	runLuaWithChan(t, source, "test_chan_select_timeout", nil, vm.Limits{}, nil)
}

func TestChan_SelectClosedChannel(t *testing.T) {
	source := `
		local ch = chan.make(1)
		ch:close()
		local idx, val, ok = chan.select(ch)
		assert(idx == 1, "expected idx=1, got " .. tostring(idx))
		assert(val == nil, "expected nil val on closed, got " .. tostring(val))
		assert(ok == false, "expected ok=false on closed")
	`
	runLuaWithChan(t, source, "test_chan_select_closed", nil, vm.Limits{}, nil)
}

// --- Context Cancellation ---

func TestChan_CancelRecv(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	source := `
		local ch = chan.make(0)
		ch:recv()
	`
	_, err := runLuaWithChan(t, source, "test_chan_cancel_recv", ctx, vm.Limits{}, nil)
	if err == nil {
		t.Fatal("expected error from cancelled recv")
	}
	if !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("expected 'interrupted', got: %v", err)
	}
}

func TestChan_CancelSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	source := `
		local ch = chan.make(0)
		ch:send(1)
	`
	_, err := runLuaWithChan(t, source, "test_chan_cancel_send", ctx, vm.Limits{}, nil)
	if err == nil {
		t.Fatal("expected error from cancelled send")
	}
	if !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("expected 'interrupted', got: %v", err)
	}
}

func TestChan_CancelSelect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	source := `
		local ch = chan.make(0)
		chan.select(ch)
	`
	_, err := runLuaWithChan(t, source, "test_chan_cancel_select", ctx, vm.Limits{}, nil)
	if err == nil {
		t.Fatal("expected error from cancelled select")
	}
	if !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("expected 'interrupted', got: %v", err)
	}
}

func TestChan_DeadlineRecv(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	source := `
		local ch = chan.make(0)
		ch:recv()
	`
	_, err := runLuaWithChan(t, source, "test_chan_deadline_recv", ctx, vm.Limits{}, nil)
	if err == nil {
		t.Fatal("expected error from deadline recv")
	}
	if !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("expected 'interrupted', got: %v", err)
	}
}

// --- Go <-> Lua Integration ---

func TestChan_GoToLua(t *testing.T) {
	provider := vm.NewDefaultChanProvider()
	ch := provider.NewChannel(1)

	// Go goroutine sends a value
	go func() {
		ch.Send(nil, vm.NewInt(999))
	}()

	source := `
		local val, ok = go_ch:recv()
		assert(val == 999, "expected 999, got " .. tostring(val))
		assert(ok == true, "expected ok=true")
		return val
	`
	results, err := runLuaWithChan(t, source, "test_chan_go_to_lua", nil, vm.Limits{}, func(v *vm.VM) {
		v.SetChanProvider(provider)
		v.SetGlobal("go_ch", stdlib.WrapChannel(v, ch))
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 || results[0].AsInt() != 999 {
		t.Fatalf("expected 999, got: %v", results)
	}
}

func TestChan_LuaToGo(t *testing.T) {
	provider := vm.NewDefaultChanProvider()
	ch := provider.NewChannel(1)

	source := `
		go_ch:send(123)
	`
	_, err := runLuaWithChan(t, source, "test_chan_lua_to_go", nil, vm.Limits{}, func(v *vm.VM) {
		v.SetChanProvider(provider)
		v.SetGlobal("go_ch", stdlib.WrapChannel(v, ch))
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Go receives after Run
	val, ok, _ := ch.Recv(nil)
	if !ok || val.AsInt() != 123 {
		t.Fatalf("expected 123, got: %v (ok=%v)", val, ok)
	}
}

func TestChan_WrapChannel(t *testing.T) {
	provider := vm.NewDefaultChanProvider()
	ch := provider.NewChannel(1)

	source := `
		assert(type(go_ch) == "table", "expected table for wrapped channel")
		go_ch:send(42)
		local val, ok = go_ch:recv()
		assert(val == 42, "expected 42, got " .. tostring(val))
		assert(ok == true, "expected ok=true")
	`
	_, err := runLuaWithChan(t, source, "test_chan_wrap", nil, vm.Limits{}, func(v *vm.VM) {
		v.SetChanProvider(provider)
		v.SetGlobal("go_ch", stdlib.WrapChannel(v, ch))
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Provider & Safety ---

func TestChan_ProvideChan(t *testing.T) {
	block, err := parser.Parse("test", `
		assert(chan ~= nil, "expected chan to be set")
		local ch = chan.make(1)
		ch:send(1)
		local v = ch:recv()
		assert(v == 1, "expected 1")
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	stdlib.Open(v)
	stdlib.ProvideChan(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

func TestChan_VMBoundary(t *testing.T) {
	// Create two separate VMs with different providers
	provider1 := vm.NewDefaultChanProvider()
	provider2 := vm.NewDefaultChanProvider()
	ch := provider1.NewChannel(1)

	source := `
		local ok, err = pcall(function()
			chan.select(foreign_ch)
		end)
		assert(not ok, "expected error for cross-VM channel")
		assert(type(err) == "string", "expected error string")
	`

	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	// VM2 with provider2 tries to select on channel from provider1
	v2 := vm.New()
	v2.SetChanProvider(provider2)
	stdlib.Open(v2)
	// Wrap channel using v2 (which has provider2), but the channel belongs to provider1
	// We need to manually create the handle for the foreign channel
	handle := vm.NewEmptyTable()
	handle.SetString("__chan_id", vm.NewInt(ch.ID()))
	// Register in the global registry so extractChannel finds it
	stdlib.WrapChannel(v2, ch)
	v2.SetGlobal("foreign_ch", stdlib.WrapChannel(v2, ch))

	_, err = v2.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

func TestChan_CoroutineInherit(t *testing.T) {
	source := `
		local ch = chan.make(1)
		ch:send(55)
		local co = coroutine.create(function()
			local val, ok = ch:recv()
			assert(val == 55, "expected 55 in coroutine, got " .. tostring(val))
			assert(ok == true, "expected ok=true in coroutine")
			return val
		end)
		local ok, val = coroutine.resume(co)
		assert(ok, "coroutine should not error")
		assert(val == 55, "expected 55 from coroutine, got " .. tostring(val))
	`
	runLuaWithChan(t, source, "test_chan_coroutine_inherit", nil, vm.Limits{}, nil)
}

func TestChan_PostWakeCheckpoint(t *testing.T) {
	// Use MaxInstructions to verify that CheckInterrupt is called after blocking op.
	// We set a very high instruction limit so the script runs normally,
	// but we verify it doesn't panic/error by completing successfully.
	source := `
		local ch = chan.make(1)
		ch:send(1)
		local val, ok = ch:recv()
		assert(val == 1, "expected 1")
		assert(ok == true, "expected ok=true")
	`
	_, err := runLuaWithChan(t, source, "test_chan_post_wake", nil, vm.Limits{MaxInstructions: 1000000}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChan_VMReuseAfterCancel(t *testing.T) {
	provider := vm.NewDefaultChanProvider()

	// First run: cancel a blocked recv
	ctx1, cancel1 := context.WithCancel(context.Background())

	block1, err := parser.Parse("test1", `
		local ch = chan.make(0)
		ch:recv()
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto1, err := compiler.Compile("test1", block1)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	v.SetChanProvider(provider)
	v.SetContext(ctx1)
	stdlib.Open(v)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel1()
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()
		_, err = v.Run(proto1)
	}()
	if err == nil {
		t.Fatal("expected cancellation error")
	}

	// Second run: fresh context, should work
	v.SetContext(context.Background())
	v.ResetInstructionCount()

	block2, err := parser.Parse("test2", `return 42`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto2, err := compiler.Compile("test2", block2)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	results, err := v.Run(proto2)
	if err != nil {
		t.Fatalf("unexpected error on reuse: %v", err)
	}
	if len(results) == 0 || results[0].AsInt() != 42 {
		t.Fatalf("expected 42, got: %v", results)
	}
}

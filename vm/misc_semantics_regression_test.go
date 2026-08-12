package vm

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
)

// compileMiscSource compiles a chunk for the regression tests below.
func compileMiscSource(t *testing.T, src string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse("<test>", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return proto
}

// TestGcFinalizerDoesNotReadMetatableOffVM covers a host abort: the Go
// finalizer closure used to resolve __gc by reading the object's metatable on
// Go's finalizer goroutine, racing with ordinary metatable writes on the VM
// goroutine ("fatal error: concurrent map read and map write", uncatchable).
// Run under -race to see the report directly; without -race the loop below
// still aborts the process on the old code given enough iterations.
func TestGcFinalizerDoesNotReadMetatableOffVM(t *testing.T) {
	v := New()
	var finalized int

	mt := NewEmptyTable()
	if err := mt.Set(metaGc, NewNativeFunc(func(*VM) int {
		finalized++
		return 0
	})); err != nil {
		t.Fatalf("set __gc: %v", err)
	}

	// Register finalizers while continuously mutating the shared metatable's
	// hash part; Go's GC schedules the finalizer goroutine somewhere in here.
	for i := 0; i < 200000; i++ {
		tbl := NewEmptyTable()
		tbl.SetMetatable(mt)
		v.RegisterGcFinalizer(tbl)
		mt.SetString("k"+strconv.Itoa(i), NewInt(int64(i)))
	}

	v.ProcessGcFinalizers()
	if finalized == 0 {
		t.Fatal("expected at least one __gc metamethod to run")
	}
}

// TestGcMetamethodResolvedAtFinalizationTime documents the consequence of
// moving the __gc lookup onto the VM goroutine: like reference Lua's GCTM,
// the metamethod is resolved when the finalizer runs, so clearing __gc first
// cancels the finalization.
func TestGcMetamethodResolvedAtFinalizationTime(t *testing.T) {
	v := New()
	called := false

	mt := NewEmptyTable()
	if err := mt.Set(metaGc, NewNativeFunc(func(*VM) int {
		called = true
		return 0
	})); err != nil {
		t.Fatalf("set __gc: %v", err)
	}
	func() {
		tbl := NewEmptyTable()
		tbl.SetMetatable(mt)
		v.RegisterGcFinalizer(tbl)
	}()
	if err := mt.Set(metaGc, Nil); err != nil {
		t.Fatalf("clear __gc: %v", err)
	}

	v.ProcessGcFinalizers()
	if called {
		t.Error("__gc ran after being removed from the metatable")
	}
}

// TestObjLenPassesObjectAsSecondArg covers the table library's __len call:
// reference luaT_callTMres passes the object twice, so #t and table.insert(t)
// must agree on what __len's second argument is.
func TestObjLenPassesObjectAsSecondArg(t *testing.T) {
	v := New()
	var second Value
	mt := NewEmptyTable()
	if err := mt.Set(metaLen, NewNativeFunc(func(vm *VM) int {
		second = vm.Get(2)
		vm.Set(0, NewInt(7))
		return 1
	})); err != nil {
		t.Fatalf("set __len: %v", err)
	}
	tbl := NewEmptyTable()
	tbl.SetMetatable(mt)
	v.SetGlobal("t", NewTable(tbl))

	// ObjLen is the entry point the table library uses; drive it from a Lua
	// frame so the metamethod dispatch has a caller frame, as it does in
	// production.
	var n int
	var objErr error
	v.SetGlobal("objlen", NewNativeFunc(func(vm *VM) int {
		n, objErr = vm.ObjLen(vm.Get(1))
		return 0
	}))
	if _, err := v.Run(compileMiscSource(t, "objlen(t)\n")); err != nil {
		t.Fatalf("run: %v", err)
	}
	if objErr != nil {
		t.Fatalf("ObjLen: %v", objErr)
	}
	if n != 7 {
		t.Errorf("ObjLen = %d, want 7", n)
	}
	if !second.IsTable() || second.AsTable() != LuaTable(tbl) {
		t.Errorf("__len second argument = %v, want the object itself", second)
	}
}

// closableGlobal builds a table whose metatable carries handler as __close.
// The bare VM under test has no stdlib, so the metatable is wired up in Go.
// handler must be a Lua closure: during error unwinding the Lua call stack has
// already been popped, and only the Lua-closure dispatch path is exercised in
// production (a <close> handler written in Lua).
func closableGlobal(t *testing.T, handler Value) Value {
	t.Helper()
	mt := NewEmptyTable()
	if err := mt.Set(metaClose, handler); err != nil {
		t.Fatalf("set __close: %v", err)
	}
	obj := NewEmptyTable()
	obj.SetMetatable(mt)
	return NewTable(obj)
}

// luaFunc compiles src (which must be `return function(...) ... end`) and
// returns the resulting Lua closure.
func luaFunc(t *testing.T, v *VM, src string) Value {
	t.Helper()
	res, err := v.Run(compileMiscSource(t, src))
	if err != nil {
		t.Fatalf("run %q: %v", src, err)
	}
	if len(res) == 0 || !res[0].IsFunction() {
		t.Fatalf("chunk %q did not return a function", src)
	}
	return res[0]
}

// TestCloseHandlerReceivesRawNilErrorObject pins the 5.4 half of a divergence
// from Lua 5.5. Reference 5.4.8 (lfunc.c callclosemethod / prepcallclosemth)
// always pushes the error object, and does NOT substitute a placeholder for a
// nil one: error(nil) reaches __close as a raw nil, and so does a normal
// close. Lua 5.5 substitutes "<no error object>" in luaG_errormsg; that is a
// 5.5-only change and must not be ported here. Verified against
// /usr/bin/lua5.4.8.
func TestCloseHandlerReceivesRawNilErrorObject(t *testing.T) {
	v := New()
	ran := false
	var got Value
	v.SetGlobal("record", NewNativeFunc(func(vm *VM) int {
		ran = true
		got = vm.Get(1)
		return 0
	}))
	v.SetGlobal("raise", NewNativeFunc(func(*VM) int {
		panic(&LuaError{Value: Nil})
	}))
	v.SetGlobal("o", closableGlobal(t, luaFunc(t, v, "return function(obj, e) record(e) end\n")))

	proto := compileMiscSource(t, `
		local function body()
			local a <close> = o
			raise()
		end
		body()
	`)
	if _, err := v.Run(proto); err == nil {
		t.Fatal("expected the error to propagate")
	}
	if !ran {
		t.Fatal("__close never ran")
	}
	if !got.IsNil() {
		t.Errorf("__close error object = %v, want nil (Lua 5.4 does not substitute)", got)
	}
}

// TestCloseHandlerNormalExitPassesNilErrorArg guards the other half: Lua 5.4
// passes (object, nil) on a normal scope exit too -- unlike 5.5, which passes
// only the object.
func TestCloseHandlerNormalExitPassesNilErrorArg(t *testing.T) {
	v := New()
	ran := false
	var got Value
	v.SetGlobal("record", NewNativeFunc(func(vm *VM) int {
		ran = true
		got = vm.Get(1)
		return 0
	}))
	v.SetGlobal("o", closableGlobal(t, luaFunc(t, v, "return function(obj, e) record(e) end\n")))
	proto := compileMiscSource(t, "do local a <close> = o end\n")
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !ran {
		t.Fatal("__close never ran")
	}
	if !got.IsNil() {
		t.Errorf("__close error object on a normal close = %v, want nil", got)
	}
}

// TestHookExitSentinelPropagates covers os.exit fired from a main-thread debug
// hook. The hook wrapper used to box every panic in a luaHookError, which hid
// the LuaExitError sentinel from ProtectedCall and turned the exit into an
// ordinary uncatchable error plus a traceback.
func TestHookExitSentinelPropagates(t *testing.T) {
	v := New()
	v.SetHook(NewNativeFunc(func(*VM) int {
		panic(&LuaExitError{Code: 7})
	}), HookMaskLine, 0)

	proto := compileMiscSource(t, "local x = 1\nx = x + 1\nreturn x\n")

	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		v.Run(proto)
	}()
	exit, ok := recovered.(*LuaExitError)
	if !ok {
		t.Fatalf("recovered %#v, want *LuaExitError", recovered)
	}
	if exit.Code != 7 {
		t.Errorf("exit code = %d, want 7", exit.Code)
	}
}

// TestHookErrorStillUncatchable guards the surrounding behaviour: a genuine
// error raised by a hook remains wrapped and uncatchable by pcall.
func TestHookErrorStillUncatchable(t *testing.T) {
	v := New()
	v.SetHook(NewNativeFunc(func(*VM) int {
		panic(&LuaError{Value: NewString("hook boom")})
	}), HookMaskLine, 0)

	proto := compileMiscSource(t, "local x = 1\nx = x + 1\nreturn x\n")
	_, err := v.Run(proto)
	if err == nil || !strings.Contains(err.Error(), "hook boom") {
		t.Fatalf("run error = %v, want it to carry the hook's error", err)
	}
}

// TestChannelCloseDuringBlockedSend covers a host panic: closing a LuaChannel
// from Lua while a host goroutine is parked in Send used to close the channel
// underneath the sender ("panic: send on closed channel") on the host's own
// goroutine, where no protected boundary exists.
func TestChannelCloseDuringBlockedSend(t *testing.T) {
	p := NewDefaultChanProvider()
	ch := p.NewChannel(context.Background(), 0)

	var wg sync.WaitGroup
	sendErr := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendErr <- ch.Send(context.Background(), NewInt(1))
		}()
	}

	// Give the producers time to park in their select before closing.
	time.Sleep(20 * time.Millisecond)
	if err := ch.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	wg.Wait()
	close(sendErr)
	for err := range sendErr {
		if err != ErrClosedChannel {
			t.Errorf("blocked Send returned %v, want ErrClosedChannel", err)
		}
	}

	if err := ch.Send(context.Background(), NewInt(1)); err != ErrClosedChannel {
		t.Errorf("Send after close returned %v, want ErrClosedChannel", err)
	}
	if ch.TrySend(NewInt(1)) {
		t.Error("TrySend after close reported success")
	}
	if _, ok, _ := ch.Recv(context.Background()); ok {
		t.Error("Recv on a closed, drained channel reported ok")
	}
}

// TestChannelSendCloseRace hammers concurrent Send/TrySend against Close;
// under -race (and without it) neither may panic the sending goroutine.
func TestChannelSendCloseRace(t *testing.T) {
	for i := 0; i < 50; i++ {
		p := NewDefaultChanProvider()
		ch := p.NewChannel(context.Background(), 1)

		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if err := ch.Send(context.Background(), NewInt(int64(j))); err != nil {
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if ch.IsClosed() {
					return
				}
				ch.TrySend(NewInt(int64(j)))
			}
		}()
		go func() {
			defer wg.Done()
			ch.Close()
		}()
		wg.Wait()

		if err := ch.Close(); err == nil {
			t.Fatal("second Close should report an error")
		}
	}
}

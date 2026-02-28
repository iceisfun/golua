package golua_test

import (
	"testing"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// runLuaWithTimeout runs Lua source in a goroutine with a timeout to detect deadlocks.
// Returns the error (if any) or fails the test on timeout.
func runLuaWithTimeout(t *testing.T, source, name string, timeout time.Duration) error {
	t.Helper()

	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	type result struct {
		err error
	}
	done := make(chan result, 1)

	go func() {
		v := vm.New()
		stdlib.Open(v)
		_, runErr := v.Run(proto)
		done <- result{err: runErr}
	}()

	select {
	case r := <-done:
		return r.err
	case <-time.After(timeout):
		t.Fatalf("deadlock: Lua code did not complete within %v", timeout)
		return nil
	}
}

// --------------------------------------------------------------------------
// BUG: coroutine.close() deadlocks when the goroutine has more yield points
// after the one it's currently suspended at. coClose sends nil to resumeCh
// to unblock the goroutine, but the goroutine continues executing and hits
// another yield, which sends to yieldCh (buffered, succeeds once) then
// blocks on <-resumeCh again. Nobody sends to resumeCh, and coClose is
// blocked on <-doneCh. Result: deadlock.
//
// The single-yield case (yield → return) works because the goroutine
// finishes and closes doneCh. Multi-yield cases deadlock.
// --------------------------------------------------------------------------

// TestCoroutineClose_Suspended_MultiYield closes a coroutine that has
// additional yield points after the current suspension. This is the
// core deadlock scenario.
func TestCoroutineClose_Suspended_MultiYield(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			coroutine.yield(1)
			coroutine.yield(2)
			return 3
		end)
		coroutine.resume(co) -- yields 1, suspended before yield(2)
		local ok, err = coroutine.close(co)
		assert(ok == true, "close should return true, got: " .. tostring(ok))
		assert(err == nil, "close should return nil, got: " .. tostring(err))
		assert(coroutine.status(co) == "dead", "should be dead after close")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_Suspended_LoopYield closes a coroutine suspended
// inside an infinite yield loop. The goroutine would yield forever if
// not properly terminated.
func TestCoroutineClose_Suspended_LoopYield(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			local i = 0
			while true do
				i = i + 1
				coroutine.yield(i)
			end
		end)
		coroutine.resume(co) -- yields 1
		local ok = coroutine.close(co)
		assert(ok == true, "close should return true")
		assert(coroutine.status(co) == "dead")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_AfterMultipleResumes closes after several resume/yield
// cycles, with more yields remaining in the body.
func TestCoroutineClose_AfterMultipleResumes(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			coroutine.yield(1)
			coroutine.yield(2)
			coroutine.yield(3)
			coroutine.yield(4)
			return 5
		end)
		coroutine.resume(co) -- yields 1
		coroutine.resume(co) -- yields 2
		-- Still has yield(3), yield(4), return 5 remaining
		local ok = coroutine.close(co)
		assert(ok == true, "close should succeed")
		assert(coroutine.status(co) == "dead")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_ResumeAfterClose_MultiYield ensures that resume
// after close returns false, even when the original body had more yields.
func TestCoroutineClose_ResumeAfterClose_MultiYield(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			coroutine.yield(1)
			coroutine.yield(2)
		end)
		coroutine.resume(co)
		coroutine.close(co)
		local ok, msg = coroutine.resume(co)
		assert(not ok, "resume after close should fail")
		assert(type(msg) == "string", "should get error string")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_YieldInNestedCall closes a coroutine that yielded
// from within a nested function call. The goroutine has more code to
// execute in the caller after the yield.
func TestCoroutineClose_YieldInNestedCall(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			local function inner()
				coroutine.yield("from_inner")
			end
			inner()
			-- More code after the yield point:
			inner()
			return "done"
		end)
		coroutine.resume(co) -- yields "from_inner" (first call)
		local ok = coroutine.close(co)
		assert(ok == true, "close should succeed")
		assert(coroutine.status(co) == "dead")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_YieldInPcall closes a coroutine that yielded from
// inside a pcall. The pcall frame is on the stack when close is called.
func TestCoroutineClose_YieldInPcall(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			pcall(function()
				coroutine.yield(1)
				coroutine.yield(2) -- extra yield inside pcall
			end)
			return "after_pcall"
		end)
		coroutine.resume(co) -- yields 1 from inside pcall
		local ok = coroutine.close(co)
		assert(ok == true, "close should succeed")
		assert(coroutine.status(co) == "dead")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_YieldInForLoop closes a coroutine yielding in a for loop.
func TestCoroutineClose_YieldInForLoop(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			for i = 1, 100 do
				coroutine.yield(i)
			end
		end)
		coroutine.resume(co) -- yields 1
		coroutine.resume(co) -- yields 2
		coroutine.resume(co) -- yields 3
		-- 97 more yields remaining
		local ok = coroutine.close(co)
		assert(ok == true, "close should succeed")
		assert(coroutine.status(co) == "dead")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_ReturnValues verifies close returns true, nil on success.
func TestCoroutineClose_ReturnValues(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			coroutine.yield(1)
			coroutine.yield(2)
		end)
		coroutine.resume(co)
		local ok, err = coroutine.close(co)
		assert(ok == true, "expected true, got: " .. tostring(ok))
		assert(err == nil, "expected nil, got: " .. tostring(err))
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_StatusTransition verifies suspended → dead transition.
func TestCoroutineClose_StatusTransition(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			coroutine.yield(1)
			coroutine.yield(2)
		end)
		coroutine.resume(co)
		assert(coroutine.status(co) == "suspended", "should be suspended")
		coroutine.close(co)
		assert(coroutine.status(co) == "dead", "should be dead after close")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// --------------------------------------------------------------------------
// Baseline tests: these currently PASS and document the yield→return case
// where close works. They serve as regression tests.
// --------------------------------------------------------------------------

// TestCoroutineClose_YieldThenReturn is the baseline case that currently works:
// single yield followed by return. The goroutine finishes naturally.
func TestCoroutineClose_YieldThenReturn(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			coroutine.yield(1)
			return 99
		end)
		coroutine.resume(co) -- yields 1
		local ok, err = coroutine.close(co)
		assert(ok == true, "close should return true")
		assert(err == nil, "close should return nil")
		assert(coroutine.status(co) == "dead")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_EmptyYieldThenReturn tests yield() with no args then return.
func TestCoroutineClose_EmptyYieldThenReturn(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			coroutine.yield()
			return "done"
		end)
		coroutine.resume(co)
		local ok = coroutine.close(co)
		assert(ok == true, "close should succeed")
		assert(coroutine.status(co) == "dead")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_Dead tests closing an already-dead coroutine (always works).
func TestCoroutineClose_Dead(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function() return 1 end)
		coroutine.resume(co)
		assert(coroutine.status(co) == "dead")
		local ok = coroutine.close(co)
		assert(ok == true, "close dead co should return true")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_NotStarted tests closing a never-resumed coroutine.
func TestCoroutineClose_NotStarted(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function()
			coroutine.yield()
		end)
		local ok = coroutine.close(co)
		assert(ok == true, "close unstarted should succeed")
		assert(coroutine.status(co) == "dead")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// TestCoroutineClose_ErroredCoroutine tests closing a coroutine that errored.
func TestCoroutineClose_ErroredCoroutine(t *testing.T) {
	err := runLuaWithTimeout(t, `
		local co = coroutine.create(function() error("boom") end)
		local ok, msg = coroutine.resume(co)
		assert(not ok, "resume should fail")
		assert(coroutine.status(co) == "dead")
		local cok, cerr = coroutine.close(co)
		assert(type(cok) == "boolean", "close should return boolean")
	`, "test", 3*time.Second)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

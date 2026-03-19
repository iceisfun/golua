package golua_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func runLuaWithTime(t *testing.T, source, name string) {
	t.Helper()

	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	v.SetTimeProvider(vm.NewDefaultTimeProvider())
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

func TestTimeNow(t *testing.T) {
	runLuaWithTime(t, `
		local t = time.now()
		assert(type(t) == "number", "time.now() should return a number, got " .. type(t))
		assert(t > 0, "time.now() should be positive")
	`, "test_time_now")
}

func TestTimeSince(t *testing.T) {
	runLuaWithTime(t, `
		local t1 = time.now()
		-- do some work
		local sum = 0
		for i = 1, 100000 do sum = sum + i end
		local elapsed = time.since(t1)
		assert(type(elapsed) == "number", "time.since() should return a number")
		assert(elapsed >= 0, "time.since() should be non-negative")
	`, "test_time_since")
}

func TestTimeTickFirstCallTrue(t *testing.T) {
	runLuaWithTime(t, `
		-- first call for a key always returns true
		assert(time.tick("test_key", 100000) == true, "first tick should be true")
		-- immediate second call should be false (interval not elapsed)
		assert(time.tick("test_key", 100000) == false, "second tick should be false")
	`, "test_time_tick_first")
}

func TestTimeTickAutoKey(t *testing.T) {
	runLuaWithTime(t, `
		-- auto-keyed by callsite: two different lines get independent keys
		local a = time.tick(100000)
		local b = time.tick(100000)
		-- both are first calls at their respective callsites
		assert(a == true, "first auto tick should be true")
		assert(b == true, "different callsite should also be true")
	`, "test_time_tick_auto")
}

func TestTimeTickExplicitKey(t *testing.T) {
	runLuaWithTime(t, `
		-- explicit name shares state regardless of callsite
		assert(time.tick("shared", 100000) == true)
		assert(time.tick("shared", 100000) == false)

		-- different name is independent
		assert(time.tick("other", 100000) == true)
	`, "test_time_tick_explicit")
}

func TestTimeTickKeyLimit(t *testing.T) {
	p := vm.NewDefaultTimeProvider()
	// Fill up to the 10,000 key limit
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key_%d", i)
		if !p.Tick(context.Background(),key, 999999999) {
			t.Fatalf("tick %d should have returned true", i)
		}
	}
	// 10,001st key should be rejected
	if p.Tick(context.Background(),"overflow", 999999999) {
		t.Fatal("tick beyond limit should return false")
	}
	// Existing keys still work
	if !p.Tick(context.Background(),"key_0", 0) {
		t.Fatal("existing key should still tick")
	}
}

func TestTimeTickKeyTruncation(t *testing.T) {
	p := vm.NewDefaultTimeProvider()
	long := string(make([]byte, 1000)) // 1000 zero bytes
	// First call succeeds (truncated to 512)
	if !p.Tick(context.Background(),long, 999999999) {
		t.Fatal("first tick with long key should return true")
	}
	// Same prefix matches (both truncate to the same 512 bytes)
	long2 := string(make([]byte, 2000))
	if p.Tick(context.Background(),long2, 999999999) {
		t.Fatal("same truncated key should return false")
	}
}

func TestTimeOnceFirstCallTrue(t *testing.T) {
	runLuaWithTime(t, `
		-- first call returns true
		assert(time.once("test_key") == true, "first once should be true")
		-- second call returns false
		assert(time.once("test_key") == false, "second once should be false")
		-- third call still false
		assert(time.once("test_key") == false, "third once should be false")
	`, "test_time_once_first")
}

func TestTimeOnceAutoKey(t *testing.T) {
	runLuaWithTime(t, `
		-- auto-keyed by callsite: different lines get independent state
		local a = time.once()
		local b = time.once()
		assert(a == true, "first auto once should be true")
		assert(b == true, "different callsite should also be true")
	`, "test_time_once_auto")
}

func TestTimeOnceExplicitKey(t *testing.T) {
	runLuaWithTime(t, `
		-- explicit name shares state regardless of callsite
		assert(time.once("shared") == true)
		assert(time.once("shared") == false)

		-- different name is independent
		assert(time.once("other") == true)
	`, "test_time_once_explicit")
}

func TestTimeOnceKeyLimit(t *testing.T) {
	p := vm.NewDefaultTimeProvider()
	// Fill up to the 10,000 key limit
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("once_%d", i)
		if !p.Once(context.Background(),key) {
			t.Fatalf("once %d should have returned true", i)
		}
	}
	// 10,001st key should be rejected
	if p.Once(context.Background(),"overflow") {
		t.Fatal("once beyond limit should return false")
	}
	// Existing keys still return false (already fired)
	if p.Once(context.Background(),"once_0") {
		t.Fatal("existing key should return false on second call")
	}
}

func TestTimeNotAvailableWithoutProvider(t *testing.T) {
	block, err := parser.Parse("test", `
		assert(time == nil, "time should not be available without provider")
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	// No time provider set
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

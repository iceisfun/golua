package main

import (
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func runLuaWithOs(t *testing.T, source, name string, provider vm.LuaOsProvider) {
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
	v.SetOsProvider(provider)
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

func TestOs_Clock(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		local c = os.clock()
		assert(type(c) == "number", "expected number, got " .. type(c))
		assert(c >= 0, "expected non-negative clock value")
	`
	runLuaWithOs(t, source, "test_os_clock", provider)
}

func TestOs_Time(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		local t = os.time()
		assert(type(t) == "number", "expected number, got " .. type(t))
		assert(t > 1000000000, "expected reasonable timestamp, got " .. tostring(t))
	`
	runLuaWithOs(t, source, "test_os_time", provider)
}

func TestOs_DateString(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		local d = os.date("%Y-%m-%d", 0)
		assert(type(d) == "string", "expected string, got " .. type(d))
		-- The date at epoch depends on timezone, so just check format
		assert(#d == 10, "expected 10-char date, got " .. tostring(#d) .. ": " .. d)
	`
	runLuaWithOs(t, source, "test_os_date_string", provider)
}

func TestOs_DateTable(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		local t = os.date("*t")
		assert(type(t) == "table", "expected table, got " .. type(t))
		assert(t.year, "expected year field")
		assert(t.month, "expected month field")
		assert(t.day, "expected day field")
		assert(t.hour ~= nil, "expected hour field")
		assert(t.min ~= nil, "expected min field")
		assert(t.sec ~= nil, "expected sec field")
		assert(t.wday, "expected wday field")
		assert(t.yday, "expected yday field")
		assert(t.isdst ~= nil, "expected isdst field")
		assert(t.year >= 2024, "expected reasonable year, got " .. tostring(t.year))
	`
	runLuaWithOs(t, source, "test_os_date_table", provider)
}

func TestOs_Difftime(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		local t1 = os.time()
		local t2 = t1 + 100
		local diff = os.difftime(t2, t1)
		assert(diff == 100, "expected 100, got " .. tostring(diff))
	`
	runLuaWithOs(t, source, "test_os_difftime", provider)
}

func TestOs_Getenv(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		-- PATH should exist on all systems
		local path = os.getenv("PATH")
		assert(path ~= nil, "expected PATH to exist")
		assert(type(path) == "string", "expected string, got " .. type(path))

		-- Non-existent var should return nil
		local nope = os.getenv("GOLUA_NONEXISTENT_VAR_12345")
		assert(nope == nil, "expected nil for nonexistent var")
	`
	runLuaWithOs(t, source, "test_os_getenv", provider)
}

func TestOs_NoProvider(t *testing.T) {
	source := `
		assert(os == nil, "expected os to be nil without provider")
	`
	runLuaSource(t, source, "test_os_no_provider")
}

func TestOs_TimeWithTable(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		local t = os.time({year=2000, month=1, day=1, hour=0, min=0, sec=0})
		assert(type(t) == "number", "expected number, got " .. type(t))
		assert(t > 0, "expected positive timestamp")
	`
	runLuaWithOs(t, source, "test_os_time_table", provider)
}

func TestOs_FilteredGetenv(t *testing.T) {
	// Test that filtered provider restricts env access
	provider := vm.NewFilteredOsProvider(func(name string) bool {
		return name == "PATH"
	})

	source := `
		-- PATH is allowed
		local path = os.getenv("PATH")
		assert(path ~= nil, "expected PATH to be accessible")

		-- HOME is not in the filter
		local home = os.getenv("HOME")
		assert(home == nil, "expected HOME to be filtered out")
	`
	runLuaWithOs(t, source, "test_os_filtered_getenv", provider)
}

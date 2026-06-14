package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
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

func TestOs_DateNonIntegralTimeError(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	// A fractional time argument has no integer representation. Reference Lua
	// reports "number has no integer representation" (via luaL_checkinteger),
	// not the nonsensical "number expected, got number".
	source := `
		local ok, err = pcall(os.date, "!%Y", 1.5)
		assert(ok == false, "expected os.date to fail on fractional time")
		assert(err:find("number has no integer representation", 1, true),
			"unexpected error: " .. tostring(err))
		-- A non-number time argument still reports the type.
		local ok2, err2 = pcall(os.date, "!%Y", {})
		assert(ok2 == false, "expected os.date to fail on table time")
		assert(err2:find("number expected, got table", 1, true),
			"unexpected error: " .. tostring(err2))
	`
	runLuaWithOs(t, source, "test_os_date_noninteger_time", provider)
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

func TestOs_TimeHonorsIsDST(t *testing.T) {
	t.Setenv("TZ", "America/New_York")
	provider := vm.NewDefaultOsProvider()
	source := `
		local a = os.time{year=2021, month=11, day=7, hour=1, min=30, sec=0, isdst=true}
		local b = os.time{year=2021, month=11, day=7, hour=1, min=30, sec=0, isdst=false}
		assert(b - a == 3600, tostring(b - a))
	`
	runLuaWithOs(t, source, "test_os_time_isdst", provider)
}

func TestOs_TimeUsesMetamethods(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		local reads, writes = {}, {}
		local proxy = setmetatable({}, {
			__index = function(_, k)
				reads[#reads+1] = k
				local src = {year=2024, month=1, day=1, hour=0, min=0, sec=0}
				return src[k]
			end,
			__newindex = function(_, k, v)
				writes[k] = v
			end,
		})
		local ts = os.time(proxy)
		assert(type(ts) == "number")
		assert(#reads >= 6, tostring(#reads))
		assert(type(writes.year) == "number")
		assert(type(writes.wday) == "number")
		assert(type(writes.yday) == "number")
		assert(type(writes.isdst) == "boolean")
	`
	runLuaWithOs(t, source, "test_os_time_metamethods", provider)
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

func TestOs_Setlocale_EmptyUsesCategoryEnv(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_TIME", "en_US.UTF-8")
	t.Setenv("LANG", "C")

	provider := vm.NewDefaultOsProvider()
	source := `
		local loc = os.setlocale("", "time")
		assert(loc == "en_US.UTF-8", "expected LC_TIME locale, got " .. tostring(loc))
	`
	runLuaWithOs(t, source, "test_os_setlocale_empty_uses_category_env", provider)
}

func TestOs_Setlocale_EmptyFallsBackToC(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_TIME", "")
	t.Setenv("LANG", "")

	provider := vm.NewDefaultOsProvider()
	source := `
		local loc = os.setlocale("", "time")
		assert(loc == "C", "expected C fallback, got " .. tostring(loc))
	`
	runLuaWithOs(t, source, "test_os_setlocale_empty_fallback_c", provider)
}

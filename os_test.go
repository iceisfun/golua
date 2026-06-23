package golua_test

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

// TestOs_Time_FieldOutOfBound verifies that os.time bounds-checks every date
// field against C's int range (the type of struct tm members), matching
// reference Lua's getfield(): out-of-range day/hour/min/sec, month-1, and
// year-1900 raise "field '<name>' is out-of-bound" instead of silently
// overflowing into a garbage timestamp.
func TestOs_Time_FieldOutOfBound(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		local INT_MAX = 2147483647
		local function oob(tbl, field)
			local ok, err = pcall(os.time, tbl)
			assert(ok == false, "expected os.time to fail for out-of-bound " .. field)
			local want = "field '" .. field .. "' is out-of-bound"
			assert(err == want, "for " .. field .. " got: " .. tostring(err))
		end
		oob({year=2000, month=1, day=INT_MAX+1}, "day")
		oob({year=2000, month=1, day=1, hour=INT_MAX+1}, "hour")
		oob({year=2000, month=1, day=1, min=INT_MAX+1}, "min")
		oob({year=2000, month=1, day=1, sec=INT_MAX+1}, "sec")
		oob({year=2000, month=1, day=1, sec=-2147483649}, "sec")
		oob({year=2000, month=INT_MAX+2, day=1}, "month")
		oob({year=INT_MAX+1901, month=1, day=1}, "year")
		assert(os.time({year=2000, month=1, day=1, sec=INT_MAX}),
			"sec=INT_MAX should be accepted")
		assert(os.time({year=2000, month=INT_MAX+1, day=1}),
			"month=INT_MAX+1 (tm_mon=INT_MAX) should be accepted")
	`
	runLuaWithOs(t, source, "test_os_time_field_out_of_bound", provider)
}

// TestOs_Date_AltModifiers verifies that os.date accepts the POSIX %E and %O
// alternate-representation modifiers for the specifier combinations allowed by
// reference Lua (LUA_STRFTIMEOPTIONS). In the C/POSIX locale these modifiers
// are no-ops that format as the base specifier, and invalid combinations still
// raise "invalid conversion specifier".
func TestOs_Date_AltModifiers(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		assert(os.date("!%Oy", 0) == os.date("!%y", 0), "%Oy should equal %y")
		assert(os.date("!%Ey", 0) == os.date("!%y", 0), "%Ey should equal %y")
		assert(os.date("!%EY", 0) == os.date("!%Y", 0), "%EY should equal %Y")
		assert(os.date("!%Od", 0) == os.date("!%d", 0), "%Od should equal %d")
		assert(os.date("!%OH%OM%OS", 0) == os.date("!%H%M%S", 0), "%OH%OM%OS")
		local function bad(fmt)
			local ok, err = pcall(os.date, fmt, 0)
			assert(ok == false, "expected " .. fmt .. " to fail")
			local want = "bad argument #1 to 'os.date' (invalid conversion specifier '"
				.. fmt:gsub("^!", "") .. "')"
			assert(err == want, "for " .. fmt .. " got: " .. tostring(err))
		end
		bad("!%Oc")
		bad("!%Ed")
		bad("!%Oz")
		bad("!%E")
		bad("!%O")
	`
	runLuaWithOs(t, source, "test_os_date_alt_modifiers", provider)
}

// TestOs_DateStrftimeParity pins strftime parity fixes against reference Lua
// 5.5 (found by the golua-conformance datefuzz tester). All cases use the '!'
// (gmtime/UTC) form so they are timezone-independent.
func TestOs_DateStrftimeParity(t *testing.T) {
	provider := vm.NewDefaultOsProvider()
	source := `
		-- %c / %F use a natural-width year (not zero-padded to 4 digits)
		assert(os.date("!%c", -62135510400) == "Tue Jan  2 00:00:00 1",
			"%c year width: " .. os.date("!%c", -62135510400))
		assert(os.date("!%F", -62135510400) == "1-01-02",
			"%F year width: " .. os.date("!%F", -62135510400))

		-- %C (century = year/100) is natural width, matching %Y/%G
		assert(os.date("!%C", -62135510400) == "0", "%C year 1: " .. os.date("!%C", -62135510400))
		assert(os.date("!%C", 1719100000) == "20", "%C year 2024: " .. os.date("!%C", 1719100000))

		-- %y/%C/%g use FLOORED division for negative years, so the invariant
		-- %C*100 + %y == %Y holds. Year -2 -> %C "-1", %y "98".
		assert(os.date("!%Y", -62200000000) == "-2", "%Y year -2: " .. os.date("!%Y", -62200000000))
		assert(os.date("!%y", -62200000000) == "98", "%y year -2: " .. os.date("!%y", -62200000000))
		assert(os.date("!%C", -62200000000) == "-1", "%C year -2: " .. os.date("!%C", -62200000000))
		local C = tonumber(os.date("!%C", -62200000000))
		local y = tonumber(os.date("!%y", -62200000000))
		assert(C * 100 + y == -2, "%C*100+%y invariant: " .. (C * 100 + y))

		-- %Z on the gmtime path names the UTC zone "GMT"
		assert(os.date("!%Z", 0) == "GMT", "%Z gmtime: " .. os.date("!%Z", 0))

		-- An invalid specifier reports the ORIGINAL format from the bad char on;
		-- compound specifiers (%R/%T) are not pre-expanded into the message.
		local ok, msg = pcall(os.date, "%Q%R", 0)
		assert(not ok and msg:find("invalid conversion specifier '%Q%R'", 1, true),
			"compound leak %R: " .. tostring(msg))
		ok, msg = pcall(os.date, "!%Q%T", 0)
		assert(not ok and msg:find("invalid conversion specifier '%Q%T'", 1, true),
			"compound leak %T: " .. tostring(msg))
	`
	runLuaWithOs(t, source, "test_os_date_strftime_parity", provider)
}

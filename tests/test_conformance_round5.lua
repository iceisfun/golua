-- Conformance tests for Lua 5.4 semantic bugs (round 5)
-- Each test documents expected Lua 5.4 behavior

---------------------------------------------------------------------
-- BUG 1: Native function == comparison panics
-- In Lua 5.4, native (C) functions can be compared with == just like
-- closures. They are equal if they are the same function object.
-- GoLua panics with "comparing uncomparable type vm.NativeFunc".
---------------------------------------------------------------------
do
    -- Same native function should be equal
    assert(print == print, "print == print should be true")
    -- Different native functions should not be equal
    assert(print ~= tostring, "print ~= tostring should be true")
    -- Native function compared with non-function should be false
    assert(print ~= 42, "print ~= 42 should be true")
    assert(print ~= "hello", "print ~= 'hello' should be true")
    assert(print ~= nil, "print ~= nil should be true")
    -- Native functions stored in tables should be comparable
    local t = {f = print}
    assert(t.f == print, "t.f == print should be true")
end

---------------------------------------------------------------------
-- BUG 2: load(number) panics instead of coercing to string
-- In Lua 5.4, load() accepts a number as the first argument by
-- converting it to a string. GoLua panics with "bad argument #1
-- to 'load' (function expected, got number)".
---------------------------------------------------------------------
do
    -- load(number) should convert number to string and compile it
    local f, err = load(123)
    -- 123 is not valid Lua, so f should be nil with an error
    assert(f == nil, "load(123) should return nil (not valid Lua)")
    assert(type(err) == "string", "load(123) should return error string")

    -- A valid number string like "return 42" won't be passed as a
    -- number, but we can test that load doesn't panic
    local ok, result = pcall(load, 42)
    assert(ok == true, "load(number) should not panic, got: " .. tostring(result))
end

---------------------------------------------------------------------
-- BUG 3: assert() doesn't prepend file:line to string error messages
-- In Lua 5.4, when assert fails with a string message, the error
-- has file:line prepended (like error() with default level).
-- GoLua returns the bare message without location info.
---------------------------------------------------------------------
do
    local ok, err = pcall(function() assert(false, "oops") end)
    assert(not ok, "assert(false) should error")
    -- In Lua 5.4, err should contain ": oops" (with file:line prefix)
    -- The message should end with ": oops" or contain it
    assert(type(err) == "string", "assert error should be string")
    assert(err:find(": oops"), "assert error should contain ': oops' (with location prefix), got: " .. err)

    -- assert with no message should also have location
    local ok2, err2 = pcall(function() assert(false) end)
    assert(not ok2, "assert(false) should error")
    assert(type(err2) == "string", "assert error should be string")
    assert(err2:find(": assertion failed!"),
        "assert error should contain ': assertion failed!' (with location prefix), got: " .. err2)

    -- assert with non-string message should NOT add location
    local ok3, err3 = pcall(function() assert(false, 42) end)
    assert(not ok3, "assert(false, 42) should error")
    assert(err3 == 42, "assert with number message should preserve the number")
end

---------------------------------------------------------------------
-- BUG 4: tonumber fails on hex floats without integer part
-- In Lua 5.4, hex float literals like "0x.1" are valid and parse
-- correctly. GoLua returns nil for these.
---------------------------------------------------------------------
do
    -- Hex floats without integer part
    assert(tonumber("0x.1") == 0.0625,
        "tonumber('0x.1') should be 0.0625, got: " .. tostring(tonumber("0x.1")))
    assert(tonumber("0x.8") == 0.5,
        "tonumber('0x.8') should be 0.5, got: " .. tostring(tonumber("0x.8")))
    assert(tonumber("0x.F") == 0.9375,
        "tonumber('0x.F') should be 0.9375, got: " .. tostring(tonumber("0x.F")))

    -- Hex floats with both integer and fractional parts
    assert(tonumber("0xA.8") == 10.5,
        "tonumber('0xA.8') should be 10.5, got: " .. tostring(tonumber("0xA.8")))
    assert(tonumber("0x1.0") == 1.0,
        "tonumber('0x1.0') should be 1.0, got: " .. tostring(tonumber("0x1.0")))

    -- Hex float with fractional part and exponent
    assert(tonumber("0x.1p4") == 1.0,
        "tonumber('0x.1p4') should be 1.0, got: " .. tostring(tonumber("0x.1p4")))
    assert(tonumber("0x.Fp0") == 0.9375,
        "tonumber('0x.Fp0') should be 0.9375, got: " .. tostring(tonumber("0x.Fp0")))

    -- Arithmetic coercion of hex float strings should also work
    assert("0x.1" + 0 == 0.0625,
        "'0x.1' + 0 should be 0.0625")
    assert("0xA.8" + 0 == 10.5,
        "'0xA.8' + 0 should be 10.5")
end

---------------------------------------------------------------------
-- BUG 5: coroutine.close() deadlocks on suspended coroutine
-- In Lua 5.4, coroutine.close() on a suspended coroutine marks it
-- dead and returns true, nil. GoLua deadlocks.
---------------------------------------------------------------------
do
    local co = coroutine.create(function()
        coroutine.yield(1)
        coroutine.yield(2)
        return 3
    end)

    -- Start the coroutine (it yields)
    local ok, v = coroutine.resume(co)
    assert(ok and v == 1, "first resume should yield 1")
    assert(coroutine.status(co) == "suspended", "should be suspended")

    -- Close the suspended coroutine
    local close_ok, close_err = coroutine.close(co)
    assert(close_ok == true, "coroutine.close should return true, got: " .. tostring(close_ok))
    assert(close_err == nil, "coroutine.close should return nil error, got: " .. tostring(close_err))
    assert(coroutine.status(co) == "dead", "closed coroutine should be dead")

    -- Resuming a closed coroutine should fail
    local ok2, err2 = coroutine.resume(co)
    assert(not ok2, "resume after close should fail")
end

---------------------------------------------------------------------
-- BUG 6: warn function not implemented
-- In Lua 5.4, warn() is a global function for emitting warnings.
-- GoLua doesn't provide it at all.
---------------------------------------------------------------------
do
    assert(type(warn) == "function", "warn should be a function, got: " .. type(warn))

    -- Basic warn call should not error
    local ok, err = pcall(warn, "test warning")
    assert(ok, "warn('test warning') should not error: " .. tostring(err))

    -- warn("@off") / warn("@on") control warning output
    local ok2 = pcall(warn, "@off")
    assert(ok2, "warn('@off') should not error")
    local ok3 = pcall(warn, "@on")
    assert(ok3, "warn('@on') should not error")
end

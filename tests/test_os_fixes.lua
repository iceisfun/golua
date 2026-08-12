-- Tests for OS library fixes

-- Fix 1: os.rename error message should NOT include filename.
-- Use os.tmpname() (then remove it) for paths under the OS temp dir so the
-- jailed test harness allows them regardless of TMPDIR; the names are
-- nonexistent so rename surfaces a real ENOENT.
do
    local src = os.tmpname()
    local dst = os.tmpname()
    os.remove(src)
    os.remove(dst)
    local ok, msg = os.rename(src, dst)
    assert(ok == nil, "os.rename should fail")
    -- error must not leak either path (basenames are "lua_..."); also assert
    -- it does not contain the full src path.
    assert(not msg:find(src, 1, true), "os.rename error should not include filename, got: " .. msg)
    -- Lua 5.4 returns just "No such file or directory" (no path prefix)
    assert(msg:find("No such file or directory"), "os.rename error wording, got: " .. msg)
end

-- Fix 2: os.difftime should reject non-integer floats
do
    local ok, err = pcall(os.difftime, 10.5, 5.5)
    assert(not ok, "os.difftime should reject float 10.5")
    assert(err:find("number has no integer representation"), "wrong error: " .. tostring(err))
end

-- Fix 3: os.time should reject strings with proper error message
do
    local ok, err = pcall(os.time, "hello")
    assert(not ok, "os.time should reject string argument")
    assert(err:find("table expected, got string"), "wrong error: " .. tostring(err))
end

-- Fix 4: os.date trailing % error should say '%' not '%\0'
do
    local ok, err = pcall(os.date, "test%")
    assert(not ok, "os.date should reject trailing %")
    assert(not err:find("\0"), "os.date error should not contain null byte, got: " .. tostring(err))
    -- Should say: invalid conversion specifier '%'
    assert(err:find("invalid conversion specifier '%%'"), "wrong error: " .. tostring(err))
end

-- Fix 5: os.date should reject out-of-range timestamps
do
    local ok, err = pcall(os.date, "%Y", 1<<62)
    assert(not ok, "os.date should reject huge timestamp")
    assert(err:find("date result cannot be represented"), "wrong error: " .. tostring(err))
    -- Should NOT have "bad argument" prefix
    assert(not err:find("bad argument"), "range error should not have bad argument prefix: " .. tostring(err))

    local ok2, err2 = pcall(os.date, "*t", 1<<62)
    assert(not ok2, "os.date *t should reject huge timestamp")
    assert(err2:find("date result cannot be represented"), "wrong error: " .. tostring(err2))
end

print("PASS")

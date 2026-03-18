-- Test io/os error message formatting
local function check(label, got, expected)
    if got ~= expected then
        error(string.format("%s:\n  got:      %s\n  expected: %s", label, tostring(got), tostring(expected)), 2)
    end
end

local function strip(err)
    return tostring(err):gsub("^.*:%d+: ", "")
end

-- Issue 1: io.read/io.write via pcall should show 'io.read'/'io.write' not 'read'/'write'
local ok, err = pcall(io.read, true)
check("io.read name", strip(err), "bad argument #1 to 'io.read' (string expected, got boolean)")

local ok2, err2 = pcall(io.write, true)
check("io.write name", strip(err2), "bad argument #1 to 'io.write' (string expected, got boolean)")

-- Issue 2: File method errors via pcall should show '?' not method name
local f = io.tmpfile()
local ok3, err3 = pcall(f.read, f, true)
check("f.read name", strip(err3), "bad argument #2 to '?' (string expected, got boolean)")

local ok4, err4 = pcall(f.write, f, true)
check("f.write name", strip(err4), "bad argument #2 to '?' (string expected, got boolean)")
f:close()

-- Issue 3: io.close(nil) should error, not close default output
local ok5, err5 = pcall(io.close, nil)
check("io.close nil ok", ok5, false)
check("io.close nil err", strip(err5), "bad argument #1 to 'io.close' (FILE* expected, got nil)")

-- Issue 4: explicit nil arg says "got nil" not "got no value"
local ok6, err6 = pcall(os.getenv, nil)
check("os.getenv nil", strip(err6), "bad argument #1 to 'os.getenv' (string expected, got nil)")

local ok7, err7 = pcall(os.remove, nil)
check("os.remove nil", strip(err7), "bad argument #1 to 'os.remove' (string expected, got nil)")

local ok8, err8 = pcall(os.rename, nil)
check("os.rename nil", strip(err8), "bad argument #1 to 'os.rename' (string expected, got nil)")

-- Issue 5: f:setvbuf(42) should show '42' not '' and '?' not 'setvbuf'
local f2 = io.tmpfile()
local ok9, err9 = pcall(f2.setvbuf, f2, 42)
check("f:setvbuf(42)", strip(err9), "bad argument #2 to '?' (invalid option '42')")
f2:close()

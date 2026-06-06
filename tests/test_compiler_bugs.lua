-- Test compiler bug fixes for differential testing

-- Bug 1: Register limit should be 255, not 249
-- Bug 2: "Too many locals" line number should be next token's line
-- Bug 3: <eof> should not be quoted in near clause
-- Bug 4: Wrong near token in function-scope "too many locals"
-- Bug 5: Duplicate label reports duplicate's line, not original's

local function check_error(code, expected_pattern, desc)
    local ok, err = load(code)
    assert(not ok, desc .. " -- expected error, got success")
    err = tostring(err)
    assert(string.find(err, expected_pattern, 1, true),
        desc .. "\n  expected pattern: " .. expected_pattern .. "\n  got: " .. err)
end

-- Bug 1: Register limit 255 (not 249)
-- 200 locals + 50 temps from additions should be within 255 register limit
do
    local lines = {}
    for i = 1, 200 do
        lines[#lines+1] = "local v" .. i .. " = " .. i
    end
    local parts = {}
    for i = 1, 50 do
        parts[#parts+1] = "v" .. i
    end
    lines[#lines+1] = "return " .. table.concat(parts, "+")
    local code = table.concat(lines, "\n")
    local fn, err = load(code)
    assert(fn, "Bug 1 - register limit too low: " .. tostring(err))
end

-- Bug 3: <eof> should not be quoted
do
    local lines = {}
    for i = 1, 201 do
        lines[#lines+1] = "local v" .. i
    end
    local code = table.concat(lines, "\n")
    local ok, err = load(code)
    err = tostring(err)
    assert(string.find(err, "near <eof>", 1, true),
        "Bug 3 - <eof> should not be quoted: " .. err)
end

-- Bug 2: "Too many locals" line number should be next token's line
do
    local lines = {}
    for i = 1, 200 do
        lines[#lines+1] = "local v" .. i
    end
    lines[#lines+1] = "local v201"
    lines[#lines+1] = "print(v1)"
    local code = table.concat(lines, "\n")
    local ok, err = load(code)
    err = tostring(err)
    assert(string.find(err, ":202:", 1, true),
        "Bug 2 - error should report line 202: " .. err)
end

-- Bug 4: Wrong near token inside function scope
do
    local lines = {"local function f()"}
    for i = 1, 201 do
        lines[#lines+1] = "local v" .. i
    end
    lines[#lines+1] = "end"
    local code = table.concat(lines, "\n")
    local ok, err = load(code)
    err = tostring(err)
    assert(string.find(err, "near 'end'", 1, true),
        "Bug 4 - should report near 'end': " .. err)
end

-- Bug 5: Duplicate label error position is the later (second) label's line,
-- and the message references the earlier label's line. Matches reference
-- lua5.5.0: "<chunk>:3: label 'A' already defined on line 1".
do
    local code = "::A::\nlocal x = 1\n::A::\n"
    local ok, err = load(code)
    err = tostring(err)
    assert(string.find(err, ":3:", 1, true) and string.find(err, "on line 1", 1, true),
        "Bug 5 - wrong error format: " .. err)
end

print("PASS")

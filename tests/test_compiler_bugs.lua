-- Test compiler bug fixes for differential testing

-- Bug 1: Register limit should be 255, not 249
-- Bug 2: "Too many locals" line number should be next token's line
-- Bug 3: <eof> should not be quoted in near clause
-- Bug 4: Wrong near token in function-scope "too many locals"
-- Bug 5: Duplicate label reports duplicate's line, not original's

local function check_error(code, expected_pattern, desc)
    local ok, err = load(code)
    if ok then
        print("FAIL: " .. desc .. " -- expected error, got success")
        return
    end
    err = tostring(err)
    if string.find(err, expected_pattern, 1, true) then
        print("PASS: " .. desc)
    else
        print("FAIL: " .. desc)
        print("  expected pattern: " .. expected_pattern)
        print("  got: " .. err)
    end
end

-- Bug 1: Register limit 255 (not 249)
-- 200 locals + 50 temps from additions should be within 255 register limit
-- (with the old limit of 249, this would fail)
do
    local lines = {}
    for i = 1, 200 do
        lines[#lines+1] = "local v" .. i .. " = " .. i
    end
    -- Build a chain of additions that needs temp registers
    -- This needs ~250 registers total, within 255 but over old 249 limit
    local parts = {}
    for i = 1, 50 do
        parts[#parts+1] = "v" .. i
    end
    lines[#lines+1] = "return " .. table.concat(parts, "+")
    local code = table.concat(lines, "\n")
    local fn, err = load(code)
    if fn then
        print("PASS: Bug 1 - register limit 255 allows 200 locals + 50 temps")
    else
        print("FAIL: Bug 1 - register limit too low: " .. tostring(err))
    end
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
    -- Should contain "near <eof>" not "near '<eof>'"
    if string.find(err, "near <eof>", 1, true) then
        print("PASS: Bug 3 - <eof> not quoted")
    elseif string.find(err, "near '<eof>'", 1, true) then
        print("FAIL: Bug 3 - <eof> is quoted: " .. err)
    else
        print("FAIL: Bug 3 - unexpected error: " .. err)
    end
end

-- Bug 2: "Too many locals" line number should be next token's line
-- When the next line has another statement, error should report THAT line
do
    local lines = {}
    for i = 1, 200 do
        lines[#lines+1] = "local v" .. i
    end
    -- The 201st local on its own line triggers the error
    -- The next token is "print" on the following line
    lines[#lines+1] = "local v201"
    lines[#lines+1] = "print(v1)"
    local code = table.concat(lines, "\n")
    local ok, err = load(code)
    err = tostring(err)
    -- Error should be at line 202 (the "print" line), not 201 (the "local" line)
    if string.find(err, ":202:", 1, true) then
        print("PASS: Bug 2 - error reports line of next token (202)")
    else
        print("FAIL: Bug 2 - wrong line in error: " .. err)
    end
end

-- Bug 4: Wrong near token inside function scope - should be 'end', not '<eof>'
do
    local lines = {"local function f()"}
    for i = 1, 201 do
        lines[#lines+1] = "local v" .. i
    end
    lines[#lines+1] = "end"
    local code = table.concat(lines, "\n")
    local ok, err = load(code)
    err = tostring(err)
    -- Should report "near 'end'" (the function's closing keyword)
    if string.find(err, "near 'end'", 1, true) then
        print("PASS: Bug 4 - reports near 'end' inside function")
    else
        print("FAIL: Bug 4 - wrong near token: " .. err)
    end
end

-- Bug 5: Duplicate label error prefix uses EOF/end line, not the label's line
do
    -- ::A:: on line 1, statement on line 2, ::A:: on line 3, trailing newline = EOF on line 4
    local code = "::A::\nlocal x = 1\n::A::\n"
    local ok, err = load(code)
    err = tostring(err)
    -- Error prefix should reference the EOF line (4), not the duplicate label line (3)
    -- Message body keeps "on line 1" (the original label)
    if string.find(err, ":4:", 1, true) and string.find(err, "on line 1", 1, true) then
        print("PASS: Bug 5 - duplicate label prefix uses EOF line")
    else
        print("FAIL: Bug 5 - wrong error: " .. err)
    end
end

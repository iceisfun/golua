-- Tests for require/load error message formatting

-- Issue 1: require() syntax error should not include @ prefix in source name
do
    local ok, err = pcall(require, "helper_bad_syntax")
    assert(not ok, "require of bad syntax file should fail")
    -- The error should NOT contain "@" before the file path
    assert(not string.find(err, "@helper_bad_syntax", 1, true),
        "require syntax error should not contain @ prefix, got: " .. err)
    -- It SHOULD contain the file path without @
    assert(string.find(err, "helper_bad_syntax.lua", 1, true),
        "require syntax error should contain file path, got: " .. err)
    -- It should have the "error loading module" wrapper
    assert(string.find(err, "error loading module", 1, true),
        "require syntax error should have 'error loading module' wrapper, got: " .. err)
    print("PASS: require syntax error has no @ prefix")
end

-- Issue 2: load() with infinite reader should report syntax error, not "not enough memory"
do
    local f, err = load(function()
        return "invalid syntax here }"
    end)
    assert(f == nil, "load with bad syntax should return nil")
    -- Should get a syntax error, not "not enough memory"
    assert(not string.find(err, "not enough memory", 1, true),
        "load with infinite reader should not say 'not enough memory', got: " .. err)
    assert(string.find(err, "syntax error", 1, true),
        "load with infinite reader should report syntax error, got: " .. err)
    print("PASS: load with infinite reader reports syntax error")
end

-- Issue 3: Circular require should wrap stack overflow with "error loading module"
do
    local ok, err = pcall(require, "helper_circ_a")
    assert(not ok, "circular require should fail")
    -- The error should be wrapped with "error loading module"
    assert(string.find(err, "error loading module", 1, true),
        "circular require should wrap error with 'error loading module', got: " .. err)
    assert(string.find(err, "stack overflow", 1, true),
        "circular require error should mention stack overflow, got: " .. err)
    print("PASS: circular require wraps stack overflow")
end

print("All require error tests passed!")

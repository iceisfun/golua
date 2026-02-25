-- test_load_mode_and_reader_contract: load() mode/reader behavior should match Lua 5.4

-- Text chunk with binary-only mode should fail
local f1, e1 = load("return 1", "x", "b")
assert(f1 == nil, "load text chunk with mode='b' should fail")
assert(type(e1) == "string" and e1:find("attempt to load a text chunk"),
       "unexpected mode='b' error: " .. tostring(e1))

-- Numeric mode is accepted via conversion and still enforced
local f2, e2 = load("return 1", "x", 123)
assert(f2 == nil, "load text chunk with numeric mode lacking 't' should fail")
assert(type(e2) == "string" and e2:find("mode is '123'"),
       "unexpected numeric mode error: " .. tostring(e2))

-- Reader returning table is invalid
local f3, e3 = load(function() return {} end)
assert(f3 == nil, "reader returning table should fail")
assert(type(e3) == "string" and e3:find("reader function must return a string"),
       "unexpected reader table error: " .. tostring(e3))

-- Reader returning number is accepted as text (Lua converts number to string)
local i = 0
local f4, e4 = load(function()
    i = i + 1
    if i == 1 then return 1 end
    return nil
end)
assert(f4 == nil, "reader returning numeric chunk '1' should parse as invalid chunk")
assert(type(e4) == "string" and e4:find("unexpected symbol"),
       "unexpected numeric reader parse error: " .. tostring(e4))

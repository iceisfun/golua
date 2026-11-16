-- The file metatable's __tostring checks its argument (reference f_tostring
-- uses luaL_checkudata) instead of answering "file (?)" for a non-file.

local mt = getmetatable(io.stdout)
local function expectError(want, ...)
  local ok, err = pcall(mt.__tostring, ...)
  assert(not ok, "expected an error, got " .. tostring(err))
  assert(err == want, "got: " .. tostring(err))
end

expectError("bad argument #1 to '?' (FILE* expected, got table)", {})
expectError("bad argument #1 to '?' (FILE* expected, got no value)")
expectError("bad argument #1 to '?' (FILE* expected, got number)", 1)
expectError("bad argument #1 to '?' (FILE* expected, got FILE*)", setmetatable({}, mt))

local ok, err = pcall(tostring, setmetatable({}, mt))
assert(not ok and err == "bad argument #1 to '?' (FILE* expected, got FILE*)", tostring(err))

-- Real handles are unaffected.
assert(tostring(io.stdout):match("^file %(0x%x+%)$"), tostring(io.stdout))
local name = os.tmpname()
local f = assert(io.open(name, "w"))
assert(mt.__tostring(f):match("^file %(0x%x+%)$"))
f:close()
assert(tostring(f) == "file (closed)")
os.remove(name)

print("PASS")

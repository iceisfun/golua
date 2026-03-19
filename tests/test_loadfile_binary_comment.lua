-- Test loading binary files with initial shebang/comment lines
-- The comment may contain null bytes; binary data follows after newline

local fname = os.tmpname()

-- Plain binary loadfile
io.open(fname, "wb"):write(string.dump(function() return 10, "hi" end)):close()
local a, b = assert(loadfile(fname))()
assert(a == 10 and b == "hi", "plain binary: a=" .. tostring(a) .. " b=" .. tostring(b))
print("plain binary loadfile: OK")

-- Binary with comment containing null byte
io.open(fname, "wb"):write(
  "#this is a comment for a binary file\0\n",
  string.dump(function() return 20, '\0\0\0' end)
):close()
a, b = assert(loadfile(fname))()
assert(a == 20, "comment+binary: a=" .. tostring(a))
assert(b == "\0\0\0", "comment+binary: b should be three null bytes")
print("binary with null-byte comment: OK")

-- Binary with simple shebang
io.open(fname, "wb"):write(
  "#!/usr/bin/lua\n",
  string.dump(function() return 30 end)
):close()
a = assert(loadfile(fname))()
assert(a == 30, "shebang+binary: a=" .. tostring(a))
print("binary with shebang: OK")

-- Binary with no-upvalue function in restricted env
io.open(fname, "wb"):write(string.dump(function() return 1 end)):close()
local f = assert(loadfile(fname, "b", {}))
assert(type(f) == "function" and f() == 1)
print("binary with env: OK")

os.remove(fname)
print("PASS")

-- A read error inside a lines iterator is raised with the strerror text, as
-- reference io_readline does; it is not taken for the end of the file, which
-- would end a loop over an unreadable file silently. f:read(0) reports a read
-- error as nil, message, errno like every other format.

local function expectError(fn, want, what)
  local ok, err = pcall(fn)
  assert(not ok, what .. ": expected an error")
  assert(err == want, what .. ": got " .. tostring(err))
end

-- A write-only stream: every read fails with EBADF.
local name = os.tmpname()
local f = assert(io.open(name, "w"))
local r, msg, errno = f:read(0)
assert(r == nil and msg == "Bad file descriptor" and errno == 9,
  "read(0): " .. tostring(r) .. " " .. tostring(msg) .. " " .. tostring(errno))
expectError(f:lines(), "Bad file descriptor", "f:lines()")
for _, fmt in ipairs({"l", "L", "n", "a", 1, 0}) do
  expectError(f:lines(fmt), "Bad file descriptor", "f:lines(" .. tostring(fmt) .. ")")
end
f:close()
os.remove(name)

-- A directory opens for reading, but reading it fails with EISDIR.
expectError(io.lines("."), "Is a directory", "io.lines(dir)")
for _, fmt in ipairs({"l", "L", "n", "a", 1, 0}) do
  expectError(io.lines(".", fmt), "Is a directory", "io.lines(dir, " .. tostring(fmt) .. ")")
end
local d = assert(io.open("."))
r, msg, errno = d:read(0)
assert(r == nil and msg == "Is a directory" and errno == 21, "dir read(0): " .. tostring(msg))
assert(not pcall(function() for _ in d:lines() do end end), "loop over a directory ended quietly")
d:close()

-- End of file still ends the loop quietly.
name = os.tmpname()
f = assert(io.open(name, "w"))
f:write("a\nb\n")
f:close()
local seen = {}
for l in io.lines(name) do seen[#seen + 1] = l end
for l in io.lines(name, "l") do seen[#seen + 1] = l end
assert(table.concat(seen, ",") == "a,b,a,b", table.concat(seen, ","))
f = assert(io.open(name))
assert(f:read(0) == "")
f:read("a")
assert(f:read(0) == nil)
f:close()
os.remove(name)

print("PASS")

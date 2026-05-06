-- broken_fuzz_io_seek_error_path_prefix:
-- f:seek() error message includes the Go *PathError prefix
-- ("seek <path>: <msg>") instead of just the strerror.
--
-- BROKEN: vm/full_io.go around line 288 returns Go's *PathError directly,
-- whose .Error() formatting is "<op> <path>: <msg>". stdlib should strip
-- the path prefix and capitalize the strerror to match reference Lua.
-- See extractLuaFileError() helper for the existing pattern.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   f:seek('cur', -100)  on an empty/short file
--     returns nil, "Invalid argument"
--
-- golua today:
--   returns nil, "seek /tmp/...: invalid argument"
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
local w = io.open(p, "w"); w:write("hi"); w:close()
local r = io.open(p, "r")

local pos, err = r:seek("cur", -100)
r:close(); os.remove(p)

assert(pos == nil, "seek before start must fail")
assert(type(err) == "string", "error must be a string")
assert(not err:find("seek "),
  "Go *PathError prefix leaked: " .. err)
-- Reference message is just "Invalid argument" (capitalized strerror).
assert(err:lower():find("invalid argument"),
  "expected 'invalid argument'; got: " .. err)

print("ok")

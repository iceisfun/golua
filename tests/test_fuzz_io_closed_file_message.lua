-- broken_fuzz_io_closed_file_message:
-- The lines() iterator emits "attempt to use a closed file" when the
-- underlying file is closed between iterations. Reference Lua emits
-- "file is already closed" specifically here (liolib.c io_readline).
--
-- Note: f:read on a closed file correctly emits "attempt to use a
-- closed file" in both reference Lua and golua (liolib.c tofile()).
-- The wording divergence is limited to the lines iterator path.
--
-- Reference (lua5.5.0 and lua 5.4.8):
--   local f = io.open(...); local it = f:lines(); f:close()
--   pcall(it) -> false, "...: file is already closed"
--
-- golua before fix:
--   -> "attempt to use a closed file"
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
local w = io.open(p, "w"); w:write("a\nb\n"); w:close()
local r = io.open(p, "r")
local iter = r:lines()
r:close()
os.remove(p)

local ok, err = pcall(iter)
assert(ok == false, "iterator on closed file must fail")
assert(type(err) == "string", "error must be a string")
assert(err:find("file is already closed"),
  "expected 'file is already closed'; got: " .. err)

print("ok")

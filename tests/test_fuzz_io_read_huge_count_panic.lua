-- broken_fuzz_io_read_huge_count_panic:
-- f:read(N) with very large N leaks a Go runtime makeslice panic through
-- pcall, instead of raising a structured Lua "not enough memory" error.
--
-- BROKEN: stdlib/io.go around line 1001 calls
--   f.ReadBytes(ctx, int(count))
-- which goes to make([]byte, count) in vm/full_io.go around line 226. Go's
-- runtime.makeslice panics with "len out of range" when count exceeds the
-- platform slice cap. The panic message ("runtime error: makeslice: ...")
-- leaks Go internals through pcall.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(f.read, f, 1<<60) -> false, "not enough memory"
--
-- golua today:
--   -> false, "runtime error: makeslice: len out of range"
--
-- Sandbox-leak class: similar to the unpack(s8) panic leak fixed earlier.
-- Fix shape: bound-check count against a sane upper limit before alloc;
-- raise the structured "not enough memory" error.
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
local w = io.open(p, "w"); w:write("hi"); w:close()
local r = io.open(p, "r")

local ok, err = pcall(r.read, r, 1 << 60)
r:close(); os.remove(p)

assert(ok == false, "huge read count must fail")
assert(type(err) == "string", "error must be a string")
assert(not err:find("runtime error"),
  "Go runtime panic leaked through pcall: " .. err)
assert(err:find("not enough memory") or err:find("memory"),
  "expected 'not enough memory' Lua error; got: " .. err)

print("ok")

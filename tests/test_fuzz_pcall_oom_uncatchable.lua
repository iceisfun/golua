-- broken_fuzz_pcall_oom_uncatchable:
-- Out-of-memory inside pcall is NOT caught — golua aborts the host process.
--
-- BROKEN: When a Lua program inside pcall exhausts memory (e.g., infinite
-- table append), golua's underlying Go allocator calls runtime.throw() from
-- runtime.growslice during Table.Set, which is unrecoverable and crashes
-- the embedding Go process. Lua 5.5 (and 5.4) cleanly raise a Lua error
-- "not enough memory" that pcall catches, returning (false, "not enough
-- memory").
--
-- Embedder impact: any sandboxed Lua script can crash the host with a
-- 4-line program. With ulimit -v 2GB the crash takes only milliseconds.
--
-- Suspect site: vm/table.go:Table.Set — the slice-growth path is
-- unprotected. A fix needs either:
--   (a) a per-VM memory budget that bounds table growth and raises a Lua
--       error before the underlying append exceeds it, or
--   (b) catching runtime.throw via debug.SetPanicOnFault-style protection
--       (Go's runtime.throw bypasses recover, so this is hard), or
--   (c) Limits.MaxTableSize / MaxBytes wired into all growth paths.
--
-- Reference (lua5.5.0):
--   pcall returns (false, "not enough memory")
--
-- golua today:
--   process aborts with `runtime: out of memory` / `fatal error: out of memory`
--   and a Go stack trace; exit code is non-zero (Go runtime fatal)
--
-- Discovered: differential fuzz 2026-05-04 (control-flow wave-1 agent).

-- This file is skipped by the broken_*.lua harness; running it directly
-- under `lua5.5.0` will print  "caught: false ... not enough memory"
-- under reasonable ulimit (e.g. ulimit -v 2097152). Running under golua
-- crashes the host.
local ok, err = pcall(function()
  local t = {}
  while true do
    t[#t + 1] = "x"
  end
end)

assert(ok == false, "pcall must return false on OOM; got ok=" .. tostring(ok))
assert(type(err) == "string" and err:find("memory"),
  "expected memory-related error string; got " .. tostring(err))

print("caught:", ok, err)

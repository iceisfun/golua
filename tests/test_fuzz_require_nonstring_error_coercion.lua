-- broken_fuzz_require_nonstring_error_coercion:
-- require() converts non-string loader errors (tables, numbers, etc.)
-- to strings. Reference Lua propagates the original value unchanged.
--
-- BROKEN: stdlib/package.go around lines 217-235 stringifies every
-- loader panic value via vm.ValueToString and wraps with "error loading
-- module ..." text. Both the wrapping AND the stringification are
-- wrong. The loader's panic value must propagate as-is.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   A loader file containing  error({code=42})  raises:
--     pcall(require, "...") -> false, <table {code=42}>
--
-- golua today:
--   pcall(require, "...") -> false, "<file>: error loading module ..."
--   (table converted to string and wrapped)
--
-- Discovered: differential fuzz 2026-05-04 (package wave-3 agent).

local tmp = os.tmpname() .. ".lua"
local f = assert(io.open(tmp, "w"))
f:write("error({code=42})\n")
f:close()

local modname = tmp:match("([^/]+)%.lua$")
local moddir = tmp:match("(.*)/")
package.path = moddir .. "/?.lua;" .. package.path

local ok, err = pcall(require, modname)
os.remove(tmp)

assert(ok == false, "loader error must propagate")
assert(type(err) == "table",
  "error value must propagate as table, not be stringified; got " ..
  type(err) .. ": " .. tostring(err))
assert(err.code == 42, "table value must propagate intact; got code=" ..
  tostring(err.code))

print("ok")

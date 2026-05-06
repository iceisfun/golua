-- broken_fuzz_require_runtime_error_wrap:
-- require() wraps loader RUNTIME errors with "error loading module ..."
-- text. Reference Lua only wraps loader COMPILE/PARSE errors via
-- luaL_loadfile; runtime errors propagate unchanged.
--
-- BROKEN: stdlib/package.go around lines 226-235 wraps every loader
-- error. Should only wrap when loadErr came from compileChunk /
-- loadBinaryChunk; runtime errors from running the chunk should
-- propagate as-is.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   A loader file containing  error("intentional")  raises:
--     "<file>:1: intentional"
--   (no "error loading module ... from file ..." prefix)
--
-- golua today:
--   "error loading module 'X' from file '<file>':
--      <file>:1: intentional"
--
-- Discovered: differential fuzz 2026-05-04 (package wave-3 agent).

local tmp = os.tmpname() .. ".lua"
local f = assert(io.open(tmp, "w"))
f:write("error('intentional')\n")
f:close()

-- Strip ".lua" suffix and use file's directory as the search root.
local modname = tmp:match("([^/]+)%.lua$")
local moddir = tmp:match("(.*)/")
package.path = moddir .. "/?.lua;" .. package.path

local ok, err = pcall(require, modname)
os.remove(tmp)

assert(ok == false, "loader error must propagate")
assert(type(err) == "string", "error must be a string")
assert(err:find("intentional"),
  "expected 'intentional' in error; got: " .. err)
assert(not err:find("error loading module"),
  "runtime errors must NOT be wrapped with 'error loading module'; got: " .. err)

print("ok")

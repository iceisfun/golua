-- broken_fuzz_load_uppercase_mode:
-- load(chunk, name, mode) accepts mode 'B' (uppercase) where Lua 5.5 errors.
--
-- BROKEN: Lua 5.5 raises  bad argument #3 to 'load' (invalid mode)  for any
-- mode string containing characters not in the {'b', 't'} set; in particular
-- "B" / "T" / "BT" / "BB" are rejected. golua silently treats these as the
-- text-only path (no binary acceptance), returning a load error like
-- "attempt to load a text chunk (mode is 'B')" instead of an arg error.
--
-- Note: BOTH impls accept other unrelated chars (e.g., "tT" passes, "btx"
-- passes) — neither does strict equality on {"t","b","bt","tb"}. The bug
-- is specifically that golua doesn't check the case-sensitive 'B' / 'T'
-- which 5.5 explicitly rejects.
--
-- Reference (lua5.5.0):
--   load("return 1", "n", "B")  -> error: bad argument #3 to 'load' (invalid mode)
--
-- golua today:
--   load("return 1", "n", "B")  -> nil, "attempt to load a text chunk (mode is 'B')"
--
-- Discovered: differential fuzz 2026-05-04 (load wave-1 agent).
-- Parity gap, low priority.

local ok, err = pcall(load, "return 1", "n", "B")
assert(ok == false,
  "load with mode='B' should raise a Lua error in 5.5; got load() returning instead")
assert(type(err) == "string" and err:find("invalid mode"),
  "expected 'invalid mode' in error message; got " .. tostring(err))

print("ok")

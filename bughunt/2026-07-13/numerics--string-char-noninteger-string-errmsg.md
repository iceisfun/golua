# checkinteger family: non-integer numeric string args get wrong error message

Systematic across the stdlib: when a *string* argument that converts to a
non-integral number (e.g. `"2.5"`) is passed at a `luaL_checkinteger` site,
reference Lua coerces to number first and reports
`(number has no integer representation)`. golua instead reports
`(number expected, got string)` — misdiagnosing the type. Success/failure
behavior is identical (both error); message text and diagnosis differ.

Integer-representable strings (`"2"`, `"2.0"`, `" 0x2 "`) coerce fine in both,
so only the message path is wrong.

## Repro

```lua
local function try(f, ...) local ok, e = pcall(f, ...) print(ok, e) end
try(string.char, "65.5")
try(string.rep, "ab", "2.5")
try(string.sub, "abc", "1.5")
try(string.byte, "abc", "1.5")
try(string.find, "abc", "b", "1.5")
try(string.gsub, "abc", "b", "x", "1.5")
try(table.insert, {1,2}, "1.5", "x")
try(table.remove, {1,2}, "1.5")
try(table.concat, {1,2}, ",", "1.5")
try(table.move, {1,2}, "1.5", 2, 1)
try(table.unpack, {1,2}, "1.5")
try(math.random, 1, "2.5")
try(math.ult, "1.5", 2)
try(utf8.char, "65.5")
try(utf8.codepoint, "A", "1.5")
try(utf8.offset, "abc", "1.5")
try(utf8.len, "abc", "1.5")
```

## golua output (every line)

```
false   bad argument #N to 'fn' (number expected, got string)
```

## lua5.5.0 output (every line)

```
false   bad argument #N to 'fn' (number has no integer representation)
```

Why wrong: reference `luaL_checkinteger` → `lua_tointegerx` converts numeric
strings via the usual coercion, then the integer-representation check fails and
`interror` distinguishes "is a number but not integral" from "not a number at
all" (lauxlib.c `interror`: `if lua_isnumber → "number has no integer
representation" else typeerror`). `lua_isnumber` is true for numeric strings,
so the reference message is the representation one. golua's equivalent helper
checks the raw type instead. Affects string.*, table.*, math.random/ult,
utf8.* — likely a single shared helper (CheckInteger) to fix.

Severity: error-message-only (behavior identical), but it is a whole-stdlib
family from one helper.

## Verification: REJECTED (ERROR-MSG-ONLY)

Verified 2026-07-13 against /usr/bin/lua5.5.0. The divergence is real and
reproduces exactly as reported, but it is error-message prose only:

- Both interpreters raise the error (pcall returns false) at the same
  argument index with the same function name; only the parenthetical differs
  ("number expected, got string" vs "number has no integer representation").
- All coercion *behavior* is identical: `string.rep("ab", "2.0")` succeeds in
  both; float `2.5` yields the identical "number has no integer
  representation" in both; non-number types yield the identical
  "number expected, got table" in both. Minimized divergence:

  ```lua
  print(pcall(string.rep, "ab", "2.5"))
  -- golua:    false  bad argument #2 to 'string.rep' (number expected, got string)
  -- lua5.5.0: false  bad argument #2 to 'string.rep' (number has no integer representation)
  ```

- Reference mechanism confirmed in lua-5.5.0/src/lauxlib.c:440 (`interror`
  uses `lua_isnumber`, which is true for numeric strings). golua's
  CheckInteger-equivalent checks the raw type before coercing.

Same error, same line, same catchability — per hunt scope rules this is
ERROR-MSG-ONLY, not CONFIRMED. A fix (shared helper: try string→number
coercion before choosing the message) would be a legitimate polish item but
is out of scope for this hunt's confirmed list.

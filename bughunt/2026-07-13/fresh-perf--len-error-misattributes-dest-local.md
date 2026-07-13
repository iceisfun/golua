# `#` error misattributes the temp operand to the destination local

Found by expression fuzzing (gen2 seeds 1016/1026/1033/1038), minimized.

## Repro

```lua
local a = 1
a = #(a + 1)
```

## golua (master, b51a63b)

```
golua: min.lua:2: attempt to get length of a number value (local 'a')
```

## lua5.5.0

```
lua5.5.0: min.lua:2: attempt to get length of a number value
```

## Why it's wrong

The operand of `#` is the temporary holding `a + 1`, not the local `a`; golua
compiles the subexpression directly into `a`'s register (assignment-target
reuse from the in-place-operand codegen work), so the error-variable
attribution names `local 'a'`, which is misleading — reference Lua names
nothing because the value is in a temp. Same shape reproduces for other unary
ops whose operand is a compound expression assigned back to a local.

Severity: error-message-only (wrong variable attribution, right line/type).

## Variants (all same root cause: unop operand compiled into the dest local's register)

```lua
local a = {} a = -(a.x or {})
-- golua: attempt to perform arithmetic on a table value (local 'a')
-- lua  : attempt to perform arithmetic on a table value

local a = 1 a = ~(a + 0.5)
-- golua: number (local 'a') has no integer representation
-- lua  : number has no integer representation
```

`local b = #(a+1)` (fresh local dest) and `t.x = #(t.y or 5)` do NOT
misattribute — only re-assignment to an existing local does.

## Verification: REJECTED (ERROR-MSG-ONLY)

Verified 2026-07-13 against golua master (b51a63b) and /usr/bin/lua5.5.0,
both under `ulimit -v 16777216; timeout 15`. The divergence is REAL and
reproduces exactly as reported (both the `#` repro and the `~(a + 0.5)`
variant), and it is not covered by any `wontfix/` entry and is not
GC/finalization-dependent. However, the ONLY difference is error-message
prose: same error kind ("attempt to get length of a number value"), same
line (2), same exit code, and identical catchability — under `pcall` both
return `false` with the same message modulo the extra `(local 'a')`
suffix:

```
golua:   false  pc.lua:3: attempt to get length of a number value (local 'a')
lua5.5:  false  pc.lua:3: attempt to get length of a number value
```

Per triage policy, prose-only divergences with matching error/line/
catchability are classified ERROR-MSG-ONLY, not CONFIRMED. The
misattribution itself is genuine (reference names nothing because the
unop operand lives in an unnamed temp register; golua's in-place codegen
reuses the destination local's register and the varinfo lookup then names
`local 'a'` whose source-level value at the error point is 1, not the 2
being length-ed), so this is a legitimate polish candidate for the error-
attribution path, just not a semantic bug.


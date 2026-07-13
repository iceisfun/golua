# and/or-produced operand loses variable name in arithmetic error message

Severity: error-message-only

When the failing operand of an arithmetic (or similar) operation was produced by an
`and`/`or` expression whose last register load was a named variable, reference Lua
back-traces the instruction and still names the variable (`(upvalue 'B')`,
`(local 'L')`, `(global 'GG')`). golua omits the name suffix entirely in this shape
(direct uses of the variable DO get the suffix — only the through-and/or path loses it).

## Repro

```lua
local B
print(pcall(function() return (1 and B) + 1 end))
print(pcall(function() return (nil or B) + 1 end))
print(pcall(function() local L; return (1 and L) + 1 end))
print(pcall(function() local L; return (nil or L) + 1 end))
local L2
print(pcall((1 and print) and function() return (1 and GG) + 1 end))
```

## golua output

```
false	m2.lua:2: attempt to perform arithmetic on a nil value
false	m2.lua:3: attempt to perform arithmetic on a nil value
false	m2.lua:4: attempt to perform arithmetic on a nil value
false	m2.lua:5: attempt to perform arithmetic on a nil value
false	m2.lua:7: attempt to perform arithmetic on a nil value
```

## lua5.5.0 output

```
false	m2.lua:2: attempt to perform arithmetic on a nil value (upvalue 'B')
false	m2.lua:3: attempt to perform arithmetic on a nil value (upvalue 'B')
false	m2.lua:4: attempt to perform arithmetic on a nil value (local 'L')
false	m2.lua:5: attempt to perform arithmetic on a nil value (local 'L')
false	m2.lua:7: attempt to perform arithmetic on a nil value (global 'GG')
```

## Why it's wrong

Reference's `varinfo`/`luaG_opinterror` traces the register back through TESTSET/MOVE
(from OP_GETUPVAL/GETTABUP/MOVE) to name the originating variable; golua's debug-name
resolution gives up when the value flowed through an and/or, dropping a diagnostic that
reference preserves. Message-suffix only; error line/type/values all match.

Found by random expression-tree differ (seeds 941, 1453), 2026-07-13 codegen-reg lens.

## Verification: REJECTED (ERROR-MSG-ONLY)

Verified 2026-07-13 (adversarial re-run, golua master vs /usr/bin/lua5.5.0). The divergence
is REAL and reproduces exactly as reported — all five lines lose the name suffix on golua
while reference names the upvalue/local/global in every case. Control confirmed the trigger
is specifically the and/or-produced operand: a direct `return L + 1` in the same closure DOES
get `(upvalue 'L')` on golua, so golua's varinfo works but gives up tracing through the
and/or register copy (TESTSET/MOVE chain).

Minimized repro (2 lines, no pcall/closure needed):

```lua
local L
return (1 and L) + 1
```

golua:    `min2.lua:2: attempt to perform arithmetic on a nil value`
lua5.5.0: `min2.lua:2: attempt to perform arithmetic on a nil value (local 'L')`

However, the ONLY difference is the error-message prose suffix: same error type
("attempt to perform arithmetic on a nil value"), same line (2), same catchability
(pcall returns false + message in both), same values/control flow. Not in wontfix/,
not GC-dependent. Per verification policy, message-prose-only divergences are classified
ERROR-MSG-ONLY rather than CONFIRMED. Legitimate polish item if debug-name resolution is
ever extended to trace through TESTSET/MOVE, but not a semantic bug.

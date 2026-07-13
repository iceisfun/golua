# error-message-only: variable-name attribution lost when errant operand flows through and/or

Reference Lua's getobjname walks back through the TESTSET/and-or copy pattern
to name the variable that supplied a bad operand; golua names the variable in
direct uses (`b % 1` → "(upvalue 'b')" works) but drops the suffix when the
value passes through an `and`/`or` expression first. Found by a dump/load
round-trip generator (seeds 72/84), but unrelated to dump — pure error-message
attribution.

## Repro
```lua
local b = {}
print(pcall(function() return (1 and b) % 1 end))
print(pcall(function() local l = {} return (1 and l) % 1 end))
print(pcall(function() return (1 and GLOB) % 1 end))
```

## golua
```
false	t.lua:2: attempt to perform arithmetic on a table value
false	t.lua:3: attempt to perform arithmetic on a table value
false	t.lua:4: attempt to perform arithmetic on a nil value
```

## lua5.5.0
```
false	t.lua:2: attempt to perform arithmetic on a table value (upvalue 'b')
false	t.lua:3: attempt to perform arithmetic on a table value (local 'l')
false	t.lua:4: attempt to perform arithmetic on a nil value (global 'GLOB')
```

Severity: error-message-only (line numbers, error type, and pcall behavior all
match; only the "(kind 'name')" suffix is missing).

## Verification: REJECTED (ERROR-MSG-ONLY)

Verified 2026-07-13 (golua master build vs /usr/bin/lua5.5.0). The divergence
is real and reproduces exactly as reported, but it falls under the
error-message-prose-only triage rule, so it is not a CONFIRMED semantic bug:

- Same error type ("attempt to perform arithmetic on a <type> value"),
  same source line, same catchability (pcall returns false in both).
- The only difference is the missing "(local/upvalue/global 'name')" suffix.
- Control case verified: without and/or, golua DOES attribute the name
  (`return l % 1` → "(local 'l')" matches the reference exactly), so the
  omission is specific to operands flowing through and/or (TESTSET copy) —
  golua's debug-name walk does not trace back through the and/or result
  register the way reference `getobjname` does.
- Not in wontfix/ index; not GC/finalization-timing dependent.

Minimized repro (2 lines, uncaught, main chunk, local variable):

```lua
local l = {}
return (1 and l) % 1
```

golua:    `min.lua:2: attempt to perform arithmetic on a table value`
lua5.5.0: `min.lua:2: attempt to perform arithmetic on a table value (local 'l')`

Legitimate polish target for an error-message parity pass, but out of scope
for this hunt's CONFIRMED bar.

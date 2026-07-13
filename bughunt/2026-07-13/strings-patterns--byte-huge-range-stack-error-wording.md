# string.byte over a huge range: error wording "C stack overflow: N slots exceeds limit" vs "stack overflow (string slice too long)"

**Severity: error-message-only** (both raise a catchable error at the same point)

## Repro

```lua
local s = string.rep("x", 1000000)
print(pcall(string.byte, s, 1, 1000000))
```

## golua output
```
false	C stack overflow: 1000028 slots exceeds limit 1000000
```

## lua5.5.0 output
```
false	stack overflow (string slice too long)
```

## Why it diverges

Reference `str_byte` calls `luaL_checkstack(L, n, "string slice too long")`
which produces `stack overflow (string slice too long)`. golua's
`stringByte` grows via `v.EnsureStack`, whose generic limit error leaks the
VM-internal slot arithmetic. Wording only; both are catchable and fire on
the same inputs (n = 100000 succeeds on both, 1000000 fails on both).

## Verification: REJECTED (ERROR-MSG-ONLY)

Verified 2026-07-13 against golua master and /usr/bin/lua5.5.0. The divergence
reproduces exactly as reported (slot count in the message varies with ambient
frame usage: observed `1000006`/`1000007` vs the report's `1000028` — same
message shape), but the ONLY difference is the error-message prose:

Minimized repro (one line):
```lua
print(pcall(string.byte, string.rep("x", 1000000), 1, 1000000))
```
- golua:    `false	C stack overflow: 1000006 slots exceeds limit 1000000`
- lua5.5.0: `false	stack overflow (string slice too long)`

Everything else is identical:
- **Same catchability**: pcall returns `false, <msg>` on both.
- **Same limit**: binary search of the largest succeeding `string.byte(s,1,n)`
  gives 999990 (golua) vs 999983 (lua5.5.0) — both are LUAI_MAXSTACK=1,000,000
  minus ambient frame overhead; reference does not specify the exact ceiling.
- **Same uncaught behavior**: both exit 1 with `file:1:` line attribution and
  an identical traceback shape (`[C]: in field 'byte'` / `in main chunk`).

Mechanism: reference `str_byte` (lstrlib.c:177) calls
`luaL_checkstack(L, n, "string slice too long")`; golua's stack-growth guard
in `vm/vm_exec.go:2042` panics with its generic
`"C stack overflow: %d slots exceeds limit %d"` wording instead. Not one of
the documented wontfix divergences (`wontfix/load-stack-overflow-traceback`
covers the parser/compile-time C-stack wording family, not this runtime path).

If wording parity were ever wanted, `stringByte` could pre-check the requested
count against the stack limit and raise `stack overflow (string slice too
long)` itself — but per campaign policy this is message-prose only, so no fix.

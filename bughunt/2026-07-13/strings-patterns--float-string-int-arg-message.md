# Error message for non-integral numeric-string integer args: "number expected, got string" vs "number has no integer representation"

**Severity: error-message-only** (behavior identical: both raise a catchable error; integral strings like "3.0", "0x3", "3e2" are accepted identically by both)

## Repro

```lua
print(pcall(string.rep, "a", "3.5"))
print(pcall(string.find, "abcdef", "c", "2.5"))
```

## golua output
```
false	...: bad argument #2 to 'string.rep' (number expected, got string)
false	...: bad argument #3 to 'string.find' (number expected, got string)
```

## lua5.5.0 output
```
false	...: bad argument #2 to 'rep' (number has no integer representation)
false	...: bad argument #3 to 'find' (number has no integer representation)
```

## Why it diverges

Reference `luaL_checkinteger` first coerces the string to a number (3.5),
then fails the float→integer conversion, producing "number has no integer
representation". golua's `getInt` fails the string→integer coercion outright
and reports the argument as a non-number. Same catchable-error behavior,
different diagnostic. (Note: `string.pack` already gets this right —
`string.pack("i4", "3.5")` reports "number has no integer representation" on
both.)

## Verification: REJECTED (ERROR-MSG-ONLY)

Verified 2026-07-13 against `/usr/bin/lua5.5.0`. The divergence is real and
reproduces exactly as reported, but the ONLY difference is error-message prose:

Minimized repro:
```lua
print(pcall(string.rep, "a", "3.5"))
```
- golua:    `false	bad argument #2 to 'string.rep' (number expected, got string)`
- lua5.5.0: `false	bad argument #2 to 'string.rep' (number has no integer representation)`

Everything else is identical between the two interpreters:
- Same error raised at the same line; uncaught form produces identical
  tracebacks (`...uncaught.lua:1: bad argument #2 to 'rep' (...)`,
  `[C]: in field 'rep'`) and identical exit code 1.
- Same catchability (pcall returns `false, <msg>` in both).
- Behavior on accepted inputs matches exactly: `"3"`, `"0x10"`, `" 3 "`
  all coerce successfully in both; a real float `3.5` already yields the
  correct "number has no integer representation" message in golua.
- Also affects `string.sub`/`string.byte`/`string.find` (any luaL_checkinteger
  string-coercion path), all message-prose-only.

Not in `wontfix/`; not GC-dependent. Per triage rules (same error, same line,
same catchability, prose-only difference) the verdict is ERROR-MSG-ONLY, not
CONFIRMED. If message parity is ever wanted, the fix is in golua's integer-arg
coercion (`getInt`-style helpers): coerce string→number first, then fail the
float→integer step with "number has no integer representation", as
`string.pack` already does.

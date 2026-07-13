# string.unpack with ≥252 values leaks a Go runtime index-out-of-range panic

**Severity: wrong-error-behavior** (internal Go runtime error surfaces to Lua; reference succeeds and returns the values — divergence begins at just 252 results)

## Repro

```lua
-- 251 values: works on both.
-- 252 values: golua raises "runtime error: index out of range [256] with length 256";
--             reference returns all 252 values + position.
local n = 252
local ok, e = pcall(function()
  return select("#", string.unpack(string.rep("B", n), string.rep("\7", n)))
end)
print(ok, e)
-- Unprotected, the whole script dies with the Go runtime error:
-- local t = {string.unpack(string.rep("B", 300), string.rep("\7", 300))}
```

## golua output
```
false	runtime error: index out of range [256] with length 256
```
Unprotected top level:
```
golua: runtime error: index out of range [256] with length 256
stack traceback:
	[C]: in field 'unpack'
	(command line):1: in main chunk
	[C]: in ?
```

## lua5.5.0 output
```
true	253
```
(Reference handles up to hundreds of thousands of results; at ~1e6 it raises a
proper catchable `stack overflow (too many results)` error.)

## Why it's wrong

`stringUnpack` (stdlib/string_pack.go) writes each decoded value with
`v.Set(nret, val)` without ever growing the stack. Unlike `stringByte`, which
calls `v.EnsureStack(v.Base()+n)` before its loop, the unpack loop runs off
the end of the fixed 256-slot frame and triggers a raw Go slice
index-out-of-range panic. The error text exposes internal Go runtime detail,
and well-formed programs that unpack ≥252 values (e.g. reading a binary
record table) fail where reference Lua succeeds.

Fix sketch: count data directives (or grow lazily) and call
`v.EnsureStack` like `stringByte` does; convert an over-limit request into
the reference's catchable "too many results"-style error.

## Verification: CONFIRMED (2026-07-13)

Independently reproduced on both interpreters (under `ulimit -v 16GB` + `timeout 15`):

- Threshold verified by sweep: n=251 (252 results) works on golua; n=252
  (253 results) panics with `runtime error: index out of range [256] with
  length 256`. Reference returns `253` at n=252.
- Unprotected variant (`{string.unpack(string.rep("B",300), ...)}`) kills the
  golua process (exit 1) with the Go runtime error in the traceback; reference
  prints `301` and exits 0.
- Oracle scaling verified: lua5.5.0 succeeds at 100,000 results (`true 100001`)
  and at 1,200,000 results raises the proper catchable
  `stack overflow (too many results)` error. Reference source confirms:
  `src/lstrlib.c:1789` in str_unpack does
  `luaL_checkstack(L, 2, "too many results")` per decoded value.
- Mechanism verified: `VM.Set` (vm/vm_access.go:31) indexes `vm.stack[base+idx]`
  with no growth; `stringUnpack` (stdlib/string_pack.go:553) calls
  `v.Set(nret, val)` per value with no `EnsureStack`. The `stringByte`
  contrast (stdlib/string.go:227 calls `v.EnsureStack(v.Base()+n)`) is accurate.
- Not a wontfix entry (checked wontfix/README.md; `load-stack-overflow-traceback`
  is about load() error prose, unrelated). Not GC/finalization-dependent. Not
  error-message-only: reference *succeeds* where golua fails, so this is a
  behavioral divergence on well-formed programs, plus a leaked Go runtime
  error message.

# debug.getupvalue on a gmatch iterator: upvalue 3 is nil (reference: userdata)

**Severity: introspection-only** (debug-library divergence; no functional impact
on iteration; no error message involved — the original "error-message-only"
label was a mislabel)

## Verification: CONFIRMED (2026-07-13)

Reproduced exactly as reported on golua master vs `/usr/bin/lua5.5.0`.
Not in `wontfix/`, not GC/finalization-timing dependent, not error-message
prose (no error is raised at all).

## Minimized repro

```lua
local n, v = debug.getupvalue(string.gmatch("", ""), 3)
print(n, type(v))
```

golua:
```
	nil
```

lua5.5.0:
```
	userdata
```

(Both agree the upvalue *exists* — name is `""`, as for all C-function
upvalues — but golua's slot holds nil while reference holds a full userdata.)

## Original repro (verified)

```lua
local it = string.gmatch("hello", "l+")
for i = 1, 4 do
  local name, val = debug.getupvalue(it, i)
  print(i, name, type(val))
end
```

golua output:
```
1		string
2		string
3		nil
4	nil	nil
```

lua5.5.0 output:
```
1		string
2		string
3		userdata
4	nil	nil
```

## Why it diverges

Reference `gmatch` (`lstrlib.c:857-869` in 5.5.0) keeps the two strings on
the stack, allocates a `GMatchState` full userdata via `lua_newuserdatauv`
(line 864), and pushes `gmatch_aux` as a C closure with **3** upvalues
(line 869): subject string, pattern string, state userdata.

golua's `stringGmatch` (`stdlib/string.go:599`) deliberately mirrors that
layout — `vm.NewNativeFuncWithNups(..., 3)` plus
`iter.SetNativeFuncUpvalue(1, subject)` / `(2, pattern)` — but leaves
upvalue 3 unpopulated; the code comment at `stdlib/string.go:656` even
notes "Upvalue 3 is internal position state (userdata in Lua 5.4)". The
actual iterator state (`pos`, `lastMatch`) lives in Go closure variables,
so the declared third upvalue slot reads as nil.

## Scope note

The upvalue layout of stdlib C functions is an implementation detail the
manual does not promise; a stricter reading could call this unspecified.
Confirmed anyway because (a) golua already commits to reference's layout
(3 declared upvalues, first two populated identically) and only the third
slot deviates, so the current state is half-done parity rather than a
different design; and (b) this project treats debug-introspection parity
as in-scope (cf. the 2026-06-06 for-loop slot-layout work done specifically
for `debug.getlocal` parity). A faithful fix would store a userdata (or
any non-nil stand-in mirroring reference's state object) in upvalue 3.

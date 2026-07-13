# string.gsub with a number subject returns the number (not a string) when nothing is replaced

**Severity: wrong-result** (wrong return TYPE — `type()` observable, breaks downstream string ops)

## Minimized repro

```lua
print(type(string.gsub(1, "x", "y")))
```

golua: `number` — lua5.5.0: `string`

## Full repro (all no-change paths)

```lua
print(type(string.gsub(123, "9", "X")))    -- no match
print(type(string.gsub(123, "1", "1")))    -- replacement text equals match
print(type(string.gsub(123, "%d", "%0")))  -- %0 keeps original text
print(type(string.gsub(123, "%d", function() return nil end)))
print(type(string.gsub(12.5, "9", "X")))
```

## golua output
```
number
number
number
number
number
```

## lua5.5.0 output
```
string
string
string
string
string
```

## Why it's wrong

Reference `str_gsub` (lstrlib.c:945, lua-5.5.0) has the same "no changes →
return original" shortcut (`if (!changed) lua_pushvalue(L, 1)`), but its
`luaL_checklstring(L, 1, &srcl)` **converts the stack slot in place**: a
number subject becomes a string on the stack before `lua_pushvalue(L, 1)`
runs, so the reference always returns a string. golua's `stringGsub`
(stdlib/string.go:458-459, the `if !changed { v.Set(0, v.Get(1)) }` identity
optimization) coerces via `getString` WITHOUT rewriting the argument slot,
so the raw number leaks through as the return value. When a substitution
does change text golua correctly returns a string, so only the
no-change paths diverge. Downstream effects: `select(1, string.gsub(123,
"9", "X"))` has type `number`; `(...):len()`, string identity comparisons,
`type()` dispatch etc. then behave differently.

Fix sketch: only take the identity shortcut when `v.Get(1)` is already a
string; otherwise return `vm.NewString(s)`.

## Verification: CONFIRMED (2026-07-13)

Reproduced as reported with the scratchpad golua binary vs `/usr/bin/lua5.5.0`
(both under `ulimit -v 16GB` + `timeout 15`): all five full-repro lines print
`number` on golua, `string` on reference; the one-line minimized repro
diverges the same way. Second return (count) is unaffected (`0` on both).
Control: `string.gsub(123, "1", "X")` (text actually changes) returns
`string` on both. Not in `wontfix/` (checked index), no GC/finalization
involvement, not an error-message issue. The reference behavior is also the
documented one: the manual specifies gsub "returns a copy of s", i.e. a
string after argument coercion.

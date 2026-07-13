# table.remove(t, #t+1) does not fetch the returned value via __index

Severity: wrong-result. Note this does NOT require `__index` to manifest: with a
`__len` that understates the raw contents (e.g. `__len` returns 2 on a raw
`{10,20,30}`), `table.remove(t, 3)` returns `30` and clears `t[3]` in reference
Lua, while golua returns `nil` and leaves the table untouched — a wrong result
on plain raw data.

`table.remove` with `pos == #t+1` (the explicitly-allowed past-the-end position)
must still return `t[pos]` fetched with a metamethod-aware read (reference
implementation does `lua_geti(L, 1, pos)` unconditionally). golua returns nil
without consulting `__index`.

## Repro

```lua
local t = setmetatable({10,20,30}, {__len=function() return 2 end})
print(table.remove(t, 3), rawget(t, 3))   -- pos == #t+1, raw value present
local u = setmetatable({}, {__len=function() return 2 end, __index=function(_,k) return k*2 end})
print(table.remove(u, 3))                 -- pos == #t+1, value via __index
```

## golua output

```
nil	30
nil
```

## lua5.5.0 output

```
30	nil
6
```

## Root cause

`stdlib/table.go` `tableRemove` (~line 212): the `pos > length` early path
returns nil without doing the `lua_geti(t, pos)` / `lua_seti(t, pos, nil)` pair
that reference `tremove` performs unconditionally.

All other paths (pos == #t, shifting reads/writes, the final nil-out via
__newindex) are metamethod-aware and match the reference; only the
`pos == size+1` early path skips the `__index` fetch.

## Additional effect: the `t[pos] = nil` store is skipped too

Reference `tremove` always executes `lua_geti(t, pos)` then `lua_seti(t, pos, nil)`,
even for `pos == size+1`. With a logging proxy (`__len`=0, logging
`__index`/`__newindex` that rawsets):

```lua
local gets, sets = {}, {}
local t = setmetatable({}, {
  __len = function() return 0 end,
  __index = function(_, k) gets[#gets+1] = k; return "G"..k end,
  __newindex = function(tt, k, v) sets[#sets+1] = k.."="..tostring(v); rawset(tt, k, v) end,
})
print(table.remove(t, 1), #gets, #sets)
```

golua: `nil 0 0` (no metamethod traffic at all)
lua5.5.0: `G1 1 1` (gets=[1], sets=[1=nil])

## Verification: CONFIRMED (2026-07-13)

Independently re-run adversarially against `/usr/bin/lua5.5.0`; all three repros
in this file diverge exactly as claimed (golua master, `bash -c 'ulimit -v
16777216; timeout 15 ...'` on both interpreters, both exit 0).

Minimized repro (2 lines, no `__index` needed — wrong result on plain raw data):

```lua
local t = setmetatable({10,20,30}, {__len=function() return 2 end})
print(table.remove(t, 3), rawget(t, 3))
```

- golua:    `nil	30`
- lua5.5.0: `30	nil`

Oracle-correctness check: reference `tremove`
(`~/Downloads/lua-5.5.0/lua-5.5.0/src/ltablib.c`) explicitly permits
`pos == size+1` via `(lua_Unsigned)pos - 1u <= (lua_Unsigned)size`, then
unconditionally runs `lua_geti(L, 1, pos)` (metamethod-aware read → the return
value) and `lua_pushnil; lua_seti(L, 1, pos)` (metamethod-aware nil-out). golua's
`tableRemove` early path at `stdlib/table.go:213-216` (`if pos > length { return
nil }`) skips both, so the raw value at `#t+1` is neither returned nor cleared,
and `__index`/`__newindex` are never consulted.

Scope checks: NOT the `length-operator-border` wontfix — `#t` here is a
deterministic `__len` metamethod (both interpreters agree `#t == 2`), no border
ambiguity. Not GC/finalization-timing dependent. Not error-message-only (values
and table state differ).


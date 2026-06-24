# length-operator-border

## What

`#t` on a table that is **not a sequence** (it has a `nil` hole in the middle)
can return a different value than reference Lua:

```lua
local t = {}
for i = 1, 100 do t[i] = i end
t[50] = nil
print(#t)   -- golua: 49   reference: 100
```

Both answers are **correct**. The Lua manual defines `#t` for a non-sequence as
*any* border — an index `n` with `t[n] ~= nil` and `t[n+1] == nil`. Here both
`49` (t[49] set, t[50] nil) and `100` (t[100] set, t[101] nil) are borders, so
either is a conforming result.

For a proper sequence (no holes), `#` is well-defined and golua and the reference
always agree.

## Why this won't change

This is explicitly implementation-defined behavior. The Lua 5.4/5.5 manual,
§3.4.7:

> The length operator applied on a table returns a border. […] When `t` is not a
> sequence, `#t` can return any of its borders.

The specific border each implementation returns falls out of its internal table
representation. golua stores the array part as a Go slice (grown via `append`)
and binary-searches for a border, whereas reference Lua tracks an array-size
hint and uses power-of-two growth — so they land on different borders for a
holed table. Matching the reference exactly would require replicating its
array-part sizing/`lenhint` machinery, a large and risky rework to mimic
behavior that is, by definition, not guaranteed.

Programs must not use `#` on tables with holes; use an explicit count or store
the length yourself.

## Where this lives in the source

- [`vm/table.go`](../../vm/table.go) — `getn` / border search and
  `hashSearchBorder` (around lines 718–800).
- [`vm/vm_table.go`](../../vm/vm_table.go) — `ObjLen`.

## Note (5.4 vs 5.5)

Lua 5.5 changed the border heuristic (first hole from `asize/2`). golua's
`master` targets 5.5 semantics and the `lua_5_4_8` branch targets 5.4 — verify
any `#`-border expectation against the matching reference (`lua5.5.0` /
`lua5.4.8`), never an in-tree assumption.

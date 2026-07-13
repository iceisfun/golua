# Metamethod call clobbers live caller registers after a table constructor consumed an open call (stale vm.top)

Severity: wrong-result (SILENT data corruption)

After a table constructor whose last element is an open (multi-return) call —
`{1, mr()}` — `vm.top` is left at the SETLIST consumption level instead of being
restored to `frame.base + proto.MaxStack`. Any later metamethod dispatch to a
**Lua closure** (`__add`, `__concat`, `__index`, ...) places the callee frame at
`base = vm.top` (vm/vm_exec.go `call()`), which now sits BELOW live caller
registers. The callee's parameter writes / post-return dead-slot clearing nil
out those live registers. Result: silently wrong values (nil) with no error, or
bogus "attempt to concatenate a nil value" errors masking the real evaluation.

## Repro (4 lines, top level)

```lua
local function mr() return 1, 2, 3 end
local obj = setmetatable({}, {__add = function() return 1000 end})
local t = {1, mr()}
print("a", "b", "c", "d", (obj + 1))
```

## golua output

```
a	b	c	nil	1000
```

## lua5.5.0 output

```
a	b	c	d	1000
```

The string constant `"d"`, already loaded into its argument register, is
silently replaced by nil.

## More shapes (same root cause)

Wrong values inside a table constructor:

```lua
local function mr() return 1,2,3 end
local obj = setmetatable({}, {__concat=function() return "CC" end})
local t = {1, mr()}
local u = {"a", "b", "c", "d", "e", (obj .. 1)}
print(u[1], u[2], u[3], u[4], u[5], u[6])
-- golua:    a  b  c  nil nil CC
-- lua5.5.0: a  b  c  d   e   CC
```

Wrong error type/line (the original fuzz hit, seeds 433/1183/2180 of gen2):

```lua
local function mr() return 1,2,3 end
local function id(...) return ... end
local obj = setmetatable({}, {__concat=function(a,b) return "CC" end})
local N = 7
local function main(...)
  local x0 = 1
  local t = {1, mr()}
  local y2 = (1 + id((x0 .. (2 .. obj .. N))))
  print(t[2], y2)
end
print(pcall(main, 1, "two", nil, 4))
-- golua:    false  FILE:8: attempt to concatenate a nil value
-- lua5.5.0: false  FILE:8: attempt to add a 'number' with a 'string'
```

## Why it's wrong

Reference Lua keeps `L->top` at `ci->top` (>= all frame registers) whenever a
metamethod can be called from the VM loop, so `luaD_call` never overlaps the
caller's register file. golua's `vm.top` is lowered by the open-call/SETLIST
sequence and never re-raised, so `callMetamethodN` -> `vm.call` (base = vm.top)
builds the callee frame on top of live caller registers; `vm.call`'s
end-of-call dead-slot clearing (`clearEnd = base + proto.MaxStack`) then nils
them.

Only metamethods implemented as Lua closures trigger it (native-func
metamethod path also starts at vm.top — likely affected the same way).

## Scope matrix (probe: `local t = {1, mr()}` then `print("a","b","c","d", X)`)

Constructs that leave vm.top stale (trigger):
- `{1, mr()}` — constructor ending in open call (OP_SETLIST B=0): TRIGGERS
- `{...}` — vararg constructor: TRIGGERS (clobbers two slots: `c` and `d`)
- `local a,b,c,d,e = mr()`, `tostring(mr())`, `select('#',...)`, `print(mr())`,
  bare `mr()`, generic-for, plain concat: do NOT trigger.

Dispatches that clobber while vm.top is stale (X):
- `(obj + 1)` __add: CLOBBERS -> `a b c nil 1000`
- `(obj .. 1)` __concat: CLOBBERS
- `obj.k` __index (closure): CLOBBERS -> `a b c nil 7`
- `(obj < obj)` __lt: CLOBBERS
- `(#obj)` __len: CLOBBERS
- `obj(1)` __call: safe (goes through register-based OP_CALL path)
- `(obj == obj)` __eq: not observed to clobber in this probe

## No user metamethods needed: built-in string-arith coercion clobbers too

The same stale-vm.top dispatch fires on plain **string arithmetic coercion** —
no metatables involved at all. Local `c` is silently overwritten with `0`
(the coercion machinery's argument/result), not even nil:

```lua
local function g() return 1, 2 end
local function main(...)
  local x0 = 1
  local t = {g()}
  local a, b, c = "A", "B", "C"
  local y = "21" % 3
  print(a, b, c, y)
end
print(pcall(main, 1, "two", nil, 4))
-- golua:    A  B  0  0   / true
-- lua5.5.0: A  B  C  0   / true
```

Also observed corrupting an OP_SELF receiver: after `local t = {obj(1, obj)}`
(open __call in constructor), `obj:m("21" % 3)` calls `m` with `self` = number
-> "attempt to index a number value (local 'self')" inside a method that
reference executes fine (gen4 seed 5586).

Found via random expression-tree differ, 2026-07-13 codegen-reg lens (gen2 seeds
433, 1183, 2180; gen4 seed 5586).

## Verification: CONFIRMED (2026-07-13, adversarial re-check)

Both headline repros re-run independently against a fresh `go build ./cmd/lua`
of master HEAD (b51a63b) and `/usr/bin/lua5.5.0`; outputs diverge exactly as
reported. Not in `wontfix/` (checked index), no GC/finalization dependence, and
it is silent wrong-result corruption, not error-message prose.

Minimized to 4 lines, top level, no functions and no metatables:

```lua
local t = {...}
local a = "A"
local y = "21" % 3
print(a, y)
```

- golua:    `0	0`   (local `a` silently overwritten)
- lua5.5.0: `A	0`

With more locals the clobber pattern shows the coercion frame's own values
being written over named locals (`local a, b = "A", "B"` + `local y = "2" + 3`
prints `5	2	5` on golua vs `A	B	5` on reference), confirming the callee
frame is being built on top of live caller registers at stale `vm.top`.

Trigger matrix re-verified: `{g()}` with a single-return `g` also triggers
(any open call in a constructor, not just multi-return), while
`local a,b = ...`-style open calls outside constructors do not. Metamethod
flavor (`__add` Lua closure after `{1, mr()}`) reproduces verbatim:
golua `a b c nil 1000` vs reference `a b c d 1000`.

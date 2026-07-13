# coroutines: debug.getlocal on frame below a suspended __index shows extra live temporary

When a coroutine is suspended inside an __index metamethod, debug.getlocal on
the *caller* frame (the Lua frame whose table access fired the metamethod)
reports one more temporary than reference, and exposes the table operand where
reference reports a single nil temporary.

## Repro

```lua
local t = setmetatable({}, {__index=function(tbl, key) local mmlocal = "MM" coroutine.yield() return 1 end})
local co = coroutine.create(function() local outer = "OUT" local v = t.someKey return v end)
coroutine.resume(co)
for lvl=0,4 do
  local info = debug.getinfo(co, lvl, "nS")
  if not info then break end
  print("lvl", lvl, info.what, info.name or "-")
  local j=1
  while true do
    local n,v = debug.getlocal(co, lvl, j)
    if not n then break end
    print("  local", j, n, tostring(v))
    j=j+1
  end
end
```

## golua (lvl 2 differs)

```
lvl	2	Lua	-
  local	1	outer	OUT
  local	2	(temporary)	table: 0xc00013b220
  local	3	(temporary)	nil
```

## lua5.5.0

```
lvl	2	Lua	-
  local	1	outer	OUT
  local	2	(temporary)	nil
```

Debug-introspection divergence: golua's stack-top accounting for a frame
suspended in a metamethod exposes an extra register (the table operand) as a
live temporary. Low severity (debug API only), but tooling that walks locals
sees different state.

## Same family: other suspension shapes

Suspended at `local s = "x" .. coroutine.yield()` (concat RHS), golua reports
temporaries `[nil, "x"]` where reference reports just `["x"]`; suspended at
`t[coroutine.yield()]=1` (index key), golua reports an extra `(temporary) nil`
that reference does not. Call-arg / return / condition / numeric-for-limit
suspension shapes match reference exactly. The pattern: whenever the frame is
suspended mid-expression with operands below the yield, golua's frame-top for
debug.getlocal is one-or-two slots higher than reference's.

```lua
local function dump(co, tag)
  print("== "..tag)
  local j=1
  while true do
    local n,v = debug.getlocal(co, 1, j)
    if not n then break end
    print(j, n, tostring(v))
    j=j+1
  end
end
local c2 = coroutine.create(function() local a="A" local s = "x" .. coroutine.yield() end)
coroutine.resume(c2); dump(c2, "concat-rhs")
local c4 = coroutine.create(function() local a="A" local t={} t[coroutine.yield()]=1 end)
coroutine.resume(c4); dump(c4, "index-key")
```

## Verification: REJECTED (NOT-A-BUG)

The output divergence is real and reproduces exactly as reported, but the
diagnosis is wrong and the behavior is not a bug.

**Not a suspension/accounting issue.** The identical divergence appears with no
coroutine, no metamethod, and no suspension — plain `debug.getlocal(2, j)` on a
live caller frame mid-expression:

```lua
local function probe()
  local j = 1
  while true do
    local n, v = debug.getlocal(2, j)
    if not n then break end
    print(j, n, tostring(v))
    j = j + 1
  end
  return ""
end
local function f() local a = "A" local s = "x" .. probe() end
f()
-- golua:   1 a A / 2 (temporary) nil / 3 (temporary) x
-- lua5.5.0: 1 a A / 2 (temporary) x
```

**Actual mechanism: register allocation, not frame-top accounting.** Bytecode
comparison (`luac5.5.0 -l` vs `go run ./cmd/luac`) shows both interpreters
apply the same "(temporary) up to frame limit" rule (reference:
`luaG_findlocal`, ldebug.c:198 — `limit = ci->next->func.p`, i.e. exactly where
the callee's function value was pushed) to their *own* frames; the frames
genuinely differ because golua's codegen uses more registers:

- `local v = t.x` (t = non-`_ENV` upvalue): reference fuses to
  `GETTABUP 1 0 1` (2 slots); golua emits `GETUPVAL 1 0; GETFIELD 1 1 1;
  MOVE 2 1` (3 slots, table materialized in R1). Hence the "extra" temporary
  and the table value.
- concat-rhs / index-key: golua evaluates the expression into a fresh temp and
  `MOVE`s it into the local's register (4 slots vs reference's 3, which
  targets the local directly). Hence the shifted/extra temporaries.

**Why this is not in-scope.** The count and contents of unnamed `(temporary)`
slots visible through `debug.getlocal` are unspecified artifacts of the code
generator (they change between reference Lua versions too). golua's
`debug.getlocal` faithfully reports golua's actual frame; there is no stale or
phantom slot. "Fixing" the observable difference would require bytecode-level
register-allocation parity with luac across all expression shapes — a codegen
project, not a defect. Severity claim "wrong-result" is overstated: only
debug-API introspection of unnamed temporaries differs; named locals, frame
count, and all program-visible semantics match.

**Possible follow-up (perf backlog, not a bug):** the `temp + MOVE`-into-local
pattern and the missing GETTABUP fusion for non-`_ENV` upvalue tables each cost
an instruction and a register; compiling into the local's register directly
(as the 2026-07 compileBinop in-place work already does for binops) would
incidentally converge these debug.getlocal outputs too.

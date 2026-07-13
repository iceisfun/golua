# coroutines: thread values are accepted by table.unpack / table.sort / ipairs (and mis-messaged by table.insert/move)

golua represents a coroutine/thread as a Go `*vm.Table` with an `IsThread()`
flag. Several table-library and iterator entry points check "is it a table?"
without also rejecting thread-flagged tables, so they silently operate on a
coroutine value where reference Lua raises "table expected, got thread".

## Repro

```lua
local co = coroutine.create(function() coroutine.yield() end)
coroutine.resume(co)
print("unpack:", pcall(table.unpack, co))
print("sort:  ", pcall(table.sort, co))
print("ipairs:", pcall(function() for _ in ipairs(co) do end end))
print("insert:", pcall(table.insert, co, 1))
print("move:  ", pcall(table.move, {1,2,3}, 1, 3, 1, co))
```

## golua

```
unpack:	true
sort:  	true
ipairs:	true
insert:	false	attempt to index a thread value
move:  	false	attempt to index a thread value
```

## lua5.5.0

```
unpack:	false	attempt to get length of a thread value
sort:  	false	bad argument #1 to 'table.sort' (table expected, got thread)
ipairs:	false	attempt to index a thread value
insert:	false	bad argument #1 to 'table.insert' (table expected, got thread)
move:  	false	bad argument #5 to 'table.move' (table expected, got thread)
```

`table.unpack`, `table.sort`, `table.concat`, `ipairs`, and `table.move` with
the thread as **source** all **succeed** in golua (wrong result: they should
error). `table.insert`, `table.remove`, and `table.move` with the thread as
**destination** do error but with the wrong message (missing the
`bad argument #N to '...' (table expected, got thread)` form) —
error-message-only for those.

Full matrix (golua succeed vs reference error):

```
                golua            lua5.5.0
table.unpack    true             error (get length of a thread value)
table.sort      true             error (table expected, got thread)
table.concat    true (empty)     error (table expected, got thread)
ipairs          true (0 iters)   error (index a thread value)
table.move src  true             error (table expected, got thread)
table.insert    wrong-msg error  error (bad argument #1 ... got thread)
table.remove    wrong-msg error  error (bad argument #1 ... got thread)
table.move dst  wrong-msg error  error (bad argument #5 ... got thread)
```

Root cause: golua's thread is a real table internally; the `AsTable()`-based
argument checks in the table lib / ipairs don't exclude `IsThread()` tables the
way reference's `lua_istable` excludes `LUA_TTHREAD`. A program that abstracts
over "container or thread" gets different control flow; `for k,v in ipairs(co)`
runs a zero-iteration loop instead of erroring.

## Verification: CONFIRMED (2026-07-13)

Reproduced exactly as reported on both interpreters (master build vs
/usr/bin/lua5.5.0). Minimized to one line:

```lua
print(pcall(table.unpack, coroutine.create(function() end)))
```

- golua:    `true`
- lua5.5.0: `false	attempt to get length of a thread value`

(No resume/yield needed; a fresh dead-simple coroutine suffices.)

Verified root-cause sites (all miss the `isThread` guard that the equivalent
opcode paths have — VM-level `#co` and `co[1]` DO error correctly):

- `vm/vm_table.go` `ObjLen` (~line 631): thread passes `val.IsTable()` and
  returns the backing table's length (0) instead of "attempt to get length of
  a thread value". The LEN opcode has a thread guard; this exported helper
  does not. → unpack/concat/sort succeed on empty range.
- `vm/vm_table.go` `IndexInt` (~line 620) (and `IndexValue`): the concrete
  fast path excludes `ct.isThread`, but the `val.IsTable()` fallback does not,
  so thread reads return nil via `tableGetInt` → `ipairs(co)` runs 0
  iterations instead of erroring.
- `stdlib/table.go` `tableCheckLike` (~line 34): `val.IsTable()` is true for
  threads, so sort/concat/move accept the thread argument. Reference
  `checktab` (ltablib.c) requires an actual table or the needed metamethods;
  threads have neither.
- `SetIndexInt` write path falls through to the `getMetafield` branch (which
  DOES special-case threads via `threadMeta`, which has no `__newindex`) →
  insert/remove/move-dst error, but with "attempt to index a thread value"
  instead of the reference `bad argument #N (table expected, got thread)` —
  that portion alone is error-message-only.

Scope check: not in `wontfix/` (no thread/table-lib entry); no
GC/finalization dependence; not error-msg-only overall — pcall returns
**true** (silent success) where reference returns **false** (error), so
control flow differs. `next(co)` already rejects threads correctly
(stdlib/globals.go luaNext has an explicit `IsThread()` check) — that same
guard is what the sites above are missing.

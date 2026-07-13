# coroutines: <close> vars in C-callback frames (gsub/sort/...) close eagerly when an error escapes the coroutine, instead of at coroutine.close

Reference 5.5 semantics: when a coroutine dies with an error, pending
to-be-closed variables are NOT closed during the failing resume — they stay
pending and are closed later by `coroutine.close` (or GC). golua honors this
for tbc vars in plain Lua frames (verified: `error()` in the coroutine body
defers, matches reference), but a tbc var declared inside a **callback frame
invoked by a C/stdlib function** — `string.gsub` replacement function,
`table.sort` comparator — is closed **eagerly during the failing resume**,
before `coroutine.resume` returns to the caller.

Affects both error kinds: plain `error()` in the callback and the implicit
"attempt to yield across a C-call boundary" error from a failed yield.
When the error is *caught inside the coroutine* (pcall around gsub, or load
swallowing a reader error), both implementations close during unwinding and
golua matches — the divergence is only for errors that escape the coroutine.

## Repro

```lua
local function mk(tag)
  return setmetatable({}, {__close=function(_,e) print("CLOSE["..tag.."]", tostring(e)) end})
end
local co1 = coroutine.create(function()
  string.gsub("a", "a", function(c)
    local t <close> = mk("gsub-err")
    error("plain-boom")
  end)
end)
print("r1:", coroutine.resume(co1))
print("c1:", coroutine.close(co1))
local co3 = coroutine.create(function()
  table.sort({2,1}, function(a,b)
    local t <close> = mk("sort-yield")
    coroutine.yield()
  end)
end)
print("r3:", coroutine.resume(co3))
print("c3:", coroutine.close(co3))
```

## golua

```
CLOSE[gsub-err]	FILE:7: plain-boom
r1:	false	FILE:7: plain-boom
c1:	false	FILE:7: plain-boom
CLOSE[sort-yield]	attempt to yield across a C-call boundary
r3:	false	attempt to yield across a C-call boundary
c3:	false	attempt to yield across a C-call boundary
```

## lua5.5.0

```
r1:	false	FILE:7: plain-boom
CLOSE[gsub-err]	FILE:7: plain-boom
c1:	false	FILE:7: plain-boom
r3:	false	attempt to yield across a C-call boundary
CLOSE[sort-yield]	attempt to yield across a C-call boundary
c3:	false	attempt to yield across a C-call boundary
```

Wrong close timing: reference guarantees resources of a dead-but-unclosed
coroutine stay live until `coroutine.close`; golua releases callback-frame
resources before resume even returns. Likely cause: the Go panic that carries
the Lua error unwinds tbc slots as it passes through the Go-implemented stdlib
frame, instead of leaving the coroutine stack intact for close.

## Verification: CONFIRMED (2026-07-13)

Independently reproduced on master golua vs `/usr/bin/lua5.5.0`. Both the
plain-`error()` (gsub) and failed-yield (sort) variants diverge exactly as
described. Not in `wontfix/`; not GC/finalization-timing dependent (the close
is triggered by an explicit `coroutine.close`); not error-message-only (the
`__close` side effect fires during `coroutine.resume` instead of during
`coroutine.close` — observable ordering and resource-lifetime difference).

Minimized repro (`__close` prints before `r:` on golua, between `r:` and `c:`
on reference):

```lua
local co = coroutine.create(function()
  string.gsub("a", "a", function()
    local t <close> = setmetatable({}, {__close=function() print("CLOSE") end})
    error("boom")
  end)
end)
print("r:", coroutine.resume(co))
print("c:", coroutine.close(co))
```

golua:

```
CLOSE
r:	false	FILE:4: boom
c:	false	FILE:4: boom
```

lua5.5.0:

```
r:	false	FILE:4: boom
CLOSE
c:	false	FILE:4: boom
```

Reference semantics confirmed in source: `lua_resume`
(`lua-5.5.0/src/ldo.c`) handles an unrecoverable error with only
`L->status = status; luaD_seterrorobj(...)` — no `luaF_close`/unwind — so tbc
variables stay pending until `lua_closethread` (`coroutine.close`) or GC.

Control cases verified to MATCH reference (scope the fix accordingly):

- tbc var in a plain Lua frame of the coroutine body + `error()` → both
  implementations defer CLOSE to `coroutine.close`. The bug is specific to
  callback frames invoked by Go-implemented stdlib functions.
- error caught inside the coroutine (`pcall(string.gsub, ...)`) → both close
  during unwinding at the pcall boundary; no divergence.
- `__close` receives the correct error object in both (golua just calls it too
  early).

# coroutines: no "C stack overflow" limit on deeply nested resume chains

Reference 5.5 bounds nested `coroutine.resume` chains by the C stack
(LUAI_MAXCCALLS): a chain of ~200+ coroutines each resuming the next fails
with "C stack overflow". golua (goroutine-backed) succeeds at any depth.

Probably wontfix-adjacent (golua is *more* permissive, similar in spirit to
load-stack-overflow-traceback), but it is an observable divergence: programs
relying on the reference limit as a guard see different behavior, and there is
no golua-side bound at all on resume nesting depth (each level parks a
goroutine).

## Repro

```lua
local function chain(n)
  if n == 0 then coroutine.yield("bottom") return "leaf" end
  local inner = coroutine.create(function() return chain(n-1) end)
  local ok, v = coroutine.resume(inner)
  if not ok then error(v, 0) end
  return v
end
for _, depth in ipairs{10, 50, 150, 190, 250} do
  local co = coroutine.create(function() return chain(depth) end)
  local ok, v = coroutine.resume(co)
  print(depth, ok, v)
end
```

## golua

```
10	true	bottom
50	true	bottom
150	true	bottom
190	true	bottom
250	true	bottom
```

## lua5.5.0

```
10	true	bottom
50	true	bottom
150	true	bottom
190	true	bottom
250	false	C stack overflow
```

## Verification: REJECTED (WONTFIX-SCOPE)

The reported output divergence at depth 250 reproduces exactly, but the
finding's core claim — "there is no golua-side bound at all on resume nesting
depth" — is **false**, and the residual divergence (a different limit
magnitude) is a documented wontfix family.

**golua DOES bound nested resume chains.** Extending the same repro to deeper
depths:

```
190	true	bottom
250	true	bottom
3000	true	bottom
6000	false	C stack overflow
```

Bisection shows the limit is crisp and deterministic: depth 4999 succeeds,
depth 5000 fails with `C stack overflow` (this repro's 2 Lua frames per level
x 5000 = `DefaultMaxCallDepth` 10000). Mechanism: coroutine VMs inherit
`callDepthBase` from the resuming parent through the native resume frame
(`vm/vm.go` `checkCallDepth`/`hasCFrame`), so resume-within-resume nesting
accumulates against the unified call-depth limit and reports
`"C stack overflow"` (not plain `"stack overflow"`). This is deliberate,
tested behavior: `tests/doctest/coroutine_recursive_cstack.lua` asserts
recursive create+resume yields `C stack overflow`, and the in-tree copy of
the official suite's cstack.lua nesting test
(`tests/stdlib/test_coroutine_resume_nesting.lua`) passes.

**The remaining difference is only the limit's magnitude** (~200 levels in
reference vs 5000 here), with identical error text and identical
catchability. That threshold is not a semantic guarantee: `LUAI_MAXCCALLS`
is an explicitly configurable implementation constant
(`ldo.h`: `#if !defined(LUAI_MAXCCALLS) #define LUAI_MAXCCALLS 200`; the
official test suite itself rebuilds with 180, and cstack.lua's header says
the tested depths shift with the constant). Reference's ~200 exists because
each nested `lua_resume` consumes real C stack; golua's goroutine-backed
coroutines have no such constraint, so it applies its unified depth limit
instead.

This is precisely the divergence family already documented in
`wontfix/load-stack-overflow-traceback/README.md`: *"Both reject; the limit
that bites first differs. This is the same goroutine/iterative-vs-C-stack
family as ... the documented nested depth divergences."* (Contrast: the
recursive `coroutine.close` chain, where reference-comparable depth matters
for a host-crash guard, IS capped at `maxCCalls` = 200 — see
`vm/vm.go` `EnterCloseChain` and `vm/limits.go`.)

Not GC-related; not error-message-only (outcomes at a fixed depth differ),
but the divergence is a resource-limit magnitude within a documented wontfix
family, so: **not a bug**.

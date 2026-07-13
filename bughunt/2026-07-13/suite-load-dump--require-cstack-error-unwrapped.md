# error-message-only: circular require loses "error loading module ... from file ..." wrapper

When a required module's execution fails with a C stack overflow (e.g. circular
require), reference Lua wraps the error as
`error loading module 'circ1' from file './circ1.lua':\n\tC stack overflow`.
golua returns the bare `C stack overflow`. Syntax errors and ordinary runtime
errors in required files ARE wrapped identically in both — only the stack
overflow path loses the wrapper (adjacent to wontfix/load-stack-overflow-traceback,
but this is require's wrapper, not load's traceback embedding).

## Repro (files in cwd, package.path = "./?.lua")
```lua
-- circ1.lua: require "circ2" return 1
-- circ2.lua: require "circ1" return 2
package.path = "./?.lua"
local ok, e = pcall(require, "circ1")
print(ok, e)
```

## golua
```
false	C stack overflow
```

## lua5.5.0
```
false	error loading module 'circ1' from file './circ1.lua':
	C stack overflow
```

Severity: error-message-only (error type/pcall behavior match).

## Verification: REJECTED (ERROR-MSG-ONLY)

Verified 2026-07-13 (golua master vs /usr/bin/lua5.5.0). The divergence is real
and reproduces, but the ONLY difference is error-message prose:

- Same error condition (unbounded require recursion -> C stack overflow).
- Same error object type (`string` in both).
- Same catchability (`pcall` returns `false, <string>` in both).
- No position/line info in either message; only the
  `error loading module '...' from file '...':\n\t` prefix differs.

Minimized repro (one self-requiring module; two files):

```lua
-- selfreq.lua: require "selfreq"
package.path = "./?.lua"
print(pcall(require, "selfreq"))
```

- golua:    `false	C stack overflow`
- lua5.5.0: `false	error loading module 'selfreq' from file './selfreq.lua':\n\tC stack overflow`

Why they differ: reference's `checkload` wrapper applies ONLY to load-phase
failures — ordinary runtime errors raised while *executing* a required module
are propagated unwrapped by BOTH interpreters (verified: `error("boom")` in a
required file prints identically). In reference, the C-stack counter happens to
trip inside the protected parser during `luaL_loadfilex` at the deepest
recursion level, so the overflow surfaces as a *load* failure and gets wrapped.
In golua, the call-depth limit trips at the VM call boundary when invoking
require/the loader chunk, so it surfaces as a *runtime* error, which
`stdlib/package.go` (lines ~216-225) deliberately re-raises unwrapped to match
reference `lua_call` semantics. Which phase the overflow lands in is an
artifact of where each implementation's depth counter bites first — the same
"which limit bites first" family as the documented
`wontfix/load-stack-overflow-traceback` divergences (that entry covers `load`'s
traceback embedding and parser-limit wording, not require's wrapper, so this is
adjacent-but-distinct, not a duplicate).

Not GC/finalization-dependent; deterministic on both sides.

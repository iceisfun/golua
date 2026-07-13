# CLI: cannot execute a precompiled (binary) chunk file; loadfile() can

Reference `lua script` uses luaL_loadfile, which auto-detects the `\27Lua`
signature and runs precompiled chunks. The golua CLI's script path parses the
file as *text only*, so running a file produced by string.dump (or golua's own
luac) fails with a lexer error — even though golua's own `loadfile()` on the
very same file loads and runs it fine. So the capability exists in the library;
only the CLI entry path never routes through binary detection.

## Repro
```lua
-- step 1 (make the chunk): golua -e 'local d=string.dump(load("print(\"hi from bytecode\")"));
--   local o=io.open("c.bin","wb") o:write(d) o:close()'
-- step 2: golua c.bin
```

## golua
```
$ golua c.bin
golua: c.bin:1: unexpected symbol near '<\27>'
(exit 1)
-- but: golua -e 'assert(loadfile("c.bin"))()'  → hi from bytecode
```

## lua5.5.0
```
$ lua5.5.0 c.bin        # (chunk dumped by lua5.5.0 itself)
hi from bytecode
```

Why wrong: the standalone interpreter contract is that `lua file` accepts both
source and precompiled chunks; golua's CLI only accepts source, while its own
loadfile proves the loader handles the binary form.

Note (separate, likely wontfix-adjacent): golua's dump format is not
byte-compatible with PUC 5.5.0 (each rejects the other's dumps). Lua does not
promise cross-implementation chunk portability, so that part is informational.

## Verification: CONFIRMED (2026-07-13)

Independently reproduced with a fresh scratch dir; each interpreter runs a
chunk dumped by *itself* (so bytecode-format cross-compatibility is not a
factor).

### Minimized repro
```sh
golua -e 'io.open("m.bin","wb"):write(string.dump(load"print(1)")):close()'
golua m.bin
```

### Verified outputs
```
$ golua m.bin
golua: m.bin:1: unexpected symbol near '<\27>'      (exit 1)
$ golua -e 'assert(loadfile("m.bin"))()'
1                                                    (exit 0)

$ lua5.5.0 -e 'io.open("mref.bin","wb"):write(string.dump(load"print(1)")):close()'
$ lua5.5.0 mref.bin
1                                                    (exit 0)
```

### Root cause
`cmd/lua/main.go` script path (line ~113–136) does `os.ReadFile` then
unconditionally `parser.Parse(displayName, source)` — it never checks the
first byte for the `\27Lua` binary signature. Reference `lua.c`'s
`handle_script` calls `luaL_loadfilex(L, fname, NULL)` (mode `NULL` = `"bt"`),
which sniffs `LUA_SIGNATURE` and routes to undump. golua's undump path
(`compiler/undump.go`) works — `loadfile()` on the same file executes it —
so only the CLI entry point lacks the binary-chunk dispatch.

### Scope checks
- Not a wontfix: `wontfix/untrusted-binary-chunks/` covers *maliciously
  crafted* chunks being unsafe to execute (matching reference stance); this
  finding is golua's own well-formed dump being unrunnable from the CLI.
- Not GC/finalization-timing dependent.
- Not error-message-only: golua functionally cannot execute the file at all
  (exit 1, chunk never runs) vs. reference executing it (exit 0).

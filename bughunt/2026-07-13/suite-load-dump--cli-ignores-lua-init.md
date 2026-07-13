# CLI: LUA_INIT / LUA_INIT_5_5 ignored (while LUA_PATH is honored)

The reference standalone interpreter runs the LUA_INIT_5_5 (or LUA_INIT)
environment variable — as code, or as a file with the `@filename` form — before
the main script, unless -E is given. The golua CLI ignores both forms entirely.
Notably golua DOES honor LUA_PATH for package.path, so env-var handling is
inconsistent: search-path env is applied, init env is silently dropped
(main.lua in the official suite tests this and would fail).

## Repro
```sh
LUA_INIT='print("init ran")' <interp> -e 'print("main")'
LUA_INIT='@initfile.lua'     <interp> -e 'print("main")'   # initfile.lua: print("initfile ran")
LUA_INIT_5_5='print("versioned init")' <interp> -e 'print("main")'
```

## golua
```
main
main
main
```

## lua5.5.0
```
init ran
main
initfile ran
main
versioned init
main
```

Why wrong: reference CLI contract (§7 of the manual, lua.c handle_luainit);
scripts and tooling rely on LUA_INIT for prelude injection. If ignoring it is a
deliberate sandbox decision, honoring LUA_PATH but not LUA_INIT is still an odd
split worth a decision/doc.

Related minimal-CLI gaps observed (not separately filed): -v, -i, -l, -E, -W,
and `-` (script from stdin) are unsupported; `lua -v` is treated as a script
filename. arg[-1] missing filed separately.

## Verification: CONFIRMED (2026-07-13)

All three repro forms verified live on both interpreters; outputs above are
exact. Minimized repro (any one form suffices):

```sh
LUA_INIT='print("init ran")' <interp> -e 'print("main")'
# golua:     main
# lua5.5.0:  init ran
#            main
```

Root cause: `cmd/lua/main.go` contains no reference to LUA_INIT or
LUA_INIT_5_5 anywhere in the repo (grep confirms zero hits in Go source).
Reference behavior is `handle_luainit` in lua.c (5.5.0 src, line 377):
check `LUA_INIT_5_5` first, fall back to `LUA_INIT`; `@file` prefix →
dofile, otherwise dostring with chunk name `=LUA_INIT_5_5`/`=LUA_INIT`;
runs after openlibs, before -e/-l/script, skipped only with -E (which the
golua CLI also lacks, so there is no opt-out path either).

The LUA_PATH asymmetry was also verified live: `LUA_PATH=... golua -e
'require("m")'` loads the module identically to reference (via
stdlib/package.go:70 getEnvPath LUA_PATH_5_5/LUA_PATH), so env-var support
is genuinely inconsistent rather than a deliberate no-env sandbox stance.
Nothing in wontfix/ covers CLI env handling; not GC/finalization related;
not an error-message-prose difference.

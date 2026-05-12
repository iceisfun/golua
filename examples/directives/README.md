# Directives Example

Parse `@`-prefixed header directives from a Lua source file. The
parser is purely source-level: it does not invoke the lexer, parser,
compiler, or VM, and it returns plain Go data.

## Non-standard Lua

Header directives (`-- @key value`) are a **golua-specific extension**
for embedders. They are **not** part of the Lua language as specified
by Lua 5.4 / Lua 5.5 and are not implemented by the reference Lua
interpreter. golua does not invent new syntax to express them — they
are deliberately encoded in ordinary Lua comments so that:

1. **Reference Lua executes the same source unchanged.** A `.lua` file
   with a directive header runs identically under `lua` / `lua5.5.0`
   and under golua. The reference interpreter ignores the comments.
2. **golua's lexer and parser are unaffected.** Directives are stripped
   by the standalone `directives` package; the lexer continues to
   discard all comments. There is no grammar change, no new tokens,
   no new AST nodes, and no bytecode change.
3. **Stripped / source-less execution is unaffected.** Directives never
   enter the bytecode pipeline; a precompiled `*compiler.Proto` carries
   no directive data.

The `directives` package has no opinion about which keys are valid.
Every directive in this example (`@tick`, `@scope`, `@disabled`,
`@import`) is an **embedder convention**, not a golua feature.

## Run

```bash
go run ./examples/directives
```

## Output

```
tick = "30s"
scope = "alias_expander"
script is disabled
import: shared/util
import: shared/log
```

# binary-integer-literals

## What

golua's lexer accepts binary integer literals with a `0b` / `0B` prefix:

```lua
print(0b101)   -- golua: 5    reference: malformed number near '0b101'
```

Reference Lua (5.4 and 5.5) defines only decimal and hexadecimal (`0x`)
numerals, so a chunk using `0b...` fails to compile there. The divergence is
also visible through `load` as a dialect probe: `load("return 0b1")` returns a
function in golua and `nil, "malformed number ..."` in reference.

Malformed forms (`0b`, `0b2`, `0b10abc`) are still rejected by golua.

## Why this won't change

The extension is deliberate (dedicated `scanBinaryNumber` in
`lexer/lexer_literal.go`, added for embedder convenience with bit-mask-heavy
configuration scripts) and removing it would break existing embedded scripts
that use it. It is strictly additive: every valid reference-Lua chunk means
the same thing in golua, and `0b` literals never enter the bytecode format
(they compile to ordinary integer constants, so `string.dump` output stays
portable).

The cost is one-directional portability: source written for golua that uses
`0b` literals must be rewritten (e.g. `0x5` or `5`) to run under reference
Lua. Scripts that need to stay portable should simply not use the extension —
like the other golua-specific extensions (`glob`, `time`, directives).

## Where the behavior lives

- `lexer/lexer_literal.go` — `scanBinaryNumber` (`0b`/`0B` prefix scan)
- `lexer/lexer_test.go`, `ast/ast_test.go` — extension tests

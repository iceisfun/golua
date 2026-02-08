# Glob Matching

Demonstrates how to use the `glob` package for case-insensitive pattern matching,
both directly from Go and exposed to Lua as native functions.

## What This Shows

- Using `glob.Match`, `glob.MatchWords`, and `glob.MatchNamed` from Go
- Exposing glob matching to Lua via a `glob` table of native functions
- Word-based matching for multi-word labels
- Named captures for extracting parts of a match (e.g. route parameters)
- A Lua utility module (`filter.lua`) that uses glob for list filtering
- Detecting whether a string contains pattern metacharacters

## Run

```sh
go run ./examples/glob
```

## Output

```
=== Go-Side Glob Matching ===
Match("hel*", "hello") = true
Match("HELLO", "hello") = true
Match("v[12].*", "v2.0") = true
MatchWords("ORG* PEACH", "ORGANIC PEACH") = true
MatchNamed = true, captures = map[resource:users version:v2]
HasPatternCharacters("hello") = false
HasPatternCharacters("h*llo") = true

=== Lua-Side Glob Matching ===
--- Basic Matching ---
Match 'hel*' vs 'hello':	true
Match 'h?llo' vs 'hello':	true
Match 'h[ae]llo' vs 'hello':	true
Match 'world' vs 'hello':	false

--- Case Insensitive ---
Match 'HELLO' vs 'hello':	true
Match 'hello' vs 'HELLO':	true

--- Word Matching ---
Pattern: ORG* PEACH
  ORGANIC PEACH -> true
  ORGANIC WHITE PEACH -> false
  CONVENTIONAL PEACH -> false
  ORGANIC APPLE -> false

--- Named Captures ---
Route pattern: /api/:version/:resource
  /api/v1/users -> version=v1, resource=users
  /api/v2/orders -> version=v2, resource=orders
  /api/v1/products -> version=v1, resource=products
  /web/home -> no match

--- Filtering (via dofile) ---
Items matching 'alpha-*':
  alpha-100
  alpha-200
Items matching '*-100':
  alpha-100
  beta-100
  gamma-100

--- Pattern Detection ---
  hello has patterns: false
  h*llo has patterns: true
  config.json has patterns: false
  *.lua has patterns: true
  [test] has patterns: true
```

## Directory Layout

```
examples/glob/
├── main.go       # Go host: uses glob directly, exposes it to Lua
├── demo.lua      # Lua script exercising all glob features
├── filter.lua    # Lua utility module for list filtering via glob
└── README.md     # This file
```

## Lua API (provided by Go host)

| Function | Signature | Returns | Description |
|----------|-----------|---------|-------------|
| `glob.match` | `(pattern, name)` | `boolean` | Single-string glob match |
| `glob.match_words` | `(pattern, name)` | `boolean` | Word-by-word glob match |
| `glob.match_named` | `(pattern, text)` | `boolean, table` | Match with named captures |
| `glob.has_pattern` | `(s)` | `boolean` | Check for metacharacters |

## Pattern Syntax

| Syntax | Meaning |
|--------|---------|
| `*` | Matches any sequence of characters |
| `?` | Matches any single character |
| `[abc]` | Matches any character in the set |
| `[a-z]` | Matches any character in the range |
| `[^abc]` | Matches any character NOT in the set |
| `\c` | Matches literal character `c` |
| `:name` | Named capture (MatchNamed only) |

All matching is case-insensitive. This is **not** Lua pattern matching -- see
the `glob` package documentation for details.

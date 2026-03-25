# glob - Go-Style Glob Matching Package

## Overview

The `glob` package provides Go-style glob matching for filesystem-like and tokenized word matching. It is a **standalone matching system** that intentionally diverges from Lua's pattern engine.

## Why Not Lua Patterns?

Lua patterns are powerful for string processing but are:
- Complex (backtracking, captures, frontier/balanced patterns)
- Unsuitable for simple case-insensitive identifier matching
- Overkill for filesystem-like globbing

The `glob` package provides a deliberately simple, safe, and fast matching primitive for domains where Lua patterns are inappropriate.

## Pattern Syntax

| Syntax | Meaning |
|--------|---------|
| `*` | Matches any sequence of characters (including empty) |
| `?` | Matches any single character |
| `[abc]` | Matches any character in the set |
| `[a-z]` | Matches any character in the range |
| `[^abc]` | Matches any character NOT in the set |
| `\c` | Matches literal character `c` (escapes metacharacters) |

All matching is **case-insensitive**.

## Lua API

The `glob` module is automatically available when `stdlib.Open` is called.

| Function | Description |
|----------|-------------|
| `glob.match(pattern, name)` | Returns true if `name` matches `pattern` |
| `glob.match_words(pattern, name)` | Splits on whitespace, matches each word |
| `glob.match_named(pattern, text)` | Returns `matched, captures_table` |
| `glob.has_pattern(s)` | Returns true if `s` contains metacharacters |

```lua
glob.match("h*o", "hello")             -- true
glob.match_words("ORG* PEACH", "ORGANIC PEACH")  -- true

local ok, caps = glob.match_named(":method /api/:id", "GET /api/42")
-- ok == true, caps.method == "GET", caps.id == "42"
```

## Go API

### `Match(pattern, name string) (bool, error)`

Reports whether `name` matches the shell `pattern`. Requires the pattern to match all of `name`, not just a substring. Returns `ErrBadPattern` on malformed patterns.

```go
matched, err := glob.Match("h*o", "hello")
// matched == true

matched, err = glob.Match("h[ea]llo", "hello")
// matched == true

matched, err = glob.Match("HELLO", "hello")
// matched == true (case-insensitive)
```

### `MatchWords(pattern, name string) (bool, error)`

Splits both `pattern` and `name` on whitespace and matches each word independently using `Match`. The number of words must match. Multiple spaces are treated as a single delimiter.

```go
matched, err := glob.MatchWords("ORG* PEACH", "ORGANIC PEACH")
// matched == true

matched, err = glob.MatchWords("ORG* PEACH", "ORGANIC WHITE PEACH")
// matched == false (word count mismatch: 2 vs 3)

matched, err = glob.MatchWords("ORG* * PEACH", "ORGANIC WHITE PEACH")
// matched == true
```

### `MatchNamed(pattern, text string) (bool, map[string]string, error)`

Reports whether `text` matches `pattern` and returns a map of named glob captures. Named globs begin with `:` followed by one or more letters.

```go
ok, caps, err := glob.MatchNamed("*/abc/:id/:name", "/service/app/abc/42/hello")
// ok == true
// caps == map[string]string{"id": "42", "name": "hello"}
```

### `HasPatternCharacters(s string) bool`

Reports whether `s` contains any glob metacharacters (`*`, `?`, `[`, `]`, `\`).

```go
glob.HasPatternCharacters("hello")    // false
glob.HasPatternCharacters("h*llo")    // true
```

## Error Handling

Malformed patterns return `glob.ErrBadPattern`. Examples:

- Unterminated character classes: `[abc`
- Invalid ranges: `[z-a]`
- Dangling escapes: `hello\`

## Limitations

- No regex features (alternation, backreferences, lookahead)
- No recursive directory globbing or filesystem traversal
- No capture groups (use `MatchNamed` for named captures)
- No Unicode normalization
- No Lua pattern compatibility (`%` escapes, frontier/balanced patterns)

## What This Package Does NOT Do

- Does **not** use Lua patterns
- Does **not** support `%` escapes, captures, or frontier/balanced patterns
- Does **not** integrate with `string.match` / `string.find`

It is a **separate matching system** for different use cases.

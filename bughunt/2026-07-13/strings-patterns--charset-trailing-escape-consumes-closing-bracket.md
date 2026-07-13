# Charset with escape pair straddling a range: trailing `%` should escape the closing `]`

**Severity: wrong-result** (silent match/no-match divergence, no error)

## Repro

```lua
print(string.find("]", "[a-%%]"))          -- who is in the set?
print(string.find("%", "[a-%%]"))
print(string.match("]", "[a-%%]+"))
print(string.gsub("a%b]c", "[a-%%]", "X"))
print(string.find("a]b", "%f[a-%%]]"))     -- frontier with same set
print(string.find("]", "[]-%%]"))
print(string.find("%", "[]-%%]"))
```

## golua output
```
nil
1	1
nil
aXb]c	1
nil
nil
1	1
```

## lua5.5.0 output
```
1	1
nil
]
a%bXc	1
2	2
1	1
nil
```

## Why it's wrong

In reference `matchbracketclass` (lstrlib.c), set items are paired strictly
left-to-right on raw bytes: in `[a-%%]` the range consumes the raw bytes
`a`, `-`, `%` (range 'a'..'%', which is empty), leaving the **second** `%` as
an escape whose operand is the very next char — the closing `]` (read at the
`ec` terminator position). So the reference set is `{range a-%, literal ']'}`.

golua's `parseCharSetElems` (stdlib/pattern.go) re-tokenizes the set contents
independently of the end-scan in `parseCharSetAt`: after the range consumes
`a`, `-`, `%`, the leftover lone `%` at the end of the (already-truncated)
contents string is treated as a **literal `%`** instead of an escape of the
closing `]`. Result: golua's set is `{range a-%, literal '%'}` — matches `%`
where reference matches `]`, and vice versa.

Root cause: `parseCharSetAt` scans for the closing `]` skipping `%X` pairs,
then hands `parseCharSetElems` only the interior; the elems parser pairs
ranges over raw bytes so an escape pair can straddle a range boundary, and the
dangling `%` at end-of-interior loses its C-semantics operand (the `]`).

## Verification: REJECTED (NOT-A-BUG)

The divergence is real and reproduces exactly as reported (verified 2026-07-13
against `/usr/bin/lua5.5.0`; minimized: `print(string.find("]", "[a-%%]"))`
→ golua `nil`, reference `1 1`). But the pattern is **explicitly undefined by
the Lua 5.5 manual**, which names this *exact* pattern
(`~/Downloads/lua-5.5.0/lua-5.5.0/doc/manual.html`, § "Character Class"):

> The interaction between ranges and classes is not defined. Therefore,
> patterns like `[%a-z]` or `[a-%%]` have no meaning.

Every diverging line in the repro (including `[]-%%]` and the `%f[a-%%]]`
frontier) has an escape (`%%`) as a range endpoint — i.e. exactly the
range/class interaction the manual declares meaningless. The reference
behavior here is an implementation accident, not defined semantics: PUC-Rio's
own two passes disagree with each other (`classEnd` pairs `%%` as one escape
and ends the class at the `]`, while `matchbracketclass` re-pairs the same
bytes as range `a-%` + escape, reading the escape operand at the `ec`
terminator position — that is how `]` sneaks into the set). golua's two
passes are mutually consistent (both read `%%` as a literal `%`).

Parity was verified on all nearby **defined** patterns — `[a-z%%]`,
`[%%a-z]`, `[%%]`, `[%]]`, `[a-c]`, `[a%-z]` against both `%` and `]`
subjects: golua and lua5.5.0 agree on every one. The divergence exists only
inside the manual's undefined region, so this is not a conformance bug.
(Also checked: not in `wontfix/`, and not GC-dependent.)

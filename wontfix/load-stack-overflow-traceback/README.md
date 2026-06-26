# load-stack-overflow-traceback

## What

When `load()` is given a chunk whose nesting depth exceeds the parser's
recursion limit, the parse fails with `"C stack overflow"`. Reference Lua, when
`load()` is called **directly** (not through `pcall`), bakes a **stack
traceback** into the error-message string that `load` returns:

```
nil   C stack overflow
      stack traceback:
          [C]: in global 'load'
          ...: in main chunk
          [C]: in ?
```

golua returns just `nil, "C stack overflow"`. Under `pcall(load, chunk)` even
reference returns the clean message, so the two agree there.

## Why this won't change

The traceback appears because reference's protected parser runs the default
message handler on the C-stack-overflow path. Reproducing it would mean
threading an error-handler/traceback through golua's parser's overflow path —
disproportionate machinery for a cosmetic difference, and golua's message is
arguably cleaner (a traceback inside a *returned* error string is unusual). The
error itself — `nil, "C stack overflow"` — is identical; only the appended
traceback differs.

## Related compiler-limit diagnostic divergences (same spirit)

These are also message-level only and not planned for change:

- **Near-token at a limit.** At a register/return/upvalue limit, golua and
  reference can name a slightly different `near '<token>'`/`near <eof>` (and
  occasionally an adjacent line) in the diagnostic. Both reject at the same
  limit; only the cited token differs.
- **Recursive-parser C-stack vs fixed limit.** A pathological construct like a
  ~200-target multiple assignment (`a,b,...,a200 = 1,...`) trips reference's
  *recursive* `restassign` against the C-stack (`"C stack overflow"`), whereas
  golua's iterative parser rejects the same program with a different fixed limit
  (`"too many registers"`). Both reject; the limit that bites first differs.
  This is the same goroutine/iterative-vs-C-stack family as
  [`libm-last-ulp`](../libm-last-ulp/)'s relatives and the documented nested
  depth divergences.

The `compilefuzz` finder in `golua-conformance` normalizes these message tails
and the C-stack-vs-register case, so its corpus stays focused on genuine
accept/reject or wrong-limit divergences.

## Where this lives in the source

- Parser recursion guard: golua's parser emits `"C stack overflow"` on deep
  nesting (matching reference's limit), in the recursive-descent expression
  parser.
- The clean (no-traceback) `load` error surfaces from `stdlib`'s `load` /
  `compiler.Compile` returning the error as a value.

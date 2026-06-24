# os-time-isdst-no-dst-zone

## What

`os.time` with an explicit `isdst = true` field, evaluated under a time zone that
has **no** daylight saving (e.g. `TZ=UTC`), errors in golua but returns a value
in reference Lua:

```lua
-- under TZ=UTC
os.time{year = 2000, month = 6, day = 1, hour = 12, isdst = true}
-- golua:    error "time result cannot be represented in this installation"
-- lua5.5.0: 959857200
```

Under a zone that *does* observe DST (`TZ=America/New_York`, etc.) golua and the
reference agree exactly, year-round. The divergence is specific to requesting
`isdst=true` where DST cannot apply.

## Why this won't change

Reference Lua passes the broken-down time straight to C `mktime()`. Under glibc,
`mktime` with `tm_isdst = 1` in a no-DST zone applies a **−3600 second** "DST is
in effect" adjustment anyway — a glibc-specific, non-portable behavior (POSIX
permits `mktime` to return `(time_t)-1` here, and other libcs do).

golua resolves local time through Go's `time` package, which has no equivalent
"force DST" knob: it looks for an actual zone offset matching the requested
wall-clock + DST state, finds none, and reports that the time cannot be
represented. Replicating the reference would mean hard-coding glibc's specific
−3600 hack, which is neither portable nor more correct.

This only affects programs that pass `isdst=true` to `os.time` while running in a
no-DST zone — an unusual combination. Omitting `isdst` (or using `isdst=false`
in a no-DST zone) works identically on both.

## Where this lives in the source

- [`vm/default_os.go`](../../vm/default_os.go) — `resolveLocalTime` (the
  "time result cannot be represented" path). os/time access stays behind the
  OS provider; the stdlib layer only validates fields.

## Related

Surfaced by `datefuzz` in `golua-conformance`; it is the dominant residual
divergence there and is recommended for allowlisting as a platform difference.

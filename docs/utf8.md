# UTF-8 Library Support

**Date:** 2026-02-08
**Status:** Implemented (strict mode only)
**Implementation:** `stdlib/utf8.go`

---

## 1. Summary

golua implements the Lua `utf8` standard library with the following
constraints:

* **Strict mode only** (the default). The `lax` parameter available in Lua 5.4
  is not supported.
* **Standard Unicode range only** (U+0000 to U+10FFFF). `utf8.char` does not
  accept codepoints above U+10FFFF.
* Surrogates (U+D800-U+DFFF) are rejected in all decoding functions, matching
  strict-mode behavior.

These limitations are documented below with rationale.

---

## 2. Function-by-Function Mapping

### 2.1 `utf8.char(...)`

**Lua 5.4:** Accepts integers 0 to 0x7FFFFFFF. Encodes each as UTF-8 (up to
6 bytes for values > U+10FFFF using original RFC 2279 encoding).

**golua:** Accepts integers 0 to 0x10FFFF. Values above U+10FFFF or surrogates
raise an error.

**Go primitive:** `utf8.AppendRune(buf, rune(code))` for encoding. Validate
with `utf8.ValidRune(rune(code))` before encoding.

**Rationale for restriction:** Go's `unicode/utf8.EncodeRune` follows RFC 3629,
which restricts UTF-8 to 4 bytes and U+10FFFF. Encoding values above this range
would require a custom encoder, which violates the project constraint of relying
exclusively on Go's standard library. In practice, codepoints above U+10FFFF are
not valid Unicode and are never encountered in real-world Lua programs.

### 2.2 `utf8.len(s [, i [, j]])`

**Lua 5.4:** Returns count of UTF-8 characters between byte positions i and j.
On invalid UTF-8, returns `nil, position`. Optional 4th arg enables lax mode.

**golua:** Implements strict mode. No lax parameter.

**Go primitives:**
* `utf8.DecodeRuneInString(s[pos:])` to advance through characters
* `RuneError` with `size == 1` detects invalid sequences
* `utf8.RuneStart(s[i])` for continuation byte detection

**Mapping is clean:** Go's decoder rejects the same sequences Lua's strict mode
rejects (surrogates, overlong encodings, > U+10FFFF, bad continuation bytes).

### 2.3 `utf8.codepoint(s [, i [, j]])`

**Lua 5.4:** Returns codepoints as integers for characters between byte
positions i and j. Raises error on invalid UTF-8. Optional 4th arg for lax mode.

**golua:** Implements strict mode. No lax parameter.

**Go primitive:** `utf8.DecodeRuneInString` returns the rune value directly.
Detect errors via `r == utf8.RuneError && size == 1`.

### 2.4 `utf8.codes(s)`

**Lua 5.4:** Returns iterator yielding `(byte_position, codepoint)` pairs.
Raises error on invalid UTF-8. Optional 2nd arg for lax mode.

**golua:** Implements strict mode. No lax parameter.

**Go primitive:** Closure over `DecodeRuneInString` advancing through the
string. Equivalent to Go's `for i, r := range s` but with error checking.

### 2.5 `utf8.offset(s, n [, i])`

**Lua 5.4:** Converts between codepoint offset and byte position. Returns nil
if target character doesn't exist in string.

**golua:** Fully implementable. No lax mode involved.

**Go primitives:**
* `utf8.RuneStart(s[i])` to detect continuation bytes
* `utf8.DecodeRuneInString(s[pos:])` for forward traversal
* Manual backward scan using `RuneStart` for `n < 0` and `n == 0`

**This function has no lax mode** in Lua 5.4, so there is no restriction.

### 2.6 `utf8.charpattern`

**Value:** `"[\0-\x7F\xC2-\xFD][\x80-\xBF]*"`

A string constant. Trivially implementable.

---

## 3. What Is Not Supported

### 3.1 Lax Mode

Lua 5.4 added an optional `lax` parameter to `utf8.len`, `utf8.codepoint`, and
`utf8.codes`. In lax mode, surrogates (U+D800-U+DFFF) and codepoints above
U+10FFFF are not treated as errors.

Go's `unicode/utf8.DecodeRuneInString` unconditionally rejects surrogates and
values > U+10FFFF by returning `RuneError`. There is no way to decode these
sequences using Go's standard library without implementing a custom decoder.

**Behavior:** Passing a truthy `lax` parameter raises a runtime error.

**Impact:** Low. Lax mode was added in Lua 5.4 for niche use cases involving
non-standard UTF-8 encodings. The vast majority of Lua programs use strict mode
(the default).

### 3.2 Extended Codepoint Range (> U+10FFFF)

Lua 5.4's `utf8.char` accepts values up to 0x7FFFFFFF, encoding them as 5-6
byte sequences per the original UTF-8 specification (RFC 2279). Go's encoder
is limited to U+10FFFF per RFC 3629.

**Impact:** Low. Values above U+10FFFF are not valid Unicode codepoints. No
standard character set assigns meaning to them. This feature exists in Lua for
completeness with the original UTF-8 spec, not for practical use.

---

## 4. Implementation Notes

### 4.1 Architecture

* New file: `stdlib/utf8.go`
* Registration: `openUtf8(v)` called from `stdlib.Open()`
* No provider needed (pure string computation, no I/O or OS access)
* Global name: `utf8`

### 4.2 Index Semantics

Lua uses 1-based byte indices with negative index support (counting from end).
This is the same convention used throughout golua's string library. The existing
`posRelat` helper (or equivalent) handles this conversion.

### 4.3 Error Behavior

| Function | On invalid UTF-8 |
|---|---|
| `utf8.len` | Returns `nil, position` (soft fail) |
| `utf8.codepoint` | Raises error (hard fail) |
| `utf8.codes` iterator | Raises error (hard fail) |
| `utf8.offset` | Error if initial position is continuation byte; nil if out of range |
| `utf8.char` | Error if codepoint > U+10FFFF or surrogate |

### 4.4 Distinguishing Real U+FFFD from Decode Errors

Go's `DecodeRuneInString` returns `(RuneError, 1)` for invalid bytes and
`(RuneError, 3)` for a genuine U+FFFD character in the input. The `size`
return value disambiguates these cases.

---

## 5. Constraints Verification

| Constraint | Status |
|---|---|
| No custom Unicode logic | Met |
| No custom UTF-8 decoding | Met |
| Exclusively Go standard library | Met (`unicode/utf8`, `unicode`) |
| Go primitives sufficient for target semantics | Met (strict mode) |
| Correctness over feature completeness | Met (strict-only is correct subset) |

---

## 6. What Users Should Expect

golua provides the `utf8` standard library for working with UTF-8 encoded
strings. All functions operate in **strict mode**, validating that sequences
represent valid Unicode codepoints (U+0000 to U+10FFFF, excluding surrogates).

The following Lua 5.4 features are **not supported**:
* The `lax` parameter on `utf8.len`, `utf8.codepoint`, and `utf8.codes`
* Codepoints above U+10FFFF in `utf8.char`

These features involve non-standard UTF-8 encodings that fall outside the
Unicode specification and Go's standard library support. Programs that rely
on standard Unicode text processing are fully supported.

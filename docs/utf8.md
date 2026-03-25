# UTF-8 Library Support

**Status:** Implemented (full Lua 5.4 compatibility including lax mode)
**Implementation:** `stdlib/utf8.go`

---

## 1. Summary

golua implements the Lua `utf8` standard library with full Lua 5.4 compatibility:

* **Strict mode** (default) validates standard Unicode (U+0000 to U+10FFFF),
  rejecting surrogates, overlong encodings, and out-of-range codepoints.
* **Lax mode** accepts the extended UTF-8 range (up to 0x7FFFFFFF) using custom
  encoding/decoding that supports 5- and 6-byte sequences per RFC 2279.
* `utf8.char` always supports the full extended range (0x00–0x7FFFFFFF).

---

## 2. Function-by-Function Mapping

### 2.1 `utf8.char(...)`

**Lua 5.4:** Accepts integers 0 to 0x7FFFFFFF. Encodes each as UTF-8 (up to
6 bytes for values > U+10FFFF using original RFC 2279 encoding).

**golua:** Matches Lua 5.4. Accepts integers 0 to 0x7FFFFFFF using a custom
`appendExtendedUTF8` encoder that produces 1- to 6-byte sequences.

### 2.2 `utf8.len(s [, i [, j [, lax]]])`

**Lua 5.4:** Returns count of UTF-8 characters between byte positions i and j.
On invalid UTF-8, returns `nil, position`. Optional 4th arg enables lax mode.

**golua:** Matches Lua 5.4. In strict mode, uses Go's `utf8.DecodeRuneInString`
which rejects surrogates, overlong encodings, and values > U+10FFFF. In lax
mode, uses `decodeExtendedUTF8` which accepts the full extended range.

### 2.3 `utf8.codepoint(s [, i [, j [, lax]]])`

**Lua 5.4:** Returns codepoints as integers for characters between byte
positions i and j. Raises error on invalid UTF-8. Optional 4th arg for lax mode.

**golua:** Matches Lua 5.4. Strict mode uses Go's decoder; lax mode uses
`decodeExtendedUTF8`.

### 2.4 `utf8.codes(s [, lax])`

**Lua 5.4:** Returns iterator yielding `(byte_position, codepoint)` pairs.
Raises error on invalid UTF-8. Optional 2nd arg for lax mode.

**golua:** Matches Lua 5.4. Strict mode uses Go's decoder; lax mode uses
`decodeExtendedUTF8`.

### 2.5 `utf8.offset(s, n [, i])`

**Lua 5.4:** Converts between codepoint offset and byte position. Returns nil
if target character doesn't exist in string.

**golua:** Matches Lua 5.4. No lax mode involved. Uses `utf8.RuneStart` for
byte classification and traversal.

### 2.6 `utf8.charpattern`

**Value:** `"[\0-\x7F\xC2-\xFD][\x80-\xBF]*"`

A string constant matching one UTF-8 byte sequence. The range `\xC2-\xFD`
covers lead bytes for 2- through 6-byte sequences, matching Lua 5.4's extended
UTF-8 support.

---

## 3. Implementation Notes

### 3.1 Custom Extended UTF-8 Codec

To support Lua 5.4's full codepoint range (0x00–0x7FFFFFFF), golua includes
two custom functions in `stdlib/utf8.go`:

* `appendExtendedUTF8` — encodes a codepoint as 1–6 byte extended UTF-8.
  Used by `utf8.char` unconditionally.
* `decodeExtendedUTF8` — decodes one extended UTF-8 character (1–6 bytes).
  Used by `utf8.len`, `utf8.codepoint`, and `utf8.codes` in lax mode only.

Both enforce minimum-length encoding (reject overlong sequences). In strict
mode, the standard Go `unicode/utf8` package is used instead.

### 3.2 Index Semantics

Lua uses 1-based byte indices with negative index support (counting from end).
The existing `posRelat` helper handles this conversion.

### 3.3 Error Behavior

| Function | On invalid UTF-8 |
|---|---|
| `utf8.len` | Returns `nil, position` (soft fail) |
| `utf8.codepoint` | Raises error (hard fail) |
| `utf8.codes` iterator | Raises error (hard fail) |
| `utf8.offset` | Error if initial position is continuation byte; nil if out of range |
| `utf8.char` | Error if codepoint > 0x7FFFFFFF or < 0 |

### 3.4 Distinguishing Real U+FFFD from Decode Errors

Go's `DecodeRuneInString` returns `(RuneError, 1)` for invalid bytes and
`(RuneError, 3)` for a genuine U+FFFD character in the input. The `size`
return value disambiguates these cases. This only applies in strict mode;
lax mode uses `decodeExtendedUTF8` which returns `(-1, 1)` on error.

---

## 4. Strict vs Lax Mode

| Aspect | Strict (default) | Lax |
|---|---|---|
| Decoder | Go `unicode/utf8` | Custom `decodeExtendedUTF8` |
| Max codepoint | U+10FFFF | 0x7FFFFFFF |
| Surrogates (U+D800–U+DFFF) | Rejected | Accepted |
| Overlong sequences | Rejected | Rejected |
| Max byte length | 4 | 6 |

`utf8.char` and `utf8.charpattern` are not affected by lax mode — they always
support the extended range.

---

## 5. What Users Should Expect

golua provides the `utf8` standard library with full Lua 5.4 compatibility.
All functions support both strict and lax modes:

* **Strict mode** (default) validates standard Unicode (U+0000 to U+10FFFF,
  excluding surrogates).
* **Lax mode** accepts the full extended UTF-8 range up to 0x7FFFFFFF,
  including surrogates and 5–6 byte sequences.

`utf8.char` always accepts the full range regardless of mode.

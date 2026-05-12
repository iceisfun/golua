// Package directives parses @-prefixed metadata directives from the
// header of a Lua source file.
//
// Directives are a golua-specific extension. They are not part of the
// Lua language and are ignored by the reference Lua interpreter, which
// sees them as ordinary comments. Source files using directives remain
// valid, portable Lua and execute identically on any Lua 5.4+
// implementation; only the directive interpretation is golua-specific,
// and that interpretation lives in the embedder, not in this package.
//
// The parser is purely source-level. It does not invoke the lexer,
// parser, compiler, or VM; it does not modify globals; it does not
// embed metadata in bytecode. A precompiled chunk carries no directive
// data, and stripped/source-less execution is unaffected.
//
// This package has no opinion about which keys are valid. All policy
// (which directives mean what, type coercion, validation) is the
// embedder's responsibility.
//
// # Header definition
//
// The directive header is the contiguous prefix of the source consisting
// of an optional shebang line (#!...), blank lines, and short comments
// (-- ...). The header ends at the first line that is neither blank nor
// a short comment. Long comments (--[[ ]]) terminate the header without
// being scanned.
//
// # Directive syntax
//
//	-- @key value...
//
// Two dashes, optional whitespace, '@', an identifier
// ([A-Za-z_][A-Za-z0-9_-]*), then either end-of-line (a flag directive)
// or whitespace followed by a value. The value is the remainder of the
// line, trimmed of leading and trailing whitespace; internal whitespace
// is preserved verbatim. Comment lines that do not match this shape are
// kept as ordinary comments and silently ignored, but do not terminate
// the header.
//
// See examples/directives and examples/directive_loader.
package directives

import (
	"iter"
	"strings"
)

// File is the parsed directive header of one Lua source file.
// A *File is immutable after Parse returns.
type File struct {
	entries []entry
	index   map[string][]int
}

type entry struct {
	key   string
	value string
}

// Parse extracts the contiguous directive header from src. It always
// returns a non-nil *File. The error return is reserved for future use
// (currently always nil).
func Parse(src string) (*File, error) {
	f := &File{index: map[string][]int{}}

	rest := src
	if strings.HasPrefix(rest, "#!") {
		if i := strings.IndexByte(rest, '\n'); i >= 0 {
			rest = rest[i+1:]
		} else {
			return f, nil
		}
	}

	for len(rest) > 0 {
		line, after := nextLine(rest)
		trimmed := strings.TrimLeft(line, " \t")

		if trimmed == "" {
			rest = after
			continue
		}
		if !strings.HasPrefix(trimmed, "--") {
			break
		}

		body := trimmed[2:]
		if strings.HasPrefix(body, "[[") || strings.HasPrefix(body, "[=") {
			break
		}

		body = strings.TrimLeft(body, " \t")
		if !strings.HasPrefix(body, "@") {
			rest = after
			continue
		}

		if k, v, ok := splitDirective(body[1:]); ok {
			f.index[k] = append(f.index[k], len(f.entries))
			f.entries = append(f.entries, entry{key: k, value: v})
		}
		rest = after
	}

	return f, nil
}

// nextLine returns the next line (without its terminating newline) and
// the remainder of s following the newline. If s has no newline, the
// whole string is returned and remainder is empty. A trailing '\r'
// (CRLF) is stripped from line.
func nextLine(s string) (line, rest string) {
	line, rest, _ = strings.Cut(s, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, rest
}

// splitDirective parses an identifier and optional value from body
// (the text following '@'). Returns ok=false if body does not begin
// with a valid identifier character.
func splitDirective(body string) (key, value string, ok bool) {
	end := 0
	for end < len(body) {
		c := body[end]
		if !isIdentChar(c, end == 0) {
			break
		}
		end++
	}
	if end == 0 {
		return "", "", false
	}
	key = body[:end]
	value = strings.TrimRight(strings.TrimLeft(body[end:], " \t"), " \t\r")
	return key, value, true
}

func isIdentChar(c byte, first bool) bool {
	switch {
	case c >= 'A' && c <= 'Z':
		return true
	case c >= 'a' && c <= 'z':
		return true
	case c == '_':
		return true
	case first:
		return false
	case c >= '0' && c <= '9':
		return true
	case c == '-':
		return true
	}
	return false
}

// Get returns the last value recorded for key and whether it was
// present. Flag directives (e.g. "-- @disabled") return ("", true).
func (f *File) Get(key string) (string, bool) {
	ix := f.index[key]
	if len(ix) == 0 {
		return "", false
	}
	return f.entries[ix[len(ix)-1]].value, true
}

// Has reports whether key was present at least once in the header.
func (f *File) Has(key string) bool {
	return len(f.index[key]) > 0
}

// Lookup returns every value recorded for key, in source order.
// Returns nil if key was not present.
func (f *File) Lookup(key string) []string {
	ix := f.index[key]
	if len(ix) == 0 {
		return nil
	}
	out := make([]string, len(ix))
	for i, j := range ix {
		out[i] = f.entries[j].value
	}
	return out
}

// Keys returns the distinct directive keys in first-occurrence order.
func (f *File) Keys() []string {
	if len(f.entries) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(f.entries))
	out := make([]string, 0, len(f.entries))
	for _, e := range f.entries {
		if _, ok := seen[e.key]; ok {
			continue
		}
		seen[e.key] = struct{}{}
		out = append(out, e.key)
	}
	return out
}

// All iterates every (key, value) pair in source order, including
// duplicates. Suitable for go1.23+ range-over-func.
func (f *File) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, e := range f.entries {
			if !yield(e.key, e.value) {
				return
			}
		}
	}
}

// Len returns the total number of directive entries, counting
// duplicates separately.
func (f *File) Len() int {
	return len(f.entries)
}

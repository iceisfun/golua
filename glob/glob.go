// Package glob provides Go-style glob matching for filesystem-like and
// tokenized word matching.
//
// This package intentionally diverges from Lua's pattern engine. It does not
// use Lua patterns, does not support % escapes, captures, or frontier/balanced
// patterns, and does not integrate with string.match or string.find.
//
// It is a separate matching system designed for:
//   - Generic word matching
//   - Filesystem globbing
//   - SKU / identifier matching
//   - Human-readable token matching
//   - Case-insensitive matching
//   - Word-by-word matching across space-separated fields
//
// Limitations:
//   - No regex features (alternation, backreferences, lookahead)
//   - No recursive directory globbing
//   - No capture groups (see MatchNamed for named glob captures)
//   - No Unicode normalization
//
// Pattern syntax:
//
//	pattern:
//	    { term }
//	term:
//	    '*'         matches any sequence of characters
//	    '?'         matches any single character
//	    '[' [ '^' ] { character-range } ']'
//	                character class (must be non-empty)
//	    c           matches character c (c != '*', '?', '\\', '[')
//	    '\\' c      matches character c
//
//	character-range:
//	    c           matches character c (c != '\\', '-', ']')
//	    '\\' c      matches character c
//	    lo '-' hi   matches character c for lo <= c <= hi
//
// All matching is case-insensitive.
package glob

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func splitWords(name string) []string {
	return strings.FieldsFunc(name, func(r rune) bool {
		return unicode.IsSpace(r)
	})
}

// MatchWords splits both pattern and name on whitespace and matches each
// word independently using Match. The number of words must match. Multiple
// spaces are treated as a single delimiter.
func MatchWords(pattern, name string) (matched bool, err error) {
	patternWords := splitWords(pattern)
	nameWords := splitWords(name)

	if len(patternWords) != len(nameWords) {
		return false, nil
	}

	for i, patternWord := range patternWords {
		matched, err = Match(patternWord, nameWords[i])
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

// Match reports whether name matches the shell pattern.
// Match requires pattern to match all of name, not just a substring.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed. All matching is case-insensitive.
func Match(pattern, name string) (matched bool, err error) {
	pattern = strings.ToUpper(pattern)
	name = strings.ToUpper(name)
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string
		star, chunk, pattern = scanChunk(pattern)
		if star && chunk == "" {
			// Trailing * matches rest of string
			return true, nil
		}
		// Look for match at current position.
		t, ok, err := matchChunk(chunk, name)
		// if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (len(t) == 0 || len(pattern) > 0) {
			name = t
			continue
		}
		if err != nil {
			return false, err
		}
		if star {
			// Look for match skipping i+1 bytes.
			for i := 0; i < len(name); i++ {
				t, ok, err := matchChunk(chunk, name[i+1:])
				if ok {
					// if we're the last chunk, make sure we exhausted the name
					if len(pattern) == 0 && len(t) > 0 {
						continue
					}
					name = t
					continue Pattern
				}
				if err != nil {
					return false, err
				}
			}
		}
		return false, nil
	}
	return len(name) == 0, nil
}

// scanChunk gets the next segment of pattern, which is a non-star string
// possibly preceded by a star.
func scanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}
	inrange := false
	var i int
Scan:
	for i = 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			// error check handled in matchChunk: bad pattern.
			if i+1 < len(pattern) {
				i++
			}
		case '[':
			inrange = true
		case ']':
			inrange = false
		case '*':
			if !inrange {
				break Scan
			}
		}
	}
	return star, pattern[0:i], pattern[i:]
}

// matchChunk checks whether chunk matches the beginning of s.
// If so, it returns the remainder of s (after the match).
// Chunk is all single-character operators: literals, char classes, and ?.
func matchChunk(chunk, s string) (rest string, ok bool, err error) {
	for len(chunk) > 0 {
		if len(s) == 0 {
			return "", false, nil
		}
		switch chunk[0] {
		case '[':
			// character class
			r, n := utf8.DecodeRuneInString(s)
			s = s[n:]
			chunk = chunk[1:]
			// possibly negated
			negated := false
			if len(chunk) > 0 && chunk[0] == '^' {
				negated = true
				chunk = chunk[1:]
			}
			// parse all ranges
			match := false
			nrange := 0
			for {
				if len(chunk) > 0 && chunk[0] == ']' && nrange > 0 {
					chunk = chunk[1:]
					break
				}
				var lo, hi rune
				if lo, chunk, err = getEsc(chunk); err != nil {
					return "", false, err
				}
				hi = lo
				if chunk[0] == '-' {
					if hi, chunk, err = getEsc(chunk[1:]); err != nil {
						return "", false, err
					}
				}
				if hi < lo {
					return "", false, ErrBadPattern
				}
				if lo <= r && r <= hi {
					match = true
				}
				nrange++
			}
			if match == negated {
				return "", false, nil
			}

		case '?':
			_, n := utf8.DecodeRuneInString(s)
			s = s[n:]
			chunk = chunk[1:]

		case '\\':
			chunk = chunk[1:]
			if len(chunk) == 0 {
				return "", false, ErrBadPattern
			}
			fallthrough

		default:
			if chunk[0] != s[0] {
				return "", false, nil
			}
			s = s[1:]
			chunk = chunk[1:]
		}
	}
	return s, true, nil
}

// getEsc gets a possibly-escaped character from chunk, for a character class.
func getEsc(chunk string) (r rune, nchunk string, err error) {
	if len(chunk) == 0 || chunk[0] == '-' || chunk[0] == ']' {
		err = ErrBadPattern
		return
	}
	if chunk[0] == '\\' {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			err = ErrBadPattern
			return
		}
	}
	r, n := utf8.DecodeRuneInString(chunk)
	if r == utf8.RuneError && n == 1 {
		err = ErrBadPattern
	}
	nchunk = chunk[n:]
	if len(nchunk) == 0 {
		err = ErrBadPattern
	}
	return
}

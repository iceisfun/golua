package stdlib

import (
	"strings"
	"unicode"
)

// luaMatch implements Lua pattern matching.
// Returns list of captures if successful, nil if no match.
// If no captures are defined, returns the whole match.
func luaMatch(s, pattern string, init int) []string {
	if init < 0 {
		init = len(s) + init + 1
	}
	if init < 1 {
		init = 1
	}
	if init > len(s)+1 {
		return nil
	}

	// Handle anchored patterns
	anchored := false
	if len(pattern) > 0 && pattern[0] == '^' {
		anchored = true
		pattern = pattern[1:]
	}

	// Try match at each position
	start := init - 1 // convert to 0-based
	if anchored {
		caps := matchPattern(s, start, pattern, 0)
		if caps != nil {
			return caps.captures(s)
		}
		return nil
	}

	for i := start; i <= len(s); i++ {
		caps := matchPattern(s, i, pattern, 0)
		if caps != nil {
			return caps.captures(s)
		}
	}
	return nil
}

// matchResult holds the match state
type matchResult struct {
	start int        // Start of match
	end   int        // End of match (exclusive)
	caps  []capture  // Captured groups
}

type capture struct {
	start int
	end   int
}

func (m *matchResult) captures(s string) []string {
	if len(m.caps) == 0 {
		// No captures, return the whole match
		return []string{s[m.start:m.end]}
	}
	result := make([]string, len(m.caps))
	for i, c := range m.caps {
		result[i] = s[c.start:c.end]
	}
	return result
}

// matchPattern tries to match pattern at position pos in s.
// Returns matchResult on success, nil on failure.
func matchPattern(s string, pos int, pattern string, patPos int) *matchResult {
	// Handle end of pattern
	if patPos >= len(pattern) {
		return &matchResult{start: pos, end: pos}
	}

	// Handle $ anchor at end
	if pattern[patPos] == '$' && patPos == len(pattern)-1 {
		if pos == len(s) {
			return &matchResult{start: pos, end: pos}
		}
		return nil
	}

	// Handle capture group start
	if pattern[patPos] == '(' {
		// Find the matching close paren
		depth := 1
		closePos := patPos + 1
		for closePos < len(pattern) && depth > 0 {
			if pattern[closePos] == '(' {
				depth++
			} else if pattern[closePos] == ')' {
				depth--
			} else if pattern[closePos] == '%' && closePos+1 < len(pattern) {
				closePos++ // skip escaped char
			}
			closePos++
		}
		if depth != 0 {
			// Unbalanced parens
			return nil
		}
		closePos-- // back to the ')'

		// Match the inner pattern
		innerPattern := pattern[patPos+1 : closePos]
		inner := matchPattern(s, pos, innerPattern, 0)
		if inner == nil {
			return nil
		}

		// Continue with rest of pattern
		rest := matchPattern(s, inner.end, pattern, closePos+1)
		if rest == nil {
			return nil
		}

		// Build result with this capture
		result := &matchResult{
			start: pos,
			end:   rest.end,
			caps:  make([]capture, 0, 1+len(inner.caps)+len(rest.caps)),
		}
		result.caps = append(result.caps, capture{start: pos, end: inner.end})
		result.caps = append(result.caps, inner.caps...)
		result.caps = append(result.caps, rest.caps...)
		return result
	}

	// Get the current pattern element
	elem, elemLen := getPatternElem(pattern, patPos)
	if elem == nil {
		return nil
	}

	// Check for repetition modifier
	modifier := byte(0)
	modPos := patPos + elemLen
	if modPos < len(pattern) {
		switch pattern[modPos] {
		case '*', '+', '-', '?':
			modifier = pattern[modPos]
			elemLen++
		}
	}

	switch modifier {
	case '*':
		// Greedy zero or more
		return matchStar(s, pos, pattern, patPos+elemLen, elem)
	case '+':
		// Greedy one or more
		if pos >= len(s) || !elem.matches(s[pos]) {
			return nil
		}
		return matchStar(s, pos+1, pattern, patPos+elemLen, elem)
	case '-':
		// Non-greedy zero or more
		return matchMinus(s, pos, pattern, patPos+elemLen, elem)
	case '?':
		// Optional
		if pos < len(s) && elem.matches(s[pos]) {
			result := matchPattern(s, pos+1, pattern, patPos+elemLen)
			if result != nil {
				result.start = pos
				return result
			}
		}
		return matchPattern(s, pos, pattern, patPos+elemLen)
	default:
		// Single match required
		if pos >= len(s) {
			return nil
		}
		if !elem.matches(s[pos]) {
			return nil
		}
		result := matchPattern(s, pos+1, pattern, patPos+elemLen)
		if result != nil {
			result.start = pos
		}
		return result
	}
}

// matchStar handles greedy * and + repetition
func matchStar(s string, pos int, pattern string, patPos int, elem patternElem) *matchResult {
	// First, eat as many matches as possible
	end := pos
	for end < len(s) && elem.matches(s[end]) {
		end++
	}

	// Then try to match the rest, backtracking as needed
	for i := end; i >= pos; i-- {
		result := matchPattern(s, i, pattern, patPos)
		if result != nil {
			result.start = pos
			return result
		}
	}
	return nil
}

// matchMinus handles non-greedy - repetition
func matchMinus(s string, pos int, pattern string, patPos int, elem patternElem) *matchResult {
	// Try to match the rest first, then consume more
	for i := pos; i <= len(s); i++ {
		result := matchPattern(s, i, pattern, patPos)
		if result != nil {
			result.start = pos
			return result
		}
		if i >= len(s) || !elem.matches(s[i]) {
			break
		}
	}
	return nil
}

// patternElem represents a single pattern element
type patternElem interface {
	matches(b byte) bool
}

type anyChar struct{}

func (anyChar) matches(b byte) bool { return true }

type literalChar struct {
	ch byte
}

func (l literalChar) matches(b byte) bool { return b == l.ch }

type classChar struct {
	class byte
	neg   bool
}

func (c classChar) matches(b byte) bool {
	matched := false
	switch c.class {
	case 'a':
		matched = isLetter(b)
	case 'd':
		matched = isDigit(b)
	case 's':
		matched = isSpace(b)
	case 'w':
		matched = isLetter(b) || isDigit(b) || b == '_'
	case 'l':
		matched = isLower(b)
	case 'u':
		matched = isUpper(b)
	case 'c':
		matched = isControl(b)
	case 'p':
		matched = isPunct(b)
	case 'x':
		matched = isHex(b)
	case 'z':
		matched = b == 0
	default:
		// Unknown class, treat as literal
		matched = b == c.class
	}
	if c.neg {
		return !matched
	}
	return matched
}

type charSet struct {
	chars string
	neg   bool
}

func (c charSet) matches(b byte) bool {
	matched := strings.ContainsRune(c.chars, rune(b))
	if c.neg {
		return !matched
	}
	return matched
}

// getPatternElem returns the pattern element at patPos and its length
func getPatternElem(pattern string, patPos int) (patternElem, int) {
	if patPos >= len(pattern) {
		return nil, 0
	}

	ch := pattern[patPos]

	// Escaped character
	if ch == '%' {
		if patPos+1 >= len(pattern) {
			return nil, 0
		}
		next := pattern[patPos+1]
		// Check for character class
		switch next {
		case 'a', 'd', 's', 'w', 'l', 'u', 'c', 'p', 'x', 'z':
			return classChar{class: next, neg: false}, 2
		case 'A', 'D', 'S', 'W', 'L', 'U', 'C', 'P', 'X', 'Z':
			return classChar{class: next + 32, neg: true}, 2 // lowercase
		default:
			// Escaped literal
			return literalChar{ch: next}, 2
		}
	}

	// Character set [...]
	if ch == '[' {
		neg := false
		start := patPos + 1
		if start < len(pattern) && pattern[start] == '^' {
			neg = true
			start++
		}
		end := start
		for end < len(pattern) && pattern[end] != ']' {
			if pattern[end] == '%' && end+1 < len(pattern) {
				end += 2
			} else {
				end++
			}
		}
		if end >= len(pattern) {
			return nil, 0
		}
		chars := expandCharSet(pattern[start:end])
		return charSet{chars: chars, neg: neg}, end - patPos + 1
	}

	// Any character
	if ch == '.' {
		return anyChar{}, 1
	}

	// Literal character
	return literalChar{ch: ch}, 1
}

// expandCharSet expands a character set definition like "a-z" to "abc...z"
func expandCharSet(set string) string {
	var result strings.Builder
	i := 0
	for i < len(set) {
		if set[i] == '%' && i+1 < len(set) {
			// Escaped character or class
			next := set[i+1]
			switch next {
			case 'a':
				result.WriteString("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
			case 'd':
				result.WriteString("0123456789")
			case 's':
				result.WriteString(" \t\n\r\f\v")
			case 'w':
				result.WriteString("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")
			default:
				result.WriteByte(next)
			}
			i += 2
		} else if i+2 < len(set) && set[i+1] == '-' {
			// Character range
			start := set[i]
			end := set[i+2]
			for c := start; c <= end; c++ {
				result.WriteByte(c)
			}
			i += 3
		} else {
			result.WriteByte(set[i])
			i++
		}
	}
	return result.String()
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}

func isLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func isUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isControl(b byte) bool {
	return b < 32 || b == 127
}

func isPunct(b byte) bool {
	return unicode.IsPunct(rune(b))
}

func isHex(b byte) bool {
	return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

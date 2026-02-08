package stdlib

import (
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

	// Handle balanced pattern %bxy
	if pattern[patPos] == '%' && patPos+1 < len(pattern) && pattern[patPos+1] == 'b' {
		if patPos+3 >= len(pattern) {
			return nil // need two characters after %b
		}
		open := pattern[patPos+2]
		close := pattern[patPos+3]
		if pos >= len(s) || s[pos] != open {
			return nil
		}
		depth := 1
		i := pos + 1
		for i < len(s) && depth > 0 {
			if s[i] == open {
				depth++
			} else if s[i] == close {
				depth--
			}
			i++
		}
		if depth != 0 {
			return nil
		}
		// Matched balanced substring s[pos:i], continue with rest of pattern
		result := matchPattern(s, i, pattern, patPos+4)
		if result != nil {
			result.start = pos
		}
		return result
	}

	// Handle frontier pattern %f[set]
	if pattern[patPos] == '%' && patPos+1 < len(pattern) && pattern[patPos+1] == 'f' {
		if patPos+2 >= len(pattern) || pattern[patPos+2] != '[' {
			return nil
		}
		set, setLen := parseCharSetAt(pattern, patPos+2)
		if set == nil {
			return nil
		}

		// Get prev and curr bytes
		var prev byte = 0
		if pos > 0 {
			prev = s[pos-1]
		}
		var curr byte = 0
		if pos < len(s) {
			curr = s[pos]
		}

		// Frontier: curr matches set, prev does not
		if set.matches(curr) && !set.matches(prev) {
			// Zero-width: continue matching at same position
			result := matchPattern(s, pos, pattern, patPos+2+setLen)
			if result != nil {
				result.start = pos
			}
			return result
		}
		return nil
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
		result := matchStar(s, pos+1, pattern, patPos+elemLen, elem)
		if result != nil {
			result.start = pos
		}
		return result
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
	elems []patternElem // individual matchers (classChar, literalChar, rangeChar)
	neg   bool
}

func (c charSet) matches(b byte) bool {
	matched := false
	for _, e := range c.elems {
		if e.matches(b) {
			matched = true
			break
		}
	}
	if c.neg {
		return !matched
	}
	return matched
}

type rangeChar struct {
	low, high byte
}

func (r rangeChar) matches(b byte) bool {
	return b >= r.low && b <= r.high
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
		set, setLen := parseCharSetAt(pattern, patPos)
		if set == nil {
			return nil, 0
		}
		return set, setLen
	}

	// Any character
	if ch == '.' {
		return anyChar{}, 1
	}

	// Literal character
	return literalChar{ch: ch}, 1
}

// parseCharSetAt parses a [...] character set starting at pattern[patPos] == '['
// Returns the charSet and the total length consumed from the pattern.
func parseCharSetAt(pattern string, patPos int) (*charSet, int) {
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
	elems := parseCharSetElems(pattern[start:end])
	return &charSet{elems: elems, neg: neg}, end - patPos + 1
}

// parseCharSetElems parses the contents of a [...] set into a list of matchers.
func parseCharSetElems(set string) []patternElem {
	var elems []patternElem
	i := 0
	for i < len(set) {
		if set[i] == '%' && i+1 < len(set) {
			next := set[i+1]
			switch next {
			case 'a', 'd', 's', 'w', 'l', 'u', 'c', 'p', 'x', 'z':
				elems = append(elems, classChar{class: next, neg: false})
			case 'A', 'D', 'S', 'W', 'L', 'U', 'C', 'P', 'X', 'Z':
				elems = append(elems, classChar{class: next + 32, neg: true})
			default:
				elems = append(elems, literalChar{ch: next})
			}
			i += 2
		} else if i+2 < len(set) && set[i+1] == '-' {
			elems = append(elems, rangeChar{low: set[i], high: set[i+2]})
			i += 3
		} else {
			elems = append(elems, literalChar{ch: set[i]})
			i++
		}
	}
	return elems
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

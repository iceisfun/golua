package stdlib

import (
	"fmt"
)

const luaMaxCaptures = 32
const capUnfinished = -1
const capPosition = -2
const maxMatchCalls = 200

// matchState tracks pattern matching state including captures.
// This implements the standard Lua pattern matching algorithm where
// '(' and ')' are inline markers in the pattern, and captures are
// tracked on a stack with backtracking support.
type matchState struct {
	s     string
	p     string
	level int
	depth int // recursion depth counter
	cap   [luaMaxCaptures]capSlot
}

type capSlot struct {
	init int
	slen int // length of capture, or capUnfinished/capPosition
}

// captureValue represents a resolved capture from a match.
type captureValue struct {
	str        string
	pos        int  // 1-based position (only valid when isPos is true)
	isPos      bool // true for position captures ()
	unfinished bool // true for unfinished captures (unclosed parenthesis)
}

// Unfinished captures are marked rather than causing a panic, so callers
// can decide whether to error (e.g., gsub with plain string replacement
// that doesn't reference captures should not error).

// getCaptures extracts resolved capture values from the match state.
func (ms *matchState) getCaptures() []captureValue {
	if ms.level == 0 {
		return nil
	}
	caps := make([]captureValue, ms.level)
	for i := 0; i < ms.level; i++ {
		c := ms.cap[i]
		if c.slen == capPosition {
			caps[i] = captureValue{isPos: true, pos: c.init + 1}
		} else if c.slen == capUnfinished {
			caps[i] = captureValue{unfinished: true}
		} else {
			caps[i] = captureValue{str: ms.s[c.init : c.init+c.slen]}
		}
	}
	return caps
}

// checkCaptures panics if any capture is unfinished. Called by find/match/gmatch
// which always need resolved captures.
func checkCaptures(caps []captureValue) {
	for _, c := range caps {
		if c.unfinished {
			panic("unfinished capture")
		}
	}
}

// match tries to match pattern starting at pp against string position si.
// Returns end position on success, -1 on failure.
func (ms *matchState) match(si int, pp int) int {
	ms.depth++
	defer func() { ms.depth-- }()
	if ms.depth > maxMatchCalls {
		panic("pattern too complex")
	}
	for {
		if pp >= len(ms.p) {
			return si
		}

		ch := ms.p[pp]

		if ch == '(' {
			if pp+1 < len(ms.p) && ms.p[pp+1] == ')' {
				return ms.matchPosCapture(si, pp)
			}
			return ms.matchOpenCapture(si, pp)
		}

		if ch == ')' {
			return ms.matchCloseCapture(si, pp)
		}

		if ch == '$' && pp+1 == len(ms.p) {
			if si == len(ms.s) {
				return si
			}
			return -1
		}

		if ch == '%' && pp+1 < len(ms.p) {
			next := ms.p[pp+1]
			if next == 'b' {
				return ms.matchBalance(si, pp)
			}
			if next == 'f' {
				return ms.matchFrontier(si, pp)
			}
			if next == '0' {
				panic("invalid capture index %0")
			}
			if next >= '1' && next <= '9' {
				return ms.matchBackRef(si, pp)
			}
		}

		// Default: single element possibly followed by modifier
		elem, elemLen := getPatternElem(ms.p, pp)
		if elem == nil {
			return -1
		}

		modPos := pp + elemLen
		var suffix byte
		if modPos < len(ms.p) {
			suffix = ms.p[modPos]
		}

		singleMatches := si < len(ms.s) && elem.matches(ms.s[si])

		if !singleMatches {
			// The element does not match here. For '*', '-' and '?' this is the
			// "zero repetitions" case: skip the element and its suffix and keep
			// going in the SAME frame (Lua's `p = ep + 1; goto init`), so a long
			// run of always-empty quantifiers — e.g. ("a*"):rep(250) against ""
			// — folds away without consuming match-recursion depth. Recursing
			// here instead would falsely trip the "pattern too complex" guard.
			if suffix == '*' || suffix == '-' || suffix == '?' {
				pp = modPos + 1
				continue
			}
			// '+' or no suffix: the element is required, so the match fails.
			return -1
		}

		// The element matches at least once here.
		switch suffix {
		case '*':
			return ms.matchGreedy(si, modPos+1, elem, 0)
		case '+':
			return ms.matchGreedy(si, modPos+1, elem, 1)
		case '-':
			return ms.matchLazy(si, modPos+1, elem)
		case '?':
			res := ms.match(si+1, modPos+1)
			if res != -1 {
				return res
			}
			// Try matching the rest without consuming (tail-call via continue).
			pp = modPos + 1
			continue
		default:
			// No suffix: a single required match. Advance (tail-call via continue).
			si++
			pp += elemLen
		}
	}
}

func (ms *matchState) matchOpenCapture(si, pp int) int {
	level := ms.level
	if level >= luaMaxCaptures {
		panic("too many captures")
	}
	ms.cap[level].init = si
	ms.cap[level].slen = capUnfinished
	ms.level++
	res := ms.match(si, pp+1)
	if res == -1 {
		ms.level = level
	}
	return res
}

func (ms *matchState) matchPosCapture(si, pp int) int {
	level := ms.level
	if level >= luaMaxCaptures {
		panic("too many captures")
	}
	ms.cap[level].init = si
	ms.cap[level].slen = capPosition
	ms.level++
	res := ms.match(si, pp+2)
	if res == -1 {
		ms.level = level
	}
	return res
}

func (ms *matchState) matchCloseCapture(si, pp int) int {
	l := ms.captureToClose()
	ms.cap[l].slen = si - ms.cap[l].init
	res := ms.match(si, pp+1)
	if res == -1 {
		ms.cap[l].slen = capUnfinished
	}
	return res
}

func (ms *matchState) captureToClose() int {
	for l := ms.level - 1; l >= 0; l-- {
		if ms.cap[l].slen == capUnfinished {
			return l
		}
	}
	panic("invalid pattern capture")
}

func (ms *matchState) matchBalance(si, pp int) int {
	if pp+3 >= len(ms.p) {
		panic("malformed pattern (missing arguments to '%b')")
	}
	open := ms.p[pp+2]
	close := ms.p[pp+3]
	if si >= len(ms.s) || ms.s[si] != open {
		return -1
	}
	if open == close {
		// Same char: match from first to next occurrence
		for i := si + 1; i < len(ms.s); i++ {
			if ms.s[i] == close {
				return ms.match(i+1, pp+4)
			}
		}
		return -1
	}
	depth := 1
	i := si + 1
	for i < len(ms.s) && depth > 0 {
		if ms.s[i] == open {
			depth++
		} else if ms.s[i] == close {
			depth--
		}
		i++
	}
	if depth != 0 {
		return -1
	}
	return ms.match(i, pp+4)
}

func (ms *matchState) matchFrontier(si, pp int) int {
	if pp+2 >= len(ms.p) || ms.p[pp+2] != '[' {
		panic("missing '[' after '%f' in pattern")
	}
	set, setLen := parseCharSetAt(ms.p, pp+2)
	if set == nil {
		return -1
	}
	var prev byte = 0
	if si > 0 {
		prev = ms.s[si-1]
	}
	var curr byte = 0
	if si < len(ms.s) {
		curr = ms.s[si]
	}
	if set.matches(curr) && !set.matches(prev) {
		return ms.match(si, pp+2+setLen)
	}
	return -1
}

func (ms *matchState) matchBackRef(si, pp int) int {
	l := int(ms.p[pp+1] - '1')
	if l < 0 || l >= ms.level {
		panic(fmt.Sprintf("invalid capture index %%%d", l+1))
	}
	c := ms.cap[l]
	if c.slen == capUnfinished {
		panic(fmt.Sprintf("invalid capture index %%%d", l+1))
	}
	if c.slen == capPosition {
		return -1 // position captures produce numbers, not strings; backref can't match
	}
	capStr := ms.s[c.init : c.init+c.slen]
	if si+len(capStr) > len(ms.s) {
		return -1
	}
	if ms.s[si:si+len(capStr)] != capStr {
		return -1
	}
	return ms.match(si+len(capStr), pp+2)
}

// matchGreedy handles * (minCount=0) and + (minCount=1).
func (ms *matchState) matchGreedy(si, pp int, elem patternElem, minCount int) int {
	count := 0
	for si+count < len(ms.s) && elem.matches(ms.s[si+count]) {
		count++
	}
	for count >= minCount {
		res := ms.match(si+count, pp)
		if res != -1 {
			return res
		}
		count--
	}
	return -1
}

// matchLazy handles - (non-greedy zero or more).
func (ms *matchState) matchLazy(si, pp int, elem patternElem) int {
	for {
		res := ms.match(si, pp)
		if res != -1 {
			return res
		}
		if si >= len(ms.s) || !elem.matches(ms.s[si]) {
			return -1
		}
		si++
	}
}

// --- Public API ---

// luaMatchAt tries to match pattern at exactly position pos (0-based) in s.
// Returns (endPos, captures, true) on match, or (0, nil, false) on failure.
func luaMatchAt(s, pattern string, pos int) (int, []captureValue, bool) {
	ms := &matchState{s: s, p: pattern}
	end := ms.match(pos, 0)
	if end == -1 {
		return 0, nil, false
	}
	return end, ms.getCaptures(), true
}

// luaMatchFrom searches for first match of pattern in s starting from init (1-based).
// Handles ^-anchor and negative init. Returns (start0, end0, captures, found).
// start0 and end0 are 0-based; end0 is exclusive.
func luaMatchFrom(s, pattern string, init int) (int, int, []captureValue, bool) {
	if init < 0 {
		init = len(s) + init + 1
	}
	if init < 1 {
		init = 1
	}
	if init > len(s)+1 {
		return 0, 0, nil, false
	}

	anchored := false
	searchPat := pattern
	if len(searchPat) > 0 && searchPat[0] == '^' {
		anchored = true
		searchPat = searchPat[1:]
	}

	start0 := init - 1
	if anchored {
		end, caps, ok := luaMatchAt(s, searchPat, start0)
		if ok {
			return start0, end, caps, true
		}
		return 0, 0, nil, false
	}

	for i := start0; i <= len(s); i++ {
		end, caps, ok := luaMatchAt(s, searchPat, i)
		if ok {
			return i, end, caps, true
		}
	}
	return 0, 0, nil, false
}

// --- Pattern element types ---

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
		matched = isLetter(b) || isDigit(b)
	case 'l':
		matched = isLower(b)
	case 'u':
		matched = isUpper(b)
	case 'c':
		matched = isControl(b)
	case 'g':
		matched = isGraph(b)
	case 'p':
		matched = isPunct(b)
	case 'x':
		matched = isHex(b)
	case 'z':
		matched = b == 0
	default:
		matched = b == c.class
	}
	if c.neg {
		return !matched
	}
	return matched
}

type charSet struct {
	elems []patternElem
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

// getPatternElem returns the pattern element at patPos and its length.
func getPatternElem(pattern string, patPos int) (patternElem, int) {
	if patPos >= len(pattern) {
		return nil, 0
	}

	ch := pattern[patPos]

	if ch == '%' {
		if patPos+1 >= len(pattern) {
			panic("malformed pattern (ends with '%')")
		}
		next := pattern[patPos+1]
		switch next {
		case 'a', 'd', 'g', 's', 'w', 'l', 'u', 'c', 'p', 'x', 'z':
			return classChar{class: next, neg: false}, 2
		case 'A', 'D', 'G', 'S', 'W', 'L', 'U', 'C', 'P', 'X', 'Z':
			return classChar{class: next + 32, neg: true}, 2
		default:
			return literalChar{ch: next}, 2
		}
	}

	if ch == '[' {
		set, setLen := parseCharSetAt(pattern, patPos)
		if set == nil {
			return nil, 0
		}
		return set, setLen
	}

	if ch == '.' {
		return anyChar{}, 1
	}

	return literalChar{ch: ch}, 1
}

func parseCharSetAt(pattern string, patPos int) (*charSet, int) {
	neg := false
	start := patPos + 1
	if start < len(pattern) && pattern[start] == '^' {
		neg = true
		start++
	}
	end := start
	// ] at start of set is literal (Lua 5.4 spec)
	if end < len(pattern) && pattern[end] == ']' {
		end++
	}
	for end < len(pattern) && pattern[end] != ']' {
		if pattern[end] == '%' && end+1 < len(pattern) {
			end += 2
		} else {
			end++
		}
	}
	if end >= len(pattern) {
		panic("malformed pattern (missing ']')")
	}
	elems := parseCharSetElems(pattern[start:end])
	return &charSet{elems: elems, neg: neg}, end - patPos + 1
}

func parseCharSetElems(set string) []patternElem {
	var elems []patternElem
	i := 0
	for i < len(set) {
		if set[i] == '%' && i+1 < len(set) {
			next := set[i+1]
			switch next {
			case 'a', 'd', 'g', 's', 'w', 'l', 'u', 'c', 'p', 'x', 'z':
				elems = append(elems, classChar{class: next, neg: false})
			case 'A', 'D', 'G', 'S', 'W', 'L', 'U', 'C', 'P', 'X', 'Z':
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

func isLetter(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }
func isDigit(b byte) bool  { return b >= '0' && b <= '9' }
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}
func isLower(b byte) bool   { return b >= 'a' && b <= 'z' }
func isUpper(b byte) bool   { return b >= 'A' && b <= 'Z' }
func isControl(b byte) bool { return b < 32 || b == 127 }
func isPunct(b byte) bool {
	return (b >= 0x21 && b <= 0x2F) || // !"#$%&'()*+,-./
		(b >= 0x3A && b <= 0x40) || // :;<=>?@
		(b >= 0x5B && b <= 0x60) || // [\]^_`
		(b >= 0x7B && b <= 0x7E) // {|}~
}
func isGraph(b byte) bool { return b > ' ' && b < 127 } // printable non-space
func isHex(b byte) bool   { return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F') }

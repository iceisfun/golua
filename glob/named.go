package glob

import (
	"unicode"
	"unicode/utf8"
)

// MatchNamed reports whether text matches pattern and, if so,
// returns a map of named glob captures. Named globs begin with ':' followed
// immediately by one or more letters.
func MatchNamed(pattern, text string) (bool, map[string]string, error) {
	caps := make(map[string]string)
	ok, finalCaps, err := matchHelper(pattern, text, caps)
	if err != nil {
		return false, make(map[string]string), err
	}
	// Ensure that on a non-match we return an empty (non-nil) map
	if !ok && finalCaps == nil {
		finalCaps = make(map[string]string)
	}
	return ok, finalCaps, nil
}

// matchHelper recursively matches pattern p against text t, carrying along the
// capture map caps.
func matchHelper(p, t string, caps map[string]string) (bool, map[string]string, error) {
	if p == "" {
		return t == "", caps, nil
	}
	r, size := utf8.DecodeRuneInString(p)
	switch r {
	case '*':
		// Skip over any consecutive '*'
		for {
			r2, s2 := utf8.DecodeRuneInString(p)
			if r2 != '*' {
				break
			}
			p = p[s2:]
			// Trailing '*' matches the rest of t
			if p == "" {
				return true, caps, nil
			}
		}
		// Try every possible split of t
		for i := 0; i <= len(t); {
			ok, newCaps, err := matchHelper(p, t[i:], copyMap(caps))
			if err != nil {
				return false, nil, err
			}
			if ok {
				return true, newCaps, nil
			}
			if i == len(t) {
				break
			}
			_, sz := utf8.DecodeRuneInString(t[i:])
			i += sz
		}
		return false, copyMap(caps), nil

	case '?':
		if t == "" {
			return false, copyMap(caps), nil
		}
		_, tsz := utf8.DecodeRuneInString(t)
		return matchHelper(p[size:], t[tsz:], caps)

	case '[':
		// Always validate the character class syntax by calling matchCharClass with a dummy rune.
		_, _, err := matchCharClass(p, 0)
		if err != nil {
			return false, nil, err
		}
		if t == "" {
			return false, copyMap(caps), nil
		}
		tr, tsz := utf8.DecodeRuneInString(t)
		ok, rest, err := matchCharClass(p, tr)
		if err != nil {
			return false, nil, err
		}
		if !ok {
			return false, copyMap(caps), nil
		}
		return matchHelper(rest, t[tsz:], caps)

	case '\\':
		p = p[size:]
		if p == "" {
			return false, nil, ErrBadPattern
		}
		rEsc, escSize := utf8.DecodeRuneInString(p)
		if t == "" {
			return false, copyMap(caps), nil
		}
		tr, tsz := utf8.DecodeRuneInString(t)
		if unicode.ToUpper(rEsc) != unicode.ToUpper(tr) {
			return false, copyMap(caps), nil
		}
		return matchHelper(p[escSize:], t[tsz:], caps)

	case ':':
		// Check if this is a named glob (':' followed by one or more letters)
		temp := p[size:]
		nameRunes := []rune{}
		totalConsumed := 0
		for len(temp) > 0 {
			rTemp, sz := utf8.DecodeRuneInString(temp)
			if unicode.IsLetter(rTemp) {
				nameRunes = append(nameRunes, rTemp)
				totalConsumed += sz
				temp = temp[sz:]
			} else {
				break
			}
		}
		// If valid named glob, try every possible split for the capture.
		if len(nameRunes) > 0 {
			name := string(nameRunes)
			pRest := p[size+totalConsumed:]
			for i := 0; i <= len(t); {
				newCaps := copyMap(caps)
				newCaps[name] = t[:i]
				ok, finalCaps, err := matchHelper(pRest, t[i:], newCaps)
				if err != nil {
					return false, nil, err
				}
				if ok {
					return true, finalCaps, nil
				}
				if i == len(t) {
					break
				}
				_, sz := utf8.DecodeRuneInString(t[i:])
				i += sz
			}
			return false, copyMap(caps), nil
		}
		// Otherwise, treat ':' as a literal.
		if t == "" {
			return false, copyMap(caps), nil
		}
		tr, tsz := utf8.DecodeRuneInString(t)
		if unicode.ToUpper(r) != unicode.ToUpper(tr) {
			return false, copyMap(caps), nil
		}
		return matchHelper(p[size:], t[tsz:], caps)

	default:
		// Literal character match.
		if t == "" {
			return false, copyMap(caps), nil
		}
		tr, tsz := utf8.DecodeRuneInString(t)
		if unicode.ToUpper(r) != unicode.ToUpper(tr) {
			return false, copyMap(caps), nil
		}
		return matchHelper(p[size:], t[tsz:], caps)
	}
}

// matchCharClass processes a character class starting with '[' in p and tests
// whether ch is in that class. If p is malformed (for example, missing a closing
// ']'), ErrBadPattern is returned.
func matchCharClass(p string, ch rune) (bool, string, error) {
	if p == "" || p[0] != '[' {
		return false, "", ErrBadPattern
	}
	// Skip the opening '['.
	p = p[1:]
	negated := false
	if p != "" {
		r, sz := utf8.DecodeRuneInString(p)
		if r == '^' {
			negated = true
			p = p[sz:]
		}
	}
	matched := false
	nrange := 0
	// Process character ranges until the closing ']'
	for {
		if p == "" {
			return false, "", ErrBadPattern
		}
		r, sz := utf8.DecodeRuneInString(p)
		// A closing ']' after at least one range ends the class.
		if r == ']' && nrange > 0 {
			p = p[sz:]
			break
		}
		var lo rune
		if r == '\\' {
			p = p[sz:]
			if p == "" {
				return false, "", ErrBadPattern
			}
			lo, sz = utf8.DecodeRuneInString(p)
			p = p[sz:]
		} else {
			lo = r
			p = p[sz:]
		}
		hi := lo
		// If a '-' follows, treat it as a range.
		if p != "" {
			r2, sz2 := utf8.DecodeRuneInString(p)
			if r2 == '-' {
				p = p[sz2:]
				if p == "" {
					return false, "", ErrBadPattern
				}
				r3, sz3 := utf8.DecodeRuneInString(p)
				if r3 == '\\' {
					p = p[sz3:]
					if p == "" {
						return false, "", ErrBadPattern
					}
					r3, sz3 = utf8.DecodeRuneInString(p)
					p = p[sz3:]
				} else {
					p = p[sz3:]
				}
				hi = r3
				if hi < lo {
					return false, "", ErrBadPattern
				}
			}
		}
		nrange++
		// For dummy ch==0 we ignore matching; otherwise do case-insensitive comparison.
		if ch != 0 && unicode.ToUpper(lo) <= unicode.ToUpper(ch) && unicode.ToUpper(ch) <= unicode.ToUpper(hi) {
			matched = true
		}
	}
	if negated {
		matched = !matched
	}
	return matched, p, nil
}

// copyMap makes a shallow copy of a map.
func copyMap(m map[string]string) map[string]string {
	nm := make(map[string]string, len(m))
	for k, v := range m {
		nm[k] = v
	}
	return nm
}

package glob

// HasPatternCharacters reports whether s contains any glob metacharacters.
func HasPatternCharacters(s string) bool {
	for _, r := range s {
		switch r {
		case '*', '?', '[', ']', '\\':
			return true
		}
	}
	return false
}

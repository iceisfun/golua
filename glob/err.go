package glob

import "errors"

// ErrBadPattern indicates a pattern was malformed.
var ErrBadPattern = errors.New("syntax error in pattern")

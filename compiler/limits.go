package compiler

// Lua 5.4 compiler limit defaults.
const (
	DefaultMaxVars   = 200 // MAXVARS in luaconf.h
	DefaultMaxRegs   = 255 // MAXREGS in luaconf.h
	DefaultMaxUpvals = 255 // MAXUPVAL
)

// CompilerLimits configures compile-time limits for a single compilation.
// Zero values mean use the corresponding default.
type CompilerLimits struct {
	MaxVars   int // max local variables per function (0 = DefaultMaxVars)
	MaxRegs   int // max registers per function (0 = DefaultMaxRegs, hard cap 255)
	MaxUpvals int // max upvalues per function (0 = DefaultMaxUpvals, hard cap 255)
}

func (cl CompilerLimits) effective() CompilerLimits {
	out := cl
	if out.MaxVars <= 0 {
		out.MaxVars = DefaultMaxVars
	}
	if out.MaxRegs <= 0 {
		out.MaxRegs = DefaultMaxRegs
	}
	if out.MaxUpvals <= 0 {
		out.MaxUpvals = DefaultMaxUpvals
	}
	// Hard caps from instruction encoding
	if out.MaxRegs > 255 {
		out.MaxRegs = 255
	}
	if out.MaxUpvals > 255 {
		out.MaxUpvals = 255
	}
	return out
}

// CompileOption is a functional option for Compile().
type CompileOption func(*compileConfig)

type compileConfig struct {
	limits  CompilerLimits
	endLine int // last line of source (for compile error messages, 0 = auto)
}

// WithLimits returns a CompileOption that sets compiler limits.
func WithLimits(limits CompilerLimits) CompileOption {
	return func(cfg *compileConfig) {
		cfg.limits = limits
	}
}

// WithEndLine returns a CompileOption that sets the last line number of the
// source, used for compile error message prefixes (matching Lua 5.4 behavior
// where compile errors reference the EOF line, not the error source line).
func WithEndLine(line int) CompileOption {
	return func(cfg *compileConfig) {
		cfg.endLine = line
	}
}

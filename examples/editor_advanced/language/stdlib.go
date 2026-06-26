package language

// This file is the curated metadata that powers completion detail and hover
// docs. Signatures and descriptions are hand-written (the VM cannot introspect
// human-readable prose), but the *set* of symbols is kept honest against the
// real library by TestStdlibMetadataMatchesVM in stdlib_drift_test.go, which
// opens a live VM and fails if this table lists a symbol the VM does not export
// (or omits one it does). Run `go test ./examples/editor_advanced/language` to
// detect drift after a stdlib change.
//
// Scope note: the entries below mirror exactly what the Run sandbox in main.go
// registers (basic globals + string, math, table, coroutine, bit32, utf8, and
// load). Provider-gated modules (io, os, debug, chan, time, exec) are not
// enabled in the sandbox, so they are intentionally absent here.

// SymbolKindStdlib is used for stdlib entries in completion sorting.
const SymbolKindStdlib = "stdlib"

// StdlibEntry describes a single stdlib symbol.
type StdlibEntry struct {
	Name      string // e.g. "print" or "len"
	Signature string // e.g. "print(...)" or "string.len(s)"
	Doc       string // short description
	Kind      string // "function", "value", "table"
	Parent    string // "" for globals, "string" for string.len, etc.
}

var globals = []StdlibEntry{
	{Name: "print", Signature: "print(...)", Doc: "Prints values to output, separated by tabs.", Kind: "function"},
	{Name: "assert", Signature: "assert(v [, message])", Doc: "Raises an error if v is falsy.", Kind: "function"},
	{Name: "type", Signature: "type(v)", Doc: "Returns the type of v as a string.", Kind: "function"},
	{Name: "tostring", Signature: "tostring(v)", Doc: "Converts v to a string.", Kind: "function"},
	{Name: "tonumber", Signature: "tonumber(v [, base])", Doc: "Converts v to a number.", Kind: "function"},
	{Name: "error", Signature: "error(message [, level])", Doc: "Raises an error with the given message.", Kind: "function"},
	{Name: "warn", Signature: "warn(msg, ...)", Doc: "Emits a warning built by concatenating its string arguments.", Kind: "function"},
	{Name: "pcall", Signature: "pcall(f, ...)", Doc: "Calls f in protected mode. Returns status and results.", Kind: "function"},
	{Name: "xpcall", Signature: "xpcall(f, handler, ...)", Doc: "Like pcall but with a custom error handler.", Kind: "function"},
	{Name: "pairs", Signature: "pairs(t)", Doc: "Returns an iterator for all key-value pairs in table t.", Kind: "function"},
	{Name: "ipairs", Signature: "ipairs(t)", Doc: "Returns an iterator for integer keys 1, 2, ... in table t.", Kind: "function"},
	{Name: "next", Signature: "next(t [, key])", Doc: "Returns the next key-value pair after key in table t.", Kind: "function"},
	{Name: "select", Signature: "select(index, ...)", Doc: "Returns arguments after index, or count with '#'.", Kind: "function"},
	{Name: "rawget", Signature: "rawget(t, k)", Doc: "Gets t[k] without invoking metamethods.", Kind: "function"},
	{Name: "rawset", Signature: "rawset(t, k, v)", Doc: "Sets t[k]=v without invoking metamethods.", Kind: "function"},
	{Name: "rawequal", Signature: "rawequal(a, b)", Doc: "Checks equality without invoking metamethods.", Kind: "function"},
	{Name: "rawlen", Signature: "rawlen(t)", Doc: "Returns the length of t without invoking metamethods.", Kind: "function"},
	{Name: "getmetatable", Signature: "getmetatable(obj)", Doc: "Returns the metatable of obj.", Kind: "function"},
	{Name: "setmetatable", Signature: "setmetatable(t, mt)", Doc: "Sets the metatable of t to mt.", Kind: "function"},
	{Name: "collectgarbage", Signature: "collectgarbage([opt [, arg]])", Doc: "Controls the garbage collector.", Kind: "function"},
	{Name: "load", Signature: "load(chunk [, name [, mode [, env]]])", Doc: "Loads a Lua chunk from a string or function.", Kind: "function"},
	{Name: "require", Signature: "require(modname)", Doc: "Loads and returns the given module (preloaded modules only in the sandbox).", Kind: "function"},
	{Name: "_G", Signature: "_G", Doc: "The global environment table.", Kind: "value"},
	{Name: "_VERSION", Signature: "_VERSION", Doc: "Lua version string.", Kind: "value"},
	// Table namespaces (for completion as identifiers)
	{Name: "string", Signature: "string", Doc: "String manipulation library.", Kind: "table"},
	{Name: "math", Signature: "math", Doc: "Mathematical functions library.", Kind: "table"},
	{Name: "table", Signature: "table", Doc: "Table manipulation library.", Kind: "table"},
	{Name: "coroutine", Signature: "coroutine", Doc: "Coroutine manipulation library.", Kind: "table"},
	{Name: "bit32", Signature: "bit32", Doc: "Bitwise operations library (Lua 5.2 compatibility).", Kind: "table"},
	{Name: "utf8", Signature: "utf8", Doc: "UTF-8 support library.", Kind: "table"},
	{Name: "glob", Signature: "glob", Doc: "Go-style glob pattern matching (GoLua extension).", Kind: "table"},
	{Name: "package", Signature: "package", Doc: "Module system control table.", Kind: "table"},
}

var tables = map[string][]StdlibEntry{
	"string": {
		{Name: "len", Signature: "string.len(s)", Doc: "Returns the length of string s.", Kind: "function", Parent: "string"},
		{Name: "sub", Signature: "string.sub(s, i [, j])", Doc: "Returns the substring from i to j.", Kind: "function", Parent: "string"},
		{Name: "upper", Signature: "string.upper(s)", Doc: "Returns s with all letters uppercase.", Kind: "function", Parent: "string"},
		{Name: "lower", Signature: "string.lower(s)", Doc: "Returns s with all letters lowercase.", Kind: "function", Parent: "string"},
		{Name: "rep", Signature: "string.rep(s, n [, sep])", Doc: "Returns s repeated n times.", Kind: "function", Parent: "string"},
		{Name: "reverse", Signature: "string.reverse(s)", Doc: "Returns s reversed.", Kind: "function", Parent: "string"},
		{Name: "byte", Signature: "string.byte(s [, i [, j]])", Doc: "Returns numeric codes of characters.", Kind: "function", Parent: "string"},
		{Name: "char", Signature: "string.char(...)", Doc: "Returns a string from numeric character codes.", Kind: "function", Parent: "string"},
		{Name: "find", Signature: "string.find(s, pattern [, init [, plain]])", Doc: "Finds the first match of pattern in s.", Kind: "function", Parent: "string"},
		{Name: "format", Signature: "string.format(fmt, ...)", Doc: "Returns a formatted string (like printf).", Kind: "function", Parent: "string"},
		{Name: "gsub", Signature: "string.gsub(s, pattern, repl [, n])", Doc: "Replaces occurrences of pattern in s.", Kind: "function", Parent: "string"},
		{Name: "match", Signature: "string.match(s, pattern [, init])", Doc: "Returns captures from the first match.", Kind: "function", Parent: "string"},
		{Name: "gmatch", Signature: "string.gmatch(s, pattern)", Doc: "Returns an iterator over all matches.", Kind: "function", Parent: "string"},
		{Name: "pack", Signature: "string.pack(fmt, ...)", Doc: "Packs values into a binary string per the format.", Kind: "function", Parent: "string"},
		{Name: "unpack", Signature: "string.unpack(fmt, s [, pos])", Doc: "Unpacks values from a binary string per the format.", Kind: "function", Parent: "string"},
		{Name: "packsize", Signature: "string.packsize(fmt)", Doc: "Returns the byte size of a fixed-size pack format.", Kind: "function", Parent: "string"},
		{Name: "dump", Signature: "string.dump(f [, strip])", Doc: "Returns a binary chunk of a Lua function.", Kind: "function", Parent: "string"},
	},
	"math": {
		{Name: "abs", Signature: "math.abs(x)", Doc: "Returns the absolute value of x.", Kind: "function", Parent: "math"},
		{Name: "acos", Signature: "math.acos(x)", Doc: "Returns the arc cosine of x (in radians).", Kind: "function", Parent: "math"},
		{Name: "asin", Signature: "math.asin(x)", Doc: "Returns the arc sine of x (in radians).", Kind: "function", Parent: "math"},
		{Name: "atan", Signature: "math.atan(y [, x])", Doc: "Returns the arc tangent of y/x (in radians).", Kind: "function", Parent: "math"},
		{Name: "ceil", Signature: "math.ceil(x)", Doc: "Returns the smallest integer >= x.", Kind: "function", Parent: "math"},
		{Name: "cos", Signature: "math.cos(x)", Doc: "Returns the cosine of x (in radians).", Kind: "function", Parent: "math"},
		{Name: "deg", Signature: "math.deg(x)", Doc: "Converts x from radians to degrees.", Kind: "function", Parent: "math"},
		{Name: "exp", Signature: "math.exp(x)", Doc: "Returns e^x.", Kind: "function", Parent: "math"},
		{Name: "floor", Signature: "math.floor(x)", Doc: "Returns the largest integer <= x.", Kind: "function", Parent: "math"},
		{Name: "fmod", Signature: "math.fmod(x, y)", Doc: "Returns the remainder of x/y.", Kind: "function", Parent: "math"},
		{Name: "frexp", Signature: "math.frexp(x)", Doc: "Returns m and e such that x = m * 2^e (GoLua extension).", Kind: "function", Parent: "math"},
		{Name: "ldexp", Signature: "math.ldexp(m, e)", Doc: "Returns m * 2^e (GoLua extension).", Kind: "function", Parent: "math"},
		{Name: "log", Signature: "math.log(x [, base])", Doc: "Returns the logarithm of x.", Kind: "function", Parent: "math"},
		{Name: "max", Signature: "math.max(x, ...)", Doc: "Returns the maximum value.", Kind: "function", Parent: "math"},
		{Name: "min", Signature: "math.min(x, ...)", Doc: "Returns the minimum value.", Kind: "function", Parent: "math"},
		{Name: "modf", Signature: "math.modf(x)", Doc: "Returns the integer and fractional parts of x.", Kind: "function", Parent: "math"},
		{Name: "rad", Signature: "math.rad(x)", Doc: "Converts x from degrees to radians.", Kind: "function", Parent: "math"},
		{Name: "random", Signature: "math.random([m [, n]])", Doc: "Returns a pseudo-random number.", Kind: "function", Parent: "math"},
		{Name: "randomseed", Signature: "math.randomseed([x [, y]])", Doc: "Seeds the random number generator.", Kind: "function", Parent: "math"},
		{Name: "sin", Signature: "math.sin(x)", Doc: "Returns the sine of x (in radians).", Kind: "function", Parent: "math"},
		{Name: "sqrt", Signature: "math.sqrt(x)", Doc: "Returns the square root of x.", Kind: "function", Parent: "math"},
		{Name: "tan", Signature: "math.tan(x)", Doc: "Returns the tangent of x (in radians).", Kind: "function", Parent: "math"},
		{Name: "tointeger", Signature: "math.tointeger(x)", Doc: "Converts x to an integer or returns nil.", Kind: "function", Parent: "math"},
		{Name: "type", Signature: "math.type(x)", Doc: "Returns \"integer\", \"float\", or false.", Kind: "function", Parent: "math"},
		{Name: "ult", Signature: "math.ult(m, n)", Doc: "Unsigned integer comparison.", Kind: "function", Parent: "math"},
		{Name: "pi", Signature: "math.pi", Doc: "The value of pi (3.14159...).", Kind: "value", Parent: "math"},
		{Name: "huge", Signature: "math.huge", Doc: "Positive infinity.", Kind: "value", Parent: "math"},
		{Name: "maxinteger", Signature: "math.maxinteger", Doc: "Maximum integer value.", Kind: "value", Parent: "math"},
		{Name: "mininteger", Signature: "math.mininteger", Doc: "Minimum integer value.", Kind: "value", Parent: "math"},
	},
	"table": {
		{Name: "concat", Signature: "table.concat(list [, sep [, i [, j]]])", Doc: "Concatenates list elements into a string.", Kind: "function", Parent: "table"},
		{Name: "insert", Signature: "table.insert(list, [pos,] value)", Doc: "Inserts a value into a list.", Kind: "function", Parent: "table"},
		{Name: "remove", Signature: "table.remove(list [, pos])", Doc: "Removes an element from a list.", Kind: "function", Parent: "table"},
		{Name: "sort", Signature: "table.sort(list [, comp])", Doc: "Sorts list elements in-place.", Kind: "function", Parent: "table"},
		{Name: "unpack", Signature: "table.unpack(list [, i [, j]])", Doc: "Returns elements from a list.", Kind: "function", Parent: "table"},
		{Name: "pack", Signature: "table.pack(...)", Doc: "Packs arguments into a table with field n.", Kind: "function", Parent: "table"},
		{Name: "move", Signature: "table.move(a1, f, e, t [, a2])", Doc: "Moves elements between tables.", Kind: "function", Parent: "table"},
		{Name: "create", Signature: "table.create(n [, nrec])", Doc: "Creates a table pre-sized for n array slots (GoLua/5.5 extension).", Kind: "function", Parent: "table"},
	},
	"coroutine": {
		{Name: "create", Signature: "coroutine.create(f)", Doc: "Creates a new coroutine from function f.", Kind: "function", Parent: "coroutine"},
		{Name: "resume", Signature: "coroutine.resume(co [, ...])", Doc: "Resumes a coroutine.", Kind: "function", Parent: "coroutine"},
		{Name: "yield", Signature: "coroutine.yield(...)", Doc: "Suspends the running coroutine.", Kind: "function", Parent: "coroutine"},
		{Name: "status", Signature: "coroutine.status(co)", Doc: "Returns the status of coroutine co.", Kind: "function", Parent: "coroutine"},
		{Name: "running", Signature: "coroutine.running()", Doc: "Returns the running coroutine and a boolean.", Kind: "function", Parent: "coroutine"},
		{Name: "wrap", Signature: "coroutine.wrap(f)", Doc: "Creates a coroutine wrapper function.", Kind: "function", Parent: "coroutine"},
		{Name: "isyieldable", Signature: "coroutine.isyieldable()", Doc: "Returns true if the coroutine can yield.", Kind: "function", Parent: "coroutine"},
		{Name: "close", Signature: "coroutine.close(co)", Doc: "Closes a coroutine.", Kind: "function", Parent: "coroutine"},
	},
	"bit32": {
		{Name: "band", Signature: "bit32.band(...)", Doc: "Bitwise AND of arguments.", Kind: "function", Parent: "bit32"},
		{Name: "bor", Signature: "bit32.bor(...)", Doc: "Bitwise OR of arguments.", Kind: "function", Parent: "bit32"},
		{Name: "bxor", Signature: "bit32.bxor(...)", Doc: "Bitwise XOR of arguments.", Kind: "function", Parent: "bit32"},
		{Name: "bnot", Signature: "bit32.bnot(x)", Doc: "Bitwise NOT of x.", Kind: "function", Parent: "bit32"},
		{Name: "lshift", Signature: "bit32.lshift(x, disp)", Doc: "Logical left shift.", Kind: "function", Parent: "bit32"},
		{Name: "rshift", Signature: "bit32.rshift(x, disp)", Doc: "Logical right shift.", Kind: "function", Parent: "bit32"},
		{Name: "arshift", Signature: "bit32.arshift(x, disp)", Doc: "Arithmetic right shift.", Kind: "function", Parent: "bit32"},
		{Name: "lrotate", Signature: "bit32.lrotate(x, disp)", Doc: "Left rotation.", Kind: "function", Parent: "bit32"},
		{Name: "rrotate", Signature: "bit32.rrotate(x, disp)", Doc: "Right rotation.", Kind: "function", Parent: "bit32"},
		{Name: "extract", Signature: "bit32.extract(n, field [, width])", Doc: "Extracts bits from n.", Kind: "function", Parent: "bit32"},
		{Name: "replace", Signature: "bit32.replace(n, v, field [, width])", Doc: "Replaces bits in n.", Kind: "function", Parent: "bit32"},
		{Name: "btest", Signature: "bit32.btest(...)", Doc: "Tests if bitwise AND is non-zero.", Kind: "function", Parent: "bit32"},
	},
	"utf8": {
		{Name: "char", Signature: "utf8.char(...)", Doc: "Returns a string from Unicode codepoints.", Kind: "function", Parent: "utf8"},
		{Name: "codepoint", Signature: "utf8.codepoint(s [, i [, j]])", Doc: "Returns codepoints from a UTF-8 string.", Kind: "function", Parent: "utf8"},
		{Name: "codes", Signature: "utf8.codes(s)", Doc: "Returns an iterator over UTF-8 codepoints.", Kind: "function", Parent: "utf8"},
		{Name: "len", Signature: "utf8.len(s [, i [, j]])", Doc: "Returns the number of UTF-8 characters.", Kind: "function", Parent: "utf8"},
		{Name: "offset", Signature: "utf8.offset(s, n [, i])", Doc: "Returns the byte position of the n-th character.", Kind: "function", Parent: "utf8"},
		{Name: "charpattern", Signature: "utf8.charpattern", Doc: "Pattern matching a single UTF-8 character.", Kind: "value", Parent: "utf8"},
	},
	"glob": {
		{Name: "match", Signature: "glob.match(pattern, name)", Doc: "Reports whether name matches the Go-style glob pattern.", Kind: "function", Parent: "glob"},
		{Name: "match_words", Signature: "glob.match_words(pattern, name)", Doc: "Splits on whitespace and matches each word against the pattern.", Kind: "function", Parent: "glob"},
		{Name: "match_named", Signature: "glob.match_named(pattern, text)", Doc: "Returns matched and a table of named captures.", Kind: "function", Parent: "glob"},
		{Name: "has_pattern", Signature: "glob.has_pattern(s)", Doc: "Reports whether s contains glob metacharacters.", Kind: "function", Parent: "glob"},
	},
	"package": {
		{Name: "loaded", Signature: "package.loaded", Doc: "Table of already-loaded modules.", Kind: "table", Parent: "package"},
		{Name: "preload", Signature: "package.preload", Doc: "Table of preload loaders, keyed by module name.", Kind: "table", Parent: "package"},
		{Name: "searchers", Signature: "package.searchers", Doc: "List of module searcher functions.", Kind: "table", Parent: "package"},
		{Name: "path", Signature: "package.path", Doc: "Search path for Lua modules.", Kind: "value", Parent: "package"},
		{Name: "cpath", Signature: "package.cpath", Doc: "Search path for C modules.", Kind: "value", Parent: "package"},
		{Name: "config", Signature: "package.config", Doc: "Compile-time path configuration string.", Kind: "value", Parent: "package"},
		{Name: "searchpath", Signature: "package.searchpath(name, path [, sep [, rep]])", Doc: "Searches for name in a path template.", Kind: "function", Parent: "package"},
		{Name: "loadlib", Signature: "package.loadlib(libname, funcname)", Doc: "Loads a dynamic library (disabled without a loadlib provider).", Kind: "function", Parent: "package"},
	},
}

var allByFQ = buildAllByFQ()

func buildAllByFQ() map[string]StdlibEntry {
	m := make(map[string]StdlibEntry)
	for _, e := range globals {
		m[e.Name] = e
	}
	for parent, members := range tables {
		for _, e := range members {
			m[parent+"."+e.Name] = e
		}
	}
	return m
}

// StdlibGlobals returns all top-level stdlib symbols.
func StdlibGlobals() []StdlibEntry { return globals }

// StdlibMembers returns the members of a stdlib table (e.g. "string").
func StdlibMembers(tableName string) []StdlibEntry { return tables[tableName] }

// StdlibLookup looks up a symbol by fully-qualified name (e.g. "string.format" or "print").
func StdlibLookup(name string) (StdlibEntry, bool) {
	e, ok := allByFQ[name]
	return e, ok
}

// StdlibGlobalNames returns the set of all stdlib global names for scope pre-population.
func StdlibGlobalNames() map[string]bool {
	m := make(map[string]bool, len(globals))
	for _, e := range globals {
		m[e.Name] = true
	}
	return m
}

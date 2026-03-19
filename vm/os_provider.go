package vm

import "context"

// LuaOsCaps declares which os-library operations are exposed to Lua.
type LuaOsCaps struct {
	// AllowTime enables os.time.
	AllowTime bool
	// AllowDate enables os.date.
	AllowDate bool
	// AllowGetenv enables os.getenv.
	AllowGetenv bool
	// AllowTmpName enables os.tmpname.
	AllowTmpName bool
	// AllowRemove enables os.remove.
	AllowRemove bool
	// AllowExecute enables os.execute when an exec provider is also present.
	AllowExecute bool
	// AllowExit enables os.exit when an exit handler is also present.
	AllowExit bool
	// AllowRename enables os.rename.
	AllowRename bool
}

// LuaDateTime is the structured date/time table shape used by os.date and os.time.
type LuaDateTime struct {
	// Year is the full year, such as 2026.
	Year int
	// Month is 1-12.
	Month int
	// Day is 1-31.
	Day int
	// Hour is 0-23.
	Hour int
	// Min is 0-59.
	Min int
	// Sec is 0-60, matching Lua's leap-second-friendly contract.
	Sec int
	// Wday is 1-7 with Sunday == 1, matching Lua.
	Wday int
	// Yday is 1-366.
	Yday int
	// IsDST reports whether daylight saving time is in effect.
	IsDST bool
	// HasDST reports whether IsDST is known for this timestamp.
	HasDST bool
}

// LuaTimeInput is the input table shape accepted by os.time.
type LuaTimeInput struct {
	// Year is required.
	Year int
	// Month is required and uses 1-12.
	Month int
	// Day is required and uses 1-31.
	Day int
	// Hour defaults to 12 when omitted, matching Lua.
	Hour int
	// Min defaults to 0 when omitted.
	Min int
	// Sec defaults to 0 when omitted.
	Sec int
	// HasIsDST reports whether IsDST was provided by the caller.
	HasIsDST bool
	// IsDST requests daylight-saving interpretation when HasIsDST is true.
	IsDST bool
}

// LuaOsProvider is a capability interface for sandboxed OS operations.
// When provided to a VM, the stdlib os library becomes available.
// Without a provider, os.* functions are not registered.
//
// Lua 5.4 Reference: §6.9 (operating system facilities).
type LuaOsProvider interface {
	// Clock returns CPU time used by the program in seconds.
	Clock(ctx context.Context) float64

	// Time returns the current time as a Unix timestamp,
	// or constructs a timestamp from the given date table fields.
	// If dateTable is nil, returns current time. When constructing from a
	// table, it also returns normalized local-time fields like Lua's mktime.
	Time(ctx context.Context, dateTable *LuaTimeInput) (int64, *LuaDateTime, error)

	// Date formats a timestamp according to a format string.
	// Returns the formatted date string.
	Date(ctx context.Context, format string, timestamp int64) (string, error)

	// DateTable returns a table of date/time components for the given timestamp.
	// Keys: "year", "month", "day", "hour", "min", "sec", "wday", "yday", "isdst"
	// If utc is true, the components are in UTC; otherwise local time.
	DateTable(ctx context.Context, timestamp int64, utc bool) *LuaDateTime

	// Getenv returns the value of an environment variable.
	Getenv(ctx context.Context, name string) (string, bool)

	// SetLocale applies/query locale state for os.setlocale.
	// Returns (locale, true) on success, or (_, false) when unsupported.
	SetLocale(ctx context.Context, locale, category string) (string, bool)

	// Capabilities declares which OS behaviors are allowed.
	Capabilities(ctx context.Context) LuaOsCaps
}

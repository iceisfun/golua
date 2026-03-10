package vm

// LuaOsCaps declares which OS operations are allowed.
type LuaOsCaps struct {
	AllowTime    bool
	AllowDate    bool
	AllowGetenv  bool
	AllowTmpName bool
	AllowRemove  bool
	AllowExecute bool
	AllowExit    bool
	AllowRename  bool
}

type LuaDateTime struct {
	Year   int
	Month  int
	Day    int
	Hour   int
	Min    int
	Sec    int
	Wday   int
	Yday   int
	IsDST  bool
	HasDST bool
}

type LuaTimeInput struct {
	Year     int
	Month    int
	Day      int
	Hour     int
	Min      int
	Sec      int
	HasIsDST bool
	IsDST    bool
}

// LuaOsProvider is a capability interface for sandboxed OS operations.
// When provided to a VM, the stdlib os library becomes available.
// Without a provider, os.* functions are not registered.
//
// Lua 5.4 Reference: §6.9 (operating system facilities).
type LuaOsProvider interface {
	// Clock returns CPU time used by the program in seconds.
	Clock() float64

	// Time returns the current time as a Unix timestamp,
	// or constructs a timestamp from the given date table fields.
	// If dateTable is nil, returns current time. When constructing from a
	// table, it also returns normalized local-time fields like Lua's mktime.
	Time(dateTable *LuaTimeInput) (int64, *LuaDateTime, error)

	// Date formats a timestamp according to a format string.
	// Returns the formatted date string.
	Date(format string, timestamp int64) (string, error)

	// DateTable returns a table of date/time components for the given timestamp.
	// Keys: "year", "month", "day", "hour", "min", "sec", "wday", "yday", "isdst"
	// If utc is true, the components are in UTC; otherwise local time.
	DateTable(timestamp int64, utc bool) *LuaDateTime

	// Getenv returns the value of an environment variable.
	Getenv(name string) (string, bool)

	// SetLocale applies/query locale state for os.setlocale.
	// Returns (locale, true) on success, or (_, false) when unsupported.
	SetLocale(locale, category string) (string, bool)

	// Capabilities declares which OS behaviors are allowed.
	Capabilities() LuaOsCaps
}

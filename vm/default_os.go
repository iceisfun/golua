package vm

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

// DefaultOsProvider provides standard OS operations.
type DefaultOsProvider struct {
	startTime time.Time
	envFilter func(string) bool // optional filter for getenv
	locale    string            // current locale, starts as "C" per Lua 5.4
}

// NewDefaultOsProvider creates a new DefaultOsProvider.
func NewDefaultOsProvider() *DefaultOsProvider {
	return &DefaultOsProvider{
		startTime: time.Now(),
		locale:    "C",
	}
}

// NewFilteredOsProvider creates a DefaultOsProvider with an environment variable filter.
// The filter function returns true for variable names that are allowed.
func NewFilteredOsProvider(filter func(string) bool) *DefaultOsProvider {
	return &DefaultOsProvider{
		startTime: time.Now(),
		envFilter: filter,
		locale:    "C",
	}
}

// Clock returns CPU time in seconds since the VM started.
func (p *DefaultOsProvider) Clock(ctx context.Context) float64 {
	return time.Since(p.startTime).Seconds()
}

// Time returns the current Unix timestamp, or constructs one from dateTable fields.
func (p *DefaultOsProvider) Time(ctx context.Context, dateTable *LuaTimeInput) (int64, *LuaDateTime, error) {
	if dateTable == nil {
		return time.Now().Unix(), nil, nil
	}

	t, err := resolveLocalTime(*dateTable)
	if err != nil {
		return 0, nil, err
	}
	return t.Unix(), dateTimeFromTime(t, true), nil
}

// Date formats a timestamp using strftime-style format specifiers.
func (p *DefaultOsProvider) Date(ctx context.Context, format string, timestamp int64) (string, error) {
	t := time.Unix(timestamp, 0)

	// Check for ! prefix (UTC)
	utc := false
	if strings.HasPrefix(format, "!") {
		t = t.UTC()
		format = format[1:]
		utc = true
	}

	// Lua 5.4 returns an error when C's gmtime/localtime returns NULL for
	// out-of-range timestamps. C's struct tm uses a 32-bit int for tm_year,
	// so years outside roughly [-2147483648+1900, 2147483647+1900] fail.
	// Go's time.Time has no such limit, so we check explicitly.
	if y := t.Year(); y > math.MaxInt32+1900 || y < math.MinInt32+1900 {
		return "", fmt.Errorf("date result cannot be represented in this installation")
	}

	return strftimeFormat(format, t, utc)
}

// DateTable returns a map of date/time components for the given timestamp.
func (p *DefaultOsProvider) DateTable(ctx context.Context, timestamp int64, utc bool) *LuaDateTime {
	t := time.Unix(timestamp, 0)
	if utc {
		t = t.UTC()
	}
	return dateTimeFromTime(t, true)
}

func dateTimeFromTime(t time.Time, hasDST bool) *LuaDateTime {
	return &LuaDateTime{
		Year:   t.Year(),
		Month:  int(t.Month()),
		Day:    t.Day(),
		Hour:   t.Hour(),
		Min:    t.Minute(),
		Sec:    t.Second(),
		Wday:   int(t.Weekday()) + 1,
		Yday:   t.YearDay(),
		IsDST:  t.IsDST(),
		HasDST: hasDST,
	}
}

func resolveLocalTime(input LuaTimeInput) (time.Time, error) {
	base := time.Date(input.Year, time.Month(input.Month), input.Day, input.Hour, input.Min, input.Sec, 0, time.Local)
	if !input.HasIsDST {
		return base, nil
	}

	// C's mktime uses tm_isdst as a hint: when isdst=1, it uses the DST
	// offset; when isdst=0, it uses the standard offset. This matters even
	// for dates that are unambiguously in one zone — mktime will use the
	// requested offset and normalize the wall clock fields.
	//
	// To replicate this in Go, we find which offsets the timezone uses
	// (standard vs DST) by sampling across the year, then pick the offset
	// matching the isdst hint and compute the Unix timestamp directly.

	type zoneInfo struct {
		offset int
		isDST  bool
	}
	var zones []zoneInfo
	seen := map[int]bool{}
	addOffset := func(t time.Time) {
		_, off := t.Zone()
		if !seen[off] {
			seen[off] = true
			zones = append(zones, zoneInfo{off, t.IsDST()})
		}
	}
	// Sample each month to find both standard and DST offsets
	for m := 1; m <= 12; m++ {
		addOffset(time.Date(input.Year, time.Month(m), 15, 12, 0, 0, 0, time.Local))
	}
	// Also sample near the target date
	for _, delta := range []time.Duration{-48 * time.Hour, -24 * time.Hour, 0, 24 * time.Hour, 48 * time.Hour} {
		addOffset(base.Add(delta))
	}

	// Find the offset matching the requested DST state
	localBase := time.Date(input.Year, time.Month(input.Month), input.Day, input.Hour, input.Min, input.Sec, 0, time.UTC)

	// First try: find an offset where the zone's DST flag matches the request
	for _, z := range zones {
		if z.isDST == input.IsDST {
			cand := time.Unix(localBase.Unix()-int64(z.offset), 0).In(time.Local)
			return cand, nil
		}
	}

	// No zone in this year matches the requested DST state. C's mktime
	// returns -1 here (e.g., requesting isdst=1 under TZ=UTC); reference
	// Lua surfaces that as a runtime error.
	return time.Time{}, fmt.Errorf("time result cannot be represented in this installation")
}

// Getenv returns an environment variable, respecting the optional filter.
func (p *DefaultOsProvider) Getenv(ctx context.Context, name string) (string, bool) {
	if p.envFilter != nil && !p.envFilter(name) {
		return "", false
	}
	return os.LookupEnv(name)
}

// SetLocale implements os.setlocale semantics for the default provider.
// Go has no process-wide locale API, so we support:
//   - query (nil/no arg → returns current locale)
//   - set "C" → set and return "C"
//   - set "" → set to system default from environment
//
// Any other locale is treated as unsupported (returns nil).
func (p *DefaultOsProvider) SetLocale(ctx context.Context, locale, category string) (string, bool) {
	if locale == "\x00query" {
		// Query current locale (nil arg in Lua maps to this sentinel)
		return p.locale, true
	}
	if locale == "C" {
		p.locale = "C"
		return "C", true
	}
	if locale == "" {
		if envLocale := localeFromEnv(category); envLocale != "" {
			p.locale = envLocale
			return envLocale, true
		}
		p.locale = "C"
		return "C", true
	}
	return "", false
}

func localeFromEnv(category string) string {
	// POSIX precedence: LC_ALL > LC_<category> > LANG
	var keys []string
	switch category {
	case "collate":
		keys = []string{"LC_ALL", "LC_COLLATE", "LANG"}
	case "ctype":
		keys = []string{"LC_ALL", "LC_CTYPE", "LANG"}
	case "monetary":
		keys = []string{"LC_ALL", "LC_MONETARY", "LANG"}
	case "numeric":
		keys = []string{"LC_ALL", "LC_NUMERIC", "LANG"}
	case "time":
		keys = []string{"LC_ALL", "LC_TIME", "LANG"}
	default:
		keys = []string{"LC_ALL", "LANG"}
	}

	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			return v
		}
	}
	return ""
}

// Capabilities returns caps with all OS operations enabled.
func (p *DefaultOsProvider) Capabilities(ctx context.Context) LuaOsCaps {
	return LuaOsCaps{
		AllowTime:    true,
		AllowDate:    true,
		AllowGetenv:  true,
		AllowTmpName: true,
		AllowRemove:  true,
		AllowExecute: true,
		AllowExit:    true,
		AllowRename:  true,
	}
}

// strftimeFormat converts a strftime format string and formats time t.
// Some specifiers map directly to Go layout strings; others require
// runtime computation and are emitted as literal text.
// altModifierValid reports whether the POSIX alternate-representation
// modifier mod ('E' or 'O') is valid before base specifier, matching the
// two-char option blocks in reference Lua's LUA_STRFTIMEOPTIONS (C99):
//
//	E: Ec EC Ex EX Ey EY
//	O: Od Oe OH OI Om OM OS Ou OU OV Ow OW Oy
func altModifierValid(mod, base byte) bool {
	switch mod {
	case 'E':
		return strings.IndexByte("cCxXyY", base) >= 0
	case 'O':
		return strings.IndexByte("deHImMSuUVwWy", base) >= 0
	}
	return false
}

func strftimeFormat(format string, t time.Time, utc bool) (string, error) {
	var result strings.Builder
	i := 0
	for i < len(format) {
		if format[i] == '%' {
			if i+1 >= len(format) {
				return "", fmt.Errorf("invalid conversion specifier '%%'")
			}
			i++
			// POSIX alternate-representation modifiers %E* and %O*. In the
			// C/POSIX locale (and in Go, which has no locale alternates) they
			// are no-ops that format as the base specifier. Only the
			// combinations in reference Lua's LUA_STRFTIMEOPTIONS are valid;
			// anything else falls through to the invalid-specifier error below.
			if format[i] == 'E' || format[i] == 'O' {
				if i+1 >= len(format) || !altModifierValid(format[i], format[i+1]) {
					return "", fmt.Errorf("invalid conversion specifier '%%%s'", format[i:])
				}
				i++ // consume modifier; format the base specifier
			}
			// Compound specifiers (%c %x %X %D %F %r %R %T) expand to a sequence
			// of primitives. Formatting the expansion recursively — rather than
			// rewriting the format string up front — keeps the original format
			// intact for the invalid-specifier error message (so '%Q%R' reports
			// '%Q%R', not the expanded '%Q%H:%M') and lets primitives like %Y
			// use their natural width.
			if exp, ok := compoundExpansion(format[i]); ok {
				s, err := strftimeFormat(exp, t, utc)
				if err != nil {
					return "", err
				}
				result.WriteString(s)
				i++
				continue
			}
			switch format[i] {
			case 'Y':
				// Natural-width year (no zero-padding); matches glibc strftime.
				result.WriteString(fmt.Sprintf("%d", t.Year()))
			case 'y':
				// Floored year-of-century (year -2 -> "98"), matching glibc's
				// floored division for negative years.
				result.WriteString(fmt.Sprintf("%02d", floorMod100(t.Year())))
			case 'm':
				result.WriteString(t.Format("01"))
			case 'd':
				result.WriteString(t.Format("02"))
			case 'H':
				result.WriteString(t.Format("15"))
			case 'M':
				result.WriteString(t.Format("04"))
			case 'S':
				result.WriteString(t.Format("05"))
			case 'p':
				result.WriteString(t.Format("PM"))
			case 'I':
				result.WriteString(t.Format("03"))
			case 'A':
				result.WriteString(t.Format("Monday"))
			case 'a':
				result.WriteString(t.Format("Mon"))
			case 'B':
				result.WriteString(t.Format("January"))
			case 'b', 'h':
				result.WriteString(t.Format("Jan"))
			case 'Z':
				zone := t.Format("MST")
				// C's gmtime path names the UTC zone "GMT"; localtime under
				// TZ=UTC names it "UTC". Only remap on the gmtime ('!') path.
				if utc && zone == "UTC" {
					zone = "GMT"
				}
				result.WriteString(zone)
			case 'z':
				result.WriteString(t.Format("-0700"))
			case '%':
				result.WriteByte('%')
			case 'C':
				// Century via floored division (natural width): year 305 -> "3",
				// year -2 -> "-1" (so %C*100 + %y == %Y holds for negative years).
				result.WriteString(fmt.Sprintf("%d", (t.Year()-floorMod100(t.Year()))/100))
			case 'e':
				// Space-padded day of month
				result.WriteString(fmt.Sprintf("%2d", t.Day()))
			case 'j':
				// Day of year, zero-padded to 3 digits
				result.WriteString(fmt.Sprintf("%03d", t.YearDay()))
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'u':
				// ISO weekday: Monday=1, Sunday=7
				wd := int(t.Weekday())
				if wd == 0 {
					wd = 7
				}
				result.WriteString(fmt.Sprintf("%d", wd))
			case 'w':
				// Weekday number: Sunday=0
				result.WriteString(fmt.Sprintf("%d", int(t.Weekday())))
			case 'g':
				// ISO 8601 2-digit year, floored for negative years
				isoYear, _ := t.ISOWeek()
				result.WriteString(fmt.Sprintf("%02d", floorMod100(isoYear)))
			case 'G':
				// ISO 8601 week-based year, natural width (matches glibc strftime).
				isoYear, _ := t.ISOWeek()
				result.WriteString(fmt.Sprintf("%d", isoYear))
			case 'V':
				// ISO 8601 week number
				_, isoWeek := t.ISOWeek()
				result.WriteString(fmt.Sprintf("%02d", isoWeek))
			case 'U':
				// Week of year, Sunday as first day
				yday := t.YearDay()
				wday := int(t.Weekday()) // Sunday=0
				result.WriteString(fmt.Sprintf("%02d", (yday+6-wday)/7))
			case 'W':
				// Week of year, Monday as first day
				yday := t.YearDay()
				wday := int(t.Weekday())
				// Shift so Monday=0, Sunday=6
				if wday == 0 {
					wday = 6
				} else {
					wday--
				}
				result.WriteString(fmt.Sprintf("%02d", (yday+6-wday)/7))
			default:
				// Match reference Lua: print the substring from this '%'
				// through end of format (lua_pushfstring("%%%s", conv) where
				// conv is a C-string starting at the '%').
				return "", fmt.Errorf("invalid conversion specifier '%%%s'", format[i:])
			}
		} else {
			result.WriteByte(format[i])
		}
		i++
	}
	return result.String(), nil
}

// floorMod100 returns n mod 100 in the range [0,99] using floored (not
// truncated) division, matching glibc strftime's %y/%g/%C for negative years
// (e.g. year -2 -> 98, century -1).
func floorMod100(n int) int {
	m := n % 100
	if m < 0 {
		m += 100
	}
	return m
}

// compoundExpansion maps a compound strftime specifier to its primitive
// (C/POSIX-locale) equivalent. The caller formats the expansion recursively, so
// primitives like %Y keep their natural width and the original format string is
// preserved for error reporting.
func compoundExpansion(conv byte) (string, bool) {
	switch conv {
	case 'c':
		return "%a %b %e %H:%M:%S %Y", true
	case 'x', 'D':
		return "%m/%d/%y", true
	case 'X', 'T':
		return "%H:%M:%S", true
	case 'F':
		return "%Y-%m-%d", true
	case 'r':
		return "%I:%M:%S %p", true
	case 'R':
		return "%H:%M", true
	}
	return "", false
}

package vm

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// DefaultOsProvider provides standard OS operations.
type DefaultOsProvider struct {
	startTime time.Time
	envFilter func(string) bool // optional filter for getenv
}

// NewDefaultOsProvider creates a new DefaultOsProvider.
func NewDefaultOsProvider() *DefaultOsProvider {
	return &DefaultOsProvider{
		startTime: time.Now(),
	}
}

// NewFilteredOsProvider creates a DefaultOsProvider with an environment variable filter.
// The filter function returns true for variable names that are allowed.
func NewFilteredOsProvider(filter func(string) bool) *DefaultOsProvider {
	return &DefaultOsProvider{
		startTime: time.Now(),
		envFilter: filter,
	}
}

// Clock returns CPU time in seconds since the VM started.
func (p *DefaultOsProvider) Clock() float64 {
	return time.Since(p.startTime).Seconds()
}

// Time returns the current Unix timestamp, or constructs one from dateTable fields.
func (p *DefaultOsProvider) Time(dateTable map[string]int) (int64, error) {
	if dateTable == nil {
		return time.Now().Unix(), nil
	}

	year := dateTable["year"]
	month := dateTable["month"]
	day := dateTable["day"]
	hour := dateTable["hour"]
	min := dateTable["min"]
	sec := dateTable["sec"]

	t := time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
	return t.Unix(), nil
}

// Date formats a timestamp using strftime-style format specifiers.
func (p *DefaultOsProvider) Date(format string, timestamp int64) (string, error) {
	t := time.Unix(timestamp, 0)

	// Check for ! prefix (UTC)
	if strings.HasPrefix(format, "!") {
		t = t.UTC()
		format = format[1:]
	}

	return strftimeFormat(format, t)
}

// DateTable returns a map of date/time components for the given timestamp.
func (p *DefaultOsProvider) DateTable(timestamp int64, utc bool) map[string]int {
	t := time.Unix(timestamp, 0)
	if utc {
		t = t.UTC()
	}

	wday := int(t.Weekday()) + 1 // Lua: Sunday = 1
	yday := t.YearDay()

	isdst := 0
	if t.IsDST() {
		isdst = 1
	}

	return map[string]int{
		"year":  t.Year(),
		"month": int(t.Month()),
		"day":   t.Day(),
		"hour":  t.Hour(),
		"min":   t.Minute(),
		"sec":   t.Second(),
		"wday":  wday,
		"yday":  yday,
		"isdst": isdst,
	}
}

// Getenv returns an environment variable, respecting the optional filter.
func (p *DefaultOsProvider) Getenv(name string) (string, bool) {
	if p.envFilter != nil && !p.envFilter(name) {
		return "", false
	}
	return os.LookupEnv(name)
}

// Capabilities returns caps with all OS operations enabled.
func (p *DefaultOsProvider) Capabilities() LuaOsCaps {
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
func strftimeFormat(format string, t time.Time) (string, error) {
	// First expand compound specifiers so the main loop only handles primitives.
	format = expandCompoundSpecifiers(format)

	var result strings.Builder
	i := 0
	for i < len(format) {
		if format[i] == '%' {
			if i+1 >= len(format) {
				return "", fmt.Errorf("invalid conversion specifier in 'date'")
			}
			i++
			switch format[i] {
			case 'Y':
				result.WriteString(t.Format("2006"))
			case 'y':
				result.WriteString(t.Format("06"))
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
			case 'c':
				result.WriteString(t.Format("Mon Jan _2 15:04:05 2006"))
			case 'X':
				result.WriteString(t.Format("15:04:05"))
			case 'x':
				result.WriteString(t.Format("01/02/06"))
			case 'Z':
				result.WriteString(t.Format("MST"))
			case 'z':
				result.WriteString(t.Format("-0700"))
			case '%':
				result.WriteByte('%')
			case 'C':
				// Century: first 2 digits of the year
				result.WriteString(fmt.Sprintf("%02d", t.Year()/100))
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
				// ISO 8601 2-digit year
				isoYear, _ := t.ISOWeek()
				result.WriteString(fmt.Sprintf("%02d", isoYear%100))
			case 'G':
				// ISO 8601 4-digit year
				isoYear, _ := t.ISOWeek()
				result.WriteString(fmt.Sprintf("%04d", isoYear))
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
				return "", fmt.Errorf("invalid conversion specifier '%c%c'", '%', format[i])
			}
		} else {
			result.WriteByte(format[i])
		}
		i++
	}
	return result.String(), nil
}

// expandCompoundSpecifiers replaces compound strftime specifiers with their
// primitive equivalents before the main formatting pass.
func expandCompoundSpecifiers(format string) string {
	var result strings.Builder
	i := 0
	for i < len(format) {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 'D':
				result.WriteString("%m/%d/%y")
				i += 2
				continue
			case 'F':
				result.WriteString("%Y-%m-%d")
				i += 2
				continue
			case 'r':
				result.WriteString("%I:%M:%S %p")
				i += 2
				continue
			case 'R':
				result.WriteString("%H:%M")
				i += 2
				continue
			case 'T':
				result.WriteString("%H:%M:%S")
				i += 2
				continue
			}
		}
		result.WriteByte(format[i])
		i++
	}
	return result.String()
}

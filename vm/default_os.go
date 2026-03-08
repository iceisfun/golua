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

	if year == 0 {
		return 0, fmt.Errorf("field 'year' missing in date table")
	}

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

	goFormat, err := strftimeToGo(format)
	if err != nil {
		return "", err
	}
	return t.Format(goFormat), nil
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

// strftimeToGo converts a strftime format string to a Go time.Format layout.
func strftimeToGo(format string) (string, error) {
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
				result.WriteString("2006")
			case 'y':
				result.WriteString("06")
			case 'm':
				result.WriteString("01")
			case 'd':
				result.WriteString("02")
			case 'H':
				result.WriteString("15")
			case 'M':
				result.WriteString("04")
			case 'S':
				result.WriteString("05")
			case 'p':
				result.WriteString("PM")
			case 'I':
				result.WriteString("03")
			case 'A':
				result.WriteString("Monday")
			case 'a':
				result.WriteString("Mon")
			case 'B':
				result.WriteString("January")
			case 'b', 'h':
				result.WriteString("Jan")
			case 'c':
				result.WriteString("Mon Jan _2 15:04:05 2006")
			case 'X':
				result.WriteString("15:04:05")
			case 'x':
				result.WriteString("01/02/06")
			case 'Z':
				result.WriteString("MST")
			case 'z':
				result.WriteString("-0700")
			case '%':
				result.WriteByte('%')
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

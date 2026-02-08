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

func (p *DefaultOsProvider) Clock() float64 {
	return time.Since(p.startTime).Seconds()
}

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
	if month == 0 {
		month = 1
	}
	if day == 0 {
		day = 1
	}

	t := time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
	return t.Unix(), nil
}

func (p *DefaultOsProvider) Date(format string, timestamp int64) (string, error) {
	t := time.Unix(timestamp, 0)

	if format == "" {
		format = "%c"
	}

	// Check for ! prefix (UTC)
	if strings.HasPrefix(format, "!") {
		t = t.UTC()
		format = format[1:]
	}

	goFormat := strftimeToGo(format)
	return t.Format(goFormat), nil
}

func (p *DefaultOsProvider) DateTable(timestamp int64) map[string]int {
	t := time.Unix(timestamp, 0)

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

func (p *DefaultOsProvider) Getenv(name string) (string, bool) {
	if p.envFilter != nil && !p.envFilter(name) {
		return "", false
	}
	return os.LookupEnv(name)
}

func (p *DefaultOsProvider) Capabilities() LuaOsCaps {
	return LuaOsCaps{
		AllowTime:   true,
		AllowDate:   true,
		AllowGetenv: true,
	}
}

// strftimeToGo converts a strftime format string to a Go time.Format layout.
func strftimeToGo(format string) string {
	var result strings.Builder
	i := 0
	for i < len(format) {
		if format[i] == '%' && i+1 < len(format) {
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
				// Unknown directive, keep as-is
				result.WriteByte('%')
				result.WriteByte(format[i])
			}
		} else {
			result.WriteByte(format[i])
		}
		i++
	}
	return result.String()
}

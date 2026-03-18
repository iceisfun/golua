package vm

import (
	"testing"
	"time"
)

func TestIsDSTHint(t *testing.T) {
	// Override time.Local to America/New_York for deterministic DST testing
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("cannot load America/New_York timezone: %v", err)
	}
	origLocal := time.Local
	time.Local = loc
	defer func() { time.Local = origLocal }()

	p := NewDefaultOsProvider()

	// July 1, 2000 in New York: EDT (UTC-4) is DST, EST (UTC-5) is standard.
	// With isdst=true (DST/EDT), mktime interprets as UTC-4 -> unix = wallclock + 4h
	// With isdst=false (standard/EST), mktime interprets as UTC-5 -> unix = wallclock + 5h
	// So t2 (isdst=false) should be 3600 seconds greater than t1 (isdst=true).
	t1, _, err := p.Time(&LuaTimeInput{
		Year: 2000, Month: 7, Day: 1, Hour: 12, Min: 0, Sec: 0,
		HasIsDST: true, IsDST: true,
	})
	if err != nil {
		t.Fatalf("Time with isdst=true failed: %v", err)
	}

	t2, _, err := p.Time(&LuaTimeInput{
		Year: 2000, Month: 7, Day: 1, Hour: 12, Min: 0, Sec: 0,
		HasIsDST: true, IsDST: false,
	})
	if err != nil {
		t.Fatalf("Time with isdst=false failed: %v", err)
	}

	diff := t2 - t1
	if diff != 3600 {
		t.Errorf("expected t2-t1=3600, got %d (t1=%d, t2=%d)", diff, t1, t2)
	}

	// Without isdst hint, July is DST in New York -> should match isdst=true
	t3, _, err := p.Time(&LuaTimeInput{
		Year: 2000, Month: 7, Day: 1, Hour: 12, Min: 0, Sec: 0,
	})
	if err != nil {
		t.Fatalf("Time without isdst failed: %v", err)
	}
	if t3 != t1 {
		t.Errorf("auto-detected DST should match isdst=true for July, t3=%d t1=%d", t3, t1)
	}
}

func TestSetLocaleTracking(t *testing.T) {
	p := NewDefaultOsProvider()

	// Initial state should be "C"
	cur, ok := p.SetLocale("\x00query", "all")
	if !ok || cur != "C" {
		t.Errorf("initial locale should be 'C', got %q (ok=%v)", cur, ok)
	}

	// Set to "C" explicitly
	r, ok := p.SetLocale("C", "all")
	if !ok || r != "C" {
		t.Errorf("setlocale('C') should return 'C', got %q", r)
	}

	// Query again
	cur, ok = p.SetLocale("\x00query", "all")
	if !ok || cur != "C" {
		t.Errorf("locale after setlocale('C') should be 'C', got %q", cur)
	}

	// Unsupported locale returns false
	_, ok = p.SetLocale("fr_FR", "all")
	if ok {
		t.Errorf("unsupported locale should return false")
	}

	// After failed set, locale should remain "C"
	cur, ok = p.SetLocale("\x00query", "all")
	if !ok || cur != "C" {
		t.Errorf("locale should remain 'C' after failed set, got %q", cur)
	}
}

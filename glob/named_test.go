package glob

import (
	"reflect"
	"testing"
)

func TestMatchNamed(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		text          string
		expectedMatch bool
		expectedCaps  map[string]string
		expectErr     bool
	}{
		{
			name:          "basic named globs",
			pattern:       "*/abc/:patterna/:patternb",
			text:          "/service/example_service/abc/1/2",
			expectedMatch: true,
			expectedCaps:  map[string]string{"patterna": "1", "patternb": "2"},
		},
		{
			name:          "named glob with literal prefix",
			pattern:       "*/abc/user:userid",
			text:          "/service/example_service/abc/user123",
			expectedMatch: true,
			expectedCaps:  map[string]string{"userid": "123"},
		},
		{
			name:          "non-matching pattern",
			pattern:       "*/abc/:patterna/:patternb",
			text:          "/service/example_service/abcfff/1/2",
			expectedMatch: false,
			expectedCaps:  map[string]string{},
		},
		{
			name:          "named glob at end",
			pattern:       "*/foo:fooid",
			text:          "/service/example_service/foo18nezf",
			expectedMatch: true,
			expectedCaps:  map[string]string{"fooid": "18nezf"},
		},
		{
			name:          "literal match with no globs",
			pattern:       "abc",
			text:          "abc",
			expectedMatch: true,
			expectedCaps:  map[string]string{},
		},
		{
			name:          "literal mismatch with no globs",
			pattern:       "abc",
			text:          "ab",
			expectedMatch: false,
			expectedCaps:  map[string]string{},
		},
		{
			name:          "empty pattern and text",
			pattern:       "",
			text:          "",
			expectedMatch: true,
			expectedCaps:  map[string]string{},
		},
		{
			name:          "empty pattern non-empty text",
			pattern:       "",
			text:          "nonempty",
			expectedMatch: false,
			expectedCaps:  map[string]string{},
		},
		{
			name:      "malformed pattern - incomplete character class",
			pattern:   "abc[",
			text:      "abc",
			expectErr: true,
		},
		{
			name:      "malformed pattern - trailing escape",
			pattern:   "abc\\",
			text:      "abc",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched, caps, err := MatchNamed(tc.pattern, tc.text)
			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error for pattern %q, got none", tc.pattern)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error for pattern %q: %v", tc.pattern, err)
				return
			}
			if matched != tc.expectedMatch {
				t.Errorf("Pattern %q with text %q: expected match %v, got %v", tc.pattern, tc.text, tc.expectedMatch, matched)
			}
			if !reflect.DeepEqual(caps, tc.expectedCaps) {
				t.Errorf("Pattern %q with text %q: expected captures %#v, got %#v", tc.pattern, tc.text, tc.expectedCaps, caps)
			}
		})
	}
}

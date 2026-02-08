package glob

import (
	"fmt"
	"testing"
)

// TestCase represents a single test case for the Match function
type TestCase struct {
	Pattern  string
	Input    string
	Expected bool
}

// TestGroup represents a group of related test cases
type TestGroup struct {
	Name      string
	TestCases []TestCase
}

// AllTestGroups contains all the test groups
var AllTestGroups = []TestGroup{
	{
		Name: "Basic Matching",
		TestCases: []TestCase{
			{"hello", "hello", true},
			{"hello", "world", false},
			{"", "", true},
			{"*", "anything", true},
		},
	},
	{
		Name: "Wildcard *",
		TestCases: []TestCase{
			{"*", "", true},
			{"*", "anything", true},
			{"he*o", "hello", true},
			{"he*o", "hey ho", true},
			{"he*o", "hello world", false},
			{"*lo", "hello", true},
			{"hel*", "hello", true},
		},
	},
	{
		Name: "Single Character Wildcard ?",
		TestCases: []TestCase{
			{"h?llo", "hello", true},
			{"h?llo", "hallo", true},
			{"h?llo", "hllo", false},
			{"h??lo", "hello", true},
			{"h??lo", "helo", false},
		},
	},
	{
		Name: "Character Classes []",
		TestCases: []TestCase{
			{"h[ea]llo", "hello", true},
			{"h[ea]llo", "hallo", true},
			{"h[ea]llo", "hillo", false},
			{"h[^ea]llo", "hillo", true},
			{"h[^ea]llo", "hello", false},
			{"h[a-z]llo", "hello", true},
			{"h[a-z]llo", "h9llo", false},
		},
	},
	{
		Name: "Combination Patterns",
		TestCases: []TestCase{
			{"h*l?o", "hello", true},
			{"h*l?o", "hllo", true},
			{"h*l?o", "helo", false},
			{"h[ea]*l?o", "hello", true},
			{"h[ea]*l?o", "hallo world", false},
		},
	},
	{
		Name: "Escape Characters",
		TestCases: []TestCase{
			{"h\\*llo", "h*llo", true},
			{"h\\*llo", "hello", false},
			{"h\\?llo", "h?llo", true},
			{"h\\?llo", "hallo", false},
			{"h\\[a-z\\]llo", "h[a-z]llo", true},
		},
	},
	{
		Name: "Edge Cases",
		TestCases: []TestCase{
			{"", "", true},
			{"*", "", true},
			{"\\", "", false},
			{"[", "", false},
			{"]", "", false},
		},
	},
	{
		Name: "User Provided Examples",
		TestCases: []TestCase{
			{"40/42*", "40/42", true},
			{"W* PEACH*", "WHITE PEACHES", true},
			{"*BAG*", "S BAGS", true},
			{"*TRAY*", "S BAGS", false},
			{"ORG* PEACH", "ORGANIC WHITE YELLOW GREEN PEACH", true},
		},
	},
}

func TestMatch(t *testing.T) {
	for _, group := range AllTestGroups {
		t.Run(group.Name, func(t *testing.T) {
			for _, tc := range group.TestCases {
				t.Run(fmt.Sprintf("%s_%s", tc.Pattern, tc.Input), func(t *testing.T) {
					result, err := Match(tc.Pattern, tc.Input)
					if err != nil {
						t.Fatalf("Error in Match(%q, %q): %v", tc.Pattern, tc.Input, err)
					}
					if result != tc.Expected {
						t.Errorf("Match(%q, %q) = %v, expected %v", tc.Pattern, tc.Input, result, tc.Expected)
					}
				})
			}
		})
	}
}

var WordsTestGroups = []TestGroup{
	{
		Name: "Globbing each word",
		TestCases: []TestCase{
			// Basic matching
			{"ORG* PEACH", "ORGANIC PEACH", true},
			{"ORG* PEACH", "ORGANIC WHITE PEACH", false},
			{"ORG* PEACH", "ORGANIC YELLOW PEACH", false},
			{"ORG* WHITE PEACH", "ORGANIC WHITE PEACH", true},
			{"ORG* * PEACH", "ORGANIC WHITE PEACH", true},

			// Identifier matching
			{"GLAB*", "Glabrezu", true},
			{"GLAB*", "GLABREZU", true},
			{"GLAB*", "Voice", false},
			{"GLAB*", "Nalfeshnee", false},

			// Different number of words
			{"ORG* PEACH", "ORGANIC", false},
			{"ORG* PEACH", "ORGANIC WHITE YELLOW GREEN PEACH", false},

			// Case sensitivity, we expect case-insensitive matching
			{"org* peach", "ORGANIC PEACH", true},
			{"ORG* PEACH", "organic peach", true},

			// Multiple patterns in a word
			{"OR*NIC PE*CH", "ORGANIC PEACH", true},
			{"OR*NIC PE*CH", "ORGANIC PUNCH", false},

			// Patterns at word boundaries
			{"*GANIC *EACH", "ORGANIC PEACH", true},
			{"ORGAN* P*", "ORGANIC PEACH", true},

			// Special characters
			{"OR?ANIC PEACH", "ORGANIC PEACH", true},
			{"OR?ANIC PEACH", "ORGXNIC PEACH", false},
			{"ORG?NIC PEACH", "ORGXNIC PEACH", true},
			{"OR?ANIC PEACH", "ORANIC PEACH", false},

			// Multiple asterisks
			{"*GA*C *EA*", "ORGANIC PEACH", true},
			{"*GA*C *EAC*", "ORGANIC PEAR", false},

			// Empty strings and spaces
			{"", "", true},
			{" ", " ", true},
			{"A B", "A  B", true}, // Multiple spaces should be treated as one

			// Complex patterns
			{"[OA]R*NIC [PB]EA*", "ORGANIC PEACH", true},
			{"[OA]R*NIC [PB]EA*", "ARGANIC BEACH", true},
			{"[OA]R*NIC [PB][EA][EA]*", "ORGANIC BEECH", true},
			{"[OA]R*NIC [PB][EA][EA]*", "ORGANIC APPLE", false},

			// Escaped characters
			{"OR\\*ANIC PEACH", "OR*ANIC PEACH", true},
			{"OR\\*ANIC PEACH", "ORGANIC PEACH", false},
		},
	},
}

func TestWordsMatch(t *testing.T) {
	for _, group := range WordsTestGroups {
		t.Run(group.Name, func(t *testing.T) {
			for _, tc := range group.TestCases {
				t.Run(fmt.Sprintf("%s_%s", tc.Pattern, tc.Input), func(t *testing.T) {
					result, err := MatchWords(tc.Pattern, tc.Input)
					if err != nil {
						t.Fatalf("Error in MatchWords(%q, %q): %v", tc.Pattern, tc.Input, err)
					}
					if result != tc.Expected {
						t.Errorf("MatchWords(%q, %q) = %v, expected %v", tc.Pattern, tc.Input, result, tc.Expected)
					}
				})
			}
		})
	}
}

func TestMatchErrors(t *testing.T) {
	errorCases := []struct {
		Pattern string
		Input   string
	}{
		{"[", "a"},
		{"[]", "a"},
		{"[a-", "a"},
		{"[\\", "a"},
		{"[z-a]", "b"},
	}

	for _, tc := range errorCases {
		t.Run(fmt.Sprintf("%s_%s", tc.Pattern, tc.Input), func(t *testing.T) {
			_, err := Match(tc.Pattern, tc.Input)
			if err == nil {
				t.Errorf("Match(%q, %q) did not return an error, expected ErrBadPattern", tc.Pattern, tc.Input)
			} else if err != ErrBadPattern {
				t.Errorf("Match(%q, %q) returned error %v, expected ErrBadPattern", tc.Pattern, tc.Input, err)
			}
		})
	}
}

func TestHasPatternCharacters(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", false},
		{"", false},
		{"h*llo", true},
		{"h?llo", true},
		{"h[e]llo", true},
		{"h]llo", true},
		{"h\\ello", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := HasPatternCharacters(tc.input)
			if result != tc.expected {
				t.Errorf("HasPatternCharacters(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

// Benchmark for Match function
func BenchmarkMatch(b *testing.B) {
	for _, group := range AllTestGroups {
		for _, tc := range group.TestCases {
			b.Run(fmt.Sprintf("%s_%s", tc.Pattern, tc.Input), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, _ = Match(tc.Pattern, tc.Input)
				}
			})
		}
	}
}

// BenchmarkMatchWords benchmarks word-based matching
func BenchmarkMatchWords(b *testing.B) {
	for _, group := range WordsTestGroups {
		for _, tc := range group.TestCases {
			b.Run(fmt.Sprintf("%s_%s", tc.Pattern, tc.Input), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, _ = MatchWords(tc.Pattern, tc.Input)
				}
			})
		}
	}
}

// BenchmarkWorstCaseWildcard benchmarks worst-case wildcard patterns
func BenchmarkWorstCaseWildcard(b *testing.B) {
	// Long string with a pattern that forces maximum backtracking
	long := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
	b.Run("star_no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = Match("a*a*a*a*b", long)
		}
	})
	b.Run("star_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = Match("a*b", long)
		}
	})
	b.Run("question_match", func(b *testing.B) {
		short := "abcdefghij"
		for i := 0; i < b.N; i++ {
			_, _ = Match("??????????", short)
		}
	})
}

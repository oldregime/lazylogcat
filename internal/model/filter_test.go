package model

import (
	"testing"
)

func TestLevelIndex(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  int
	}{
		{name: "Verbose", level: "V", want: 0},
		{name: "Debug", level: "D", want: 1},
		{name: "Info", level: "I", want: 2},
		{name: "Warn", level: "W", want: 3},
		{name: "Error", level: "E", want: 4},
		{name: "Fatal", level: "F", want: 5},
		{name: "Unknown", level: "X", want: -1},
		{name: "Empty", level: "", want: -1},
		{name: "Lowercase", level: "v", want: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevelIndex(tt.level)
			if got != tt.want {
				t.Errorf("LevelIndex(%q) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestTextFilterMode_Next(t *testing.T) {
	tests := []struct {
		name string
		mode TextFilterMode
		want TextFilterMode
	}{
		{name: "ContainsToExact", mode: FilterModeContains, want: FilterModeExact},
		{name: "ExactToRegex", mode: FilterModeExact, want: FilterModeRegex},
		{name: "RegexToContains", mode: FilterModeRegex, want: FilterModeContains},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.Next()
			if got != tt.want {
				t.Errorf("TextFilterMode(%d).Next() = %d, want %d", tt.mode, got, tt.want)
			}
		})
	}
}

func TestTextFilterMode_String(t *testing.T) {
	tests := []struct {
		name string
		mode TextFilterMode
		want string
	}{
		{name: "Contains", mode: FilterModeContains, want: "contains"},
		{name: "Exact", mode: FilterModeExact, want: "exact"},
		{name: "Regex", mode: FilterModeRegex, want: "regex"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("TextFilterMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestTextFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter TextFilter
		want   bool
	}{
		{name: "Empty", filter: TextFilter{}, want: true},
		{name: "EmptyValue", filter: TextFilter{Value: "", Mode: FilterModeExact}, want: true},
		{name: "NonEmpty", filter: TextFilter{Value: "hello"}, want: false},
		{name: "NonEmptyWithMode", filter: TextFilter{Value: "hello", Mode: FilterModeRegex}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.IsEmpty()
			if got != tt.want {
				t.Errorf("TextFilter{%q, %d}.IsEmpty() = %v, want %v", tt.filter.Value, tt.filter.Mode, got, tt.want)
			}
		})
	}
}

func TestTextFilter_Match(t *testing.T) {
	tests := []struct {
		name     string
		filter   TextFilter
		haystack string
		want     bool
	}{
		// Empty filter matches everything
		{name: "EmptyFilterMatchesAnything", filter: TextFilter{}, haystack: "anything", want: true},
		{name: "EmptyFilterMatchesEmpty", filter: TextFilter{}, haystack: "", want: true},

		// Contains mode
		{name: "ContainsMatch", filter: TextFilter{Value: "error", Mode: FilterModeContains}, haystack: "fatal error occurred", want: true},
		{name: "ContainsCaseInsensitive", filter: TextFilter{Value: "ERROR", Mode: FilterModeContains}, haystack: "fatal error occurred", want: true},
		{name: "ContainsNoMatch", filter: TextFilter{Value: "warning", Mode: FilterModeContains}, haystack: "fatal error occurred", want: false},
		{name: "ContainsEmptyHaystack", filter: TextFilter{Value: "error", Mode: FilterModeContains}, haystack: "", want: false},

		// Exact mode
		{name: "ExactMatch", filter: TextFilter{Value: "com.example.app", Mode: FilterModeExact}, haystack: "com.example.app", want: true},
		{name: "ExactCaseInsensitive", filter: TextFilter{Value: "com.example.APP", Mode: FilterModeExact}, haystack: "com.example.app", want: true},
		{name: "ExactNoMatchSubstring", filter: TextFilter{Value: "example", Mode: FilterModeExact}, haystack: "com.example.app", want: false},
		{name: "ExactNoMatchSuperset", filter: TextFilter{Value: "com.example.app.extra", Mode: FilterModeExact}, haystack: "com.example.app", want: false},

		// Regex mode
		{name: "RegexMatch", filter: TextFilter{Value: "err.*occurred", Mode: FilterModeRegex}, haystack: "fatal error occurred", want: true},
		{name: "RegexCaseInsensitive", filter: TextFilter{Value: "ERR.*OCCURRED", Mode: FilterModeRegex}, haystack: "fatal error occurred", want: true},
		{name: "RegexNoMatch", filter: TextFilter{Value: "^error$", Mode: FilterModeRegex}, haystack: "fatal error occurred", want: false},
		{name: "RegexAnchoredMatch", filter: TextFilter{Value: "^fatal", Mode: FilterModeRegex}, haystack: "fatal error occurred", want: true},
		{name: "RegexInvalidPattern", filter: TextFilter{Value: "[invalid", Mode: FilterModeRegex}, haystack: "anything", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Match(tt.haystack)
			if got != tt.want {
				t.Errorf("TextFilter{%q, %s}.Match(%q) = %v, want %v",
					tt.filter.Value, tt.filter.Mode, tt.haystack, got, tt.want)
			}
		})
	}
}

func TestTextFilter_Compile(t *testing.T) {
	t.Run("RegexValid", func(t *testing.T) {
		tf := TextFilter{Value: "error.*line", Mode: FilterModeRegex}
		tf.Compile()
		if tf.compiled == nil {
			t.Fatal("expected compiled regexp, got nil")
		}
		if tf.compileErr != nil {
			t.Fatalf("expected no compile error, got %v", tf.compileErr)
		}
		// Verify the compiled regex works through Match
		if !tf.Match("an error on this line") {
			t.Error("expected match after Compile()")
		}
		if tf.Match("no match here") {
			t.Error("expected no match after Compile()")
		}
	})

	t.Run("RegexInvalid", func(t *testing.T) {
		tf := TextFilter{Value: "[bad", Mode: FilterModeRegex}
		tf.Compile()
		if tf.compileErr == nil {
			t.Fatal("expected compile error for invalid regex")
		}
		// Match should return false for all inputs when regex is invalid
		if tf.Match("anything") {
			t.Error("expected false for invalid compiled regex")
		}
	})

	t.Run("ContainsModeNoOp", func(t *testing.T) {
		tf := TextFilter{Value: "hello", Mode: FilterModeContains}
		tf.Compile()
		if tf.compiled != nil {
			t.Error("expected nil compiled for contains mode")
		}
		// Match should still work normally
		if !tf.Match("say hello world") {
			t.Error("expected contains match to work after Compile()")
		}
	})

	t.Run("ExactModeNoOp", func(t *testing.T) {
		tf := TextFilter{Value: "hello", Mode: FilterModeExact}
		tf.Compile()
		if tf.compiled != nil {
			t.Error("expected nil compiled for exact mode")
		}
		if !tf.Match("hello") {
			t.Error("expected exact match to work after Compile()")
		}
	})
}

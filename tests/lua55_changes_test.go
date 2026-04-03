package tests

import (
	"strings"
	"testing"
)

// TestErrorNilReplacedByString verifies Lua 5.5 behavior where error(nil)
// replaces the nil with the string "<no error object>".
func TestErrorNilReplacedByString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		output string
	}{
		{
			name:   "error_nil",
			code:   `local ok, msg = pcall(error, nil); print(type(msg), msg)`,
			output: "string\t<no error object>",
		},
		{
			name:   "error_no_args",
			code:   `local ok, msg = pcall(error); print(type(msg), msg)`,
			output: "string\t<no error object>",
		},
		{
			name:   "error_nil_level_2",
			code:   `local ok, msg = pcall(error, nil, 2); print(type(msg), msg)`,
			output: "string\t<no error object>",
		},
		{
			name:   "error_nil_level_0",
			code:   `local ok, msg = pcall(error, nil, 0); print(type(msg), msg)`,
			output: "string\t<no error object>",
		},
		{
			name:   "error_false_unchanged",
			code:   `local ok, msg = pcall(error, false); print(type(msg), msg)`,
			output: "boolean\tfalse",
		},
		{
			name:   "error_string_unchanged",
			code:   `local ok, msg = pcall(error, "oops"); print(ok)`,
			output: "false",
		},
		{
			name:   "xpcall_handler_sees_nil",
			code: `local ok, msg = xpcall(function() error(nil) end, function(e) return type(e) end)
print(ok, msg)`,
			output: "false\tnil",
		},
		{
			name:   "xpcall_handler_returns_nil_replaced",
			code: `local ok, msg = xpcall(function() error(nil) end, function(e) return nil end)
print(ok, type(msg), msg)`,
			output: "false\tstring\t<no error object>",
		},
		{
			name:   "xpcall_handler_returns_value_kept",
			code: `local ok, msg = xpcall(function() error(nil) end, function(e) return "custom" end)
print(ok, msg)`,
			output: "false\tcustom",
		},
		{
			name:   "assert_nil_nil",
			code:   `local ok, msg = pcall(assert, nil, nil); print(type(msg), msg)`,
			output: "string\t<no error object>",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, lines, err := runLua(t, tt.code)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			got := strings.Join(lines, "\n")
			if got != tt.output {
				t.Errorf("got %q, want %q", got, tt.output)
			}
		})
	}
}

// TestCollectgarbageParam verifies Lua 5.5 "param" option for collectgarbage.
func TestCollectgarbageParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		output string
	}{
		{
			name:   "param_pause",
			code:   `print(collectgarbage("param", "pause"))`,
			output: "250",
		},
		{
			name:   "param_minormul",
			code:   `print(collectgarbage("param", "minormul"))`,
			output: "20",
		},
		{
			name:   "param_majorminor",
			code:   `print(collectgarbage("param", "majorminor"))`,
			output: "50",
		},
		{
			name:   "param_minormajor",
			code:   `print(collectgarbage("param", "minormajor"))`,
			output: "68",
		},
		{
			name:   "param_stepmul",
			code:   `print(collectgarbage("param", "stepmul"))`,
			output: "200",
		},
		{
			name:   "param_stepsize",
			code:   `print(collectgarbage("param", "stepsize"))`,
			output: "9600",
		},
		{
			name:   "param_set_returns_old",
			code:   `print(collectgarbage("param", "pause", 100))`,
			output: "250",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, lines, err := runLua(t, tt.code)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			got := strings.Join(lines, "\n")
			if got != tt.output {
				t.Errorf("got %q, want %q", got, tt.output)
			}
		})
	}
}

// TestCollectgarbageRemovedOptions verifies setpause and setstepmul are removed in 5.5.
func TestCollectgarbageRemovedOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    string
		wantErr string
	}{
		{
			name:    "setpause_removed",
			code:    `collectgarbage("setpause", 200)`,
			wantErr: "invalid option 'setpause'",
		},
		{
			name:    "setstepmul_removed",
			code:    `collectgarbage("setstepmul", 200)`,
			wantErr: "invalid option 'setstepmul'",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := runLua(t, tt.code)
			if err == nil {
				t.Fatalf("Expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestCollectgarbageParamErrors verifies error handling in the "param" option.
func TestCollectgarbageParamErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    string
		wantErr string
	}{
		{
			name:    "invalid_param_name",
			code:    `collectgarbage("param", "invalid")`,
			wantErr: "invalid option 'invalid'",
		},
		{
			name:    "missing_param_name",
			code:    `collectgarbage("param")`,
			wantErr: "string expected, got no value",
		},
		{
			name:    "nil_param_name",
			code:    `collectgarbage("param", nil)`,
			wantErr: "string expected, got nil",
		},
		{
			name:    "non_string_value_arg",
			code:    `collectgarbage("param", "pause", "abc")`,
			wantErr: "number expected, got string",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := runLua(t, tt.code)
			if err == nil {
				t.Fatalf("Expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

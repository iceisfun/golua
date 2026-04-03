package tests

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
)

func TestForReadonly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		code      string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "numeric for assign to loop var",
			code:      "for i = 1, 10 do i = 5 end",
			wantErr:   true,
			errSubstr: "attempt to assign to const variable 'i'",
		},
		{
			name:      "generic for assign to first var",
			code:      "for k, v in pairs({}) do k = 1 end",
			wantErr:   true,
			errSubstr: "attempt to assign to const variable 'k'",
		},
		{
			name:    "generic for assign to second var is OK",
			code:    "for k, v in pairs({}) do v = 1 end",
			wantErr: false,
		},
		{
			name:    "generic for assign to third var is OK",
			code:    "for a, b, c in pairs({}) do c = 1 end",
			wantErr: false,
		},
		{
			name:      "generic for assign to first of three vars",
			code:      "for a, b, c in pairs({}) do a = 1 end",
			wantErr:   true,
			errSubstr: "attempt to assign to const variable 'a'",
		},
		{
			name:    "numeric for read is OK",
			code:    "for i = 1, 10 do local x = i end",
			wantErr: false,
		},
		{
			name:    "numeric for shadow is OK",
			code:    "for i = 1, 10 do local i = i + 1 end",
			wantErr: false,
		},
		{
			name:    "generic for read first var is OK",
			code:    "for k, v in pairs({}) do local x = k end",
			wantErr: false,
		},
		{
			name:      "numeric for op-assign to loop var",
			code:      "for i = 1, 10 do i = i + 1 end",
			wantErr:   true,
			errSubstr: "attempt to assign to const variable 'i'",
		},
		{
			name:    "generic for single var assign",
			code:    "for k in pairs({}) do k = 1 end",
			wantErr: true,
			errSubstr: "attempt to assign to const variable 'k'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			block, err := parser.Parse("test", tt.code)
			if err != nil {
				if tt.wantErr {
					// Parser error is acceptable if it matches
					if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
						t.Errorf("parse error %q does not contain %q", err, tt.errSubstr)
					}
					return
				}
				t.Fatalf("unexpected parse error: %v", err)
			}

			_, err = compiler.Compile("test", block)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected compile error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("compile error %q does not contain %q", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected compile error: %v", err)
				}
			}
		})
	}
}

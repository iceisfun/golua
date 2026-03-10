package stdlib

import "testing"

func TestFormatAltGeneralFloat(t *testing.T) {
	tests := []struct {
		spec string
		conv byte
		val  float64
		want string
	}{
		{"%#.2g", 'g', 99.99995, "1.e+02"},
		{"%#.6g", 'g', 999999.5, "1.e+06"},
		{"%#010.6g", 'g', 999999.5, "00001.e+06"},
		{"%#+012.6g", 'g', 999999.5, "+000001.e+06"},
		{"%#.6G", 'G', 999999.5, "1.E+06"},
		{"%#.5g", 'g', 9.999995e-05, "0.00010000"},
		{"%#.6g", 'g', 123.0, "123.000"},
	}

	for _, tc := range tests {
		if got := formatAltGeneralFloat(tc.spec, tc.conv, tc.val); got != tc.want {
			t.Fatalf("formatAltGeneralFloat(%q, %q, %v) = %q, want %q", tc.spec, tc.conv, tc.val, got, tc.want)
		}
	}
}

package compiler

import "testing"

func TestMetamethodTagOrdinals(t *testing.T) {
	tests := []struct {
		tag  MetamethodTag
		want int
	}{
		{TM_ADD, 0},
		{TM_SUB, 1},
		{TM_MUL, 2},
		{TM_MOD, 3},
		{TM_POW, 4},
		{TM_DIV, 5},
		{TM_IDIV, 6},
		{TM_BAND, 7},
		{TM_BOR, 8},
		{TM_BXOR, 9},
		{TM_SHL, 10},
		{TM_SHR, 11},
		{TM_UNM, 12},
		{TM_BNOT, 13},
		{TM_LT, 14},
		{TM_LE, 15},
		{TM_CONCAT, 16},
		{TM_LEN, 17},
		{TM_EQ, 18},
	}
	for _, tt := range tests {
		if int(tt.tag) != tt.want {
			t.Errorf("%s: got ordinal %d, want %d", tt.tag, int(tt.tag), tt.want)
		}
	}
}

func TestMetamethodTagString(t *testing.T) {
	tests := []struct {
		tag  MetamethodTag
		want string
	}{
		{TM_ADD, "__add"},
		{TM_SUB, "__sub"},
		{TM_MUL, "__mul"},
		{TM_MOD, "__mod"},
		{TM_POW, "__pow"},
		{TM_DIV, "__div"},
		{TM_IDIV, "__idiv"},
		{TM_BAND, "__band"},
		{TM_BOR, "__bor"},
		{TM_BXOR, "__bxor"},
		{TM_SHL, "__shl"},
		{TM_SHR, "__shr"},
		{TM_UNM, "__unm"},
		{TM_BNOT, "__bnot"},
		{TM_LT, "__lt"},
		{TM_LE, "__le"},
		{TM_CONCAT, "__concat"},
		{TM_LEN, "__len"},
		{TM_EQ, "__eq"},
	}
	for _, tt := range tests {
		if got := tt.tag.String(); got != tt.want {
			t.Errorf("MetamethodTag(%d).String() = %q, want %q", int(tt.tag), got, tt.want)
		}
	}
}

func TestMetamethodTagOutOfRange(t *testing.T) {
	if got := MetamethodTag(-1).String(); got != "???" {
		t.Errorf("MetamethodTag(-1).String() = %q, want %q", got, "???")
	}
	if got := MetamethodTag(99).String(); got != "???" {
		t.Errorf("MetamethodTag(99).String() = %q, want %q", got, "???")
	}
}

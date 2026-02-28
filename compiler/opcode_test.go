package compiler

import (
	"fmt"
	"testing"
)

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
	if got := MetamethodTag(-1).String(); got != "MetamethodTag(-1)" {
		t.Errorf("MetamethodTag(-1).String() = %q, want %q", got, "MetamethodTag(-1)")
	}
	if got := MetamethodTag(99).String(); got != "MetamethodTag(99)" {
		t.Errorf("MetamethodTag(99).String() = %q, want %q", got, "MetamethodTag(99)")
	}
}

func TestOpCodeString(t *testing.T) {
	// All valid opcodes should have non-empty names.
	for op := OpCode(0); op < NumOps; op++ {
		name := op.String()
		if name == "" {
			t.Errorf("OpCode(%d).String() returned empty string", op)
		}
		if name == "???" {
			t.Errorf("OpCode(%d).String() returned %q, expected a registered name", op, name)
		}
	}
}

func TestOpCodeStringOutOfRange(t *testing.T) {
	if got := OpCode(255).String(); got != "OpCode(255)" {
		t.Errorf("OpCode(255).String() = %q, want %q", got, "OpCode(255)")
	}
}

func TestOpModeString(t *testing.T) {
	tests := []struct {
		mode OpMode
		want string
	}{
		{IABC, "iABC"},
		{IvABC, "ivABC"},
		{IABx, "iABx"},
		{IAsBx, "iAsBx"},
		{IAx, "iAx"},
		{IsJ, "isJ"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("OpMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestOpModeStringOutOfRange(t *testing.T) {
	if got := OpMode(99).String(); got != "OpMode(99)" {
		t.Errorf("OpMode(99).String() = %q, want %q", got, "OpMode(99)")
	}
}

func TestGetOpModeAllOpcodes(t *testing.T) {
	for op := OpCode(0); op < NumOps; op++ {
		mode := GetOpMode(op)
		if mode > IsJ {
			t.Errorf("GetOpMode(%s) returned invalid mode %d", op, mode)
		}
	}
}

func TestGetOpModeInvalidPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("GetOpMode(255) did not panic")
		}
		msg := fmt.Sprint(r)
		if msg != "compiler bug: GetOpMode called with invalid opcode 255" {
			t.Errorf("unexpected panic message: %s", msg)
		}
	}()
	GetOpMode(255)
}

func TestOpCodeValid(t *testing.T) {
	for op := OpCode(0); op < NumOps; op++ {
		if !op.Valid() {
			t.Errorf("OpCode(%d).Valid() = false, want true", op)
		}
	}
	if NumOps.Valid() {
		t.Errorf("NumOps.Valid() = true, want false")
	}
	if OpCode(255).Valid() {
		t.Errorf("OpCode(255).Valid() = true, want false")
	}
}

func TestInitCompletenessGuard(t *testing.T) {
	for op := OpCode(0); op < NumOps; op++ {
		if OpName(op) == "" {
			t.Errorf("opcode %d has no name registered", op)
		}
	}
}

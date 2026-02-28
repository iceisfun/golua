package compiler

import "testing"

// TestABCEncodeDecode verifies round-trip symmetry for iABC instructions.
func TestABCEncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		op      OpCode
		a, b, c int
		k       int
	}{
		{"zeros", OP_MOVE, 0, 0, 0, 0},
		{"max fields", OP_MOVE, MaxArgA, MaxArgB, MaxArgC, 1},
		{"typical", OP_ADD, 3, 1, 2, 0},
		{"k flag set", OP_EQ, 10, 20, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := ABC(tt.op, tt.a, tt.b, tt.c, tt.k)
			if got := inst.OpCode(); got != tt.op {
				t.Errorf("OpCode() = %v, want %v", got, tt.op)
			}
			if got := inst.A(); got != tt.a {
				t.Errorf("A() = %d, want %d", got, tt.a)
			}
			if got := inst.B(); got != tt.b {
				t.Errorf("B() = %d, want %d", got, tt.b)
			}
			if got := inst.C(); got != tt.c {
				t.Errorf("C() = %d, want %d", got, tt.c)
			}
			if got := inst.K(); got != tt.k {
				t.Errorf("K() = %d, want %d", got, tt.k)
			}
		})
	}
}

// TestABxEncodeDecode verifies round-trip symmetry for iABx instructions.
func TestABxEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		a    int
		bx   int
	}{
		{"zeros", 0, 0},
		{"max", MaxArgA, MaxArgBx},
		{"typical", 5, 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := ABx(OP_LOADK, tt.a, tt.bx)
			if got := inst.A(); got != tt.a {
				t.Errorf("A() = %d, want %d", got, tt.a)
			}
			if got := inst.Bx(); got != tt.bx {
				t.Errorf("Bx() = %d, want %d", got, tt.bx)
			}
		})
	}
}

// TestAsBxEncodeDecode verifies round-trip symmetry for signed sBx.
func TestAsBxEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		a    int
		sbx  int
	}{
		{"zero", 0, 0},
		{"positive max", 0, OffsetSBx},
		{"negative max", 0, -OffsetSBx},
		{"positive", 3, 100},
		{"negative", 3, -100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := AsBx(OP_LOADI, tt.a, tt.sbx)
			if got := inst.A(); got != tt.a {
				t.Errorf("A() = %d, want %d", got, tt.a)
			}
			if got := inst.SBx(); got != tt.sbx {
				t.Errorf("SBx() = %d, want %d", got, tt.sbx)
			}
		})
	}
}

// TestAxEncodeDecode verifies round-trip symmetry for iAx instructions.
func TestAxEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		ax   int
	}{
		{"zero", 0},
		{"max", MaxArgAx},
		{"typical", 12345},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := Ax(OP_EXTRAARG, tt.ax)
			if got := inst.Ax(); got != tt.ax {
				t.Errorf("Ax() = %d, want %d", got, tt.ax)
			}
		})
	}
}

// TestSJEncodeDecode verifies round-trip symmetry for isJ instructions.
func TestSJEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		sj   int
	}{
		{"zero", 0},
		{"positive max", OffsetSJ},
		{"negative max", -OffsetSJ},
		{"positive", 42},
		{"negative", -42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := SJ(OP_JMP, tt.sj, 0)
			if got := inst.SJ(); got != tt.sj {
				t.Errorf("SJ() = %d, want %d", got, tt.sj)
			}
		})
	}
}

// TestSCEncodeDecode verifies round-trip symmetry for the signed C field.
func TestSCEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		sc   int
	}{
		{"zero", 0},
		{"positive max", OffsetSC},
		{"negative max", -OffsetSC},
		{"positive", 10},
		{"negative", -10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SC is encoded as C = sc + OffsetSC
			raw := tt.sc + OffsetSC
			inst := ABC(OP_ADDI, 0, 0, raw, 0)
			if got := inst.SC(); got != tt.sc {
				t.Errorf("SC() = %d, want %d", got, tt.sc)
			}
		})
	}
}

// TestNoRegEqualsMaxArgA ensures the sentinel relationship holds.
func TestNoRegEqualsMaxArgA(t *testing.T) {
	if NoReg != MaxArgA {
		t.Errorf("NoReg = %d, want MaxArgA = %d", NoReg, MaxArgA)
	}
	if NoReg != 255 {
		t.Errorf("NoReg = %d, want 255", NoReg)
	}
}

// TestSetAPreservesOtherFields verifies SetA only modifies the A field.
func TestSetAPreservesOtherFields(t *testing.T) {
	inst := ABC(OP_ADD, 10, 20, 30, 1)
	inst = inst.SetA(99)
	if got := inst.A(); got != 99 {
		t.Errorf("A() = %d, want 99", got)
	}
	if got := inst.OpCode(); got != OP_ADD {
		t.Errorf("OpCode() = %v, want OP_ADD", got)
	}
	if got := inst.B(); got != 20 {
		t.Errorf("B() = %d, want 20", got)
	}
	if got := inst.C(); got != 30 {
		t.Errorf("C() = %d, want 30", got)
	}
	if got := inst.K(); got != 1 {
		t.Errorf("K() = %d, want 1", got)
	}
}

// TestSetSBxPreservesOpAndA verifies SetSBx only modifies the Bx field.
func TestSetSBxPreservesOpAndA(t *testing.T) {
	inst := AsBx(OP_LOADI, 42, 100)
	inst = inst.SetSBx(-50)
	if got := inst.SBx(); got != -50 {
		t.Errorf("SBx() = %d, want -50", got)
	}
	if got := inst.A(); got != 42 {
		t.Errorf("A() = %d, want 42", got)
	}
	if got := inst.OpCode(); got != OP_LOADI {
		t.Errorf("OpCode() = %v, want OP_LOADI", got)
	}
}

// TestSetSJPreservesOp verifies SetSJ only modifies the sJ field.
func TestSetSJPreservesOp(t *testing.T) {
	inst := SJ(OP_JMP, 100, 0)
	inst = inst.SetSJ(-200)
	if got := inst.SJ(); got != -200 {
		t.Errorf("SJ() = %d, want -200", got)
	}
	if got := inst.OpCode(); got != OP_JMP {
		t.Errorf("OpCode() = %v, want OP_JMP", got)
	}
}

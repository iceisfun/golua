package compiler

// Instruction is a 32-bit Lua VM instruction.
//
// Bit layout (from LSB to MSB):
//
//	iABC:   Op(7) | A(8) | k(1) | B(8) | C(8)
//	iABx:   Op(7) | A(8) | Bx(17)
//	iAsBx:  Op(7) | A(8) | sBx(17)         where sBx = Bx - OffsetSBx
//	iAx:    Op(7) | Ax(25)
//	isJ:    Op(7) | sJ(25)                 where sJ = J - OffsetSJ
//
// Bit positions:
//
//	 31       24 23       16 15  14        7 6        0
//	+----------+-----------+---+-----------+----------+
//	| C (8)    | B (8)     | k | A (8)     | Op (7)   |   iABC
//	+----------+-----------+---+-----------+----------+
//	|        Bx (17)           | A (8)     | Op (7)   |   iABx / iAsBx
//	+--------------------------+-----------+----------+
//	|             Ax (25)                  | Op (7)   |   iAx
//	+--------------------------------------+----------+
//	|             sJ (25)                  | Op (7)   |   isJ
//	+--------------------------------------+----------+
//
// The opcode occupies the low 7 bits. Field A occupies bits 7-14.
// Remaining bits are format-dependent.
type Instruction uint32

// Field sizes.
const (
	SizeOP = 7
	SizeA  = 8
	SizeK  = 1
	SizeB  = 8
	SizeC  = 8
	SizeBx = SizeK + SizeB + SizeC // 17
	SizeAx = SizeA + SizeBx        // 25
	SizeSJ = SizeAx                // 25

	// IvABC format (NEWTABLE, SETLIST) reuses the B/C bit region as two
	// variable-width fields: a 6-bit vB and a 10-bit vC.
	SizeVB = 6
	SizeVC = 10
)

// Field positions.
const (
	PosOP = 0
	PosA  = PosOP + SizeOP // 7
	PosK  = PosA + SizeA   // 15
	PosB  = PosK + SizeK   // 16
	PosC  = PosB + SizeB   // 24
	PosBx = PosK           // 15
	PosAx = PosA           // 7
	PosSJ = PosA           // 7
	PosVB = PosB           // 16 — IvABC vB starts where B does
	PosVC = PosVB + SizeVB // 22 — IvABC vC follows the 6-bit vB
)

// Limits.
const (
	MaxArgA  = (1 << SizeA) - 1  // 255
	MaxArgB  = (1 << SizeB) - 1  // 255
	MaxArgC  = (1 << SizeC) - 1  // 255
	MaxArgBx = (1 << SizeBx) - 1 // 131071
	MaxArgAx = (1 << SizeAx) - 1
	MaxArgSJ = (1 << SizeSJ) - 1
	MaxArgVB = (1 << SizeVB) - 1 // 63
	MaxArgVC = (1 << SizeVC) - 1 // 1023

	OffsetSBx = MaxArgBx >> 1 // 65535
	OffsetSC  = MaxArgC >> 1  // 127
	OffsetSJ  = MaxArgSJ >> 1

	// MaxSJ is the largest positive sJ offset that fits in the 25-bit
	// signed jump field (isJ format). MinSJ is the most negative.
	MaxSJ = MaxArgSJ - OffsetSJ // +16777216  = 1<<24
	MinSJ = -OffsetSJ           // -16777215  = -((1<<24)-1)

	// NoReg is the invalid-register sentinel. It equals MaxArgA (255),
	// the largest value that fits in the 8-bit A field, and is used to
	// signal "no register" in instruction operands.
	NoReg = MaxArgA // 255
)

// mask1 returns a bitmask with n 1-bits starting at position p.
func mask1(n, p uint) uint32 {
	return ((1 << n) - 1) << p
}

// ---------------------------------------------------------------------------
// Getters
// ---------------------------------------------------------------------------

// OpCode extracts the 7-bit opcode from the instruction.
func (i Instruction) OpCode() OpCode {
	return OpCode(uint32(i) & mask1(SizeOP, PosOP))
}

// A extracts the 8-bit A field (register or operand).
func (i Instruction) A() int {
	return int((uint32(i) >> PosA) & mask1(SizeA, 0))
}

// B extracts the 8-bit B field.
func (i Instruction) B() int {
	return int((uint32(i) >> PosB) & mask1(SizeB, 0))
}

// C extracts the 8-bit C field.
func (i Instruction) C() int {
	return int((uint32(i) >> PosC) & mask1(SizeC, 0))
}

// K extracts the 1-bit k flag (used for comparison polarity and RK mode).
func (i Instruction) K() int {
	return int((uint32(i) >> PosK) & 1)
}

// VB extracts the 6-bit vB field of an IvABC instruction (NEWTABLE, SETLIST).
func (i Instruction) VB() int {
	return int((uint32(i) >> PosVB) & mask1(SizeVB, 0))
}

// VC extracts the 10-bit vC field of an IvABC instruction (NEWTABLE, SETLIST).
func (i Instruction) VC() int {
	return int((uint32(i) >> PosVC) & mask1(SizeVC, 0))
}

// Bx extracts the unsigned 17-bit Bx field (iABx format).
func (i Instruction) Bx() int {
	return int((uint32(i) >> PosBx) & mask1(SizeBx, 0))
}

// SBx extracts the signed 17-bit sBx field (iAsBx format), offset-decoded.
func (i Instruction) SBx() int {
	return i.Bx() - OffsetSBx
}

// Ax extracts the 25-bit Ax field (iAx format, used by OP_EXTRAARG).
func (i Instruction) Ax() int {
	return int((uint32(i) >> PosAx) & mask1(SizeAx, 0))
}

// SJ extracts the signed 25-bit jump offset (isJ format, used by OP_JMP).
func (i Instruction) SJ() int {
	return int((uint32(i)>>PosSJ)&mask1(SizeSJ, 0)) - OffsetSJ
}

// SC returns the signed C field (C - OffsetSC).
func (i Instruction) SC() int {
	return i.C() - OffsetSC
}

// SB returns the signed B field (B - OffsetSC), used by EQI/LTI/etc.
func (i Instruction) SB() int {
	return i.B() - OffsetSC
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// ABC creates an iABC instruction.
func ABC(op OpCode, a, b, c, k int) Instruction {
	return Instruction(uint32(op)<<PosOP |
		uint32(a)<<PosA |
		uint32(b)<<PosB |
		uint32(c)<<PosC |
		uint32(k)<<PosK)
}

// ABx creates an iABx instruction.
func ABx(op OpCode, a, bx int) Instruction {
	return Instruction(uint32(op)<<PosOP |
		uint32(a)<<PosA |
		uint32(bx)<<PosBx)
}

// AsBx creates an iAsBx instruction.
func AsBx(op OpCode, a, sbx int) Instruction {
	return ABx(op, a, sbx+OffsetSBx)
}

// Ax creates an iAx instruction.
func Ax(op OpCode, ax int) Instruction {
	return Instruction(uint32(op)<<PosOP |
		uint32(ax)<<PosAx)
}

// SJ creates an isJ instruction.
func SJ(op OpCode, sj, k int) Instruction {
	return Instruction(uint32(op)<<PosOP |
		uint32(sj+OffsetSJ)<<PosSJ |
		uint32(k)<<PosK)
}

// SetA returns a copy of the instruction with field A replaced.
func (i Instruction) SetA(a int) Instruction {
	u := uint32(i)
	u &^= mask1(SizeA, PosA)
	u |= uint32(a) << PosA
	return Instruction(u)
}

// SetSBx returns a copy with sBx replaced.
func (i Instruction) SetSBx(sbx int) Instruction {
	u := uint32(i)
	u &^= mask1(SizeBx, PosBx)
	u |= uint32(sbx+OffsetSBx) << PosBx
	return Instruction(u)
}

// SetSJ returns a copy with sJ replaced.
func (i Instruction) SetSJ(sj int) Instruction {
	u := uint32(i)
	u &^= mask1(SizeSJ, PosSJ)
	u |= uint32(sj+OffsetSJ) << PosSJ
	return Instruction(u)
}

// SetBx returns a copy with Bx replaced.
func (i Instruction) SetBx(bx int) Instruction {
	u := uint32(i)
	u &^= mask1(SizeBx, PosBx)
	u |= uint32(bx) << PosBx
	return Instruction(u)
}

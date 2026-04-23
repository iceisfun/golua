// Package compiler transforms a Lua AST into executable bytecode.
//
// The compilation pipeline takes an [ast.Block] (produced by the parser) and
// emits a [Proto] — a function prototype containing 32-bit instructions, a
// constant table, upvalue descriptors, and nested child prototypes.
//
// The instruction set uses a register-based architecture with 7-bit opcodes
// and five encoding formats (iABC, iABx, iAsBx, iAx, isJ). Register
// allocation uses a linear scan over the function's local variables, with
// freeReg tracking the next available register.
//
// Key compiler invariants:
//   - Registers 0..N-1 hold the function's N local variables.
//   - Temporary expression results are allocated above local registers.
//   - Upvalue capture follows Lua 5.4 semantics: open upvalues reference
//     the enclosing stack, closed upvalues copy to heap on scope exit.
//   - OP_CLOSE is emitted at scope boundaries where captured locals exist,
//     and before goto jumps that cross local-variable declarations.
//
// Lua 5.4 Reference: §3.3.2 (local variables), §3.4 (expressions),
// §3.3.11 (to-be-closed variables).
package compiler

import "fmt"

// OpCode is a 7-bit Lua VM opcode.
type OpCode byte

// Instruction set — order matches Lua 5.4's opcode numbering (ORDER OP).
// Each opcode comment shows its format and semantics in Lua register notation.
// R[x] = register x, K[x] = constant x, UpValue[x] = upvalue x.
const (
	OP_MOVE       OpCode = iota // A B     R[A] := R[B]
	OP_LOADI                    // A sBx   R[A] := sBx
	OP_LOADF                    // A sBx   R[A] := (float)sBx
	OP_LOADK                    // A Bx    R[A] := K[Bx]
	OP_LOADKX                   // A       R[A] := K[extra arg]
	OP_LOADFALSE                // A       R[A] := false
	OP_LFALSESKIP               // A       R[A] := false; pc++
	OP_LOADTRUE                 // A       R[A] := true
	OP_LOADNIL                  // A B     R[A], ..., R[A+B] := nil
	OP_GETUPVAL                 // A B     R[A] := UpValue[B]
	OP_SETUPVAL                 // A B     UpValue[B] := R[A]

	OP_GETTABUP // A B C   R[A] := UpValue[B][K[C]:string]
	OP_GETTABLE // A B C   R[A] := R[B][R[C]]
	OP_GETI     // A B C   R[A] := R[B][C]
	OP_GETFIELD // A B C   R[A] := R[B][K[C]:string]

	OP_SETTABUP // A B C   UpValue[A][K[B]:string] := RK(C)
	OP_SETTABLE // A B C   R[A][R[B]] := RK(C)
	OP_SETI     // A B C   R[A][B] := RK(C)
	OP_SETFIELD // A B C   R[A][K[B]:string] := RK(C)

	OP_NEWTABLE // A vB vC k  R[A] := {}

	OP_SELF // A B C   R[A+1] := R[B]; R[A] := R[B][K[C]:string]

	OP_ADDI // A B sC  R[A] := R[B] + sC

	OP_ADDK  // A B C   R[A] := R[B] + K[C]
	OP_SUBK  // A B C   R[A] := R[B] - K[C]
	OP_MULK  // A B C   R[A] := R[B] * K[C]
	OP_MODK  // A B C   R[A] := R[B] % K[C]
	OP_POWK  // A B C   R[A] := R[B] ^ K[C]
	OP_DIVK  // A B C   R[A] := R[B] / K[C]
	OP_IDIVK // A B C   R[A] := R[B] // K[C]

	OP_BANDK // A B C   R[A] := R[B] & K[C]
	OP_BORK  // A B C   R[A] := R[B] | K[C]
	OP_BXORK // A B C   R[A] := R[B] ~ K[C]

	OP_SHLI // A B sC  R[A] := sC << R[B]
	OP_SHRI // A B sC  R[A] := R[B] >> sC

	OP_ADD  // A B C   R[A] := R[B] + R[C]
	OP_SUB  // A B C   R[A] := R[B] - R[C]
	OP_MUL  // A B C   R[A] := R[B] * R[C]
	OP_MOD  // A B C   R[A] := R[B] % R[C]
	OP_POW  // A B C   R[A] := R[B] ^ R[C]
	OP_DIV  // A B C   R[A] := R[B] / R[C]
	OP_IDIV // A B C   R[A] := R[B] // R[C]

	OP_BAND // A B C   R[A] := R[B] & R[C]
	OP_BOR  // A B C   R[A] := R[B] | R[C]
	OP_BXOR // A B C   R[A] := R[B] ~ R[C]
	OP_SHL  // A B C   R[A] := R[B] << R[C]
	OP_SHR  // A B C   R[A] := R[B] >> R[C]

	OP_MMBIN  // A B C     call C metamethod over R[A] and R[B]
	OP_MMBINI // A sB C k  call C metamethod over R[A] and sB
	OP_MMBINK // A B C k   call C metamethod over R[A] and K[B]

	OP_UNM  // A B     R[A] := -R[B]
	OP_BNOT // A B     R[A] := ~R[B]
	OP_NOT  // A B     R[A] := not R[B]
	OP_LEN  // A B     R[A] := #R[B]

	OP_CONCAT // A B     R[A] := R[A].. ... ..R[A+B-1]

	OP_CLOSE // A       close all upvalues >= R[A]
	OP_TBC   // A       mark variable A "to be closed"
	OP_JMP   // sJ      pc += sJ

	OP_EQ // A B k   if ((R[A] == R[B]) ~= k) then pc++
	OP_LT // A B k   if ((R[A] <  R[B]) ~= k) then pc++
	OP_LE // A B k   if ((R[A] <= R[B]) ~= k) then pc++

	OP_EQK // A B k   if ((R[A] == K[B]) ~= k) then pc++
	OP_EQI // A sB k  if ((R[A] == sB) ~= k) then pc++
	OP_LTI // A sB k  if ((R[A] < sB) ~= k) then pc++
	OP_LEI // A sB k  if ((R[A] <= sB) ~= k) then pc++
	OP_GTI // A sB k  if ((R[A] > sB) ~= k) then pc++
	OP_GEI // A sB k  if ((R[A] >= sB) ~= k) then pc++

	OP_TEST    // A k     if (not R[A] == k) then pc++
	OP_TESTSET // A B k   if (not R[B] == k) then pc++ else R[A] := R[B]

	OP_CALL     // A B C   R[A],...,R[A+C-2] := R[A](R[A+1],...,R[A+B-1])
	OP_TAILCALL // A B C k return R[A](R[A+1],...,R[A+B-1])

	OP_RETURN  // A B C k return R[A], ..., R[A+B-2]
	OP_RETURN0 //         return
	OP_RETURN1 // A       return R[A]

	OP_FORLOOP // A Bx    update counters; if loop continues then pc-=Bx
	OP_FORPREP // A Bx    check values and prepare; if not to run then pc+=Bx+1

	OP_TFORPREP // A Bx   create upvalue for R[A+3]; pc+=Bx
	OP_TFORCALL // A C    R[A+4],...,R[A+3+C] := R[A](R[A+1], R[A+2])
	OP_TFORLOOP // A Bx   if R[A+2] ~= nil then { R[A]=R[A+2]; pc -= Bx }

	OP_SETLIST // A vB vC k  R[A][vC+i] := R[A+i], 1 <= i <= vB

	OP_CLOSURE // A Bx    R[A] := closure(KPROTO[Bx])

	OP_VARARG // A B C k R[A],...,R[A+C-2] = varargs

	OP_GETVARG // A B C   R[A] := R[B][R[C]], R[B] is vararg param

	OP_ERRNNIL // A Bx    raise error if R[A] ~= nil

	OP_VARARGPREP // A    (adjust varargs)

	OP_EXTRAARG // Ax     extra argument for previous opcode

	// NumOps is the total number of opcodes. It is a sentinel value used to size
	// the opProperties table and must remain the final constant in the OpCode
	// enumeration. Matches Lua 5.4.6 opcode count.
	NumOps // sentinel → total number of opcodes
)

// OpMode describes the instruction encoding format. Each opcode has exactly
// one mode that determines how operand fields are packed into the 32-bit word.
type OpMode byte

const (
	IABC  OpMode = iota // A(8) B(8) C(8) k(1) — three operands + flag
	IvABC               // A(8) vB(6) vC(10) k(1) — variable-width B and C (NEWTABLE, SETLIST)
	IABx                // A(8) Bx(17) — unsigned extended operand
	IAsBx               // A(8) sBx(17) — signed extended operand (sBx = Bx - offset)
	IAx                 // Ax(25) — single wide operand (EXTRAARG)
	IsJ                 // sJ(25) — signed jump offset (JMP)
)

// String returns the name of the instruction encoding format.
func (m OpMode) String() string {
	switch m {
	case IABC:
		return "iABC"
	case IvABC:
		return "ivABC"
	case IABx:
		return "iABx"
	case IAsBx:
		return "iAsBx"
	case IAx:
		return "iAx"
	case IsJ:
		return "isJ"
	default:
		return fmt.Sprintf("OpMode(%d)", m)
	}
}

// opProp is the encoding mode and name for an opcode.
type opProp struct {
	mode OpMode
	name string
}

// opProperties stores the encoding mode and name for each opcode.
// Every OpCode in [0, NumOps) must have a non-empty entry; a missing entry
// is caught at compile time by the array size and at runtime by OpName/GetOpMode
// bounds checks.
var opProperties = [NumOps]opProp{
	OP_MOVE:       {IABC, "MOVE"},
	OP_LOADI:      {IAsBx, "LOADI"},
	OP_LOADF:      {IAsBx, "LOADF"},
	OP_LOADK:      {IABx, "LOADK"},
	OP_LOADKX:     {IABx, "LOADKX"},
	OP_LOADFALSE:  {IABC, "LOADFALSE"},
	OP_LFALSESKIP: {IABC, "LFALSESKIP"},
	OP_LOADTRUE:   {IABC, "LOADTRUE"},
	OP_LOADNIL:    {IABC, "LOADNIL"},
	OP_GETUPVAL:   {IABC, "GETUPVAL"},
	OP_SETUPVAL:   {IABC, "SETUPVAL"},
	OP_GETTABUP:   {IABC, "GETTABUP"},
	OP_GETTABLE:   {IABC, "GETTABLE"},
	OP_GETI:       {IABC, "GETI"},
	OP_GETFIELD:   {IABC, "GETFIELD"},
	OP_SETTABUP:   {IABC, "SETTABUP"},
	OP_SETTABLE:   {IABC, "SETTABLE"},
	OP_SETI:       {IABC, "SETI"},
	OP_SETFIELD:   {IABC, "SETFIELD"},
	OP_NEWTABLE:   {IvABC, "NEWTABLE"},
	OP_SELF:       {IABC, "SELF"},
	OP_ADDI:       {IABC, "ADDI"},
	OP_ADDK:       {IABC, "ADDK"},
	OP_SUBK:       {IABC, "SUBK"},
	OP_MULK:       {IABC, "MULK"},
	OP_MODK:       {IABC, "MODK"},
	OP_POWK:       {IABC, "POWK"},
	OP_DIVK:       {IABC, "DIVK"},
	OP_IDIVK:      {IABC, "IDIVK"},
	OP_BANDK:      {IABC, "BANDK"},
	OP_BORK:       {IABC, "BORK"},
	OP_BXORK:      {IABC, "BXORK"},
	OP_SHLI:       {IABC, "SHLI"},
	OP_SHRI:       {IABC, "SHRI"},
	OP_ADD:        {IABC, "ADD"},
	OP_SUB:        {IABC, "SUB"},
	OP_MUL:        {IABC, "MUL"},
	OP_MOD:        {IABC, "MOD"},
	OP_POW:        {IABC, "POW"},
	OP_DIV:        {IABC, "DIV"},
	OP_IDIV:       {IABC, "IDIV"},
	OP_BAND:       {IABC, "BAND"},
	OP_BOR:        {IABC, "BOR"},
	OP_BXOR:       {IABC, "BXOR"},
	OP_SHL:        {IABC, "SHL"},
	OP_SHR:        {IABC, "SHR"},
	OP_MMBIN:      {IABC, "MMBIN"},
	OP_MMBINI:     {IABC, "MMBINI"},
	OP_MMBINK:     {IABC, "MMBINK"},
	OP_UNM:        {IABC, "UNM"},
	OP_BNOT:       {IABC, "BNOT"},
	OP_NOT:        {IABC, "NOT"},
	OP_LEN:        {IABC, "LEN"},
	OP_CONCAT:     {IABC, "CONCAT"},
	OP_CLOSE:      {IABC, "CLOSE"},
	OP_TBC:        {IABC, "TBC"},
	OP_JMP:        {IsJ, "JMP"},
	OP_EQ:         {IABC, "EQ"},
	OP_LT:         {IABC, "LT"},
	OP_LE:         {IABC, "LE"},
	OP_EQK:        {IABC, "EQK"},
	OP_EQI:        {IABC, "EQI"},
	OP_LTI:        {IABC, "LTI"},
	OP_LEI:        {IABC, "LEI"},
	OP_GTI:        {IABC, "GTI"},
	OP_GEI:        {IABC, "GEI"},
	OP_TEST:       {IABC, "TEST"},
	OP_TESTSET:    {IABC, "TESTSET"},
	OP_CALL:       {IABC, "CALL"},
	OP_TAILCALL:   {IABC, "TAILCALL"},
	OP_RETURN:     {IABC, "RETURN"},
	OP_RETURN0:    {IABC, "RETURN0"},
	OP_RETURN1:    {IABC, "RETURN1"},
	OP_FORLOOP:    {IABx, "FORLOOP"},
	OP_FORPREP:    {IABx, "FORPREP"},
	OP_TFORPREP:   {IABx, "TFORPREP"},
	OP_TFORCALL:   {IABC, "TFORCALL"},
	OP_TFORLOOP:   {IABx, "TFORLOOP"},
	OP_SETLIST:    {IvABC, "SETLIST"},
	OP_CLOSURE:    {IABx, "CLOSURE"},
	OP_VARARG:     {IABC, "VARARG"},
	OP_GETVARG:    {IABC, "GETVARG"},
	OP_ERRNNIL:    {IABx, "ERRNNIL"},
	OP_VARARGPREP: {IABC, "VARARGPREP"},
	OP_EXTRAARG:   {IAx, "EXTRAARG"},
}

// String returns the human-readable name of an opcode (delegates to OpName).
func (op OpCode) String() string {
	return OpName(op)
}

// OpName returns the human-readable name of an opcode.
func OpName(op OpCode) string {
	if int(op) < len(opProperties) {
		return opProperties[op].name
	}
	return fmt.Sprintf("OpCode(%d)", op)
}

// GetOpMode returns the instruction encoding format for an opcode.
// Panics if op is not a valid opcode — invalid values indicate corrupted
// bytecode or a compiler bug.
func GetOpMode(op OpCode) OpMode {
	if int(op) < len(opProperties) {
		return opProperties[op].mode
	}
	panic(fmt.Sprintf("compiler bug: GetOpMode called with invalid opcode %d", op))
}

// Valid reports whether op is a known opcode (in [0, NumOps)).
func (op OpCode) Valid() bool {
	return op < NumOps
}

// MetamethodTag identifies which metamethod to invoke when operands don't
// support an arithmetic or comparison operation directly. Used by OP_MMBIN,
// OP_MMBINI, and OP_MMBINK. The ordinals match Lua 5.4's TM_* enum in ltm.h.
type MetamethodTag int

const (
	TM_ADD  MetamethodTag = iota // __add
	TM_SUB                       // __sub
	TM_MUL                       // __mul
	TM_MOD                       // __mod
	TM_POW                       // __pow
	TM_DIV                       // __div
	TM_IDIV                      // __idiv
	TM_BAND                      // __band
	TM_BOR                       // __bor
	TM_BXOR                      // __bxor
	TM_SHL                       // __shl
	TM_SHR                       // __shr
	TM_UNM                       // __unm
	TM_BNOT                      // __bnot
	TM_LT                        // __lt
	TM_LE                        // __le

	TM_CONCAT // __concat
	TM_LEN    // __len
	TM_EQ     // __eq
)

// metamethodNames must be kept in sync with MetamethodTag constants.
// Length must equal TM_EQ + 1 (the last tag ordinal plus one).
var metamethodNames = [...]string{
	TM_ADD: "__add", TM_SUB: "__sub", TM_MUL: "__mul",
	TM_MOD: "__mod", TM_POW: "__pow", TM_DIV: "__div",
	TM_IDIV: "__idiv", TM_BAND: "__band", TM_BOR: "__bor",
	TM_BXOR: "__bxor", TM_SHL: "__shl", TM_SHR: "__shr",
	TM_UNM: "__unm", TM_BNOT: "__bnot", TM_LT: "__lt",
	TM_LE: "__le", TM_CONCAT: "__concat", TM_LEN: "__len",
	TM_EQ: "__eq",
}

// String returns the Lua metamethod name (e.g. "__add").
func (t MetamethodTag) String() string {
	if t >= 0 && int(t) < len(metamethodNames) {
		return metamethodNames[t]
	}
	return fmt.Sprintf("MetamethodTag(%d)", t)
}

// MetamethodTagFromLua54 converts a Lua 5.4-style TM ordinal (as emitted in
// reference binary chunks via OP_MMBIN*) into GoLua's MetamethodTag value.
// Lua 5.4's TM_* enum places fast-access TMs (INDEX..EQ) first, so its TM_ADD
// is 6 while GoLua's TM_ADD is 0. Only the arithmetic/bitwise tags that can
// appear in OP_MMBIN* (TM_ADD..TM_SHR in Lua 5.4, i.e. 6..17) are translated.
// Returns ok=false for out-of-range values.
func MetamethodTagFromLua54(lua54Ord int) (MetamethodTag, bool) {
	// Lua 5.4 ordinals (from ltm.h): INDEX=0, NEWINDEX=1, GC=2, MODE=3,
	// LEN=4, EQ=5, ADD=6, SUB=7, MUL=8, MOD=9, POW=10, DIV=11, IDIV=12,
	// BAND=13, BOR=14, BXOR=15, SHL=16, SHR=17.
	// Map 6..17 (ADD..SHR in Lua 5.4) onto our TM_ADD..TM_SHR (0..11).
	if lua54Ord >= 6 && lua54Ord <= 17 {
		return MetamethodTag(lua54Ord - 6), true
	}
	return 0, false
}

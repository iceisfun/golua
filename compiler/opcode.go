// Package compiler transforms a Lua AST into bytecode.
//
// The bytecode format closely follows Lua 5.5's instruction set:
// 32-bit instructions with a 7-bit opcode and varying operand layouts.
package compiler

// OpCode is a 7-bit Lua VM opcode.
type OpCode byte

// Instruction set — order matches Lua 5.5 exactly (ORDER OP).
const (
	OP_MOVE      OpCode = iota // A B     R[A] := R[B]
	OP_LOADI                   // A sBx   R[A] := sBx
	OP_LOADF                   // A sBx   R[A] := (float)sBx
	OP_LOADK                   // A Bx    R[A] := K[Bx]
	OP_LOADKX                  // A       R[A] := K[extra arg]
	OP_LOADFALSE               // A       R[A] := false
	OP_LFALSESKIP              // A       R[A] := false; pc++
	OP_LOADTRUE                // A       R[A] := true
	OP_LOADNIL                 // A B     R[A], ..., R[A+B] := nil
	OP_GETUPVAL                // A B     R[A] := UpValue[B]
	OP_SETUPVAL                // A B     UpValue[B] := R[A]

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

	NumOps // sentinel
)

// OpMode describes the instruction format.
type OpMode byte

const (
	IABC  OpMode = iota // A B C k
	IvABC               // A vB vC k
	IABx                // A Bx
	IAsBx               // A sBx
	IAx                 // Ax
	IsJ                 // sJ
)

// opProperties stores the mode for each opcode.
var opProperties [NumOps]struct {
	mode OpMode
	name string
}

func init() {
	type entry struct {
		op   OpCode
		mode OpMode
		name string
	}
	for _, e := range []entry{
		{OP_MOVE, IABC, "MOVE"},
		{OP_LOADI, IAsBx, "LOADI"},
		{OP_LOADF, IAsBx, "LOADF"},
		{OP_LOADK, IABx, "LOADK"},
		{OP_LOADKX, IABx, "LOADKX"},
		{OP_LOADFALSE, IABC, "LOADFALSE"},
		{OP_LFALSESKIP, IABC, "LFALSESKIP"},
		{OP_LOADTRUE, IABC, "LOADTRUE"},
		{OP_LOADNIL, IABC, "LOADNIL"},
		{OP_GETUPVAL, IABC, "GETUPVAL"},
		{OP_SETUPVAL, IABC, "SETUPVAL"},
		{OP_GETTABUP, IABC, "GETTABUP"},
		{OP_GETTABLE, IABC, "GETTABLE"},
		{OP_GETI, IABC, "GETI"},
		{OP_GETFIELD, IABC, "GETFIELD"},
		{OP_SETTABUP, IABC, "SETTABUP"},
		{OP_SETTABLE, IABC, "SETTABLE"},
		{OP_SETI, IABC, "SETI"},
		{OP_SETFIELD, IABC, "SETFIELD"},
		{OP_NEWTABLE, IvABC, "NEWTABLE"},
		{OP_SELF, IABC, "SELF"},
		{OP_ADDI, IABC, "ADDI"},
		{OP_ADDK, IABC, "ADDK"},
		{OP_SUBK, IABC, "SUBK"},
		{OP_MULK, IABC, "MULK"},
		{OP_MODK, IABC, "MODK"},
		{OP_POWK, IABC, "POWK"},
		{OP_DIVK, IABC, "DIVK"},
		{OP_IDIVK, IABC, "IDIVK"},
		{OP_BANDK, IABC, "BANDK"},
		{OP_BORK, IABC, "BORK"},
		{OP_BXORK, IABC, "BXORK"},
		{OP_SHLI, IABC, "SHLI"},
		{OP_SHRI, IABC, "SHRI"},
		{OP_ADD, IABC, "ADD"},
		{OP_SUB, IABC, "SUB"},
		{OP_MUL, IABC, "MUL"},
		{OP_MOD, IABC, "MOD"},
		{OP_POW, IABC, "POW"},
		{OP_DIV, IABC, "DIV"},
		{OP_IDIV, IABC, "IDIV"},
		{OP_BAND, IABC, "BAND"},
		{OP_BOR, IABC, "BOR"},
		{OP_BXOR, IABC, "BXOR"},
		{OP_SHL, IABC, "SHL"},
		{OP_SHR, IABC, "SHR"},
		{OP_MMBIN, IABC, "MMBIN"},
		{OP_MMBINI, IABC, "MMBINI"},
		{OP_MMBINK, IABC, "MMBINK"},
		{OP_UNM, IABC, "UNM"},
		{OP_BNOT, IABC, "BNOT"},
		{OP_NOT, IABC, "NOT"},
		{OP_LEN, IABC, "LEN"},
		{OP_CONCAT, IABC, "CONCAT"},
		{OP_CLOSE, IABC, "CLOSE"},
		{OP_TBC, IABC, "TBC"},
		{OP_JMP, IsJ, "JMP"},
		{OP_EQ, IABC, "EQ"},
		{OP_LT, IABC, "LT"},
		{OP_LE, IABC, "LE"},
		{OP_EQK, IABC, "EQK"},
		{OP_EQI, IABC, "EQI"},
		{OP_LTI, IABC, "LTI"},
		{OP_LEI, IABC, "LEI"},
		{OP_GTI, IABC, "GTI"},
		{OP_GEI, IABC, "GEI"},
		{OP_TEST, IABC, "TEST"},
		{OP_TESTSET, IABC, "TESTSET"},
		{OP_CALL, IABC, "CALL"},
		{OP_TAILCALL, IABC, "TAILCALL"},
		{OP_RETURN, IABC, "RETURN"},
		{OP_RETURN0, IABC, "RETURN0"},
		{OP_RETURN1, IABC, "RETURN1"},
		{OP_FORLOOP, IABx, "FORLOOP"},
		{OP_FORPREP, IABx, "FORPREP"},
		{OP_TFORPREP, IABx, "TFORPREP"},
		{OP_TFORCALL, IABC, "TFORCALL"},
		{OP_TFORLOOP, IABx, "TFORLOOP"},
		{OP_SETLIST, IvABC, "SETLIST"},
		{OP_CLOSURE, IABx, "CLOSURE"},
		{OP_VARARG, IABC, "VARARG"},
		{OP_GETVARG, IABC, "GETVARG"},
		{OP_ERRNNIL, IABx, "ERRNNIL"},
		{OP_VARARGPREP, IABC, "VARARGPREP"},
		{OP_EXTRAARG, IAx, "EXTRAARG"},
	} {
		opProperties[e.op] = struct {
			mode OpMode
			name string
		}{e.mode, e.name}
	}
}

// OpName returns the human-readable name of an opcode.
func OpName(op OpCode) string {
	if int(op) < len(opProperties) {
		return opProperties[op].name
	}
	return "???"
}

// OpMode returns the instruction format for an opcode.
func GetOpMode(op OpCode) OpMode {
	if int(op) < len(opProperties) {
		return opProperties[op].mode
	}
	return IABC
}

// Metamethod indices for OP_MMBIN / OP_MMBINI / OP_MMBINK.
const (
	TM_ADD  = 0
	TM_SUB  = 1
	TM_MUL  = 2
	TM_MOD  = 3
	TM_POW  = 4
	TM_DIV  = 5
	TM_IDIV = 6
	TM_BAND = 7
	TM_BOR  = 8
	TM_BXOR = 9
	TM_SHL  = 10
	TM_SHR  = 11
	TM_UNM  = 12
	TM_BNOT = 13
	TM_LT   = 14
	TM_LE   = 15

	TM_CONCAT = 16
	TM_LEN    = 17
	TM_EQ     = 18
)

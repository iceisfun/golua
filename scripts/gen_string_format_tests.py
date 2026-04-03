#!/usr/bin/env python3
import itertools
import math
import os
import shutil
import subprocess
from typing import Any
from dataclasses import dataclass


@dataclass(frozen=True)
class Arg:
    kind: str
    value: Any


@dataclass(frozen=True)
class Case:
    name: str
    family: str
    fmt: str
    args: tuple[Arg, ...]


def lua_bytes_string(s: str) -> str:
    bs = list(s.encode("utf-8", "surrogateescape"))
    if not bs:
        return '""'
    return "string.char(" + ", ".join(str(b) for b in bs) + ")"


def go_string_literal(s: str) -> str:
    out = ['"']
    for b in s.encode("utf-8", "surrogateescape"):
        if b == 0x22:
            out.append('\\"')
        elif b == 0x5C:
            out.append('\\\\')
        elif 0x20 <= b <= 0x7E:
            out.append(chr(b))
        else:
            out.append(f"\\x{b:02x}")
    out.append('"')
    return "".join(out)


def lua_arg(arg: Arg) -> str:
    if arg.kind == "int":
        return str(arg.value)
    if arg.kind == "float":
        v = float(arg.value)
        if math.isnan(v):
            return "0/0"
        if math.isinf(v):
            return "math.huge" if v > 0 else "-math.huge"
        return repr(v)
    if arg.kind == "string":
        return lua_bytes_string(str(arg.value))
    if arg.kind == "bool":
        return "true" if arg.value else "false"
    if arg.kind == "nil":
        return "nil"
    raise ValueError(arg.kind)


def go_arg(arg: Arg) -> str:
    if arg.kind == "int":
        return f'{{kind: "int", intVal: {arg.value}}}'
    if arg.kind == "float":
        v = float(arg.value)
        if math.isnan(v):
            return '{kind: "float", floatExpr: "NaN"}'
        if math.isinf(v):
            expr = "+Inf" if v > 0 else "-Inf"
            return f'{{kind: "float", floatExpr: "{expr}"}}'
        return f'{{kind: "float", floatVal: {repr(v)}}}'
    if arg.kind == "string":
        return f'{{kind: "string", strVal: {go_string_literal(str(arg.value))}}}'
    if arg.kind == "bool":
        return f'{{kind: "bool", boolVal: {str(arg.value).lower()}}}'
    if arg.kind == "nil":
        return '{kind: "nil"}'
    raise ValueError(arg.kind)


def canonical_flag_sets(chars: str):
    out = [""]
    for r in range(1, len(chars) + 1):
        for combo in itertools.combinations(chars, r):
            out.append("".join(combo))
    return out


def add_case(cases: dict, case: Case):
    key = (case.family, case.fmt, case.args)
    cases[key] = case


def build_cases() -> list[Case]:
    cases: dict[tuple, Case] = {}

    signed_vals = [Arg("int", v) for v in (-1, 0, 1, 42)] + [Arg("float", 3.5), Arg("string", "12")]
    unsigned_vals = [Arg("int", v) for v in (0, 1, 255)] + [Arg("float", 2.25)]
    hex_vals = [Arg("int", v) for v in (0, 1, 15, 255)] + [Arg("float", 15.0)]
    float_vals = [Arg("float", v) for v in (-0.0, 0.0, 1.5, math.pi, 99.99995, 999999.5, 9.999995e-05, float("inf"), float("-inf"))] + [Arg("string", "1.5")]
    string_vals = [Arg("string", v) for v in ("", "a", "hello", "abc\x00def", "h\u00e9")] + [Arg("int", 42), Arg("bool", True)]
    char_vals = [Arg("int", v) for v in (0, 65, 255, 256, -1)] + [Arg("float", 65.0)]
    quote_vals = [Arg("string", v) for v in ("", "abc", '"\\\n', "a\x00b", "1\x07")]
    quote_vals += [Arg("bool", True), Arg("bool", False), Arg("int", 0), Arg("float", math.pi), Arg("float", float("inf"))]

    small_widths = ["", "1", "5", "10"]
    int_precs = ["", ".0", ".1", ".2", ".6"]
    float_precs = ["", ".0", ".1", ".2", ".6"]
    string_precs = ["", ".0", ".1", ".2", ".5"]

    for conv in ("d", "i"):
        for flags in ["", "0", "-", "+", " ", "+0", "-0"]:
            for width in small_widths:
                for prec in int_precs:
                    for arg in signed_vals:
                        add_case(cases, Case(f"{conv}_{flags or 'plain'}_{width or 'w0'}_{prec or 'p0'}_{arg.kind}_{str(arg.value).replace('-', 'm')}", "integer", f"%{flags}{width}{prec}{conv}", (arg,)))

    for conv in ("u",):
        for flags in ["", "0", "-", "-0"]:
            for width in small_widths:
                for prec in int_precs:
                    for arg in unsigned_vals:
                        add_case(cases, Case(f"{conv}_{flags or 'plain'}_{width or 'w0'}_{prec or 'p0'}_{arg.kind}_{str(arg.value).replace('-', 'm')}", "unsigned", f"%{flags}{width}{prec}{conv}", (arg,)))

    for conv in ("o", "x", "X"):
        for flags in ["", "#", "0", "-", "#0", "-#"]:
            for width in small_widths:
                for prec in int_precs:
                    for arg in hex_vals:
                        add_case(cases, Case(f"{conv}_{flags or 'plain'}_{width or 'w0'}_{prec or 'p0'}_{arg.kind}_{str(arg.value).replace('-', 'm')}", "radix", f"%{flags}{width}{prec}{conv}", (arg,)))

    for conv in ("e", "E", "f"):
        for flags in ["", "0", "-", "+", " ", "+0", "#"]:
            for width in small_widths:
                for prec in float_precs:
                    for arg in float_vals:
                        add_case(cases, Case(f"{conv}_{flags or 'plain'}_{width or 'w0'}_{prec or 'p0'}_{arg.kind}", "float", f"%{flags}{width}{prec}{conv}", (arg,)))

    for conv in ("g", "G"):
        for flags in ["", "#", "0", "-", "+", " ", "#0", "+#", " #"]:
            for width in small_widths:
                for prec in ["", ".0", ".1", ".2", ".5", ".6"]:
                    for arg in float_vals:
                        add_case(cases, Case(f"{conv}_{flags or 'plain'}_{width or 'w0'}_{prec or 'p0'}_{arg.kind}", "general", f"%{flags}{width}{prec}{conv}", (arg,)))

    for conv in ("a", "A"):
        for flags in ["", "#", "0", "-", "+", " ", "#0", "+#"]:
            for width in small_widths:
                for prec in ["", ".0", ".1", ".2", ".6"]:
                    for arg in [Arg("float", 1.5), Arg("float", -2.25), Arg("float", float.fromhex("0x1p-1022")), Arg("float", float.fromhex("0x0.fffffffffffffp-1022")), Arg("float", float("inf"))]:
                        add_case(cases, Case(f"{conv}_{flags or 'plain'}_{width or 'w0'}_{prec or 'p0'}", "hexfloat", f"%{flags}{width}{prec}{conv}", (arg,)))

    for flags in ["", "-"]:
        for width in small_widths:
            for prec in string_precs:
                for arg in string_vals:
                    add_case(cases, Case(f"s_{flags or 'plain'}_{width or 'w0'}_{prec or 'p0'}_{arg.kind}", "string", f"%{flags}{width}{prec}s", (arg,)))

    for flags in ["", "-"]:
        for width in ["", "1", "5", "10"]:
            for arg in char_vals:
                add_case(cases, Case(f"c_{flags or 'plain'}_{width or 'w0'}_{arg.kind}", "char", f"%{flags}{width}c", (arg,)))

    for arg in quote_vals:
        add_case(cases, Case(f"q_{arg.kind}", "quote", "%q", (arg,)))

    invalid = [
        Case("invalid_trailing_percent", "invalid", "%", (Arg("int", 1),)),
        Case("invalid_conv_w", "invalid", "%w", (Arg("int", 42),)),
        Case("invalid_hash_d", "invalid", "%#d", (Arg("int", 42),)),
        Case("invalid_plus_u", "invalid", "%+u", (Arg("int", 42),)),
        Case("invalid_space_o", "invalid", "% o", (Arg("int", 42),)),
        Case("invalid_zero_s", "invalid", "%0s", (Arg("string", "x"),)),
        Case("invalid_prec_p", "invalid", "%.2p", (Arg("int", 1),)),
        Case("invalid_mod_q", "invalid", "%1q", (Arg("string", "x"),)),
        Case("invalid_F", "invalid", "%F", (Arg("float", 1.0),)),
        Case("invalid_width_100", "invalid", "%100d", (Arg("int", 1),)),
        Case("invalid_prec_100", "invalid", "%.100f", (Arg("float", 1.5),)),
        Case("invalid_missing_arg", "invalid", "%s %d", (Arg("int", 1),)),
    ]
    for case in invalid:
        add_case(cases, case)

    return sorted(cases.values(), key=lambda c: (c.family, c.name, c.fmt, len(c.args)))


def run_lua_oracle(cases: list[Case]):
    lua = shutil.which("lua5.4") or shutil.which("lua")
    if not lua:
        raise SystemExit("lua5.4 not found")

    lines = ["local cases = {}"]
    for idx, case in enumerate(cases, 1):
        args = ", ".join(lua_arg(arg) for arg in case.args)
        lines.append(f"cases[{idx}] = {{ fmt = {lua_bytes_string(case.fmt)}, args = {{ {args} }} }}")
    lines.append(
        r"""
local function enc(s)
  return (s:gsub('.', function(c) return string.format('%02x', string.byte(c)) end))
end
for i, case in ipairs(cases) do
  local out = {pcall(string.format, case.fmt, table.unpack(case.args))}
  local ok = out[1]
  if ok then
    io.write(i, "\t1\t", enc(out[2]), "\n")
  else
    io.write(i, "\t0\t", enc(tostring(out[2])), "\n")
  end
end
"""
    )
    src = "\n".join(lines)
    out = subprocess.run([lua, "-"], input=src.encode(), stdout=subprocess.PIPE, check=True).stdout.decode()
    result = {}
    for line in out.splitlines():
        idx, ok, data = line.split("\t")
        result[int(idx)] = (ok == "1", bytes.fromhex(data).decode("utf-8", "surrogateescape"))
    return result


def emit_go(cases: list[Case], oracle: dict[int, tuple[bool, str]]) -> str:
    groups: dict[str, list[str]] = {}
    for idx, case in enumerate(cases, 1):
        ok, want = oracle[idx]
        groups.setdefault(case.family, []).append(
            "\n".join([
                "\t\t{",
                f"\t\t\tname: {go_string_literal(case.name)},",
                f"\t\t\tformat: {go_string_literal(case.fmt)},",
                "\t\t\targs: []generatedFormatArg{" + ", ".join(go_arg(arg) for arg in case.args) + "},",
                f"\t\t\twantOK: {str(ok).lower()},",
                f"\t\t\twant: {go_string_literal(want)},",
                "\t\t},",
            ])
        )

    parts = [
        "// Code generated by scripts/gen_string_format_tests.py; DO NOT EDIT.",
        "package stdlib",
        "",
        "import (",
        '\t"fmt"',
        '\t"math"',
        '\t"testing"',
        "",
        '\t"github.com/iceisfun/golua/v2/vm"',
        ")",
        "",
        "type generatedFormatArg struct {",
        '\tkind string',
        '\tintVal int64',
        '\tfloatVal float64',
        '\tfloatExpr string',
        '\tstrVal string',
        '\tboolVal bool',
        "}",
        "",
        "type generatedFormatCase struct {",
        '\tname string',
        '\tformat string',
        '\targs []generatedFormatArg',
        '\twantOK bool',
        '\twant string',
        "}",
        "",
        "func buildGeneratedFormatArgs(args []generatedFormatArg) []vm.Value {",
        '\tout := make([]vm.Value, 0, len(args))',
        '\tfor _, arg := range args {',
        '\t\tswitch arg.kind {',
        '\t\tcase "int":',
        '\t\t\tout = append(out, vm.NewInt(arg.intVal))',
        '\t\tcase "float":',
        '\t\t\tif arg.floatExpr != "" {',
        '\t\t\t\tswitch arg.floatExpr {',
        '\t\t\t\tcase "NaN":',
        '\t\t\t\t\tout = append(out, vm.NewFloat(math.NaN()))',
        '\t\t\t\tcase "+Inf":',
        '\t\t\t\t\tout = append(out, vm.NewFloat(math.Inf(1)))',
        '\t\t\t\tcase "-Inf":',
        '\t\t\t\t\tout = append(out, vm.NewFloat(math.Inf(-1)))',
        '\t\t\t\tdefault:',
        '\t\t\t\t\tpanic("unknown float expr: " + arg.floatExpr)',
        '\t\t\t\t}',
        '\t\t\t} else {',
        '\t\t\t\tout = append(out, vm.NewFloat(arg.floatVal))',
        '\t\t\t}',
        '\t\tcase "string":',
        '\t\t\tout = append(out, vm.NewString(arg.strVal))',
        '\t\tcase "bool":',
        '\t\t\tout = append(out, vm.NewBool(arg.boolVal))',
        '\t\tcase "nil":',
        '\t\t\tout = append(out, vm.Nil)',
        '\t\tdefault:',
        '\t\t\tpanic("unknown generated arg kind: " + arg.kind)',
        '\t\t}',
        '\t}',
        '\treturn out',
        "}",
        "",
        "func runGeneratedFormatCase(tc generatedFormatCase) (ok bool, got string) {",
        '\tv := vm.New()',
        '\tdefer func() {',
        '\t\tif r := recover(); r != nil {',
        '\t\t\tok = false',
        '\t\t\tswitch x := r.(type) {',
        '\t\t\tcase *vm.LuaError:',
        '\t\t\t\tgot = x.Value.String()',
        '\t\t\tcase error:',
        '\t\t\t\tgot = x.Error()',
        '\t\t\tcase string:',
        '\t\t\t\tgot = x',
        '\t\t\tdefault:',
        '\t\t\t\tgot = fmt.Sprint(x)',
        '\t\t\t}',
        '\t\t}',
        '\t}()',
        '\tgot = luaFormatValues(v, tc.format, buildGeneratedFormatArgs(tc.args))',
        '\treturn true, got',
        "}",
    ]

    for family in sorted(groups):
        fn = "TestGeneratedStringFormat" + "".join(part.capitalize() for part in family.split("_"))
        parts += [
            "",
            f"func {fn}(t *testing.T) {{",
            "\ttests := []generatedFormatCase{",
            *groups[family],
            "\t}",
            "\tfor _, tc := range tests {",
            "\t\tt.Run(tc.name, func(t *testing.T) {",
            "\t\t\tok, got := runGeneratedFormatCase(tc)",
            "\t\t\tif ok != tc.wantOK || got != tc.want {",
            '\t\t\t\tt.Fatalf("format=%q ok=%v got=%q wantOK=%v want=%q", tc.format, ok, got, tc.wantOK, tc.want)',
            "\t\t\t}",
            "\t\t})",
            "\t}",
            "}",
        ]
    return "\n".join(parts) + "\n"


def main() -> None:
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    out_path = os.path.join(root, "stdlib", "string_format_generated_test.go")
    cases = build_cases()
    oracle = run_lua_oracle(cases)
    content = emit_go(cases, oracle)
    with open(out_path, "w", encoding="utf-8") as f:
        f.write(content)
    print(f"wrote {out_path} with {len(cases)} cases")


if __name__ == "__main__":
    main()

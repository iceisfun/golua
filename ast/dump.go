package ast

import (
	"fmt"
	"io"
	"strings"
)

// DumpOptions controls AST dump output.
type DumpOptions struct {
	// ShowPos annotates each node with its source position (@line:col).
	ShowPos bool
}

// Dump writes a human-readable, indented representation of the AST to w.
func Dump(w io.Writer, block *Block) {
	d := &dumper{w: w}
	d.block(block)
}

// DumpWith writes a human-readable AST representation with configurable options.
func DumpWith(w io.Writer, block *Block, opts DumpOptions) {
	d := &dumper{w: w, showPos: opts.ShowPos}
	d.block(block)
}

// DumpString returns a human-readable string representation of the AST.
func DumpString(block *Block) string {
	var buf strings.Builder
	Dump(&buf, block)
	return buf.String()
}

type dumper struct {
	w       io.Writer
	depth   int
	showPos bool
}

func (d *dumper) indent() {
	for i := 0; i < d.depth; i++ {
		fmt.Fprint(d.w, "  ")
	}
}

func (d *dumper) printf(format string, args ...any) {
	d.indent()
	fmt.Fprintf(d.w, format, args...)
	fmt.Fprintln(d.w)
}

func (d *dumper) pos(n Node) string {
	if !d.showPos {
		return ""
	}
	p := n.Pos()
	return fmt.Sprintf(" @%d:%d", p.Line, p.Column)
}

func (d *dumper) block(b *Block) {
	if b == nil {
		d.printf("(block)")
		return
	}
	if len(b.Stmts) == 0 {
		d.printf("(block%s)", d.pos(b))
		return
	}
	d.printf("(block%s", d.pos(b))
	d.depth++
	for _, s := range b.Stmts {
		d.stmt(s)
	}
	d.depth--
	d.printf(")")
}

func (d *dumper) stmt(s Stmt) {
	switch s := s.(type) {
	case *AssignStmt:
		d.printf("(assign%s", d.pos(s))
		d.depth++
		d.printf("(targets")
		d.depth++
		for _, t := range s.Targets {
			d.expr(t)
		}
		d.depth--
		d.printf(")")
		d.printf("(values")
		d.depth++
		for _, v := range s.Values {
			d.expr(v)
		}
		d.depth--
		d.printf(")")
		d.depth--
		d.printf(")")

	case *LocalStmt:
		d.printf("(local%s", d.pos(s))
		d.depth++
		d.printf("(names")
		d.depth++
		for i, n := range s.Names {
			attr := ""
			if i < len(s.Attribs) && s.Attribs[i] != "" {
				attr = fmt.Sprintf(" <%s>", s.Attribs[i])
			}
			d.printf("%s%s", n.Name, attr)
		}
		d.depth--
		d.printf(")")
		if len(s.Values) > 0 {
			d.printf("(values")
			d.depth++
			for _, v := range s.Values {
				d.expr(v)
			}
			d.depth--
			d.printf(")")
		}
		d.depth--
		d.printf(")")

	case *DoStmt:
		d.printf("(do%s", d.pos(s))
		d.depth++
		d.block(s.Body)
		d.depth--
		d.printf(")")

	case *WhileStmt:
		d.printf("(while%s", d.pos(s))
		d.depth++
		d.expr(s.Cond)
		d.block(s.Body)
		d.depth--
		d.printf(")")

	case *RepeatStmt:
		d.printf("(repeat%s", d.pos(s))
		d.depth++
		d.block(s.Body)
		d.printf("(until")
		d.depth++
		d.expr(s.Cond)
		d.depth--
		d.printf(")")
		d.depth--
		d.printf(")")

	case *IfStmt:
		d.printf("(if%s", d.pos(s))
		d.depth++
		d.expr(s.Cond)
		d.printf("(then")
		d.depth++
		d.block(s.Then)
		d.depth--
		d.printf(")")
		for _, elif := range s.ElseIfs {
			d.printf("(elseif")
			d.depth++
			d.expr(elif.Cond)
			d.printf("(then")
			d.depth++
			d.block(elif.Then)
			d.depth--
			d.printf(")")
			d.depth--
			d.printf(")")
		}
		if s.Else != nil {
			d.printf("(else")
			d.depth++
			d.block(s.Else)
			d.depth--
			d.printf(")")
		}
		d.depth--
		d.printf(")")

	case *ForNumStmt:
		d.printf("(for-num%s %s", d.pos(s), s.Name.Name)
		d.depth++
		d.printf("(start")
		d.depth++
		d.expr(s.Start)
		d.depth--
		d.printf(")")
		d.printf("(stop")
		d.depth++
		d.expr(s.Stop)
		d.depth--
		d.printf(")")
		if s.Step != nil {
			d.printf("(step")
			d.depth++
			d.expr(s.Step)
			d.depth--
			d.printf(")")
		}
		d.block(s.Body)
		d.depth--
		d.printf(")")

	case *ForInStmt:
		names := make([]string, len(s.Names))
		for i, n := range s.Names {
			names[i] = n.Name
		}
		d.printf("(for-in%s [%s]", d.pos(s), strings.Join(names, ", "))
		d.depth++
		d.printf("(iters")
		d.depth++
		for _, it := range s.Iters {
			d.expr(it)
		}
		d.depth--
		d.printf(")")
		d.block(s.Body)
		d.depth--
		d.printf(")")

	case *ReturnStmt:
		if len(s.Values) == 0 {
			d.printf("(return%s)", d.pos(s))
		} else {
			d.printf("(return%s", d.pos(s))
			d.depth++
			for _, v := range s.Values {
				d.expr(v)
			}
			d.depth--
			d.printf(")")
		}

	case *BreakStmt:
		d.printf("(break%s)", d.pos(s))

	case *GotoStmt:
		d.printf("(goto%s %s)", d.pos(s), s.Label)

	case *LabelStmt:
		d.printf("(label%s %s)", d.pos(s), s.Name)

	case *ExprStmt:
		d.printf("(expr-stmt%s", d.pos(s))
		d.depth++
		d.expr(s.Expr)
		d.depth--
		d.printf(")")

	case *FuncStmt:
		method := ""
		if s.IsMethod {
			method = " :method"
		}
		d.printf("(func-stmt%s%s", d.pos(s), method)
		d.depth++
		d.printf("(name")
		d.depth++
		d.expr(s.Name)
		d.depth--
		d.printf(")")
		d.funcExpr(s.Func)
		d.depth--
		d.printf(")")

	case *LocalFuncStmt:
		d.printf("(local-func%s %s", d.pos(s), s.Name.Name)
		d.depth++
		d.funcExpr(s.Func)
		d.depth--
		d.printf(")")

	case *GlobalStmt:
		if s.Star {
			d.printf("(global%s *)", d.pos(s))
		} else {
			d.printf("(global%s", d.pos(s))
			d.depth++
			d.printf("(names")
			d.depth++
			for i, n := range s.Names {
				attr := ""
				if i < len(s.Attribs) && s.Attribs[i] != "" {
					attr = fmt.Sprintf(" <%s>", s.Attribs[i])
				}
				d.printf("%s%s", n.Name, attr)
			}
			d.depth--
			d.printf(")")
			if len(s.Values) > 0 {
				d.printf("(values")
				d.depth++
				for _, v := range s.Values {
					d.expr(v)
				}
				d.depth--
				d.printf(")")
			}
			d.depth--
			d.printf(")")
		}

	case *GlobalFuncStmt:
		d.printf("(global-func%s %s", d.pos(s), s.Name.Name)
		d.depth++
		d.funcExpr(s.Func)
		d.depth--
		d.printf(")")

	case *EmptyStmt:
		d.printf("(;%s)", d.pos(s))

	default:
		d.printf("(?unknown-stmt)")
	}
}

func (d *dumper) expr(e Expr) {
	switch e := e.(type) {
	case *NilExpr:
		d.printf("nil%s", d.pos(e))
	case *TrueExpr:
		d.printf("true%s", d.pos(e))
	case *FalseExpr:
		d.printf("false%s", d.pos(e))
	case *NumberExpr:
		d.printf("(int%s %s)", d.pos(e), e.Raw)
	case *FloatExpr:
		d.printf("(float%s %s)", d.pos(e), e.Raw)
	case *StringExpr:
		d.printf("(string%s %q)", d.pos(e), e.Value)
	case *VarArgExpr:
		d.printf("...%s", d.pos(e))
	case *NameExpr:
		d.printf("(name%s %s)", d.pos(e), e.Name)

	case *BinopExpr:
		d.printf("(binop%s %s", d.pos(e), e.Op)
		d.depth++
		d.expr(e.Left)
		d.expr(e.Right)
		d.depth--
		d.printf(")")

	case *UnopExpr:
		d.printf("(unop%s %s", d.pos(e), e.Op)
		d.depth++
		d.expr(e.Operand)
		d.depth--
		d.printf(")")

	case *IndexExpr:
		d.printf("(index%s", d.pos(e))
		d.depth++
		d.expr(e.Table)
		d.expr(e.Key)
		d.depth--
		d.printf(")")

	case *FieldExpr:
		d.printf("(field%s .%s", d.pos(e), e.Field)
		d.depth++
		d.expr(e.Table)
		d.depth--
		d.printf(")")

	case *MethodCallExpr:
		d.printf("(method-call%s :%s", d.pos(e), e.Method)
		d.depth++
		d.expr(e.Object)
		if len(e.Args) > 0 {
			d.printf("(args")
			d.depth++
			for _, a := range e.Args {
				d.expr(a)
			}
			d.depth--
			d.printf(")")
		}
		d.depth--
		d.printf(")")

	case *FuncCallExpr:
		d.printf("(call%s", d.pos(e))
		d.depth++
		d.expr(e.Func)
		if len(e.Args) > 0 {
			d.printf("(args")
			d.depth++
			for _, a := range e.Args {
				d.expr(a)
			}
			d.depth--
			d.printf(")")
		}
		d.depth--
		d.printf(")")

	case *FuncExpr:
		d.funcExpr(e)

	case *TableConstructor:
		if len(e.Fields) == 0 {
			d.printf("(table%s {})", d.pos(e))
		} else {
			d.printf("(table%s {", d.pos(e))
			d.depth++
			for _, f := range e.Fields {
				if f.Key != nil {
					d.printf("(field")
					d.depth++
					d.printf("(key")
					d.depth++
					d.expr(f.Key)
					d.depth--
					d.printf(")")
					d.printf("(val")
					d.depth++
					d.expr(f.Value)
					d.depth--
					d.printf(")")
					d.depth--
					d.printf(")")
				} else {
					d.printf("(item")
					d.depth++
					d.expr(f.Value)
					d.depth--
					d.printf(")")
				}
			}
			d.depth--
			d.printf("})")
		}

	case *ParenExpr:
		d.printf("(paren%s", d.pos(e))
		d.depth++
		d.expr(e.Inner)
		d.depth--
		d.printf(")")

	default:
		d.printf("(?unknown-expr)")
	}
}

func (d *dumper) funcExpr(f *FuncExpr) {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.Name
	}
	va := ""
	if f.VarArg {
		if f.VarArgName != "" {
			va = fmt.Sprintf(", ...%s", f.VarArgName)
		} else {
			va = ", ..."
		}
	}
	if len(params) == 0 && f.VarArg {
		va = strings.TrimPrefix(va, ", ")
	}
	d.printf("(function%s (%s%s)", d.pos(f), strings.Join(params, ", "), va)
	d.depth++
	d.block(f.Body)
	d.depth--
	d.printf(")")
}

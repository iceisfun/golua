package parser

import (
	"fmt"

	"github.com/iceisfun/golua/v2/ast"
	"github.com/iceisfun/golua/v2/token"
)

// ---------------------------------------------------------------------------
// Expression parsing (precedence climbing)
// ---------------------------------------------------------------------------
//
// The parser uses Lua's precedence-climbing scheme:
// - every binary operator has a left and right binding priority
// - higher numbers bind tighter
// - right-associative operators use right = left-1 (e.g. "^", "..")
//
// parseSubExpr(limit) parses an expression whose next operator must have
// left priority strictly greater than limit to continue binding.

type precedence int

const (
	precedenceLowest precedence = iota
	precedenceOr
	precedenceAnd
	precedenceCompare
	precedenceBitOr
	precedenceBitXor
	precedenceBitAnd
	precedenceShift
	precedenceConcatRight
	precedenceConcatLeft
	precedenceAdd
	precedenceMul
	precedenceUnary
	precedencePowRight
	precedencePowLeft
)

// String returns the precedence level's name for debugging.
func (p precedence) String() string {
	switch p {
	case precedenceLowest:
		return "precedenceLowest"
	case precedenceOr:
		return "precedenceOr"
	case precedenceAnd:
		return "precedenceAnd"
	case precedenceCompare:
		return "precedenceCompare"
	case precedenceBitOr:
		return "precedenceBitOr"
	case precedenceBitXor:
		return "precedenceBitXor"
	case precedenceBitAnd:
		return "precedenceBitAnd"
	case precedenceShift:
		return "precedenceShift"
	case precedenceConcatRight:
		return "precedenceConcatRight"
	case precedenceConcatLeft:
		return "precedenceConcatLeft"
	case precedenceAdd:
		return "precedenceAdd"
	case precedenceMul:
		return "precedenceMul"
	case precedenceUnary:
		return "precedenceUnary"
	case precedencePowRight:
		return "precedencePowRight"
	case precedencePowLeft:
		return "precedencePowLeft"
	default:
		return fmt.Sprintf("precedence(%d)", int(p))
	}
}

type binPriority struct {
	left  precedence
	right precedence
}

func leftAssoc(level precedence) binPriority {
	return binPriority{left: level, right: level}
}

func rightAssoc(level precedence) binPriority {
	return binPriority{left: level, right: level - 1}
}

func (p binPriority) binds(limit precedence) bool {
	return p.left > limit
}

type binPriorities map[string]binPriority

func (bp binPriorities) lookup(op string) (binPriority, bool) {
	pri, ok := bp[op]
	return pri, ok
}

var priorities = binPriorities{
	"+":   leftAssoc(precedenceAdd),
	"-":   leftAssoc(precedenceAdd),
	"*":   leftAssoc(precedenceMul),
	"%":   leftAssoc(precedenceMul),
	"^":   rightAssoc(precedencePowLeft),
	"/":   leftAssoc(precedenceMul),
	"//":  leftAssoc(precedenceMul),
	"&":   leftAssoc(precedenceBitAnd),
	"|":   leftAssoc(precedenceBitOr),
	"~":   leftAssoc(precedenceBitXor),
	"<<":  leftAssoc(precedenceShift),
	">>":  leftAssoc(precedenceShift),
	"..":  rightAssoc(precedenceConcatLeft),
	"==":  leftAssoc(precedenceCompare),
	"<":   leftAssoc(precedenceCompare),
	"<=":  leftAssoc(precedenceCompare),
	"~=":  leftAssoc(precedenceCompare),
	">":   leftAssoc(precedenceCompare),
	">=":  leftAssoc(precedenceCompare),
	"and": leftAssoc(precedenceAnd),
	"or":  leftAssoc(precedenceOr),
}

func (p *parser) getBinop() string {
	switch p.tok.Type {
	case token.Type('+'):
		return "+"
	case token.Type('-'):
		return "-"
	case token.Type('*'):
		return "*"
	case token.Type('/'):
		return "/"
	case token.IDIV:
		return "//"
	case token.Type('%'):
		return "%"
	case token.Type('^'):
		return "^"
	case token.Type('&'):
		return "&"
	case token.Type('|'):
		return "|"
	case token.Type('~'):
		return "~"
	case token.SHL:
		return "<<"
	case token.SHR:
		return ">>"
	case token.CONCAT:
		return ".."
	case token.EQ:
		return "=="
	case token.NE:
		return "~="
	case token.Type('<'):
		return "<"
	case token.LE:
		return "<="
	case token.Type('>'):
		return ">"
	case token.GE:
		return ">="
	case token.AND:
		return "and"
	case token.OR:
		return "or"
	default:
		return ""
	}
}

func (p *parser) getUnop() string {
	switch p.tok.Type {
	case token.NOT:
		return "not"
	case token.Type('-'):
		return "-"
	case token.Type('~'):
		return "~"
	case token.Type('#'):
		return "#"
	default:
		return ""
	}
}

func (p *parser) parseExpr() ast.Expr {
	return p.parseSubExpr(precedenceLowest)
}

// parseSubExpr -> (simpleexp | unop subexpr) { binop subexpr }
func (p *parser) parseSubExpr(limit precedence) ast.Expr {
	p.incDepth()
	defer p.decDepth()
	if p.err != nil {
		return ast.NewNilExpr(p.pos())
	}

	var expr ast.Expr

	if uop := p.getUnop(); uop != "" {
		pos := p.pos()
		p.advance()
		operand := p.parseSubExpr(precedenceUnary)
		expr = ast.NewUnopExpr(pos, uop, operand)
	} else {
		expr = p.parseSimpleExpr()
	}

	return p.continueBinExpr(expr, limit)
}

func (p *parser) parseSimpleExpr() ast.Expr {
	if p.err != nil {
		return ast.NewNilExpr(p.pos())
	}
	pos := p.pos()
	switch p.tok.Type {
	case token.INT:
		v := p.tok
		p.advance()
		return ast.NewNumberExpr(pos, v.IntVal, v.Literal)
	case token.FLOAT:
		v := p.tok
		p.advance()
		return ast.NewFloatExpr(pos, v.FltVal, v.Literal)
	case token.STRING:
		v := p.tok
		p.advance()
		s := ast.NewStringExpr(pos, v.Literal)
		s.EndP = tokenEnd(v)
		return s
	case token.NIL:
		p.advance()
		return ast.NewNilExpr(pos)
	case token.TRUE:
		p.advance()
		return ast.NewTrueExpr(pos)
	case token.FALSE:
		p.advance()
		return ast.NewFalseExpr(pos)
	case token.DOTS:
		p.advance()
		return ast.NewVarArgExpr(pos)
	case token.Type('{'):
		return p.parseTableConstructor()
	case token.FUNCTION:
		p.advance()
		return p.parseFuncBody(false)
	default:
		return p.parseSuffixedExpr()
	}
}

func (p *parser) parseSuffixedExpr() ast.Expr {
	return p.continueSuffixedExpr(p.parsePrimaryExpr())
}

func (p *parser) parsePrimaryExpr() ast.Expr {
	if p.err != nil {
		return ast.NewNilExpr(p.pos())
	}
	switch p.tok.Type {
	case token.NAME:
		return p.parseName()
	case token.Type('('):
		pos := p.pos()
		openLine := p.tok.Pos.Line
		p.advance()
		inner := p.parseExpr()
		closeTok := p.tok
		p.checkMatch(token.Type(')'), "(", openLine)
		pe := ast.NewParenExpr(pos, inner)
		pe.EndP = tokenEnd(closeTok)
		return pe
	default:
		p.errorf("unexpected symbol%s", p.nearClause())
		return ast.NewNilExpr(p.pos())
	}
}

func (p *parser) continueSuffixedExpr(expr ast.Expr) ast.Expr {
	for {
		if p.err != nil {
			return expr
		}
		switch {
		case p.check(token.Type('.')):
			pos := p.pos()
			p.advance()
			field := p.parseName()
			fe := ast.NewFieldExpr(pos, expr, field.Name)
			fe.EndP = field.End()
			expr = fe
		case p.check(token.Type('[')):
			pos := p.pos()
			p.advance()
			key := p.parseExpr()
			closeTok, _ := p.expect(token.Type(']'))
			ie := ast.NewIndexExpr(pos, expr, key)
			ie.EndP = tokenEnd(closeTok)
			expr = ie
		case p.check(token.Type(':')):
			pos := p.pos()
			p.advance()
			method := p.parseName()
			args, argsEnd := p.parseFuncArgs()
			mc := ast.NewMethodCallExpr(pos, expr, method.Name, args)
			mc.EndP = argsEnd
			expr = mc
		case p.check(token.Type('(')) || p.check(token.STRING) || p.check(token.Type('{')):
			pos := p.pos()
			args, argsEnd := p.parseFuncArgs()
			fc := ast.NewFuncCallExpr(pos, expr, args)
			fc.EndP = argsEnd
			expr = fc
		default:
			return expr
		}
	}
}

// parseFuncArgs parses a call's argument list and returns the arguments along
// with the position just past the closing delimiter (')' for parenthesised
// calls, or the end of the single string/table argument for the f"..." / f{}
// call forms).
func (p *parser) parseFuncArgs() ([]ast.Expr, token.Pos) {
	switch p.tok.Type {
	case token.Type('('):
		openLine := p.tok.Pos.Line
		p.advance()
		var args []ast.Expr
		if !p.check(token.Type(')')) {
			args = p.parseExprList()
		}
		closeTok := p.tok
		p.checkMatch(token.Type(')'), "(", openLine)
		return args, tokenEnd(closeTok)
	case token.Type('{'):
		tc := p.parseTableConstructor()
		return []ast.Expr{tc}, tc.End()
	case token.STRING:
		pos := p.pos()
		v := p.tok
		p.advance()
		s := ast.NewStringExpr(pos, v.Literal)
		s.EndP = tokenEnd(v)
		return []ast.Expr{s}, s.EndP
	default:
		p.errorf("function arguments expected%s", p.nearClause())
		return nil, p.pos()
	}
}

func (p *parser) parseFuncBody(isMethod bool) *ast.FuncExpr {
	return p.parseFuncBodyAt(isMethod, p.tok.Pos.Line)
}

func (p *parser) parseFuncBodyAt(isMethod bool, funcLine int) *ast.FuncExpr {
	p.incDepth()
	defer p.decDepth()
	p.pushFuncScope(funcLine)
	defer p.popFuncScope()
	pos := p.pos()
	p.expect(token.Type('('))

	var params []*ast.NameExpr
	vararg := false
	varargName := ""

	if isMethod {
		params = append(params, ast.NewNameExpr(pos, "self"))
	}

	if !p.check(token.Type(')')) {
		for {
			if p.check(token.DOTS) {
				vararg = true
				p.advance()
				// Lua 5.5: named vararg parameter — `... name`
				if p.check(token.NAME) {
					varargName = p.tok.Literal
					p.advance()
					p.trackLocals(1) // count the named vararg as a local
				}
				break
			}
			if !p.check(token.NAME) {
				p.errorf("<name> or '...' expected%s", p.nearClause())
				break
			}
			params = append(params, p.parseName())
			if !p.match(token.Type(',')) {
				break
			}
		}
	}
	// Count parameters as locals in this function scope.
	p.trackLocals(len(params))
	p.expect(token.Type(')'))
	body := p.parseBlock()
	endTok := p.tok
	endLine := p.tok.Pos.Line
	p.checkMatch(token.END, "function", funcLine)
	fe := ast.NewFuncExpr(pos, params, vararg, varargName, body)
	fe.EndLine = endLine
	fe.EndP = tokenEnd(endTok)
	return fe
}

func (p *parser) parseTableConstructor() *ast.TableConstructor {
	pos := p.pos()
	openLine := p.tok.Pos.Line
	p.expect(token.Type('{'))
	var fields []*ast.TableField
	for !p.check(token.Type('}')) {
		if p.err != nil {
			break
		}
		fields = append(fields, p.parseField())
		if !p.match(token.Type(',')) && !p.match(token.Type(';')) {
			break
		}
	}
	closeTok := p.tok
	p.checkMatch(token.Type('}'), "{", openLine)
	tc := ast.NewTableConstructor(pos, fields)
	tc.EndP = tokenEnd(closeTok)
	return tc
}

// parseField: '[' expr ']' '=' expr | NAME '=' expr | expr
func (p *parser) parseField() *ast.TableField {
	pos := p.pos()

	// [expr] = expr
	if p.check(token.Type('[')) {
		p.advance()
		key := p.parseExpr()
		p.expect(token.Type(']'))
		p.expect(token.Type('='))
		val := p.parseExpr()
		return ast.NewTableField(pos, key, val)
	}

	// NAME '=' expr  (lookahead: consume NAME, check for '=')
	if p.check(token.NAME) {
		nameTok := p.tok
		p.advance() // consume NAME
		if p.check(token.Type('=')) {
			// Record field
			p.advance() // skip '='
			val := p.parseExpr()
			key := ast.NewStringExpr(pos, nameTok.Literal)
			return ast.NewTableField(pos, key, val)
		}
		// Not a record field — build the expression starting from the name
		nameExpr := ast.NewNameExpr(pos, nameTok.Literal)
		expr := p.continueSuffixedExpr(nameExpr)
		expr = p.continueBinExpr(expr, precedenceLowest)
		return ast.NewTableField(pos, nil, expr)
	}

	// List expression
	val := p.parseExpr()
	return ast.NewTableField(pos, nil, val)
}

// continueBinExpr handles residual binary operators after partial expression parsing.
func (p *parser) continueBinExpr(expr ast.Expr, limit precedence) ast.Expr {
	for {
		op := p.getBinop()
		if op == "" {
			break
		}
		pri, ok := priorities.lookup(op)
		if !ok || !pri.binds(limit) {
			break
		}
		pos := p.pos()
		p.advance()
		right := p.parseSubExpr(pri.right)
		expr = ast.NewBinopExpr(pos, op, expr, right)
	}
	return expr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (p *parser) parseName() *ast.NameExpr {
	pos := p.pos()
	tok, _ := p.expect(token.NAME)
	return ast.NewNameExpr(pos, tok.Literal)
}

func (p *parser) parseExprList() []ast.Expr {
	exprs := []ast.Expr{p.parseExpr()}
	for p.match(token.Type(',')) {
		exprs = append(exprs, p.parseExpr())
	}
	return exprs
}

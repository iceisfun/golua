// Package parser implements a recursive descent parser for Lua 5.5,
// producing an AST from lexer tokens.
package parser

import (
	"fmt"

	"github.com/iceisfun/golua/ast"
	"github.com/iceisfun/golua/lexer"
	"github.com/iceisfun/golua/token"
)

// ParsePartial parses the given Lua source and returns a partial AST block
// alongside any error. Unlike Parse, it never returns a nil block — on error,
// the block contains all statements that were successfully parsed before the
// first error.
func ParsePartial(source, input string) (*ast.Block, error) {
	p := &parser{
		lex:    lexer.New(source, input),
		source: source,
	}
	if err := p.advance(); err != nil {
		return &ast.Block{Start: token.Pos{Source: source, Line: 1, Column: 1}}, err
	}
	block := p.parseBlock()
	if p.err != nil {
		return block, p.err
	}
	if p.tok.Type != token.EOS {
		return block, p.errorf("unexpected symbol near '%s'", p.tok.Literal)
	}
	return block, nil
}

// Parse parses the given Lua source and returns the top-level block.
func Parse(source, input string) (*ast.Block, error) {
	p := &parser{
		lex:    lexer.New(source, input),
		source: source,
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	block := p.parseBlock()
	if p.err != nil {
		return nil, p.err
	}
	if p.tok.Type != token.EOS {
		return nil, p.errorf("unexpected symbol near '%s'", p.tok.Literal)
	}
	return block, nil
}

// ---------------------------------------------------------------------------
// Parser state
// ---------------------------------------------------------------------------

type parser struct {
	lex    *lexer.Lexer
	tok    token.Token // current (lookahead) token
	source string
	err    error
}

func (p *parser) advance() error {
	tok, err := p.lex.Next()
	if err != nil {
		p.err = err
		return err
	}
	p.tok = tok
	return nil
}

func (p *parser) expect(typ token.Type) (token.Token, error) {
	if p.tok.Type != typ {
		return token.Token{}, p.errorf("'%s' expected near '%s'", typ, p.tok)
	}
	tok := p.tok
	if err := p.advance(); err != nil {
		return token.Token{}, err
	}
	return tok, nil
}

func (p *parser) check(typ token.Type) bool {
	return p.tok.Type == typ
}

func (p *parser) match(typ token.Type) bool {
	if p.tok.Type == typ {
		p.advance()
		return true
	}
	return false
}

func (p *parser) errorf(format string, args ...any) error {
	if p.err != nil {
		return p.err
	}
	p.err = &token.PosError{Pos: p.tok.Pos, Msg: fmt.Sprintf(format, args...)}
	return p.err
}

func (p *parser) pos() token.Pos { return p.tok.Pos }

// ---------------------------------------------------------------------------
// Block / statement list
// ---------------------------------------------------------------------------

func (p *parser) blockFollow(withUntil bool) bool {
	switch p.tok.Type {
	case token.ELSE, token.ELSEIF, token.END, token.EOS:
		return true
	case token.UNTIL:
		return withUntil
	default:
		return false
	}
}

func (p *parser) parseBlock() *ast.Block {
	block := &ast.Block{Start: p.pos()}
	for !p.blockFollow(true) {
		s := p.parseStatement()
		if s == nil {
			break
		}
		block.Stmts = append(block.Stmts, s)
		if p.err != nil {
			break
		}
	}
	return block
}

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

func (p *parser) parseStatement() ast.Stmt {
	if p.err != nil {
		return nil
	}
	switch p.tok.Type {
	case token.Type(';'):
		s := ast.NewEmptyStmt(p.pos())
		p.advance()
		return s
	case token.IF:
		return p.parseIfStmt()
	case token.WHILE:
		return p.parseWhileStmt()
	case token.DO:
		return p.parseDoStmt()
	case token.FOR:
		return p.parseForStmt()
	case token.REPEAT:
		return p.parseRepeatStmt()
	case token.FUNCTION:
		return p.parseFuncStmt()
	case token.LOCAL:
		return p.parseLocalStmt()
	case token.GLOBAL:
		return p.parseGlobalStmt()
	case token.RETURN:
		return p.parseReturnStmt()
	case token.BREAK:
		s := ast.NewBreakStmt(p.pos())
		p.advance()
		return s
	case token.GOTO:
		return p.parseGotoStmt()
	case token.DBCOLON:
		return p.parseLabelStmt()
	default:
		return p.parseExprStat()
	}
}

func (p *parser) parseIfStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'if'
	cond := p.parseExpr()
	p.expect(token.THEN)
	then := p.parseBlock()

	var elseifs []*ast.ElseIf
	for p.check(token.ELSEIF) {
		eiPos := p.pos()
		p.advance()
		eiCond := p.parseExpr()
		p.expect(token.THEN)
		eiThen := p.parseBlock()
		elseifs = append(elseifs, ast.NewElseIf(eiPos, eiCond, eiThen))
	}

	var elseb *ast.Block
	if p.match(token.ELSE) {
		elseb = p.parseBlock()
	}
	p.expect(token.END)
	return ast.NewIfStmt(pos, cond, then, elseifs, elseb)
}

func (p *parser) parseWhileStmt() ast.Stmt {
	pos := p.pos()
	p.advance()
	cond := p.parseExpr()
	p.expect(token.DO)
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewWhileStmt(pos, cond, body)
}

func (p *parser) parseDoStmt() ast.Stmt {
	pos := p.pos()
	p.advance()
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewDoStmt(pos, body)
}

func (p *parser) parseForStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'for'
	name := p.parseName()
	if p.check(token.Type('=')) {
		return p.parseForNumStmt(pos, name)
	}
	return p.parseForInStmt(pos, name)
}

func (p *parser) parseForNumStmt(pos token.Pos, name *ast.NameExpr) ast.Stmt {
	p.expect(token.Type('='))
	start := p.parseExpr()
	p.expect(token.Type(','))
	stop := p.parseExpr()
	var step ast.Expr
	if p.match(token.Type(',')) {
		step = p.parseExpr()
	}
	p.expect(token.DO)
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewForNumStmt(pos, name, start, stop, step, body)
}

func (p *parser) parseForInStmt(pos token.Pos, firstName *ast.NameExpr) ast.Stmt {
	names := []*ast.NameExpr{firstName}
	for p.match(token.Type(',')) {
		names = append(names, p.parseName())
	}
	p.expect(token.IN)
	iters := p.parseExprList()
	p.expect(token.DO)
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewForInStmt(pos, names, iters, body)
}

func (p *parser) parseRepeatStmt() ast.Stmt {
	pos := p.pos()
	p.advance()
	body := p.parseBlock()
	p.expect(token.UNTIL)
	cond := p.parseExpr()
	return ast.NewRepeatStmt(pos, body, cond)
}

// parseFuncStmt: function funcname body
// funcname: NAME {'.' NAME} [':' NAME]
func (p *parser) parseFuncStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'function'
	name := p.parseName()
	var nameExpr ast.Expr = name
	isMethod := false
	for p.match(token.Type('.')) {
		field := p.parseName()
		nameExpr = ast.NewFieldExpr(nameExpr.Pos(), nameExpr, field.Name)
	}
	if p.match(token.Type(':')) {
		method := p.parseName()
		nameExpr = ast.NewFieldExpr(nameExpr.Pos(), nameExpr, method.Name)
		isMethod = true
	}
	fn := p.parseFuncBody(isMethod)
	return ast.NewFuncStmt(pos, nameExpr, isMethod, fn)
}

func (p *parser) parseLocalStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'local'

	if p.match(token.FUNCTION) {
		name := p.parseName()
		fn := p.parseFuncBody(false)
		return ast.NewLocalFuncStmt(pos, name, fn)
	}

	defAttrib := p.parseAttrib()
	names := []*ast.NameExpr{p.parseName()}
	attribs := []string{p.parseAttribOr(defAttrib)}
	for p.match(token.Type(',')) {
		names = append(names, p.parseName())
		attribs = append(attribs, p.parseAttribOr(defAttrib))
	}
	var values []ast.Expr
	if p.match(token.Type('=')) {
		values = p.parseExprList()
	}
	return ast.NewLocalStmt(pos, names, attribs, values)
}

func (p *parser) parseGlobalStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'global'

	if p.match(token.FUNCTION) {
		name := p.parseName()
		fn := p.parseFuncBody(false)
		return ast.NewGlobalFuncStmt(pos, name, fn)
	}

	defAttrib := p.parseAttrib()
	if p.match(token.Type('*')) {
		return ast.NewGlobalStarStmt(pos, defAttrib)
	}

	names := []*ast.NameExpr{p.parseName()}
	attribs := []string{p.parseAttribOr(defAttrib)}
	for p.match(token.Type(',')) {
		names = append(names, p.parseName())
		attribs = append(attribs, p.parseAttribOr(defAttrib))
	}
	var values []ast.Expr
	if p.match(token.Type('=')) {
		values = p.parseExprList()
	}
	return ast.NewGlobalStmt(pos, names, attribs, values)
}

func (p *parser) parseAttrib() string {
	if p.match(token.Type('<')) {
		tok, _ := p.expect(token.NAME)
		p.expect(token.Type('>'))
		return tok.Literal
	}
	return ""
}

func (p *parser) parseAttribOr(def string) string {
	if a := p.parseAttrib(); a != "" {
		return a
	}
	return def
}

func (p *parser) parseReturnStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'return'
	var values []ast.Expr
	if !p.blockFollow(true) && !p.check(token.Type(';')) {
		values = p.parseExprList()
	}
	p.match(token.Type(';'))
	return ast.NewReturnStmt(pos, values)
}

func (p *parser) parseGotoStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'goto'
	name, _ := p.expect(token.NAME)
	return ast.NewGotoStmt(pos, name.Literal)
}

func (p *parser) parseLabelStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip '::'
	name, _ := p.expect(token.NAME)
	p.expect(token.DBCOLON)
	return ast.NewLabelStmt(pos, name.Literal)
}

func (p *parser) parseExprStat() ast.Stmt {
	pos := p.pos()
	expr := p.parseSuffixedExpr()

	if p.check(token.Type('=')) || p.check(token.Type(',')) {
		targets := []ast.Expr{expr}
		for p.match(token.Type(',')) {
			targets = append(targets, p.parseSuffixedExpr())
		}
		p.expect(token.Type('='))
		values := p.parseExprList()
		return ast.NewAssignStmt(pos, targets, values)
	}

	switch expr.(type) {
	case *ast.FuncCallExpr, *ast.MethodCallExpr:
		return ast.NewExprStmt(pos, expr)
	default:
		p.errorf("syntax error: unexpected expression statement")
		return nil
	}
}

// ---------------------------------------------------------------------------
// Expression parsing (precedence climbing)
// ---------------------------------------------------------------------------

type binPriority struct{ left, right int }

var priorities = map[string]binPriority{
	"+":   {10, 10},
	"-":   {10, 10},
	"*":   {11, 11},
	"%":   {11, 11},
	"^":   {14, 13}, // right-assoc
	"/":   {11, 11},
	"//":  {11, 11},
	"&":   {6, 6},
	"|":   {4, 4},
	"~":   {5, 5},
	"<<":  {7, 7},
	">>":  {7, 7},
	"..":  {9, 8}, // right-assoc
	"==":  {3, 3},
	"<":   {3, 3},
	"<=":  {3, 3},
	"~=":  {3, 3},
	">":   {3, 3},
	">=":  {3, 3},
	"and": {2, 2},
	"or":  {1, 1},
}

const unaryPriority = 12

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
	return p.parseSubExpr(0)
}

// parseSubExpr -> (simpleexp | unop subexpr) { binop subexpr }
func (p *parser) parseSubExpr(limit int) ast.Expr {
	if p.err != nil {
		return ast.NewNilExpr(p.pos())
	}

	var expr ast.Expr

	if uop := p.getUnop(); uop != "" {
		pos := p.pos()
		p.advance()
		operand := p.parseSubExpr(unaryPriority)
		expr = ast.NewUnopExpr(pos, uop, operand)
	} else {
		expr = p.parseSimpleExpr()
	}

	for {
		op := p.getBinop()
		if op == "" {
			break
		}
		pri := priorities[op]
		if pri.left <= limit {
			break
		}
		pos := p.pos()
		p.advance()
		right := p.parseSubExpr(pri.right)
		expr = ast.NewBinopExpr(pos, op, expr, right)
	}

	return expr
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
		return ast.NewStringExpr(pos, v.Literal)
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
		p.advance()
		inner := p.parseExpr()
		p.expect(token.Type(')'))
		return ast.NewParenExpr(pos, inner)
	default:
		p.errorf("unexpected symbol near '%s'", p.tok)
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
			expr = ast.NewFieldExpr(pos, expr, field.Name)
		case p.check(token.Type('[')):
			pos := p.pos()
			p.advance()
			key := p.parseExpr()
			p.expect(token.Type(']'))
			expr = ast.NewIndexExpr(pos, expr, key)
		case p.check(token.Type(':')):
			pos := p.pos()
			p.advance()
			method := p.parseName()
			args := p.parseFuncArgs()
			expr = ast.NewMethodCallExpr(pos, expr, method.Name, args)
		case p.check(token.Type('(')) || p.check(token.STRING) || p.check(token.Type('{')):
			pos := p.pos()
			args := p.parseFuncArgs()
			expr = ast.NewFuncCallExpr(pos, expr, args)
		default:
			return expr
		}
	}
}

func (p *parser) parseFuncArgs() []ast.Expr {
	switch p.tok.Type {
	case token.Type('('):
		p.advance()
		var args []ast.Expr
		if !p.check(token.Type(')')) {
			args = p.parseExprList()
		}
		p.expect(token.Type(')'))
		return args
	case token.Type('{'):
		return []ast.Expr{p.parseTableConstructor()}
	case token.STRING:
		pos := p.pos()
		v := p.tok
		p.advance()
		return []ast.Expr{ast.NewStringExpr(pos, v.Literal)}
	default:
		p.errorf("function arguments expected near '%s'", p.tok)
		return nil
	}
}

func (p *parser) parseFuncBody(isMethod bool) *ast.FuncExpr {
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
				// Lua 5.5: ... can optionally be followed by a name
				if p.check(token.NAME) {
					varargName = p.tok.Literal
					p.advance()
				}
				break
			}
			params = append(params, p.parseName())
			if !p.match(token.Type(',')) {
				break
			}
		}
	}
	p.expect(token.Type(')'))
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewFuncExpr(pos, params, vararg, varargName, body)
}

func (p *parser) parseTableConstructor() *ast.TableConstructor {
	pos := p.pos()
	p.expect(token.Type('{'))
	var fields []*ast.TableField
	for !p.check(token.Type('}')) && !p.check(token.EOS) {
		if p.err != nil {
			break
		}
		fields = append(fields, p.parseField())
		if !p.match(token.Type(',')) && !p.match(token.Type(';')) {
			break
		}
	}
	p.expect(token.Type('}'))
	return ast.NewTableConstructor(pos, fields)
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
		expr = p.continueBinExpr(expr, 0)
		return ast.NewTableField(pos, nil, expr)
	}

	// List expression
	val := p.parseExpr()
	return ast.NewTableField(pos, nil, val)
}

// continueBinExpr handles residual binary operators after partial expression parsing.
func (p *parser) continueBinExpr(expr ast.Expr, limit int) ast.Expr {
	for {
		op := p.getBinop()
		if op == "" {
			break
		}
		pri := priorities[op]
		if pri.left <= limit {
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

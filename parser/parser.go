// Package parser implements a recursive descent parser for Lua 5.4, producing
// an abstract syntax tree (AST) from source text.
//
// The parser uses the lexer to tokenize input and builds an AST following
// Lua 5.4's grammar with precedence-climbing for expressions. It supports
// all Lua 5.4 syntax including goto/labels, local attributes (<const>, <close>),
// and shebang line stripping for script loading.
//
// Lua 5.4 Reference: §3 – The Language (§3.3 Statements, §3.4 Expressions).
package parser

import (
	"fmt"

	"github.com/iceisfun/golua/v2/ast"
	"github.com/iceisfun/golua/v2/lexer"
	"github.com/iceisfun/golua/v2/token"
)

// ParsePartial parses the given Lua source and returns a partial AST block
// alongside any error. Unlike Parse, it never returns a nil block — on error,
// the block contains all statements that were successfully parsed before the
// first error.
func ParsePartial(source, input string) (*ast.Block, error) {
	p := &parser{
		lex:    lexer.New(source, input, true),
		source: source,
		funcs:  []funcScope{{nLocals: 0, lineDef: 0}}, // main chunk
	}
	if err := p.advance(); err != nil {
		return &ast.Block{Start: token.Pos{Source: source, Line: 1, Column: 1}}, err
	}
	block := p.parseBlock()
	if p.err != nil {
		block.EndLine = p.tok.Pos.Line
		return block, p.err
	}
	if p.tok.Type != token.EOS {
		return block, p.errorf("<eof> expected%s", p.nearClause())
	}
	block.EndLine = p.tok.Pos.Line
	return block, nil
}

// Parse parses the given Lua source and returns the top-level block.
// An optional third argument controls shebang stripping (default true).
func Parse(source, input string, stripShebang ...bool) (*ast.Block, error) {
	strip := true
	if len(stripShebang) > 0 {
		strip = stripShebang[0]
	}
	return parseImpl(source, input, strip, 0)
}

// ParseWithMaxVars is like Parse but overrides the per-function local variable
// limit (Lua 5.4 MAXVARS). When maxVars is 0 the default (200) is used.
func ParseWithMaxVars(source, input string, maxVars int) (*ast.Block, error) {
	return parseImpl(source, input, true, maxVars)
}

func parseImpl(source, input string, strip bool, maxVars int) (*ast.Block, error) {
	p := &parser{
		lex:     lexer.New(source, input, strip),
		source:  source,
		funcs:   []funcScope{{nLocals: 0, lineDef: 0}}, // main chunk
		maxVars: maxVars,
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	block := p.parseBlock()
	if p.err != nil {
		return nil, p.err
	}
	if p.tok.Type != token.EOS {
		return nil, p.errorf("<eof> expected%s", p.nearClause())
	}
	block.EndLine = p.tok.Pos.Line
	return block, nil
}

// ---------------------------------------------------------------------------
// Parser state
// ---------------------------------------------------------------------------

// maxSyntaxLevels limits parser recursion depth to match Lua 5.4's
// LUAI_MAXCCALLS. Lua 5.4 hits a C stack overflow at ~196 levels of
// nesting; we use a similar limit. The error message matches Lua 5.4.
const maxSyntaxLevels = 200

// maxLocalVars matches Lua 5.4's MAXVARS limit (200 locals per function).
const maxLocalVars = 200

// funcScope tracks per-function local variable counts for MAXVARS checking.
type funcScope struct {
	nLocals    int   // current number of active locals in this function
	lineDef    int   // line where the function was defined (0 for main chunk)
	blockSaves []int // saved nLocals at block entry for scope restore
}

type parser struct {
	lex          *lexer.Lexer
	tok          token.Token // current (lookahead) token
	lookahead    token.Token // second lookahead token (valid when hasLookahead)
	hasLookahead bool
	source       string
	err          error
	depth        int         // recursion depth counter
	funcs        []funcScope // stack of function scopes for local var limit checking
	maxVars      int         // maximum local variables per function (0 = maxLocalVars)
}

func (p *parser) maxVarsLimit() int {
	if p.maxVars > 0 {
		return p.maxVars
	}
	return maxLocalVars
}

// isLvalue returns true if the expression is a valid assignment target
// (name, field access, or index expression). Matches Lua 5.4's check_lhs.
func isLvalue(e ast.Expr) bool {
	switch e.(type) {
	case *ast.NameExpr, *ast.FieldExpr, *ast.IndexExpr:
		return true
	}
	return false
}

func (p *parser) advance() error {
	if p.hasLookahead {
		p.tok = p.lookahead
		p.hasLookahead = false
		return nil
	}
	tok, err := p.lex.Next()
	if err != nil {
		p.err = err
		return err
	}
	p.tok = tok
	return nil
}

// peekNext returns the token after the current one without consuming it.
func (p *parser) peekNext() token.Token {
	if p.hasLookahead {
		return p.lookahead
	}
	tok, err := p.lex.Next()
	if err != nil {
		p.err = err
		return token.Token{Type: token.EOS}
	}
	p.lookahead = tok
	p.hasLookahead = true
	return tok
}

func (p *parser) expect(typ token.Type) (token.Token, error) {
	if p.tok.Type != typ {
		return token.Token{}, p.errorf("%s expected%s", tokenForError(typ), p.nearClause())
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

// incDepth increments the parser recursion depth and returns an error
// if it exceeds maxSyntaxLevels.
func (p *parser) incDepth() {
	p.depth++
	if p.depth >= maxSyntaxLevels {
		p.errorf("C stack overflow")
	}
}

func (p *parser) decDepth() {
	p.depth--
}

// pushFuncScope starts a new function scope for local variable tracking.
func (p *parser) pushFuncScope(lineDef int) {
	p.funcs = append(p.funcs, funcScope{nLocals: 0, lineDef: lineDef})
}

// popFuncScope removes the current function scope.
func (p *parser) popFuncScope() {
	if len(p.funcs) > 0 {
		p.funcs = p.funcs[:len(p.funcs)-1]
	}
}

// addLocals registers n new locals in the current function scope and
// checks the MAXVARS limit, producing an error matching Lua 5.4's format.
func (p *parser) addLocals(n int) {
	if len(p.funcs) == 0 {
		return
	}
	fs := &p.funcs[len(p.funcs)-1]
	fs.nLocals += n
	limit := p.maxVarsLimit()
	if fs.nLocals > limit {
		msg := fmt.Sprintf("too many local variables (limit is %d)", limit)
		if fs.lineDef == 0 {
			msg += " in main function"
		} else {
			msg += fmt.Sprintf(" in function at line %d", fs.lineDef)
		}
		msg += p.nearClause()
		p.errorf("%s", msg)
	}
}

// enterBlock saves the current local count for later restoration.
func (p *parser) enterBlock() {
	if len(p.funcs) == 0 {
		return
	}
	fs := &p.funcs[len(p.funcs)-1]
	fs.blockSaves = append(fs.blockSaves, fs.nLocals)
}

// leaveBlock restores the local count saved by the matching enterBlock.
func (p *parser) leaveBlock() {
	if len(p.funcs) == 0 {
		return
	}
	fs := &p.funcs[len(p.funcs)-1]
	n := len(fs.blockSaves)
	if n > 0 {
		fs.nLocals = fs.blockSaves[n-1]
		fs.blockSaves = fs.blockSaves[:n-1]
	}
}

func (p *parser) pos() token.Pos { return p.tok.Pos }

// checkMatch expects a closing token (e.g. '}', ')', 'end') that matches
// an opening token. If the close is missing and on a different line from the
// open, the error includes "(to close 'OPEN' at line N)" for better diagnostics.
// Matches Lua 5.4's check_match.
func (p *parser) checkMatch(close token.Type, openLiteral string, openLine int) error {
	if p.tok.Type == close {
		return p.advance()
	}
	if openLine == p.tok.Pos.Line {
		return p.errorf("%s expected%s", tokenForError(close), p.nearClause())
	}
	return p.errorf("%s expected (to close '%s' at line %d)%s",
		tokenForError(close), openLiteral, openLine, p.nearClause())
}

// tokenForError formats a token type for error messages.
// Lua 5.4 quotes keywords and single-char tokens but not <name>/<eof>/etc.
func tokenForError(typ token.Type) string {
	s := typ.String()
	if len(s) > 0 && s[0] == '<' {
		return s
	}
	return "'" + s + "'"
}

// nearToken returns the current token formatted for error messages:
// quoted for names/strings/numbers, unquoted for <eof> and keywords.
// Returns "" for null byte tokens (Lua 5.4 omits the near clause for \0
// because the null byte terminates the C string buffer).
func (p *parser) nearToken() string {
	switch p.tok.Type {
	case token.NAME, token.INT, token.FLOAT:
		return "'" + p.tok.Literal + "'"
	case token.STRING:
		// Use raw source text (with delimiters) if available, matching Lua 5.4.
		if p.tok.Raw != "" {
			return "'" + p.tok.Raw + "'"
		}
		return "'" + p.tok.Literal + "'"
	case token.EOS:
		return "<eof>"
	default:
		lit := p.tok.Literal
		// Escape non-printable/non-ASCII chars using Lua 5.4's <\NNN> format.
		// Lua 5.4 is byte-oriented: report the first source byte value.
		// For single-byte tokens (Type < 256), use byte(Type) directly since
		// raw bytes (0x80-0xFF) get re-encoded as multi-byte UTF-8 in the
		// literal, making lit[0] incorrect (e.g., 0xC2 for raw byte 0x80).
		// For multi-byte Unicode chars (Type >= 256, e.g., BOM U+FEFF),
		// use lit[0] which is the first byte of the UTF-8 encoding.
		if len(lit) > 0 {
			var b byte
			if int(p.tok.Type) < 256 {
				b = byte(p.tok.Type)
			} else {
				b = lit[0]
			}
			// Null byte: return empty to omit the "near" clause,
			// matching Lua 5.4 where \0 terminates the token buffer.
			if b == 0 {
				return ""
			}
			if b < 0x20 || b >= 0x7f {
				return fmt.Sprintf("'<\\%d>'", b)
			}
		}
		return "'" + lit + "'"
	}
}

// nearClause returns " near TOKEN" for error messages, or "" if the
// near token is empty (e.g. for null byte tokens). This mirrors Lua 5.4's
// luaX_syntaxerror which omits the near clause when txtToken returns "".
func (p *parser) nearClause() string {
	tok := p.nearToken()
	if tok == "" {
		return ""
	}
	return " near " + tok
}

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
		// return is the last statement in a block; no more statements allowed
		if _, ok := s.(*ast.ReturnStmt); ok {
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
		// Lua 5.5: "global" is a soft keyword — only treated as the global
		// declaration keyword at statement start when followed by a name,
		// attribute (<), wildcard (*), or 'function'. Otherwise it's a
		// regular name (e.g. global = 10, global(), global.field).
		if p.tok.Type == token.NAME && p.tok.Literal == "global" {
			next := p.peekNext()
			switch next.Type {
			case token.NAME, token.FUNCTION, token.Type('*'), token.Type('<'):
				return p.parseGlobalStmt()
			}
		}
		return p.parseExprStat()
	}
}

func (p *parser) parseIfStmt() ast.Stmt {
	p.incDepth()
	defer p.decDepth()
	pos := p.pos()
	openLine := p.tok.Pos.Line
	p.advance() // skip 'if'
	cond := p.parseExpr()
	thenLine := p.tok.Pos.Line // line of 'then' keyword
	p.expect(token.THEN)
	p.enterBlock()
	then := p.parseBlock()
	p.leaveBlock()

	var elseifs []*ast.ElseIf
	for p.check(token.ELSEIF) {
		eiPos := p.pos()
		p.advance()
		eiCond := p.parseExpr()
		eiThenLine := p.tok.Pos.Line // line of 'then' keyword
		p.expect(token.THEN)
		p.enterBlock()
		eiThen := p.parseBlock()
		p.leaveBlock()
		elseifs = append(elseifs, ast.NewElseIf(eiPos, eiCond, eiThenLine, eiThen))
	}

	var elseb *ast.Block
	if p.match(token.ELSE) {
		p.enterBlock()
		elseb = p.parseBlock()
		p.leaveBlock()
	}
	endLine := p.tok.Pos.Line // line of 'end' keyword
	p.checkMatch(token.END, "if", openLine)
	return ast.NewIfStmt(pos, cond, thenLine, then, elseifs, elseb, endLine)
}

func (p *parser) parseWhileStmt() ast.Stmt {
	p.incDepth()
	defer p.decDepth()
	pos := p.pos()
	openLine := p.tok.Pos.Line
	p.advance()
	cond := p.parseExpr()
	p.expect(token.DO)
	p.enterBlock()
	body := p.parseBlock()
	p.leaveBlock()
	endLine := p.tok.Pos.Line
	p.checkMatch(token.END, "while", openLine)
	return ast.NewWhileStmt(pos, cond, body, endLine)
}

func (p *parser) parseDoStmt() ast.Stmt {
	p.incDepth()
	defer p.decDepth()
	pos := p.pos()
	openLine := p.tok.Pos.Line
	p.advance()
	p.enterBlock()
	body := p.parseBlock()
	p.leaveBlock()
	endLine := p.tok.Pos.Line // line of 'end' keyword
	p.checkMatch(token.END, "do", openLine)
	return ast.NewDoStmt(pos, body, endLine)
}

func (p *parser) parseForStmt() ast.Stmt {
	p.incDepth()
	defer p.decDepth()
	pos := p.pos()
	openLine := p.tok.Pos.Line
	p.advance() // skip 'for'
	name := p.parseName()
	if p.check(token.Type('=')) {
		return p.parseForNumStmt(pos, openLine, name)
	}
	// If not '=' and not ',' or 'in', the for statement is malformed.
	// Report both options like Lua 5.4 does.
	if !p.check(token.Type(',')) && !p.check(token.IN) {
		p.errorf("'=' or 'in' expected%s", p.nearClause())
		return &ast.DoStmt{P: pos}
	}
	return p.parseForInStmt(pos, openLine, name)
}

func (p *parser) parseForNumStmt(pos token.Pos, openLine int, name *ast.NameExpr) ast.Stmt {
	p.expect(token.Type('='))
	start := p.parseExpr()
	p.expect(token.Type(','))
	stop := p.parseExpr()
	var step ast.Expr
	if p.match(token.Type(',')) {
		step = p.parseExpr()
	}
	p.expect(token.DO)
	p.enterBlock()
	p.addLocals(4) // for-loop internal variables: (for index), (for limit), (for step), name
	body := p.parseBlock()
	p.leaveBlock()
	endLine := p.tok.Pos.Line
	p.checkMatch(token.END, "for", openLine)
	return ast.NewForNumStmt(pos, name, start, stop, step, body, endLine)
}

func (p *parser) parseForInStmt(pos token.Pos, openLine int, firstName *ast.NameExpr) ast.Stmt {
	names := []*ast.NameExpr{firstName}
	for p.match(token.Type(',')) {
		names = append(names, p.parseName())
	}
	p.expect(token.IN)
	iters := p.parseExprList()
	p.expect(token.DO)
	p.enterBlock()
	p.addLocals(4 + len(names)) // for-in internal variables: (for state), (for control), (for toclose), (for iterator) + user names
	body := p.parseBlock()
	p.leaveBlock()
	endLine := p.tok.Pos.Line
	p.checkMatch(token.END, "for", openLine)
	return ast.NewForInStmt(pos, names, iters, body, endLine)
}

func (p *parser) parseRepeatStmt() ast.Stmt {
	p.incDepth()
	defer p.decDepth()
	pos := p.pos()
	openLine := p.tok.Pos.Line
	p.advance()
	p.enterBlock()
	body := p.parseBlock()
	p.checkMatch(token.UNTIL, "repeat", openLine)
	cond := p.parseExpr()
	p.leaveBlock()
	return ast.NewRepeatStmt(pos, body, cond)
}

// parseFuncStmt: function funcname body
// funcname: NAME {'.' NAME} [':' NAME]
func (p *parser) parseFuncStmt() ast.Stmt {
	pos := p.pos()
	funcLine := p.tok.Pos.Line
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
	fn := p.parseFuncBodyAt(isMethod, funcLine)
	return ast.NewFuncStmt(pos, nameExpr, isMethod, fn)
}

func (p *parser) parseLocalStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'local'

	if p.check(token.FUNCTION) {
		funcLine := p.tok.Pos.Line
		p.advance()
		name := p.parseName()
		p.addLocals(1) // local function name is a local in outer scope
		fn := p.parseFuncBodyAt(false, funcLine)
		return ast.NewLocalFuncStmt(pos, name, fn)
	}

	// Lua 5.5: local<const> x, y = ... — prefix attribute applies to all names
	defAttrib := p.parseAttrib()

	names := []*ast.NameExpr{p.parseName()}
	p.addLocals(1)
	if p.err != nil {
		return nil
	}
	attribs := []string{p.parseAttribOr(defAttrib)}
	for p.match(token.Type(',')) {
		names = append(names, p.parseName())
		p.addLocals(1)
		if p.err != nil {
			return nil
		}
		attribs = append(attribs, p.parseAttribOr(defAttrib))
		if p.err != nil {
			return nil
		}
	}
	if p.err != nil {
		return nil
	}

	// Lua 5.4: at most one <close> variable per local statement.
	closeCount := 0
	for _, a := range attribs {
		if a == "close" {
			closeCount++
		}
	}
	if closeCount > 1 {
		p.errorf("multiple to-be-closed variables in local list")
		return nil
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

	if p.check(token.FUNCTION) {
		funcLine := p.tok.Pos.Line
		p.advance()
		name := p.parseName()
		fn := p.parseFuncBodyAt(false, funcLine)
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
		tok, err := p.expect(token.NAME)
		if err != nil {
			return ""
		}
		// Lua 5.5's getvarattribute requires '>' BEFORE validating the
		// attribute name. So `local x<weird>=5` reports the missing '>'
		// (the lexer reads `>=` as a single token) rather than the unknown
		// attribute name.
		if _, err := p.expect(token.Type('>')); err != nil {
			return ""
		}
		if tok.Literal != "const" && tok.Literal != "close" {
			p.err = &token.PosError{Pos: tok.Pos, Msg: fmt.Sprintf("unknown attribute '%s'", tok.Literal)}
		}
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
	// Capture line after closing :: — Lua 5.4 stores the lexer's current
	// line number at this point, which may have advanced past a newline.
	endLine := p.pos().Line
	return ast.NewLabelStmt(pos, name.Literal, endLine)
}

func (p *parser) parseExprStat() ast.Stmt {
	pos := p.pos()
	expr := p.parseSuffixedExpr()

	if p.check(token.Type('=')) || p.check(token.Type(',')) {
		targets := []ast.Expr{expr}
		for p.match(token.Type(',')) {
			p.incDepth()
			targets = append(targets, p.parseSuffixedExpr())
			if p.err != nil {
				break
			}
		}
		// Validate all targets are lvalues (name, field, or index).
		// Lua 5.4 rejects non-lvalue assignment targets with "syntax error near '='".
		for _, t := range targets {
			if !isLvalue(t) {
				p.errorf("syntax error%s", p.nearClause())
				return nil
			}
		}
		p.depth -= len(targets) - 1 // unwind depth for all targets added
		p.expect(token.Type('='))
		values := p.parseExprList()
		return ast.NewAssignStmt(pos, targets, values)
	}

	switch expr.(type) {
	case *ast.FuncCallExpr, *ast.MethodCallExpr:
		return ast.NewExprStmt(pos, expr)
	default:
		p.errorf("syntax error%s", p.nearClause())
		return nil
	}
}

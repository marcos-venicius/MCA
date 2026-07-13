// TODO: fix some bugs:
//   - does it make sense to keep `{}` as map initializer instead of being a block scope?
//   - should we stack prefix unary operators?
//   - power is right-associative (shouldn't it be left?)
package parser

import (
	"fmt"
	"slices"
	"strconv"

	"mca/internal/ast"
	"mca/internal/lexer"
)

// TODO: we have position present here but spreaded, shouldn't I share the same object?
type Error struct {
	Filename  string
	Line, Col int
	Message   string
	Info      string // optional extra help text (ast_info), "" if none
}

func (e Error) Error() string {
	loc := fmt.Sprintf("%d:%d", e.Line, e.Col)
	if e.Filename != "" {
		loc = fmt.Sprintf("%s:%d:%d", e.Filename, e.Line, e.Col)
	}

	msg := fmt.Sprintf("%s \033[0;31msyntax error\033[0m: %s", loc, e.Message)
	if e.Info != "" {
		msg += "\n" + fmt.Sprintf("%s \033[0;36minfo\033[0m: %s", loc, e.Info)
	}

	return msg
}

// Program is the result of a full parse: the top-level statement list, plus
// any errors encountered.
type Program struct {
	// even though I called statements, the last expression is evaluated and returns,
	// btw, that's how import and exports works today. We leave an object at the end of the file
	// with the exported fields, functions, etc we want and then the import just evaluate the file and
	// loads the last evaluated expression (which is the 'export' object)
	Stmts  []ast.Expr
	Errors []Error
}

type parser struct {
	filename string
	tokens   []lexer.Token
	pos      int // index of the "current" token; pos >= len(tokens) means EOF
	lastPos  int // index of the last successfully consumed token, for EOF-error locations

	errors []Error
}

func Parse(filename string, tokens []lexer.Token) *Program {
	p := &parser{filename: filename, tokens: tokens}

	var stmts []ast.Expr

	for {
		for p.cur() != nil {
			if p.cur().Kind == lexer.Semi {
				p.next()
				continue
			}

			expr := p.parseExpr()

			// note: expr may be nil on error
			if expr != nil {
				stmts = append(stmts, expr)
			}
		}

		if p.cur() != nil {
			// only reachable if parseExpr stopped short of full consumption without erroring itself
			p.errorAt(p.cur(), fmt.Sprintf("expected EOF but got '%s'", p.cur().Value))
			p.synchronize()
			continue
		}

		break
	}

	return &Program{Stmts: stmts, Errors: p.errors}
}

// ---- token stream helpers (mirror token/ntoken/check/checkahead/next_token) ----

func (p *parser) cur() *lexer.Token {
	if p.pos >= len(p.tokens) {
		return nil
	}
	return &p.tokens[p.pos]
}

func (p *parser) peek() *lexer.Token {
	if p.pos+1 >= len(p.tokens) {
		return nil
	}
	return &p.tokens[p.pos+1]
}

func (p *parser) check(kind lexer.TokenKind) bool {
	c := p.cur()
	return c != nil && c.Kind == kind
}

func (p *parser) checkAhead(kind lexer.TokenKind) bool {
	n := p.peek()
	return n != nil && n.Kind == kind
}

func isLogicalAnd(tok *lexer.Token) bool {
	return tok != nil && tok.Kind == lexer.Ident && tok.Value == "and"
}

func isLogicalOr(tok *lexer.Token) bool {
	return tok != nil && tok.Kind == lexer.Ident && tok.Value == "or"
}

func isKeyword(tok *lexer.Token, kw string) bool {
	return tok != nil && tok.Kind == lexer.Ident && tok.Value == kw
}

func (p *parser) next() *lexer.Token {
	if p.cur() == nil {
		return nil
	}

	p.pos++

	if p.cur() != nil {
		p.lastPos = p.pos
	}

	return p.cur()
}

func (p *parser) lastConsumed() *lexer.Token {
	if p.lastPos < len(p.tokens) {
		return &p.tokens[p.lastPos]
	}
	if len(p.tokens) > 0 {
		return &p.tokens[len(p.tokens)-1]
	}
	return nil
}

func (p *parser) synchronize() {
	// TODO: is this enough?
	for p.cur() != nil && p.cur().Kind != lexer.Semi {
		p.next()
	}
}

func (p *parser) errorAt(tok *lexer.Token, message string) {
	e := Error{Filename: p.filename, Message: message}
	if tok != nil {
		e.Line, e.Col = tok.Loc.Line, tok.Loc.Col
	}
	p.errors = append(p.errors, e)
}

func (p *parser) errorInfoAt(tok *lexer.Token, message, info string) {
	e := Error{Filename: p.filename, Message: message, Info: info}
	if tok != nil {
		e.Line, e.Col = tok.Loc.Line, tok.Loc.Col
	}
	p.errors = append(p.errors, e)
}

func (p *parser) expect(kind lexer.TokenKind) bool {
	if p.cur() == nil || p.cur().Kind != kind {
		p.errorAt(p.lastConsumed(), fmt.Sprintf("expected '%s' but got '%s'", kind.DisplayName(), lexer.KindName(p.cur())))
		p.synchronize()
		return false
	}

	return true
}

func pos(filename string, tok *lexer.Token) ast.Pos {
	if tok == nil {
		return ast.Pos{Filename: filename}
	}
	return ast.Pos{Filename: filename, Line: tok.Loc.Line, Col: tok.Loc.Col}
}

func (p *parser) posOf(tok *lexer.Token) ast.Pos { return pos(p.filename, tok) }

// ---- operator mapping ----

func binaryOpForToken(kind lexer.TokenKind) ast.BinaryOp {
	switch kind {
	case lexer.Plus:
		return ast.PlusOp
	case lexer.Times:
		return ast.TimesOp
	case lexer.Minus:
		return ast.SubtractOp
	case lexer.Divide:
		return ast.DivideOp
	case lexer.Mod:
		return ast.ModOp
	case lexer.Pow:
		return ast.PowOp
	case lexer.Equal:
		return ast.EqualOp
	case lexer.NotEqual:
		return ast.NotEqualOp
	case lexer.Gt:
		return ast.GtOp
	case lexer.Lt:
		return ast.LtOp
	case lexer.Gte:
		return ast.GteOp
	case lexer.Lte:
		return ast.LteOp
	}

	panic("binaryOpForToken: invalid token kind as binary operator")
}

// ---- grammar: primary building blocks ----

func (p *parser) parseIdentifier() *ast.Ident {
	if !p.expect(lexer.Ident) {
		return nil
	}

	tok := p.cur()
	id := &ast.Ident{Base: ast.NewBase(p.posOf(tok)), Name: tok.Value}
	p.next()

	return id
}

// parseBlock mirrors parse_block_expression. Returns (body, ok); ok=false
// means an unrecoverable parse error occurred.
// A nil body with ok=true is a legitimately empty block (bare ';' form).
func (p *parser) parseBlock() ([]ast.Expr, bool) {
	if p.cur() == nil {
		p.errorAt(p.lastConsumed(), "invalid block expression. expected '{' but got EOF")
		p.synchronize()
		return nil, false
	}

	switch p.cur().Kind {
	case lexer.Semi:
		p.next() // empty block: `while true;`, `\(a, b) ->;`, etc
		return nil, true

	case lexer.LCurly:
		p.next() // skip '{'

		var body []ast.Expr

		for p.cur() != nil && p.cur().Kind != lexer.RCurly {
			if p.cur().Kind == lexer.Semi {
				p.next()
				continue
			}

			expr := p.parseExpr()
			if expr == nil {
				return nil, false
			}

			body = append(body, expr)
		}

		if p.cur() == nil || p.cur().Kind != lexer.RCurly {
			p.errorAt(p.lastConsumed(), fmt.Sprintf("unterminated block expression. expected '}' but got '%s'", lexer.KindName(p.cur())))
			p.synchronize()
			return nil, false
		}

		p.next() // skip '}'

		return body, true

	default:
		expr := p.parseExpr() // single inline expression as the block body

		if expr == nil {
			return nil, false
		}

		return []ast.Expr{expr}, true
	}
}

// parseCallExpr parses a '(' arg-list ')' suffix and wraps callee in a
// CallExpr. It's invoked from parsePostfixExpr as a general postfix
// operator, so callee can be any expression already parsed there (an Ident,
// a DotExpr field, an indexed value, a parenthesized expression, ...).
func (p *parser) parseCallExpr(callee ast.Expr) ast.Expr {
	lparen := p.cur()
	p.next() // skip '('

	next := p.cur()
	if next == nil {
		p.errorAt(lparen, "expected ')' or an expression but got EOF")
		p.synchronize()
		return nil
	}

	call := &ast.CallExpr{Base: ast.NewBase(callee.Pos()), Callee: callee}

	if next.Kind == lexer.RParen {
		p.next() // skip ')'
		return call
	}

	for p.cur() != nil && p.cur().Kind != lexer.RParen {
		expr := p.parseExpr()
		if expr == nil {
			return nil
		}

		call.Args = append(call.Args, expr)

		if p.cur() == nil {
			p.errorAt(lparen, "expected ',' or ')' but got EOF")
			p.synchronize()
			return nil
		}

		if p.cur().Kind == lexer.RParen {
			break
		}

		if p.cur().Kind != lexer.Comma {
			p.errorAt(lparen, fmt.Sprintf("expected ',' but got '%s'", p.cur().Value))
			p.synchronize()
			return nil
		}

		p.next() // skip ','
	}

	next = p.cur()
	if next == nil {
		p.errorAt(lparen, "expected ')' but got EOF")
		p.synchronize()
		return nil
	}

	if next.Kind != lexer.RParen {
		p.errorAt(lparen, fmt.Sprintf("expected ')' but got '%s'", next.Value))
		p.synchronize()
		return nil
	}

	p.next()

	return call
}

func (p *parser) parseFnExpr() ast.Expr {
	startTok := p.cur()
	fn := &ast.FnExpr{Base: ast.NewBase(p.posOf(startTok))}

	p.next() // skip '\'

	if !p.expect(lexer.LParen) {
		return nil
	}
	p.next() // skip '('

	for p.cur() != nil && p.cur().Kind != lexer.RParen {
		if !p.expect(lexer.Ident) {
			return nil
		}

		arg := p.parseIdentifier()
		fn.Params = append(fn.Params, arg)

		if p.cur() == nil {
			p.errorAt(p.lastConsumed(), "expected ',' or ')' but got EOF")
			p.synchronize()
			return nil
		}

		if p.cur().Kind == lexer.RParen {
			break
		}

		if p.cur().Kind != lexer.Comma {
			p.errorAt(p.lastConsumed(), fmt.Sprintf("expected ',' but got '%s'", p.cur().Kind.DisplayName()))
			p.synchronize()
			return nil
		}

		p.next() // skip ','
	}

	if !p.expect(lexer.RParen) {
		return nil
	}
	p.next() // skip ')'

	if !p.expect(lexer.Arrow) {
		return nil
	}
	p.next() // skip '->'

	body, _ := p.parseBlock()
	fn.Body = body

	return fn
}

func (p *parser) parseBreakExpr() ast.Expr {
	firstTok := p.cur()
	p.next() // jump 'break'

	var value ast.Expr

	if p.cur() != nil && p.cur().Kind != lexer.Semi {
		value = p.parseExpr()

		if value == nil {
			p.errorInfoAt(firstTok,
				"invalid break expression. Isn't it missing a ';'? 'break;'",
				"all break expressions that don't have a value should be terminated with a ';'")
			p.synchronize()
			return nil
		}
	}

	return &ast.BreakExpr{Base: ast.NewBase(p.posOf(firstTok)), Value: value}
}

func (p *parser) parseReturnExpr() ast.Expr {
	firstTok := p.cur()
	p.next() // jump 'return'

	var value ast.Expr

	if p.cur() != nil && p.cur().Kind != lexer.Semi && p.cur().Kind != lexer.RCurly {
		value = p.parseExpr()

		if value == nil {
			p.errorInfoAt(firstTok,
				"invalid return expression. Isn't it missing a ';'? 'return;'",
				"all return expressions that don't have a value should be terminated with a ';'")
			p.synchronize()
			return nil
		}
	}

	return &ast.ReturnExpr{Base: ast.NewBase(p.posOf(firstTok)), Value: value}
}

func (p *parser) parseWhileExpr() ast.Expr {
	firstTok := p.cur()
	p.next() // jump keyword

	if p.cur() == nil {
		p.errorAt(firstTok, "unterminated loop expression. expected an expression or a '{' but got EOF")
		p.synchronize()
		return nil
	}

	var condition ast.Expr

	if p.cur().Kind != lexer.LCurly {
		condition = p.parseExpr()
		if condition == nil {
			return nil
		}
	}

	body, _ := p.parseBlock()

	return &ast.WhileExpr{Base: ast.NewBase(p.posOf(firstTok)), Condition: condition, Body: body}
}

func (p *parser) parseForExpr() ast.Expr {
	keywordTok := p.cur()
	p.next() // skip 'for'

	if !p.check(lexer.Ident) {
		p.errorAt(p.lastConsumed(), fmt.Sprintf("expected an identifier but received %s", lexer.KindName(p.cur())))
		p.synchronize()
		return nil
	}

	index := p.parseIdentifier()

	var target, from, to, by ast.Expr
	var value *ast.Ident

	switch {
	case p.check(lexer.Colon): // for i : ... (range for loop)
		p.next() // skip ':'

		if p.check(lexer.LBracket) {
			p.next() // skip '['

			from = p.parseExpr()
			if from == nil {
				p.errorAt(p.lastConsumed(), "invalid for range expression. expected a primary expression")
				return nil
			}

			if p.check(lexer.RBracket) {
				p.errorInfoAt(p.lastConsumed(),
					"invalid for range expression",
					"in for loop ranges you should either use an integer or an array in this format `[from, to]` or `[from, to, by]`")
				p.synchronize()
				return nil
			}

			if !p.check(lexer.Comma) {
				p.errorAt(p.lastConsumed(), fmt.Sprintf("invalid for range expression. Expected ',' but got '%s'", lexer.KindName(p.cur())))
				p.synchronize()
				return nil
			}
			p.next() // skip ','

			to = p.parseExpr()
			if to == nil {
				p.errorAt(p.lastConsumed(), "invalid for range expression. expected a primary expression")
				return nil
			}

			if p.check(lexer.Comma) {
				p.next() // skip ','

				by = p.parseExpr()
				if by == nil {
					p.errorAt(p.lastConsumed(), "invalid for range expression. expected a primary expression")
					return nil
				}
			}

			if !p.check(lexer.RBracket) {
				p.errorAt(p.lastConsumed(), "invalid for range expression. missing close range ']'")
				p.synchronize()
				return nil
			}
			p.next() // skip ']'
		} else {
			from = p.parseExpr()
			if from == nil {
				p.errorAt(p.lastConsumed(), "invalid for range expression. expected a primary expression")
				return nil
			}
		}

	case p.check(lexer.Comma): // for k, v : target
		p.next() // skip ','

		if !p.check(lexer.Ident) {
			p.errorAt(p.lastConsumed(), fmt.Sprintf("invalid for loop expression. expected an identifier got '%s'", lexer.KindName(p.cur())))
			p.synchronize()
			return nil
		}

		value = p.parseIdentifier()

		if !p.check(lexer.Colon) {
			p.errorAt(p.lastConsumed(), fmt.Sprintf("invalid for loop expression. expected ':' got '%s'", lexer.KindName(p.cur())))
			p.synchronize()
			return nil
		}
		p.next() // skip ':'

		target = p.parseExpr()
		if target == nil {
			p.errorAt(p.lastConsumed(), "invalid for loop expression. expected a primary expression")
			return nil
		}

	default:
		p.errorInfoAt(p.lastConsumed(), "invalid for loop expression.",
			"you can implement for loops like: for k : [from, to, by?] {...\n"+
				"you can also implement for loops like: for k, v : <array|map|string> {...")
		p.synchronize()
		return nil
	}

	body, _ := p.parseBlock()

	if target != nil {
		return &ast.ForOfExpr{Base: ast.NewBase(p.posOf(keywordTok)), Key: index, Value: value, Target: target, Body: body}
	}

	return &ast.ForRangeExpr{Base: ast.NewBase(p.posOf(keywordTok)), Index: index, From: from, To: to, By: by, Body: body}
}

func (p *parser) parseIfExpr() ast.Expr {
	firstTok := p.cur()
	p.next() // jump 'if'

	if p.cur() == nil {
		p.errorAt(firstTok, "unterminated if expression. expected an expression but got EOF")
		p.synchronize()
		return nil
	}

	condition := p.parseExpr()
	if condition == nil {
		p.errorAt(firstTok, "unterminated if expression. expected an expression 'if ... {'")
		p.synchronize()
		return nil
	}

	if p.cur() == nil {
		p.errorAt(firstTok, "unterminated if expression. expected '{' but got EOF")
		p.synchronize()
		return nil
	}

	thenBody, _ := p.parseBlock()

	var elifs []ast.ElifBlock

	for isKeyword(p.cur(), "elif") {
		p.next() // skip 'elif'

		if p.cur() == nil {
			p.errorAt(firstTok, "unterminated elif expression. expected a condition 'elif ... {' but got EOF")
			p.synchronize()
			return nil
		}

		elifCond := p.parseExpr()
		if elifCond == nil {
			return nil
		}

		elifBody, _ := p.parseBlock()

		elifs = append(elifs, ast.ElifBlock{Condition: elifCond, Body: elifBody})
	}

	var elseBody []ast.Expr

	if isKeyword(p.cur(), "else") {
		p.next() // jump 'else'

		if p.cur() == nil {
			p.errorAt(firstTok, "unterminated if expression. expected 'else' block but got EOF")
			p.synchronize()
			return nil
		}

		elseBody, _ = p.parseBlock()
	}

	return &ast.IfExpr{
		Base:      ast.NewBase(p.posOf(firstTok)),
		Condition: condition,
		Then:      thenBody,
		Elifs:     elifs,
		Else:      elseBody,
	}
}

func (p *parser) parseStringLiteral() ast.Expr {
	tok := p.cur()

	// decode the limited escape set (\\ \' \n); the lexer already rejected
	// any other escape sequence, so this default branch is unreachable in
	// practice. TODO: add more escaping sequences
	var out []byte
	raw := tok.Value

	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' && i+1 < len(raw) {
			i++
			switch raw[i] {
			case '\'':
				out = append(out, '\'')
			case 'n':
				out = append(out, '\n')
			case '\\':
				out = append(out, '\\')
			default:
				out = append(out, raw[i])
			}
			continue
		}

		out = append(out, raw[i])
	}

	p.next() // jump over the string token

	return &ast.StringLit{Base: ast.NewBase(p.posOf(tok)), Value: string(out)}
}

func (p *parser) parseMapExpr() ast.Expr {
	startTok := p.cur()
	p.next() // skip '{'

	m := &ast.MapExpr{Base: ast.NewBase(p.posOf(startTok))}

	for !p.check(lexer.RCurly) {
		keyStartTok := p.cur()
		key := p.parseUnaryExpr()

		if !p.check(lexer.Colon) {
			p.errorAt(keyStartTok, "missing value for key inside the map")
			p.synchronize()
			return nil
		}
		p.next() // skip ':'

		value := p.parseUnaryExpr()
		if value == nil {
			return nil
		}

		if p.check(lexer.Comma) {
			p.next()
		} else if !p.check(lexer.RCurly) {
			p.errorAt(p.lastConsumed(), fmt.Sprintf("expected ',' but got unexpected '%s'", p.lastConsumed().Kind.DisplayName()))
			p.synchronize()
			return nil
		}

		m.Keys = append(m.Keys, key)
		m.Values = append(m.Values, value)
	}

	if !p.check(lexer.RCurly) {
		p.errorAt(p.lastConsumed(), "unclosed curly expression '{...'")
		p.synchronize()
		return nil
	}
	p.next() // skip '}'

	return m
}

func (p *parser) parsePrimaryExpr() ast.Expr {
	tok := p.cur()
	if tok == nil {
		return nil
	}

	switch tok.Kind {
	case lexer.Int:
		v, _ := strconv.ParseInt(tok.Value, 10, 64)
		p.next()
		return &ast.IntLit{Base: ast.NewBase(p.posOf(tok)), Value: v}

	case lexer.QuestionMark:
		p.next()
		return &ast.UnitLit{Base: ast.NewBase(p.posOf(tok))}

	case lexer.Float:
		v, _ := strconv.ParseFloat(tok.Value, 64)
		p.next()
		return &ast.FloatLit{Base: ast.NewBase(p.posOf(tok)), Value: v}

	case lexer.Ident:
		switch tok.Value {
		case "while":
			return p.parseWhileExpr()
		case "for":
			return p.parseForExpr()
		case "break":
			return p.parseBreakExpr()
		case "return":
			return p.parseReturnExpr()
		case "if":
			return p.parseIfExpr()
		case "true":
			p.next()
			return &ast.BoolLit{Base: ast.NewBase(p.posOf(tok)), Value: true}
		case "false":
			p.next()
			return &ast.BoolLit{Base: ast.NewBase(p.posOf(tok)), Value: false}
		}

		return p.parseIdentifier()

	case lexer.String:
		return p.parseStringLiteral()

	case lexer.LParen:
		firstTok := tok
		p.next()

		expr := p.parseExpr()

		if p.cur() == nil {
			p.errorAt(firstTok, "unterminated parenthesis expression. expected ')' but got EOF")
			p.synchronize()
			return nil
		}

		if p.cur().Kind != lexer.RParen {
			p.errorAt(firstTok, fmt.Sprintf("unterminated parenthesis expression. expected ')' but got '%s'", p.cur().Value))
			p.synchronize()
			return nil
		}

		p.next()

		return expr

	case lexer.Backslash:
		return p.parseFnExpr()

	case lexer.LBracket:
		return p.parseArrayExpr()

	case lexer.LCurly:
		return p.parseMapExpr()

	default:
		p.errorAt(tok, fmt.Sprintf("expected a primary expression but got '%s'", lexer.KindName(tok)))
		p.synchronize()
		return nil
	}
}

func (p *parser) parseArrayExpr() ast.Expr {
	startTok := p.cur()
	p.next() // skip '['

	arr := &ast.ArrayExpr{Base: ast.NewBase(p.posOf(startTok))}

	if p.cur() != nil && p.cur().Kind != lexer.RBracket {
		for {
			item := p.parseExpr()
			if item == nil {
				return nil
			}

			arr.Items = append(arr.Items, item)

			if p.cur() == nil {
				p.errorAt(startTok, "expected ']' but got EOF")
				p.synchronize()
				return nil
			}
			if p.cur().Kind == lexer.RBracket {
				break
			}
			if p.cur().Kind != lexer.Comma {
				p.errorAt(p.cur(), "expected ',' or ']'")
				p.synchronize()
				return nil
			}
			p.next() // skip ','
		}
	}

	if !p.expect(lexer.RBracket) {
		return nil
	}
	p.next() // skip ']'

	return arr
}

func (p *parser) parsePostfixExpr() ast.Expr {
	left := p.parsePrimaryExpr()
	if left == nil {
		return nil
	}

	for p.check(lexer.LBracket) || p.check(lexer.Dot) || p.check(lexer.LParen) {
		switch {
		case p.check(lexer.LBracket):
			bracketTok := p.cur()
			p.next() // skip '['

			index := p.parseExpr()

			if !p.expect(lexer.RBracket) {
				return nil
			}
			p.next() // skip ']'

			left = &ast.SquareExpr{Base: ast.NewBase(p.posOf(bracketTok)), Left: left, Index: index}

		case p.check(lexer.LParen):
			// '(' is a general postfix operator: whatever `left` evaluates
			// to (an Ident, a DotExpr field, an indexed value, a
			// parenthesized fn literal, ...) is the thing being called.
			left = p.parseCallExpr(left)
			if left == nil {
				return nil
			}

		default: // Dot
			dotTok := p.cur()
			p.next() // skip '.'

			if !p.expect(lexer.Ident) {
				return nil
			}

			left = &ast.DotExpr{Base: ast.NewBase(p.posOf(dotTok)), Left: left, Index: p.parseIdentifier()}
		}
	}

	return left
}

func (p *parser) parseFactorialExpr() ast.Expr {
	if p.cur() == nil {
		return nil
	}

	left := p.parsePostfixExpr()
	if left == nil {
		return nil
	}

	for p.cur() != nil && p.cur().Kind == lexer.Exclamation {
		opTok := p.cur()
		p.next()

		left = &ast.UnaryExpr{Base: ast.NewBase(p.posOf(opTok)), Op: ast.FactorialOp, Operand: left}
	}

	return left
}

func (p *parser) parseUnaryExpr() ast.Expr {
	if p.cur() == nil {
		return nil
	}

	firstTok := p.cur()

	if p.cur().Kind == lexer.Minus || p.cur().Kind == lexer.Exclamation {
		opTok := p.cur()
		isNot := opTok.Kind == lexer.Exclamation
		p.next()

		operand := p.parseFactorialExpr()
		if operand == nil {
			opName := "-"
			if isNot {
				opName = "!"
			}
			p.errorAt(firstTok, fmt.Sprintf("missing operand for unary '%s'", opName))
			p.synchronize()
			return nil
		}

		op := ast.MinusOp
		if isNot {
			op = ast.NotOp // a prefix '!' always means "not", never factorial
		}

		return &ast.UnaryExpr{Base: ast.NewBase(p.posOf(opTok)), Op: op, Operand: operand}
	}

	return p.parseFactorialExpr()
}

// parseBinaryLevel is a small helper factoring the repeated
// left-assoc-binary-chain shape shared by power/term/additive/relational/equality.
func (p *parser) parseBinaryLevel(next func() ast.Expr, kinds ...lexer.TokenKind) ast.Expr {
	left := next()
	if left == nil {
		return nil
	}

	for p.cur() != nil && containsKind(kinds, p.cur().Kind) {
		opTok := p.cur()
		op := binaryOpForToken(opTok.Kind)
		p.next()

		right := next()
		if right == nil {
			p.errorAt(opTok, fmt.Sprintf("missing right operand for '%s'", op))
			p.synchronize()
			return nil
		}

		left = &ast.BinaryExpr{Base: ast.NewBase(p.posOf(opTok)), Op: op, Left: left, Right: right}
	}

	return left
}

func containsKind(kinds []lexer.TokenKind, k lexer.TokenKind) bool {
	return slices.Contains(kinds, k)
}

// parsePowerExpr is right-associative, unlike the other binary levels, so it
// isn't expressed via parseBinaryLevel. TODO: I think that's wrong (mathematically speaking).
// But, I'm not gonna fix it right now.
func (p *parser) parsePowerExpr() ast.Expr {
	if p.cur() == nil {
		return nil
	}

	left := p.parseUnaryExpr()
	if left == nil {
		return nil
	}

	if p.cur() != nil && p.cur().Kind == lexer.Pow {
		opTok := p.cur()
		p.next()

		right := p.parsePowerExpr() // recurse into self: right-associative

		if right == nil {
			p.errorAt(opTok, fmt.Sprintf("missing right operand for '%s'", ast.PowOp))
			p.synchronize()
			return nil
		}

		return &ast.BinaryExpr{Base: ast.NewBase(p.posOf(opTok)), Op: ast.PowOp, Left: left, Right: right}
	}

	return left
}

func (p *parser) parseTermExpr() ast.Expr {
	return p.parseBinaryLevel(p.parsePowerExpr, lexer.Times, lexer.Divide, lexer.Mod)
}

func (p *parser) parseAdditiveExpr() ast.Expr {
	return p.parseBinaryLevel(p.parseTermExpr, lexer.Plus, lexer.Minus)
}

func (p *parser) parseRelationalExpr() ast.Expr {
	return p.parseBinaryLevel(p.parseAdditiveExpr, lexer.Lt, lexer.Lte, lexer.Gt, lexer.Gte)
}

func (p *parser) parseEqualityExpr() ast.Expr {
	return p.parseBinaryLevel(p.parseRelationalExpr, lexer.Equal, lexer.NotEqual)
}

func (p *parser) parseLogicalExpr() ast.Expr {
	if p.cur() == nil {
		return nil
	}

	left := p.parseEqualityExpr()

	for p.cur() != nil {
		var and bool
		switch {
		case isLogicalAnd(p.cur()):
			and = true
		case isLogicalOr(p.cur()):
			and = false
		default:
			return left
		}

		opTok := p.cur()
		p.next()

		right := p.parseEqualityExpr()
		if right == nil {
			name := "or"
			if and {
				name = "and"
			}
			p.errorAt(opTok, fmt.Sprintf("missing right operand for '%s' operator", name))
			p.synchronize()
			return nil
		}

		op := ast.OrOp
		if and {
			op = ast.AndOp
		}

		left = &ast.BinaryExpr{Base: ast.NewBase(p.posOf(opTok)), Op: op, Left: left, Right: right}
	}

	return left
}

func acceptableAssignTarget(e ast.Expr) bool {
	switch e.(type) {
	case *ast.Ident, *ast.SquareExpr, *ast.DotExpr:
		return true
	}
	return false
}

func (p *parser) parseAssignmentExpr() ast.Expr {
	if p.cur() == nil {
		return nil
	}

	// `const` is a soft keyword, like if/while/else: it is only a declaration
	// when an identifier follows it, so an existing variable named `const`
	// keeps working in every other position.
	var constTok *lexer.Token
	if isKeyword(p.cur(), "const") && p.checkAhead(lexer.Ident) {
		constTok = p.cur()
		p.next()
	}

	firstTok := p.cur()

	left := p.parseLogicalExpr()

	if left != nil && acceptableAssignTarget(left) &&
		(p.check(lexer.Assign) || p.check(lexer.PlusEqual) || p.check(lexer.MinusEqual)) {

		opTok := p.cur()

		var op ast.AssignOp
		var missingMsg string

		switch opTok.Kind {
		case lexer.Assign:
			op = ast.Assign
			missingMsg = fmt.Sprintf("missing right operand for assignment '%s = ...'", firstTok.Value)
		case lexer.PlusEqual:
			op = ast.AddAssign
			missingMsg = fmt.Sprintf("missing right operand for addition assignment '%s += ...'", firstTok.Value)
		case lexer.MinusEqual:
			op = ast.SubAssign
			missingMsg = fmt.Sprintf("missing right operand for subtraction assignment '%s -= ...'", firstTok.Value)
		}

		if constTok != nil {
			if op != ast.Assign {
				p.errorAt(opTok, fmt.Sprintf("a constant must be initialized with '=', not '%s'", op))
				p.synchronize()
				return nil
			}

			if _, isIdent := left.(*ast.Ident); !isIdent {
				p.errorAt(constTok, "only a plain identifier can be declared 'const'")
				p.synchronize()
				return nil
			}
		}

		p.next() // skip '=', '+=', '-='

		right := p.parseExpr()
		if right == nil {
			p.errorAt(firstTok, missingMsg)
			p.synchronize()
			return nil
		}

		return &ast.AssignExpr{
			Base:  ast.NewBase(p.posOf(opTok)),
			Op:    op,
			Const: constTok != nil,
			Left:  left,
			Right: right,
		}
	}

	if constTok != nil {
		p.errorAt(constTok, "a constant must be given a value: 'const name = ...'")
		p.synchronize()
		return nil
	}

	return left
}

// parseExpr is the entry point for any expression.
func (p *parser) parseExpr() ast.Expr {
	return p.parseAssignmentExpr()
}

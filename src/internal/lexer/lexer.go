package lexer

import "fmt"

type Error struct {
	Filename  string
	Line, Col int
	Message   string
}

func (e Error) Error() string {
	if e.Filename != "" {
		return fmt.Sprintf("%s:%d:%d: \033[1;31merror\033[0m %s", e.Filename, e.Line, e.Col, e.Message)
	}

	return fmt.Sprintf("%d:%d: \033[1;31merror\033[0m %s", e.Line, e.Col, e.Message)
}

type Lexer struct {
	filename string
	content  string

	cursor    int
	bot       int
	line, col int

	// position at the start of the token currently being scanned
	tokLine, tokCol int

	tokens []Token
	Errors []Error
}

var singleTokensMapping = map[byte]TokenKind{
	'*':  Times,
	'/':  Divide,
	'(':  LParen,
	')':  RParen,
	'[':  LBracket,
	']':  RBracket,
	'{':  LCurly,
	'}':  RCurly,
	'+':  Plus,
	'-':  Minus,
	'%':  Mod,
	'^':  Pow,
	'!':  Exclamation,
	';':  Semi,
	':':  Colon,
	',':  Comma,
	'?':  QuestionMark,
	'.':  Dot,
	'\\': Backslash,
}

func New(filename, content string) *Lexer {
	return &Lexer{
		filename: filename,
		content:  content,
		line:     1,
		col:      1,
	}
}

func isIdentifierStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func keepBeingIdentifier(c byte) bool {
	return isIdentifierStart(c) || (c >= '0' && c <= '9')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func (l *Lexer) chr() byte {
	if l.cursor < len(l.content) {
		return l.content[l.cursor]
	}

	return 0
}

func (l *Lexer) nchr() byte {
	if l.cursor+1 < len(l.content) {
		return l.content[l.cursor+1]
	}

	return 0
}

func (l *Lexer) updateBot() {
	l.bot = l.cursor
}

func (l *Lexer) advance() {
	if l.cursor < len(l.content) {
		if l.chr() == '\n' {
			l.col = 1
			l.line++
		} else {
			l.col++
		}

		l.cursor++
	}
}

func (l *Lexer) trimWhitespaceAndLineBreaks() {
	for l.chr() == ' ' || l.chr() == '\t' || l.chr() == '\r' || l.chr() == '\n' {
		l.advance()
	}
}

func (l *Lexer) errorf(format string, args ...any) {
	l.Errors = append(l.Errors, Error{
		Filename: l.filename,
		Line:     l.line,
		Col:      l.col,
		Message:  fmt.Sprintf(format, args...),
	})

	l.advance()
}

func (l *Lexer) saveToken(kind TokenKind) {
	l.tokens = append(l.tokens, Token{
		Kind:  kind,
		Value: l.content[l.bot:l.cursor],
		Loc:   Location{Line: l.tokLine, Col: l.tokCol},
	})
}

func (l *Lexer) saveTokenRange(kind TokenKind, bot, cursor int) {
	l.tokens = append(l.tokens, Token{
		Kind:  kind,
		Value: l.content[bot:cursor],
		Loc:   Location{Line: l.tokLine, Col: l.tokCol},
	})
}

func (l *Lexer) tokenizeN(n int, kind TokenKind) {
	for range n {
		l.advance()
	}

	l.saveToken(kind)
}

func (l *Lexer) tokenizeNumber() {
	digits := 0

	for isDigit(l.chr()) {
		l.advance()
		digits++
	}

	kind := Int

	if l.chr() == '.' {
		kind = Float

		if digits == 0 {
			l.invalidFloatingNumberError()
			return
		}

		if !isDigit(l.nchr()) {
			l.invalidFloatingNumberError()
			return
		}

		l.advance()

		for isDigit(l.chr()) {
			l.advance()
		}
	}

	l.saveToken(kind)
}

func (l *Lexer) invalidFloatingNumberError() {
	end := min(l.cursor+1, len(l.content))

	l.errorf("invalid floating number %s", l.content[l.bot:end])
}

func (l *Lexer) tokenizeString() {
	l.advance() // skip opening '

	for l.chr() != 0 && l.chr() != '\'' {
		if l.chr() == '\n' {
			l.errorf("you cannot have line breaks inside literal strings")
		}

		if l.chr() == '\\' {
			switch l.nchr() {
			case '\\', '\'', 'n', 'r':
				l.advance()
			default:
				l.errorf("invalid scaping sequence '\\%c'", l.nchr())
			}
		}

		l.advance()
	}

	if l.chr() != '\'' {
		l.errorf("unterminated string literal")
		return
	}

	l.advance() // skip closing '

	l.saveTokenRange(String, l.bot+1, l.cursor-1)
}

func (l *Lexer) tokenizeSingle() {
	if v, ok := singleTokensMapping[l.chr()]; ok {
		l.advance()
		l.saveToken(v)
	} else {
		l.errorf("unrecognized character %c", l.chr())
	}
}

func (l *Lexer) tokenizeIdentifier() {
	for keepBeingIdentifier(l.chr()) {
		l.advance()
	}

	l.saveToken(Ident)
}

func (l *Lexer) skipComment() {
	for l.chr() != 0 && l.chr() != '\n' {
		l.advance()
	}

	l.advance()
}

func (l *Lexer) Tokenize() []Token {
	for l.chr() != 0 {
		l.trimWhitespaceAndLineBreaks()
		l.updateBot()

		l.tokLine = l.line
		l.tokCol = l.col

		switch c := l.chr(); {
		case isDigit(c):
			l.tokenizeNumber()
		case c == '\'':
			l.tokenizeString()
		case c == '-':
			switch l.nchr() {
			case '=':
				l.tokenizeN(2, MinusEqual)
			case '>':
				l.tokenizeN(2, Arrow)
			default:
				l.tokenizeSingle()
			}
		case c == '+':
			if l.nchr() == '=' {
				l.tokenizeN(2, PlusEqual)
			} else {
				l.tokenizeSingle()
			}
		case c == '!':
			if l.nchr() == '=' {
				l.tokenizeN(2, NotEqual)
			} else {
				l.tokenizeSingle()
			}
		case c == '=':
			if l.nchr() == '=' {
				l.tokenizeN(2, Equal)
			} else {
				l.tokenizeN(1, Assign)
			}
		case c == '<':
			switch l.nchr() {
			case '=':
				l.tokenizeN(2, Lte)
			case '<':
				l.tokenizeN(2, Shl)
			default:
				l.tokenizeN(1, Lt)
			}
		case c == '>':
			switch l.nchr() {
			case '=':
				l.tokenizeN(2, Gte)
			case '>':
				l.tokenizeN(2, Shr)
			default:
				l.tokenizeN(1, Gt)
			}
		case c == '&':
			l.tokenizeN(1, Amp)
		case c == '|':
			l.tokenizeN(1, Pipe)
		case c == '~':
			l.tokenizeN(1, Tilde)
		case c == '#':
			l.skipComment()
		case c == 0:
			// EOF, nothing to do
		default:
			if isIdentifierStart(c) {
				l.tokenizeIdentifier()
			} else {
				l.tokenizeSingle()
			}
		}
	}

	if len(l.Errors) > 0 {
		return nil
	}

	return l.tokens
}

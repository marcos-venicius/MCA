package lexer

import (
	"testing"
)

func kinds(toks []Token) []TokenKind {
	ks := make([]TokenKind, len(toks))
	for i, t := range toks {
		ks[i] = t.Kind
	}
	return ks
}

func eqKinds(t *testing.T, src string, want []TokenKind) []Token {
	t.Helper()

	l := New("test.mca", src)
	toks := l.Tokenize()

	if len(l.Errors) > 0 {
		t.Fatalf("tokenize(%q): unexpected errors: %v", src, l.Errors)
	}

	got := kinds(toks)

	if len(got) != len(want) {
		t.Fatalf("tokenize(%q): got %d tokens %v, want %d %v", src, len(got), got, len(want), want)
	}

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("tokenize(%q): token %d = %v, want %v (full got=%v want=%v)", src, i, got[i], want[i], got, want)
		}
	}

	return toks
}

func TestNumbers(t *testing.T) {
	toks := eqKinds(t, "42", []TokenKind{Int})
	if toks[0].Value != "42" {
		t.Fatalf("want value 42, got %q", toks[0].Value)
	}

	toks = eqKinds(t, "3.14", []TokenKind{Float})
	if toks[0].Value != "3.14" {
		t.Fatalf("want value 3.14, got %q", toks[0].Value)
	}
}

func TestInvalidFloat(t *testing.T) {
	l := New("test.mca", "3.")
	toks := l.Tokenize()

	if toks != nil {
		t.Fatalf("expected nil tokens on lex error, got %v", toks)
	}

	if len(l.Errors) == 0 {
		t.Fatalf("expected an error for '3.'")
	}
}

func TestStringLiteralAndEscapes(t *testing.T) {
	toks := eqKinds(t, `'hello world'`, []TokenKind{String})
	if toks[0].Value != "hello world" {
		t.Fatalf("want %q got %q", "hello world", toks[0].Value)
	}

	toks = eqKinds(t, `'a\nb\'c\\d'`, []TokenKind{String})
	if toks[0].Value != `a\nb\'c\\d` {
		t.Fatalf("raw escape value mismatch: got %q", toks[0].Value)
	}

	// the lexer keeps escapes raw; \r is a valid escape and must not error.
	toks = eqKinds(t, `'a\rb'`, []TokenKind{String})
	if toks[0].Value != `a\rb` {
		t.Fatalf("raw \\r escape value mismatch: got %q", toks[0].Value)
	}

	// \r sitting next to the other escapes stays raw too.
	toks = eqKinds(t, `'\r\n'`, []TokenKind{String})
	if toks[0].Value != `\r\n` {
		t.Fatalf("raw \\r\\n escape value mismatch: got %q", toks[0].Value)
	}
}

func TestInvalidEscapeIsError(t *testing.T) {
	l := New("test.mca", `'a\tb'`)
	toks := l.Tokenize()

	if toks != nil {
		t.Fatalf("expected nil tokens on lex error")
	}

	if len(l.Errors) == 0 {
		t.Fatalf("expected an error for invalid escape \\t")
	}
}

func TestUnterminatedString(t *testing.T) {
	l := New("test.mca", `'abc`)
	l.Tokenize()

	if len(l.Errors) == 0 {
		t.Fatalf("expected unterminated string error")
	}
}

func TestNoLineBreakInString(t *testing.T) {
	l := New("test.mca", "'ab\ncd'")
	l.Tokenize()

	if len(l.Errors) == 0 {
		t.Fatalf("expected line-break-in-string error")
	}
}

func TestIdentifiersAreNotKeywords(t *testing.T) {
	// while/if/and/etc are plain identifiers at the lexer level; the parser
	// is the one that gives them meaning positionally.
	eqKinds(t, "while", []TokenKind{Ident})
	eqKinds(t, "and", []TokenKind{Ident})
	eqKinds(t, "_foo123", []TokenKind{Ident})
}

func TestOperatorsAndMultiChar(t *testing.T) {
	eqKinds(t, "+ += - -= -> * / % ^ ! != = == < <= > >= << >> & | ~", []TokenKind{
		Plus, PlusEqual, Minus, MinusEqual, Arrow, Times, Divide, Mod, Pow,
		Exclamation, NotEqual, Assign, Equal, Lt, Lte, Gt, Gte, Shl, Shr,
		Amp, Pipe, Tilde,
	})
}

func TestShiftOperators(t *testing.T) {
	eqKinds(t, "1 << 2 >> 3", []TokenKind{Int, Shl, Int, Shr, Int})

	// maximal munch: '<<'/'>>' win over '<'/'>', without stealing '<='/'>='
	eqKinds(t, "a < b << c <= d", []TokenKind{Ident, Lt, Ident, Shl, Ident, Lte, Ident})
	eqKinds(t, "a > b >> c >= d", []TokenKind{Ident, Gt, Ident, Shr, Ident, Gte, Ident})
}

func TestBitwiseOperators(t *testing.T) {
	eqKinds(t, "a & b | c ~ d", []TokenKind{Ident, Amp, Ident, Pipe, Ident, Tilde, Ident})
	eqKinds(t, "~a", []TokenKind{Tilde, Ident}) // same token whether unary or binary
}

func TestSymbols(t *testing.T) {
	eqKinds(t, "? . : \\ ( ) { } [ ] ; ,", []TokenKind{
		QuestionMark, Dot, Colon, Backslash, LParen, RParen, LCurly, RCurly,
		LBracket, RBracket, Semi, Comma,
	})
}

func TestComment(t *testing.T) {
	eqKinds(t, "1 # this is a comment\n2", []TokenKind{Int, Int})
}

func TestWhitespaceAndNewlinesInsignificant(t *testing.T) {
	// newlines carry no statement-terminating meaning at the lexer level
	eqKinds(t, "1\n+\n2", []TokenKind{Int, Plus, Int})
}

func TestLambdaSyntaxTokens(t *testing.T) {
	eqKinds(t, `\(x) -> x + 1`, []TokenKind{
		Backslash, LParen, Ident, RParen, Arrow, Ident, Plus, Int,
	})
}

func TestFactorialAndNotAreSameTokenKind(t *testing.T) {
	// disambiguation between "not" and "factorial" happens in the parser,
	// not the lexer -- both produce Exclamation.
	eqKinds(t, "!true", []TokenKind{Exclamation, Ident})
	eqKinds(t, "5!", []TokenKind{Int, Exclamation})
}

func TestUnrecognizedSymbol(t *testing.T) {
	l := New("test.mca", "1 @ 2")
	l.Tokenize()

	if len(l.Errors) == 0 {
		t.Fatalf("expected unrecognized-symbol error for '@'")
	}
}

func TestLocationTracking(t *testing.T) {
	l := New("test.mca", "1\n  22")
	toks := l.Tokenize()

	if toks[0].Loc.Line != 1 || toks[0].Loc.Col != 1 {
		t.Fatalf("first token loc = %+v, want line 1 col 1", toks[0].Loc)
	}

	if toks[1].Loc.Line != 2 || toks[1].Loc.Col != 3 {
		t.Fatalf("second token loc = %+v, want line 2 col 3", toks[1].Loc)
	}
}

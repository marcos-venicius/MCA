package lexer

type TokenKind int

const (
	Int TokenKind = iota
	Float
	String

	Ident

	// binary operators
	Plus
	PlusEqual
	Divide
	Times
	Minus // may be unary too, for negative numbers
	MinusEqual
	Mod
	Pow
	Shl
	Shr
	Amp
	Pipe
	Tilde // binary xor, or unary bitwise not

	// unary operators
	Exclamation

	// '=' operator
	Assign

	// logical operators
	Equal
	NotEqual
	Gt
	Lt
	Gte
	Lte

	// unit literal '?'
	QuestionMark

	// symbols
	Dot
	Colon
	Backslash
	Arrow
	LParen
	RParen
	LCurly
	RCurly
	LBracket
	RBracket
	Semi
	Comma
)

var tokenKindDisplayNameMapping = map[TokenKind]string{
	Int:          "int",
	Float:        "float",
	String:       "string",
	Ident:        "identifier",
	Plus:         "+",
	PlusEqual:    "+=",
	Divide:       "/",
	Times:        "*",
	Mod:          "%",
	Pow:          "^",
	Shl:          "<<",
	Shr:          ">>",
	Amp:          "&",
	Pipe:         "|",
	Tilde:        "~",
	Minus:        "-",
	MinusEqual:   "-=",
	Exclamation:  "!",
	QuestionMark: "?",
	Assign:       "=",
	Equal:        "==",
	NotEqual:     "!=",
	Gt:           ">",
	Lt:           "<",
	Gte:          ">=",
	Lte:          "<=",
	Colon:        ":",
	Dot:          ".",
	Backslash:    "\\",
	Arrow:        "->",
	LParen:       "(",
	RParen:       ")",
	LBracket:     "[",
	RBracket:     "]",
	LCurly:       "{",
	RCurly:       "}",
	Comma:        ",",
	Semi:         ";",
}

func (k TokenKind) DisplayName() string {
	if name, ok := tokenKindDisplayNameMapping[k]; ok {
		return name
	}

	panic("token.DisplayName: unhandled TokenKind")
}

// TODO: don't we need to have file name here?
type Location struct {
	Line, Col int
}

type Token struct {
	Kind  TokenKind
	Value string
	Loc   Location
}

func KindName(tok *Token) string {
	if tok == nil {
		return "eof"
	}

	return tok.Kind.DisplayName()
}

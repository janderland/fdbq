package parser

import (
	"bufio"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const (
	runesWhitespace = "\t "
	runesNewline    = "\n\r"
)

type TokenKind int

const (
	TokenKindInvalid TokenKind = iota
	TokenKindKVSep
	TokenKindDirSep
	TokenKindTupStart
	TokenKindTupEnd
	TokenKindTupSep
	TokenKindVarStart
	TokenKindVarEnd
	TokenKindVarSep
	TokenKindStrMark
	TokenKindWhitespace
	TokenKindNewLine
	TokenKindOther
	TokenKindEnd
)

var specialTokensByRune = map[rune]TokenKind{
	KVSep:    TokenKindKVSep,
	DirSep:   TokenKindDirSep,
	TupStart: TokenKindTupStart,
	TupEnd:   TokenKindTupEnd,
	TupSep:   TokenKindTupSep,
	VarStart: TokenKindVarStart,
	VarEnd:   TokenKindVarEnd,
	VarSep:   TokenKindVarSep,
	StrMark:  TokenKindStrMark,
}

type state int

const (
	stateWhitespace state = iota
	stateNewline
	stateDirPart
	stateString
	stateOther
)

var primaryKindByState = map[state]TokenKind{
	stateWhitespace: TokenKindWhitespace,
	stateNewline:    TokenKindNewLine,
	stateDirPart:    TokenKindOther,
	stateString:     TokenKindOther,
	stateOther:      TokenKindOther,
}

type Scanner struct {
	reader *bufio.Reader
	token  strings.Builder
	state  state
}

func New(rd io.Reader) Scanner {
	return Scanner{reader: bufio.NewReader(rd)}
}

func (x *Scanner) Token() string {
	return x.token.String()
}

func (x *Scanner) Scan() (kind TokenKind, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				kind = TokenKindInvalid
				err = e
				return
			}
			panic(r)
		}
	}()

	x.token.Reset()

	for {
		r, eof := x.read()
		if eof {
			if x.token.Len() == 0 {
				return TokenKindEnd, nil
			}
			return primaryKindByState[x.state], nil
		}

		if kind, ok := specialTokensByRune[r]; ok {
			if x.token.Len() > 0 {
				x.unread()
				return primaryKindByState[x.state], nil
			}

			switch r {
			case DirSep:
				switch x.state {
				case stateString:
					break
				default:
					x.state = stateDirPart
				}

			case StrMark:
				switch x.state {
				case stateString:
					x.state = stateWhitespace
				default:
					x.state = stateString
				}

			default:
				switch x.state {
				case stateString:
					break
				default:
					x.state = stateWhitespace
				}
			}

			x.append(r)
			return kind, nil
		}

		if strings.ContainsRune(runesWhitespace, r) {
			switch x.state {
			case stateOther:
				x.unread()
				kind := primaryKindByState[x.state]
				x.state = stateWhitespace
				return kind, nil

			default:
				x.append(r)
				continue
			}
		}

		if strings.ContainsRune(runesNewline, r) {
			switch x.state {
			case stateWhitespace:
				x.state = stateNewline
				x.append(r)
				continue

			case stateOther:
				x.unread()
				kind := primaryKindByState[x.state]
				x.state = stateNewline
				return kind, nil

			default:
				x.append(r)
				continue
			}
		}

		switch x.state {
		case stateWhitespace, stateNewline:
			if x.token.Len() == 0 {
				x.state = stateOther
				x.append(r)
				continue
			}
			x.unread()
			kind := primaryKindByState[x.state]
			x.state = stateOther
			return kind, nil

		default:
			x.append(r)
			continue
		}
	}
}

func (x *Scanner) append(r rune) {
	_, err := x.token.WriteRune(r)
	if err != nil {
		panic(errors.Wrap(err, "failed to append rune"))
	}
}

func (x *Scanner) read() (rune, bool) {
	r, _, err := x.reader.ReadRune()
	if err == io.EOF {
		return 0, true
	}
	if err != nil {
		panic(errors.Wrap(err, "failed to read rune"))
	}
	return r, false
}

func (x *Scanner) unread() {
	err := x.reader.UnreadRune()
	if err != nil {
		panic(errors.Wrap(err, "failed to unread rune"))
	}
}

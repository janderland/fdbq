// Package scanner tokenizes query strings.
package scanner

import (
	"bufio"
	"io"
	"strings"
	"unicode"

	"github.com/pkg/errors"

	"github.com/janderland/fql/parser/internal"
)

// TokenKind represents the kind of token read during a call to Scanner.Scan.
type TokenKind int

const (
	// TokenKindUnassigned is used to identify a TokenKind variable
	// which hasn't been assigned to yet. For this purpose, it must
	// be the zero-value of its type.
	TokenKindUnassigned TokenKind = iota

	// TokenKindWhitespace identifies a token which only contains
	// runes found in the runesWhitespace constant.
	TokenKindWhitespace

	// TokenKindNewline identifies a token which only contains
	// runes found in the runesWhitespace or runesNewline
	// constants.
	TokenKindNewline

	// TokenKindEscape identifies a 2-rune token which always
	// starts with the Escape rune.
	TokenKindEscape

	// TokenKindOther identifies all other possible tokens which are
	// not identified by the given TokenKind constants. This kind of
	// token is used to represent directory names, value types, and
	// data elements (numbers, strings, UUIDs, etc...).
	// TODO: Is there a better name than TokenKindOther?
	TokenKindOther

	// TokenKindEnd is returned from Scanner.Scan when the wrapped
	// io.Reader has been read to completion.
	TokenKindEnd

	// TokenKindKeyValSep identifies a token equal to KeyValSep.
	TokenKindKeyValSep

	// TokenKindDirSep identifies a token equal to DirSep.
	TokenKindDirSep

	// TokenKindTupStart identifies a token equal to TupStart.
	TokenKindTupStart

	// TokenKindTupEnd identifies a token equal to TupEnd.
	TokenKindTupEnd

	// TokenKindTupSep identifies a token equal to TupSep.
	TokenKindTupSep

	// TokenKindVarStart identifies a token equal to VarStart.
	TokenKindVarStart

	// TokenKindVarEnd identifies a token equal to VarEnd.
	TokenKindVarEnd

	// TokenKindVarSep identifies a token equal to VarSep.
	TokenKindVarSep

	// TokenKindStrMark identifies a token equal to StrMark.
	TokenKindStrMark

	// TokenKindReserved identifies a single-rune token which
	// isn't currently used by the language but reserved for
	// later use.
	TokenKindReserved
)

type state int

const (
	// stateUnassigned is used to identify a state variable
	// which hasn't been assigned to yet. For this purpose, it must
	// be the zero-value of its type.
	stateUnassigned state = iota

	// stateWhitespace is the root state of the scanner. The scanner
	// remains in this state as long as the current token only contains runes
	// found in the runesWhitespace constant.
	stateWhitespace

	// stateNewline follows stateWhitespace if any of the runes
	// found in the runesNewline constant are encountered. The scanner remains
	// in this state as long os the current token only contains runes found in
	// the runesWhitespace or runesNewline constants.
	stateNewline

	// stateString follows any state, save for stateDirPart, if a
	// StrMark rune is encountered. During this state, the scanner doesn't
	// separate whitespace or newlines into separate tokens. The scanner remains
	// in this state until another StrMark is encountered.
	stateString

	// stateOther follows stateWhitespace or stateNewline
	// if a non-significant character is encountered. The scanner remains in
	// this state until a significant rune is encountered. Significant runes
	// include any of the single-rune constants from special.go, or any of
	// the runes in the constants runesWhitespace and runesNewline.
	stateOther
)

// singleRuneKind returns a TokenKind which identifies a token equal
// to the given rune, if such a TokenKind exists. There are a subset
// of TokenKind whose tokens are a single rune. One of these is
// returned. If no TokenKind exists for the given rune then
// TokenKindUnassigned is returned.
func singleRuneKind(r rune) TokenKind {
	switch r {
	case internal.KeyValSep:
		return TokenKindKeyValSep
	case internal.DirSep:
		return TokenKindDirSep
	case internal.TupStart:
		return TokenKindTupStart
	case internal.TupEnd:
		return TokenKindTupEnd
	case internal.TupSep:
		return TokenKindTupSep
	case internal.VarStart:
		return TokenKindVarStart
	case internal.VarEnd:
		return TokenKindVarEnd
	case internal.VarSep:
		return TokenKindVarSep
	case internal.StrMark:
		return TokenKindStrMark

	// While the following aren't currently used by
	// the language, the following symbols have been
	// reserved for future use.
	case internal.Exclamation:
		return TokenKindReserved
	case internal.Hashtag:
		return TokenKindReserved
	case internal.Dollar:
		return TokenKindReserved
	case internal.Percent:
		return TokenKindReserved
	case internal.Ampersand:
		return TokenKindReserved
	case internal.CurlyStart:
		return TokenKindReserved
	case internal.CurlyEnd:
		return TokenKindReserved
	case internal.Star:
		return TokenKindReserved
	case internal.Plus:
		return TokenKindReserved
	case internal.Colon:
		return TokenKindReserved
	case internal.Semicolon:
		return TokenKindReserved
	case internal.Question:
		return TokenKindReserved
	case internal.At:
		return TokenKindReserved
	case internal.BraceStart:
		return TokenKindReserved
	case internal.BraceEnd:
		return TokenKindReserved
	case internal.Caret:
		return TokenKindReserved
	case internal.Grave:
		return TokenKindReserved
	case internal.Tilde:
		return TokenKindReserved

	default:
		return TokenKindUnassigned
	}
}

// primaryKind maps a state to the TokenKind usually returned
// by Scanner.Scan during said state. If the scanner encounters an escape
// or any of runes accepted by singleRuneKind, then the Scanner.Scan
// method may return a different TokenKind than what this function returns.
func primaryKind(state state) TokenKind {
	switch state {
	case stateWhitespace:
		return TokenKindWhitespace
	case stateNewline:
		return TokenKindNewline
	case stateString:
		return TokenKindOther
	case stateOther:
		return TokenKindOther
	default:
		// Its expected that this panic is recovered in Scanner.Scan.
		err := errors.Errorf("unrecognized scanner state '%v'", state)
		panic(errors.Wrap(err, "failed to get primary kind"))
	}
}

// Scanner splits the bytes from an io.Reader into tokens
// for a Parser. For each call to the Scan method, a token
// worth of bytes is read from the io.Reader and the kind
// of token is returned. After a call to Scan, the Token
// method may be called to obtain the token string.
type Scanner struct {
	source *bufio.Reader
	token  *strings.Builder
	state  state

	// escape is true if the next rune should be
	// included in a TokenKindEscape token.
	escape bool
}

// New creates a Scanner which reads from the given io.Reader.
func New(src io.Reader) Scanner {
	return Scanner{
		source: bufio.NewReader(src),
		token:  &strings.Builder{},
		state:  stateWhitespace,
	}
}

// Token returns token obtained from the wrapped io.Reader by the last call to Scan.
func (x *Scanner) Token() string {
	return x.token.String()
}

// Scan reads a token from the wrapped io.Reader and returns the kind of token read.
// The Scanner is meant to work with any input and should never fail as long as the
// io.Reader doesn't fail. io.Reader errors are wrapped and returned by this method.
func (x *Scanner) Scan() (kind TokenKind, err error) {
	// To keep from obscuring this method's complex logic all the low level
	// functions propagate their errors via a panic which is recovered here.
	// This works out nicely because all errors are fatal in this method. If
	// this method's logic made decisions based on returned errors then this
	// would need to change.
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				kind = TokenKindUnassigned
				err = e
				return
			}
			panic(r)
		}
	}()

	x.token.Reset()

	// This loop reads runes from the wrapped io.Reader, appending
	// them to the Scanner.token string builder. If the most recently
	// read rune should be a part of the next token, the rune is
	// unread and the type of the current token is returned:
	//
	//  if x.token.Len() > 0 {
	//  	x.unread()
	//      return primaryKind(x.state), nil
	//  }
	//
	// This unread is conditional because it shouldn't occur if the
	// current rune is the first one read.
	for {
		r, eof := x.read()
		if eof {
			if x.token.Len() > 0 {
				return primaryKind(x.state), nil
			}
			return TokenKindEnd, nil
		}

		// No matter what state the scanner is in, if the Escape rune
		// is encountered it starts a new 2-rune escape token.
		if x.escape {
			x.escape = false
			x.append(r)
			return TokenKindEscape, nil
		} else if r == internal.Escape {
			if x.token.Len() > 0 {
				x.unread()
				return primaryKind(x.state), nil
			}
			x.escape = true
			x.append(r)
			continue
		}

		// Check if the current rune should start a single-rune token.
		// These kinds of tokens are always equal to a specific rune.
		if kind := singleRuneKind(r); kind != TokenKindUnassigned {
			newState := stateUnassigned

			// TODO: Simplify this branch.
			// This branch used handle two special states,
			// strings & directory elements. Directory
			// elements are no longer a special state, so
			// this branch could maybe be simplified.

			switch r {
			case internal.StrMark:
				switch x.state {
				case stateString:
					newState = stateWhitespace
				default:
					newState = stateString
				}

			default:
				switch x.state {
				case stateString:
					break
				default:
					newState = stateWhitespace
				}
			}

			if newState != stateUnassigned {
				if x.token.Len() > 0 {
					x.unread()
					return primaryKind(x.state), nil
				}
				x.state = newState
			}

			x.append(r)
			return kind, nil
		}

		// Check if the current rune should start a new
		// TokenKindWhitespace token.
		if strings.ContainsRune(internal.Whitespace, r) {
			switch x.state {
			case stateOther:
				x.unread()
				kind := primaryKind(x.state)
				x.state = stateWhitespace
				return kind, nil

			default:
				x.append(r)
				continue
			}
		}

		// Check if the current rune should either start a
		// TokenKindNewline token or promote the current
		// TokenKindWhitespace token into a TokenKindNewline
		// token.
		if strings.ContainsRune(internal.Newline, r) {
			switch x.state {
			case stateWhitespace:
				x.state = stateNewline
				x.append(r)
				continue

			case stateOther:
				x.unread()
				kind := primaryKind(x.state)
				x.state = stateNewline
				return kind, nil

			default:
				x.append(r)
				continue
			}
		}

		// If the current rune didn't match any of the above
		// checks, then it should start a TokenKindOther token.
		switch x.state {
		case stateWhitespace, stateNewline:
			if x.token.Len() == 0 {
				x.state = stateOther
				x.append(r)
				continue
			}
			x.unread()
			kind := primaryKind(x.state)
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
	r, _, err := x.source.ReadRune()
	if err == io.EOF {
		return 0, true
	}
	if err != nil {
		panic(errors.Wrap(err, "failed to read rune"))
	}
	if r > unicode.MaxASCII {
		panic(errors.Errorf("read non-ASCII rune with code '%d'", r))
	}
	if !unicode.IsPrint(r) && !strings.ContainsRune(internal.Whitespace+internal.Newline, r) {
		panic(errors.Errorf("read unsupported ASCII rune with code '%d'", r))
	}
	return r, false
}

func (x *Scanner) unread() {
	err := x.source.UnreadRune()
	if err != nil {
		panic(errors.Wrap(err, "failed to unread rune"))
	}
}

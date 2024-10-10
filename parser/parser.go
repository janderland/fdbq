// Package parser converts query strings into key-values.
package parser

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/janderland/fql/keyval"
	"github.com/janderland/fql/parser/internal"
	"github.com/janderland/fql/parser/scanner"
)

type state int

const (
	stateInitial state = iota
	stateDirHead
	stateDirTail
	stateDirVarEnd
	stateTupleHead
	stateTupleTail
	stateSeparator
	stateValue
	stateString
	stateVarHead
	stateVarTail
	stateFinished
)

type stringState int

const (
	stringStateDir stringState = iota
	stringStateTup
	stringStateVal
)

func stateName(state state) string {
	switch state {
	case stateInitial:
		return "Initial"
	case stateDirHead:
		return "DirHead"
	case stateDirTail:
		return "DirTail"
	case stateDirVarEnd:
		return "DirVarEnd"
	case stateTupleHead:
		return "TupleHead"
	case stateTupleTail:
		return "TupleTail"
	case stateSeparator:
		return "Separator"
	case stateValue:
		return "Value"
	case stateString:
		return "String"
	case stateVarHead:
		return "VarHead"
	case stateVarTail:
		return "VarTail"
	case stateFinished:
		return "Finished"
	default:
		return fmt.Sprintf("[unknown parser state %v]", state)
	}
}

func tokenKindName(kind scanner.TokenKind) string {
	switch kind {
	case scanner.TokenKindEscape:
		return "Escape"
	case scanner.TokenKindKeyValSep:
		return "KeyValSep"
	case scanner.TokenKindDirSep:
		return "DirSep"
	case scanner.TokenKindTupStart:
		return "TupStart"
	case scanner.TokenKindTupEnd:
		return "TupEnd"
	case scanner.TokenKindTupSep:
		return "TupSeparator"
	case scanner.TokenKindVarStart:
		return "VarStart"
	case scanner.TokenKindVarEnd:
		return "VarEnd"
	case scanner.TokenKindVarSep:
		return "VarSep"
	case scanner.TokenKindStrMark:
		return "StrMark"
	case scanner.TokenKindWhitespace:
		return "Whitespace"
	case scanner.TokenKindNewline:
		return "Newline"
	case scanner.TokenKindOther:
		return "Other"
	case scanner.TokenKindEnd:
		return "End"
	case scanner.TokenKindReserved:
		return "Reserved"
	default:
		return fmt.Sprintf("[unknown token kind %v]", kind)
	}
}

// Token is a categorized piece of the query string
// returned from [scanner.Scanner].
type Token struct {
	Kind  scanner.TokenKind
	Token string
}

// Error represents a problem encountered during parsing.
// Included with the error is the entire list of tokens
// returned by the [scanner.Scanner] and the index of the
// token which caused the parsing error.
type Error struct {
	// Tokens is the tokens returned from
	// the scanner.Scanner for the string
	// being parsed.
	Tokens []Token

	// Index is the index of the token
	// where the parsing failed.
	Index int

	// Err is the error encountered
	// at the failing token.
	Err error
}

// Error returns a string made up of all the tokens
// with the invalid token marked as such.
func (x *Error) Error() string {
	var msg strings.Builder
	for i, token := range x.Tokens {
		if i+1 == x.Index {
			msg.WriteString(" --> ")
		}
		msg.WriteString(token.Token)
		if i+1 == x.Index {
			msg.WriteString(" <--invalid-token--- ")
		}
	}
	return errors.Wrap(x.Err, msg.String()).Error()
}

// Parser obtains tokens from the given [scanner.Scanner]
// and attempts to parse them into a keyval.Query.
type Parser struct {
	scanner scanner.Scanner
	tokens  []Token
	state   state
}

func New(s scanner.Scanner) Parser {
	return Parser{scanner: s}
}

// Parse consumes all the tokens from the given
// [scanner.Scanner] and either returns a [keyval.Query]
// or the first error encountered during parsing.
func (x *Parser) Parse() (keyval.Query, error) {
	var (
		kv  internal.KeyValBuilder
		tup internal.TupBuilder

		// TODO: Work into the state machine?
		// If true, when internal.TupBuilder ends its
		// root tuple, the tuple is copied into the query's
		// value. Otherwise, it's copied into the key.
		valTup bool

		// TODO: Work into the state machine?
		// If true, stateVarHead & stateVarTail are building
		// a variable for use as a value. Otherwise, the
		// variable is for use in a tuple.
		valVar bool

		// TODO: Work into the state machine?
		// If < 0 then the string is a directory part.
		// If == 0 then the string is in a tuple.
		// If > 0 then the string is for a value.
		stringState stringState
	)

	for {
		kind, err := x.scanner.Scan()
		if err != nil {
			return nil, err
		}

		// We make sure to add the token to our running
		// list before handling it below. The withTokens
		// method assumes the last token added is the
		// problematic one.
		token := x.scanner.Token()
		x.tokens = append(x.tokens, Token{
			Kind:  kind,
			Token: token,
		})

		switch x.state {
		// The Parser should be at stateInitial when it begins
		// parsing a query. Because all queries begin with a
		// TokenKindDirSep, this is the only accepted token.
		case stateInitial:
			switch kind {
			case scanner.TokenKindDirSep:
				x.state = stateDirHead

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// During stateDirHead, the Parser creates a new
		// keyval.DirElement. If the new element is a variable,
		// the Parser transitions to stateDirVarEnd. Otherwise,
		// it transitions to stateDirTail.
		case stateDirHead:
			switch kind {
			case scanner.TokenKindVarStart:
				x.state = stateDirVarEnd
				kv.AppendVarToDirectory()

			case scanner.TokenKindStrMark:
				x.state = stateString
				stringState = stringStateDir
				kv.AppendPartToDirectory("")

			case scanner.TokenKindOther:
				x.state = stateDirTail
				kv.AppendPartToDirectory(token)

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// During stateDirTail the Parser transitions to create
		// a new directory element, start parsing the key's
		// tuple, or finishes the query as a directory query.
		case stateDirTail:
			switch kind {
			case scanner.TokenKindDirSep:
				x.state = stateDirHead

			case scanner.TokenKindTupStart:
				x.state = stateTupleHead
				tup = internal.TupBuilder{}
				valTup = false

			case scanner.TokenKindEnd:
				return kv.Get().Key.Directory, nil

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// stateDirVarEnd ensures that a TokenKindVarEnd
		// follows a TokenKindVarStart which was read
		// during the previous stateDirHead.
		case stateDirVarEnd:
			switch kind {
			case scanner.TokenKindVarEnd:
				x.state = stateDirTail

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// During stateTupleHead, the Parser creates a new
		// keyval.TupElement current tuple, starts a new
		// sub-tuple, or ends the current tuple. Allowing
		// tuples to end in this state lets us parse empty
		// tuples and ignore trailing commas on the final
		// element.
		case stateTupleHead:
			switch kind {
			case scanner.TokenKindTupStart:
				tup.StartSubTuple()

			case scanner.TokenKindTupEnd:
				if tup.EndTuple() {
					if valTup {
						x.state = stateFinished
						kv.SetValue(tup.Get())
						break
					}
					x.state = stateSeparator
					kv.SetKeyTuple(tup.Get())
				}

			case scanner.TokenKindVarStart:
				x.state = stateVarHead
				valVar = false
				tup.Append(keyval.Variable{})

			case scanner.TokenKindStrMark:
				x.state = stateString
				stringState = stringStateTup
				tup.Append(keyval.String(""))

			case scanner.TokenKindWhitespace, scanner.TokenKindNewline:
				break

			case scanner.TokenKindOther:
				x.state = stateTupleTail
				if token == internal.MaybeMore {
					tup.Append(keyval.MaybeMore{})
					break
				}
				data, err := parseData(token)
				if err != nil {
					return nil, x.withTokens(err)
				}
				tup.Append(data)

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// During stateTupleTail, the Parser either transitions state
		// to create a new keyval.TupElement or finishes the current
		// tuple. The tuple being constructed may be a sub-tuple. If
		// the tuple is finished and is not a sub-tuple, the tuple
		// is copied from its build into the query.
		case stateTupleTail:
			switch kind {
			case scanner.TokenKindTupEnd:
				if tup.EndTuple() {
					if valTup {
						x.state = stateFinished
						kv.SetValue(tup.Get())
					} else {
						x.state = stateSeparator
						kv.SetKeyTuple(tup.Get())
					}
				}

			case scanner.TokenKindTupSep:
				x.state = stateTupleHead

			case scanner.TokenKindWhitespace, scanner.TokenKindNewline:
				break

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// stateSeparator occurs after the key's tuple is completed.
		// The Parser then either begins parsing the value or
		// returns the key as the query.
		case stateSeparator:
			switch kind {
			case scanner.TokenKindEnd:
				return kv.Get().Key, nil

			case scanner.TokenKindKeyValSep:
				x.state = stateValue

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// stateValue begins the parsing of the value. The Parser
		// either begins to parse a tuple, begins to parse a
		// variable, or parses the tokens as a raw value.
		case stateValue:
			switch kind {
			case scanner.TokenKindTupStart:
				x.state = stateTupleHead
				valTup = true
				tup = internal.TupBuilder{}

			case scanner.TokenKindVarStart:
				x.state = stateVarHead
				valVar = true
				kv.SetValue(keyval.Variable{})

			case scanner.TokenKindStrMark:
				x.state = stateString
				stringState = stringStateVal
				kv.SetValue(keyval.String(""))

			case scanner.TokenKindOther:
				x.state = stateFinished
				if token == internal.Clear {
					kv.SetValue(keyval.Clear{})
					break
				}
				data, err := parseData(token)
				if err != nil {
					return nil, x.withTokens(err)
				}
				kv.SetValue(data)

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// During stateString, tokens are appended to the
		// string currently being constructed. The string
		// being constructed may be a directory element,
		// tuple element, or a value. If a TokenKindStrMark
		// is encountered, the Parser moves out of this
		// state.
		case stateString:
			switch kind {
			case scanner.TokenKindEnd:
				return nil, x.withTokens(x.tokenErr(kind))

			case scanner.TokenKindStrMark:
				switch stringState {
				case stringStateDir:
					x.state = stateDirTail

				case stringStateTup:
					x.state = stateTupleTail

				case stringStateVal:
					x.state = stateFinished

				default:
					return nil, errors.Errorf("unexpected string state '%v'", stringState)
				}

			default:
				if kind == scanner.TokenKindEscape {
					switch token[1] {
					case internal.Escape, internal.StrMark:
						// Get rid of the leading backslash.
						token = token[1:]

					default:
						return nil, x.withTokens(x.escapeErr(token))
					}
				}

				switch stringState {
				case stringStateDir:
					if err := kv.AppendToLastDirPart(token); err != nil {
						return nil, x.withTokens(errors.Wrap(err, "failed to append to last directory element"))
					}

				case stringStateTup:
					if err := tup.AppendToLastElemStr(token); err != nil {
						return nil, x.withTokens(errors.Wrap(err, "failed to append to last tuple element"))
					}

				case stringStateVal:
					if err := kv.AppendToValueStr(token); err != nil {
						return nil, x.withTokens(errors.Wrap(err, "failed to append to value"))
					}

				default:
					return nil, errors.Errorf("unexpected string state '%v'", stringState)
				}
			}

		// During stateVarHead, the Parser adds a value type
		// to the current keyval.Variable which may be in a
		// tuple or the value.
		case stateVarHead:
			switch kind {
			case scanner.TokenKindVarEnd:
				if valVar {
					x.state = stateFinished
				} else {
					x.state = stateTupleTail
				}

			case scanner.TokenKindOther:
				x.state = stateVarTail
				v, err := parseValueType(token)
				if err != nil {
					return nil, x.withTokens(err)
				}

				if valVar {
					if err := kv.AppendToValueVar(v); err != nil {
						return nil, x.withTokens(errors.Wrap(err, "failed to append to value variable"))
					}
				} else {
					if err := tup.AppendToLastElemVar(v); err != nil {
						return nil, x.withTokens(errors.Wrap(err, "failed to append to last tuple element"))
					}
				}

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// During stateVarTail, the Parser either begins
		// parsing another value type or finishes the
		// current keyval.Variable.
		case stateVarTail:
			switch kind {
			case scanner.TokenKindVarEnd:
				if valVar {
					x.state = stateFinished
				} else {
					x.state = stateTupleTail
				}

			case scanner.TokenKindVarSep:
				x.state = stateVarHead

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		// During stateFinished, the query is finished and
		// the Parser isn't expecting any tokens except
		// for TokenKindWhitespace.
		case stateFinished:
			switch kind {
			case scanner.TokenKindWhitespace:
				break

			case scanner.TokenKindEnd:
				return kv.Get(), nil

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		default:
			return nil, errors.Errorf("unexpected state '%v'", stateName(x.state))
		}
	}
}

// withTokens wraps the given generic error with an Error.
func (x *Parser) withTokens(err error) error {
	out := Error{
		Index: len(x.tokens),
		Err:   err,
	}

	for {
		kind, err := x.scanner.Scan()
		if err != nil {
			return err
		}

		if kind == scanner.TokenKindEnd {
			out.Tokens = x.tokens
			return &out
		}

		x.tokens = append(x.tokens, Token{
			Kind:  kind,
			Token: x.scanner.Token(),
		})
	}
}

func (x *Parser) escapeErr(token string) error {
	return errors.Errorf("unexpected escape '%v' at parser state '%v'", token, stateName(x.state))
}

func (x *Parser) tokenErr(kind scanner.TokenKind) error {
	return errors.Errorf("unexpected '%v' token at parser state '%v'", tokenKindName(kind), stateName(x.state))
}

func parseValueType(token string) (keyval.ValueType, error) {
	for _, v := range keyval.AllTypes() {
		if string(v) == token {
			return v, nil
		}
	}
	return keyval.AnyType, errors.Errorf("unrecognized value type")
}

func parseData(token string) (
	interface {
		keyval.TupElement
		keyval.Value
	},
	error,
) {
	if token == internal.Nil {
		return keyval.Nil{}, nil
	}
	if token == internal.True {
		return keyval.Bool(true), nil
	}
	if token == internal.False {
		return keyval.Bool(false), nil
	}

	if strings.HasPrefix(token, internal.HexStart) {
		data, err := hex.DecodeString(token[len(internal.HexStart):])
		if err != nil {
			return nil, errors.Wrapf(err, "token begins with '%s' but cannot be parsed as a hex string", internal.HexStart)
		}
		return keyval.Bytes(data), nil
	}

	if strings.Count(token, "-") == 4 {
		var uuid keyval.UUID
		_, err := hex.Decode(uuid[:], []byte(strings.ReplaceAll(token, "-", "")))
		if err != nil {
			return nil, errors.Wrap(err, "token contains four '-' but cannot be parsed as a UUID")
		}
		return uuid, nil
	}

	if strings.ContainsRune(token, '.') {
		data, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return nil, errors.Wrap(err, "token contains a '.' but cannot be parsed as a float")
		}
		return keyval.Float(data), nil
	}

	if strings.HasPrefix(token, "-") {
		data, err := strconv.ParseInt(token, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "token starts with a '-' but cannot be parsed as an int")
		}
		return keyval.Int(data), nil
	}

	// We attempt to parse as Int before Uint to mimic the
	// way tuple.Unpack decodes integers: if the value fits
	// within an int then it's parsed as such, regardless
	// of the value's type during formatting.
	i, iErr := strconv.ParseInt(token, 10, 64)
	if iErr == nil {
		return keyval.Int(i), nil
	}
	u, uErr := strconv.ParseUint(token, 10, 64)
	if uErr == nil {
		return keyval.Uint(u), nil
	}
	err := errors.Errorf("while parsing int - %s, while parsing uint - %s", iErr, uErr)
	return nil, errors.Wrap(err, "failed to parse token as int or uint")
}

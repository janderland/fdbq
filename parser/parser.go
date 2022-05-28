package parser

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	q "github.com/janderland/fdbq/keyval"
	"github.com/janderland/fdbq/parser/internal"
	"github.com/janderland/fdbq/parser/scanner"
	"github.com/pkg/errors"
)

type state int

const (
	stateInitial state = iota
	stateDirHead
	stateDirTail
	stateDirVarEnd
	stateTupleHead
	stateTupleTail
	stateTupleVarHead
	stateTupleVarTail
	stateTupleString
	stateSeparator
	stateValue
	stateValueVarHead
	stateValueVarTail
	stateFinished
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
	case stateTupleVarHead:
		return "TupleVarHead"
	case stateTupleVarTail:
		return "TupleVarTail"
	case stateTupleString:
		return "String"
	case stateSeparator:
		return "Separator"
	case stateValue:
		return "Value"
	case stateValueVarHead:
		return "ValueVarHead"
	case stateValueVarTail:
		return "ValueVarTail"
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
	default:
		return fmt.Sprintf("[unknown token kind %v]", kind)
	}
}

type Token struct {
	Kind  scanner.TokenKind
	Token string
}

type Error struct {
	Tokens []Token
	Index  int
	Err    error
}

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

type Parser struct {
	scanner scanner.Scanner
	tokens  []Token
	state   state
}

func New(s scanner.Scanner) Parser {
	return Parser{scanner: s}
}

func (x *Parser) Parse() (q.Query, error) {
	var (
		kv  internal.KeyValBuilder
		tup internal.TupBuilder

		valTup bool
	)

	for {
		kind, err := x.scanner.Scan()
		if err != nil {
			return nil, err
		}

		token := x.scanner.Token()
		x.tokens = append(x.tokens, Token{
			Kind:  kind,
			Token: token,
		})

		switch x.state {
		case stateInitial:
			switch kind {
			case scanner.TokenKindDirSep:
				x.state = stateDirHead

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		case stateDirTail:
			switch kind {
			case scanner.TokenKindDirSep:
				x.state = stateDirHead

			case scanner.TokenKindTupStart:
				x.state = stateTupleHead
				tup = internal.TupBuilder{}
				valTup = false

			case scanner.TokenKindEscape, scanner.TokenKindOther:
				if kind == scanner.TokenKindEscape {
					switch token[1] {
					case internal.DirSep:
					default:
						return nil, x.withTokens(x.escapeErr(token))
					}
				}
				if err := kv.AppendToLastDirPart(token); err != nil {
					return nil, x.withTokens(errors.Wrap(err, "failed to append to last directory part"))
				}

			case scanner.TokenKindEnd:
				return kv.Get().Key.Directory, nil

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		case stateDirVarEnd:
			switch kind {
			case scanner.TokenKindVarEnd:
				x.state = stateDirTail
				kv.AppendVarToDirectory()

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		case stateDirHead:
			switch kind {
			case scanner.TokenKindVarStart:
				x.state = stateDirVarEnd

			case scanner.TokenKindEscape, scanner.TokenKindOther:
				x.state = stateDirTail
				kv.AppendPartToDirectory(token)

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

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
				x.state = stateTupleVarHead
				tup.Append(q.Variable{})

			case scanner.TokenKindStrMark:
				x.state = stateTupleString
				tup.Append(q.String(""))

			case scanner.TokenKindWhitespace, scanner.TokenKindNewline:
				break

			case scanner.TokenKindOther:
				x.state = stateTupleTail
				if token == internal.MaybeMore {
					tup.Append(q.MaybeMore{})
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

		case stateTupleTail:
			switch kind {
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

			case scanner.TokenKindTupSep:
				x.state = stateTupleHead

			case scanner.TokenKindWhitespace, scanner.TokenKindNewline:
				break

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		case stateTupleString:
			if kind == scanner.TokenKindEnd {
				return nil, x.withTokens(x.tokenErr(kind))
			}
			if kind == scanner.TokenKindStrMark {
				x.state = stateTupleTail
				break
			}
			if err := tup.AppendToLastElemStr(token); err != nil {
				return nil, x.withTokens(errors.Wrap(err, "failed to append to last tuple element"))
			}

		case stateTupleVarHead:
			switch kind {
			case scanner.TokenKindVarEnd:
				x.state = stateTupleTail

			case scanner.TokenKindOther:
				x.state = stateTupleVarTail
				v, err := parseValueType(token)
				if err != nil {
					return nil, x.withTokens(err)
				}
				if err := tup.AppendToLastElemVar(v); err != nil {
					return nil, x.withTokens(errors.Wrap(err, "failed to append to last tuple element"))
				}

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		case stateTupleVarTail:
			switch kind {
			case scanner.TokenKindVarEnd:
				x.state = stateTupleTail

			case scanner.TokenKindVarSep:
				x.state = stateTupleVarHead

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		case stateSeparator:
			switch kind {
			case scanner.TokenKindEnd:
				return kv.Get().Key, nil

			case scanner.TokenKindKeyValSep:
				x.state = stateValue

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		case stateValue:
			switch kind {
			case scanner.TokenKindTupStart:
				x.state = stateTupleHead
				tup = internal.TupBuilder{}
				valTup = true

			case scanner.TokenKindVarStart:
				x.state = stateValueVarHead
				kv.SetValue(q.Variable{})

			case scanner.TokenKindOther:
				x.state = stateFinished
				if token == internal.Clear {
					kv.SetValue(q.Clear{})
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

		case stateValueVarHead:
			switch kind {
			case scanner.TokenKindVarEnd:
				x.state = stateFinished

			case scanner.TokenKindOther:
				x.state = stateValueVarTail
				v, err := parseValueType(token)
				if err != nil {
					return nil, x.withTokens(err)
				}
				if err := kv.AppendToValueVar(v); err != nil {
					return nil, x.withTokens(errors.Wrap(err, "failed to append to value variable"))
				}

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

		case stateValueVarTail:
			switch kind {
			case scanner.TokenKindVarEnd:
				x.state = stateFinished

			case scanner.TokenKindVarSep:
				x.state = stateValueVarHead

			default:
				return nil, x.withTokens(x.tokenErr(kind))
			}

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

func parseValueType(token string) (q.ValueType, error) {
	for _, v := range q.AllTypes() {
		if string(v) == token {
			return v, nil
		}
	}
	return q.AnyType, errors.Errorf("unrecognized value type")
}

func parseData(token string) (
	interface {
		q.TupElement
		q.Value
	},
	error,
) {
	if token == internal.Nil {
		return q.Nil{}, nil
	}
	if token == internal.True {
		return q.Bool(true), nil
	}
	if token == internal.False {
		return q.Bool(false), nil
	}

	if strings.HasPrefix(token, internal.HexStart) {
		data, err := hex.DecodeString(token[len(internal.HexStart):])
		if err != nil {
			return nil, err
		}
		return q.Bytes(data), nil
	}

	if strings.Count(token, "-") == 4 {
		var uuid q.UUID
		_, err := hex.Decode(uuid[:], []byte(strings.ReplaceAll(token, "-", "")))
		if err != nil {
			return nil, err
		}
		return uuid, nil
	}

	// We attempt to parse as Int before Uint to mimic the
	// way tuple.Unpack decodes integers: if the value fits
	// within an int then it's parsed a such, regardless
	// of the value's type during formatting.
	i, err := strconv.ParseInt(token, 10, 64)
	if err == nil {
		return q.Int(i), nil
	}
	u, err := strconv.ParseUint(token, 10, 64)
	if err == nil {
		return q.Uint(u), nil
	}

	f, err := strconv.ParseFloat(token, 64)
	if err == nil {
		return q.Float(f), nil
	}

	return nil, errors.New("unrecognized data element")
}

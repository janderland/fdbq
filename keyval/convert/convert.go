// Package convert converts between FQL and FDB types.
package convert

import (
	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/pkg/errors"

	q "github.com/janderland/fql/keyval"
)

// ToStringArray attempts to convert a keyval.Directory to a string
// array. If the keyval.Directory contains non-string elements, an
// error is returned.
func ToStringArray(in q.Directory) ([]string, error) {
	out := make([]string, len(in))

	for i, element := range in {
		switch e := element.(type) {
		case q.String:
			out[i] = string(e)
		default:
			return nil, errors.Errorf("index '%d' has type '%T'", i, e)
		}
	}

	return out, nil
}

// FromStringArray converts a string array into a keyval.Directory.
func FromStringArray(in []string) q.Directory {
	out := make(q.Directory, len(in))

	for i := range in {
		out[i] = q.String(in[i])
	}

	return out
}

// ToFDBTuple converts a keyval.Tuple into a tuple.Tuple. If the
// keyval.Tuple contains a keyval.Variable or keyval.MaybeMore
// then an error is returned.
func ToFDBTuple(in q.Tuple) (tuple.Tuple, error) {
	out := make(tuple.Tuple, len(in))
	var err error

	for i, element := range in {
		var c conversion
		element.TupElement(&c)
		out[i], err = c.out, c.err
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert index %d", i)
		}
	}

	return out, nil
}

// FromFDBTuple converts a tuple.Tuple into a keyval.Tuple.
// This function panics if an invalid tuple.Tuple is provided.
func FromFDBTuple(in tuple.Tuple) q.Tuple {
	out := make(q.Tuple, len(in))

	for i, element := range in {
		switch element := element.(type) {
		case tuple.Tuple:
			out[i] = FromFDBTuple(element)
		default:
			out[i] = FromFDBElement(element)
		}
	}

	return out
}

// FromFDBElement converts a tuple.TupleElement into a keyval.TupElement.
// This function panics if an invalid tuple.TupleElement is provided.
func FromFDBElement(in tuple.TupleElement) q.TupElement {
	switch in := in.(type) {
	case tuple.Tuple:
		return FromFDBTuple(in)

	case []byte:
		return q.Bytes(in)
	case fdb.KeyConvertible:
		return q.Bytes(in.FDBKey())

	case string:
		return q.String(in)

	case int64:
		return q.Int(in)
	case int:
		return q.Int(in)

	case uint64:
		return q.Uint(in)
	case uint:
		return q.Uint(in)

		// TODO: Add support for BigInt.
		/*
			case big.Int:
				return q.BigInt(in)
			case *big.Int:
				return q.BigInt(*in)
		*/

	case float64:
		return q.Float(in)
	case float32:
		return q.Float(in)

	case bool:
		return q.Bool(in)

	case tuple.UUID:
		return q.UUID(in)

	case nil:
		return q.Nil{}

	default:
		panic(errors.Errorf("cannot convert type %T", in))
	}
}

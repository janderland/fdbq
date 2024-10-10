// Package values serializes and deserializes values.
package values

import (
	"encoding/binary"
	"math"

	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/pkg/errors"

	"github.com/janderland/fql/keyval"
	"github.com/janderland/fql/keyval/convert"
)

// UnexpectedValueTypeErr is returned by Unpack if the provided keyval.ValueType is
// not implemented. This error should only occur if a bug is present in the code.
type UnexpectedValueTypeErr struct {
	error
}

// Pack serializes keyval.Value into a bytes string for writing to the DB.
func Pack(val keyval.Value, order binary.ByteOrder) ([]byte, error) {
	if val == nil {
		return nil, errors.New("value cannot be nil")
	}
	s := serialization{order: order}
	val.Value(&s)
	return s.out, s.err
}

// Unpack deserializes keyval.Value from a byte string read from the DB.
func Unpack(val []byte, typ keyval.ValueType, order binary.ByteOrder) (keyval.Value, error) {
	switch typ {
	case keyval.AnyType:
		return keyval.Bytes(val), nil

	case keyval.BoolType:
		if len(val) != 1 {
			return nil, errors.New("not 1 byte")
		}
		if val[0] == 1 {
			return keyval.Bool(true), nil
		}
		return keyval.Bool(false), nil

	case keyval.IntType:
		if len(val) != 8 {
			return nil, errors.New("not 8 bytes")
		}
		return keyval.Int(order.Uint64(val)), nil

	case keyval.UintType:
		if len(val) != 8 {
			return nil, errors.New("not 8 bytes")
		}
		return keyval.Uint(order.Uint64(val)), nil

	case keyval.FloatType:
		if len(val) != 8 {
			return nil, errors.New("not 8 bytes")
		}
		return keyval.Float(math.Float64frombits(order.Uint64(val))), nil

	case keyval.StringType:
		return keyval.String(val), nil

	case keyval.BytesType:
		return keyval.Bytes(val), nil

	case keyval.UUIDType:
		var uuid keyval.UUID
		if n := copy(uuid[:], val); n != 16 {
			return nil, errors.New("not 16 bytes")
		}
		return uuid, nil

	case keyval.TupleType:
		tup, err := tuple.Unpack(val)
		return convert.FromFDBTuple(tup), errors.Wrap(err, "failed to unpack tuple")

	default:
		return nil, UnexpectedValueTypeErr{errors.Errorf("unknown ValueType '%v'", typ)}
	}
}

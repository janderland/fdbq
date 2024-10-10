package internal

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"

	"github.com/janderland/fql/keyval"
	"github.com/janderland/fql/keyval/values"
)

type (
	// ValHandler processes the FDB value byte-strings during a FQL
	// read operation. Depending on the operation, the byte-strings
	// aren't always deserialized.
	ValHandler interface {
		Handle([]byte) (keyval.Value, error)
	}

	// pass is a ValHandler which passes the []byte through
	// as keyval.Bytes. It never returns an error.
	pass struct{}

	// unpack is a ValHandler which attempts to deserialize the given
	// []byte into one of the types specified in the variable. When
	// the first successful deserialization occurs, the resultant
	// value is returned. If the bytes cannot be deserialized to
	// any of the given types and filter=false, then an error is
	// returned. If filter=true, errors are not returned.
	unpack struct {
		variable keyval.Variable
		order    binary.ByteOrder
		filter   bool
	}

	// compare is a ValHandler which compares the given []byte to
	// the packed bytes of the query. If the bytes match, the original
	// query is returned. If the bytes don't match and filter=false,
	// then an error is returned. If filter=true, errors are not
	// returned.
	compare struct {
		query  keyval.Value
		packed []byte
		filter bool
	}
)

func NewValueHandler(query keyval.Value, order binary.ByteOrder, filter bool) (ValHandler, error) {
	if variable, ok := query.(keyval.Variable); ok {
		if len(variable) == 0 {
			return &pass{}, nil
		}
		return &unpack{
			variable: variable,
			order:    order,
			filter:   filter,
		}, nil
	} else {
		packed, err := values.Pack(query, order)
		if err != nil {
			return nil, errors.Wrap(err, "failed to pack query")
		}
		return &compare{
			query:  query,
			packed: packed,
			filter: filter,
		}, nil
	}
}

func (x *pass) Handle(val []byte) (keyval.Value, error) {
	if val == nil {
		return nil, nil
	}
	return keyval.Bytes(val), nil
}

func (x *unpack) Handle(val []byte) (keyval.Value, error) {
	if val == nil {
		return nil, nil
	}
	for _, typ := range x.variable {
		out, err := values.Unpack(val, typ, x.order)
		if err != nil {
			if _, ok := err.(values.UnexpectedValueTypeErr); ok {
				return nil, err
			}
			continue
		}
		return out, nil
	}
	if x.filter {
		return nil, nil
	}
	return nil, errors.New("unexpected value")
}

func (x *compare) Handle(val []byte) (keyval.Value, error) {
	if val == nil {
		return nil, nil
	}
	if bytes.Equal(x.packed, val) {
		return x.query, nil
	}
	if x.filter {
		return nil, nil
	}
	return nil, errors.New("unexpected value")
}

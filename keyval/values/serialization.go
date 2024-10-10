package values

import (
	"encoding/binary"
	"math"

	q "github.com/janderland/fql/keyval"
	"github.com/janderland/fql/keyval/convert"
	"github.com/pkg/errors"
)

type serialization struct {
	order binary.ByteOrder
	out   []byte
	err   error
}

var _ q.ValueOperation = &serialization{}

func (x *serialization) ForTuple(v q.Tuple) {
	tup, err := convert.ToFDBTuple(v)
	if err != nil {
		x.err = errors.Wrap(err, "failed to convert to FDB tuple")
		return
	}
	x.out = tup.Pack()
}

func (x *serialization) ForInt(v q.Int) {
	x.out = make([]byte, 8)
	x.order.PutUint64(x.out, uint64(v))
}

func (x *serialization) ForUint(v q.Uint) {
	x.out = make([]byte, 8)
	x.order.PutUint64(x.out, uint64(v))
}

func (x *serialization) ForBool(v q.Bool) {
	if v {
		x.out = []byte{1}
	} else {
		x.out = []byte{0}
	}
}

func (x *serialization) ForFloat(v q.Float) {
	x.out = make([]byte, 8)
	x.order.PutUint64(x.out, math.Float64bits(float64(v)))
}

func (x *serialization) ForString(v q.String) {
	x.out = []byte(v)
}

func (x *serialization) ForUUID(v q.UUID) {
	x.out = v[:]
}

func (x *serialization) ForBytes(v q.Bytes) {
	x.out = v
}

func (x *serialization) ForNil(_ q.Nil) {}

func (x *serialization) ForVariable(_ q.Variable) {
	x.err = errors.New("cannot serialize a variable")
}

func (x *serialization) ForClear(_ q.Clear) {
	x.err = errors.New("cannot serialize a clear")
}

package convert

import (
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/pkg/errors"

	q "github.com/janderland/fql/keyval"
)

var _ q.TupleOperation = &conversion{}

type conversion struct {
	out tuple.TupleElement
	err error
}

func (x *conversion) ForTuple(in q.Tuple) {
	x.out, x.err = ToFDBTuple(in)
}

func (x *conversion) ForNil(q.Nil) {}

func (x *conversion) ForInt(in q.Int) {
	x.out = int64(in)
}

func (x *conversion) ForUint(in q.Uint) {
	x.out = uint64(in)
}

func (x *conversion) ForBool(in q.Bool) {
	x.out = bool(in)
}

func (x *conversion) ForFloat(in q.Float) {
	x.out = float64(in)
}

// TODO: Add support for BigInt.
/*
func (x *conversion) ForBigInt(in q.BigInt) {
	x.out = big.Int(in)
}
*/

func (x *conversion) ForString(in q.String) {
	x.out = string(in)
}

func (x *conversion) ForUUID(in q.UUID) {
	x.out = tuple.UUID(in)
}

func (x *conversion) ForBytes(in q.Bytes) {
	x.out = []byte(in)
}

func (x *conversion) ForVariable(q.Variable) {
	x.err = errors.New("cannot convert variable")
}

func (x *conversion) ForMaybeMore(q.MaybeMore) {
	x.err = errors.New("cannot convert maybe-more")
}

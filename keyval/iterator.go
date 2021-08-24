package keyval

import (
	"fmt"
	"math/big"

	"github.com/apple/foundationdb/bindings/go/src/fdb"

	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/pkg/errors"
)

// A TupleErrorMode is passed to ReadTuple and
// modifies the way ReadTuple fails.
type TupleErrorMode = int

const (
	// AllErrors tells ReadTuple to check for
	// a LongTupleError.
	AllErrors TupleErrorMode = iota

	// AllowLong tells ReadTuple to not check
	// for a LongTupleError.
	AllowLong
)

// A ConversionError is returned by ReadTuple when
// the TupleIterator fails to convert a Tuple element
// to the requested type.
type ConversionError struct {
	InValue interface{}
	OutType interface{}
	Index   int
}

func (t ConversionError) Error() string {
	return fmt.Sprintf("failed to convert element %d from %v to %T", t.Index, t.InValue, t.OutType)
}

var (
	// ShortTupleError is returned by ReadTuple when the TupleIterator
	// reads beyond the length of the Tuple.
	ShortTupleError = errors.New("read past end of tuple")

	// LongTupleError is returned by ReadTuple when the entire Tuple
	// is not consumed. This error isn't returned when ReadTuple is
	// given the AllowLong flag.
	LongTupleError = errors.New("did not parse entire tuple")
)

// ReadTuple provides a way to iterate over a Tuple's elements and convert each element to it's
// expected type. The caller-provided function uses a TupleIterator to read each of the Tuple's
// elements is sequential order. Any errors generated by the caller-provided function are returned
// as is. Additionally, ReadTuple may return ShortTupleError, LongTupleError, or an instance of
// ConversionError. See these errors for more information on when they are returned.
func ReadTuple(t Tuple, mode TupleErrorMode, f func(*TupleIterator) error) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if e, ok := e.(ConversionError); ok {
				err = e
				return
			}
			if e == ShortTupleError {
				err = ShortTupleError
				return
			}
			panic(e)
		}
	}()

	p := TupleIterator{t: t}
	if err := f(&p); err != nil {
		return err
	}

	if mode == AllErrors && p.i != len(t) {
		return LongTupleError
	}
	return nil
}

// TupleIterator provides methods for reading each Tuple element
// and converting the read element to an expected type. It is
// meant to be created by the ReadTuple function. For more
// information, see the ReadTuple documentation.
type TupleIterator struct {
	t Tuple
	i int
}

func (i *TupleIterator) getIndex() int {
	if i.i >= len(i.t) {
		panic(ShortTupleError)
	}

	i.i++
	return i.i - 1
}

func (i *TupleIterator) Any() interface{} {
	return i.t[i.getIndex()]
}

func (i *TupleIterator) BoolErr() (out bool, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].(bool); ok {
		return val, nil
	}
	return false, ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) Bool() (out bool) {
	val, err := i.BoolErr()
	if err != nil {
		panic(err)
	}
	return val
}

func (i *TupleIterator) IntErr() (out int64, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].(int64); ok {
		return val, nil
	}
	if val, ok := i.t[index].(int); ok {
		return int64(val), nil
	}
	return 0, ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) Int() (out int64) {
	val, err := i.IntErr()
	if err != nil {
		panic(err)
	}
	return val
}

func (i *TupleIterator) UintErr() (out uint64, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].(int64); ok {
		return uint64(val), nil
	}
	if val, ok := i.t[index].(uint64); ok {
		return val, nil
	}
	if val, ok := i.t[index].(int); ok {
		return uint64(val), nil
	}
	if val, ok := i.t[index].(uint); ok {
		return uint64(val), nil
	}
	return 0, ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) Uint() (out uint64) {
	val, err := i.UintErr()
	if err != nil {
		panic(err)
	}
	return val
}

func (i *TupleIterator) BigIntErr() (out *big.Int, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].(int64); ok {
		return big.NewInt(val), nil
	}
	if val, ok := i.t[index].(int); ok {
		return big.NewInt(int64(val)), nil
	}
	if val, ok := i.t[index].(uint64); ok {
		out = big.NewInt(0)
		out.SetUint64(val)
		return out, nil
	}
	if val, ok := i.t[index].(uint); ok {
		out = big.NewInt(0)
		out.SetUint64(uint64(val))
		return out, nil
	}
	if val, ok := i.t[index].(big.Int); ok {
		return &val, nil
	}
	if val, ok := i.t[index].(*big.Int); ok {
		return val, nil
	}
	return nil, ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) BigInt() (out *big.Int) {
	val, err := i.BigIntErr()
	if err != nil {
		panic(err)
	}
	return val
}

func (i *TupleIterator) FloatErr() (out float64, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].(float64); ok {
		return val, nil
	}
	if val, ok := i.t[index].(float32); ok {
		return float64(val), nil
	}
	return 0, ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) Float() (out float64) {
	val, err := i.FloatErr()
	if err != nil {
		panic(err)
	}
	return val
}

func (i *TupleIterator) StringErr() (out string, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].(string); ok {
		return val, nil
	}
	return "", ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) String() (out string) {
	val, err := i.StringErr()
	if err != nil {
		panic(err)
	}
	return val
}

func (i *TupleIterator) BytesErr() (out []byte, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].([]byte); ok {
		return val, nil
	}
	if val, ok := i.t[index].(fdb.KeyConvertible); ok {
		return val.FDBKey(), nil
	}
	return nil, ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) Bytes() (out []byte) {
	val, err := i.BytesErr()
	if err != nil {
		panic(err)
	}
	return val
}

func (i *TupleIterator) UUIDErr() (out tuple.UUID, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].(tuple.UUID); ok {
		return val, nil
	}
	return tuple.UUID{}, ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) UUID() (out tuple.UUID) {
	val, err := i.UUIDErr()
	if err != nil {
		panic(err)
	}
	return val
}

func (i *TupleIterator) TupleErr() (out Tuple, err error) {
	index := i.getIndex()
	if val, ok := i.t[index].(Tuple); ok {
		return val, nil
	}
	if val, ok := i.t[index].(tuple.Tuple); ok {
		return FromFDBTuple(val), nil
	}
	return nil, ConversionError{
		InValue: i.t[index],
		OutType: out,
		Index:   index,
	}
}

func (i *TupleIterator) Tuple() (out Tuple) {
	val, err := i.TupleErr()
	if err != nil {
		panic(err)
	}
	return val
}

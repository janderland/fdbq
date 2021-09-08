package iterator

import (
	"fmt"
	"math/big"

	q "github.com/janderland/fdbq/keyval/keyval"
	"github.com/pkg/errors"
)

// Generate the TupleIterator.Must...() methods.
//go:generate go run ./must -types Bool,Int,Uint,BigInt,Float,String,Bytes,UUID,Tuple

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

// A ConversionError is returned by ReadTuple when the
// TupleIterator fails to convert a Tuple element to
// the requested type.
type ConversionError struct {
	InValue interface{}
	OutType interface{}
	Index   int
}

func (x ConversionError) Error() string {
	return fmt.Sprintf("failed to convert element %d from %v to %T", x.Index, x.InValue, x.OutType)
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
func ReadTuple(t q.Tuple, mode TupleErrorMode, f func(*TupleIterator) error) (err error) {
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
	t q.Tuple
	i int
}

func (x *TupleIterator) getIndex() int {
	if x.i >= len(x.t) {
		panic(ShortTupleError)
	}

	x.i++
	return x.i - 1
}

func (x *TupleIterator) Any() q.TupElement {
	return x.t[x.getIndex()]
}

func (x *TupleIterator) Bool() (out q.Bool, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.Bool); ok {
		return val, nil
	}
	return false, ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}

func (x *TupleIterator) Int() (out q.Int, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.Int); ok {
		return val, nil
	}
	return 0, ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}

func (x *TupleIterator) Uint() (out q.Uint, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.Uint); ok {
		return val, nil
	}
	return 0, ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}

func (x *TupleIterator) BigInt() (out q.BigInt, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.Int); ok {
		return q.BigInt(*big.NewInt(int64(val))), nil
	}
	if val, ok := x.t[index].(q.Uint); ok {
		bi := big.NewInt(0)
		bi.SetUint64(uint64(val))
		return q.BigInt(*bi), nil
	}
	if val, ok := x.t[index].(q.BigInt); ok {
		return val, nil
	}
	return q.BigInt{}, ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}

func (x *TupleIterator) Float() (out q.Float, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.Float); ok {
		return val, nil
	}
	return 0, ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}

func (x *TupleIterator) String() (out q.String, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.String); ok {
		return val, nil
	}
	return "", ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}

func (x *TupleIterator) Bytes() (out q.Bytes, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.Bytes); ok {
		return val, nil
	}
	return nil, ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}

func (x *TupleIterator) UUID() (out q.UUID, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.UUID); ok {
		return val, nil
	}
	return q.UUID{}, ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}

func (x *TupleIterator) Tuple() (out q.Tuple, err error) {
	index := x.getIndex()
	if val, ok := x.t[index].(q.Tuple); ok {
		return val, nil
	}
	return nil, ConversionError{
		InValue: x.t[index],
		OutType: out,
		Index:   index,
	}
}
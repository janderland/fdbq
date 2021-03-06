package keyval

import (
	tup "github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/pkg/errors"
)

// ToStringArray attempts to convert a Directory to a string
// array. If the Directory contains non-string elements, an
// error is returned.
func ToStringArray(in Directory) ([]string, error) {
	out := make([]string, len(in))
	var ok bool

	for i := range in {
		out[i], ok = in[i].(string)
		if !ok {
			return nil, errors.Errorf("index '%d' has type '%T'", i, in[i])
		}
	}

	return out, nil
}

// FromStringArray converts a string array into a Directory.
func FromStringArray(in []string) Directory {
	out := make(Directory, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

// ToFDBTuple converts a Tuple into a tuple.Tuple. Note that
// the resultant tuple.Tuple will be invalid if the original
// Tuple contains a Variable.
func ToFDBTuple(in Tuple) tup.Tuple {
	out := make(tup.Tuple, len(in))
	for i := range in {
		switch e := in[i].(type) {
		case Tuple:
			out[i] = ToFDBTuple(e)
		default:
			out[i] = tup.TupleElement(in[i])
		}
	}
	return out
}

// FromFDBTuple converts a tuple.Tuple into a Tuple.
func FromFDBTuple(in tup.Tuple) Tuple {
	out := make(Tuple, len(in))
	for i := range in {
		switch e := in[i].(type) {
		case tup.Tuple:
			out[i] = FromFDBTuple(e)
		default:
			out[i] = in[i]
		}
	}
	return out
}

// SplitAtFirstVariable accepts either a Directory or Tuple and returns a slice of the elements
// before the first variable, the first variable, and a slice of the elements after the variable.
func SplitAtFirstVariable(list []interface{}) ([]interface{}, *Variable, []interface{}) {
	for i, segment := range list {
		if segment, ok := segment.(Variable); ok {
			return list[:i], &segment, list[i+1:]
		}
	}
	return list, nil, nil
}

package internal

import (
	"github.com/pkg/errors"

	"github.com/janderland/fql/keyval"
)

// KeyValBuilder is used by parser.Parser to construct the
// resultant key-value. parser.Parser doesn't interact with
// keyval.KeyValue directly, so these methods outline all
// the key-value state changes performed by the parser.
type KeyValBuilder struct {
	kv keyval.KeyValue
}

// Get returns the [keyval.KeyValue] being constructed.
func (x *KeyValBuilder) Get() keyval.KeyValue {
	return x.kv
}

// AppendVarToDirectory appends a keyval.Variable to the end of the directory.
func (x *KeyValBuilder) AppendVarToDirectory() {
	x.kv.Key.Directory = append(x.kv.Key.Directory, keyval.Variable{})
}

// AppendPartToDirectory appends a keyval.String to the end of the directory.
func (x *KeyValBuilder) AppendPartToDirectory(token string) {
	x.kv.Key.Directory = append(x.kv.Key.Directory, keyval.String(token))
}

// AppendToLastDirPart appends the given string to the last element of the
// directory, which is assumed to be a keyval.String. If the last element is
// not a keyval.String then this method panics.
func (x *KeyValBuilder) AppendToLastDirPart(token string) error {
	i := len(x.kv.Key.Directory) - 1
	str, ok := x.kv.Key.Directory[i].(keyval.String)
	if !ok {
		return errors.Errorf("expected element %d to be string, actually is %T", i, x.kv.Key.Directory[i])
	}
	x.kv.Key.Directory[i] = str + keyval.String(token)
	return nil
}

// AppendToValueVar appends the given keyval.ValueType to the keyval.Variable
// assigned as the value. If the value is not a keyval.Variable then this
// method panics.
func (x *KeyValBuilder) AppendToValueVar(typ keyval.ValueType) error {
	val, ok := x.kv.Value.(keyval.Variable)
	if !ok {
		return errors.Errorf("expected value to be variable, actually is %T", x.kv.Value)
	}
	x.kv.Value = append(val, typ)
	return nil
}

// AppendToValueStr appends the given string to the keyval.String assigned
// as the value. If the value is not a keyval.String then this method panics.
func (x *KeyValBuilder) AppendToValueStr(token string) error {
	str, ok := x.kv.Value.(keyval.String)
	if !ok {
		return errors.Errorf("expected value to be string, actually is %T", x.kv.Value)
	}
	x.kv.Value = str + keyval.String(token)
	return nil
}

// SetKeyTuple sets the tuple portion of the keyval.Key.
func (x *KeyValBuilder) SetKeyTuple(tup keyval.Tuple) {
	x.kv.Key.Tuple = tup
}

// SetValue sets the value portion of the keyval.KeyValue.
func (x *KeyValBuilder) SetValue(val keyval.Value) {
	x.kv.Value = val
}

// TupBuilder is used by parser.Parser to construct tuples.
// parser.Parser doesn't interact with keyval.Tuple directly,
// so these methods outline all the key-value state changes
// performed by the parser.
type TupBuilder struct {
	root  keyval.Tuple
	depth int
}

// Get returns the [keyval.Tuple] being constructed.
func (x *TupBuilder) Get() keyval.Tuple {
	return x.root
}

// StartSubTuple starts a new sub-tuple which will be appended
// to the currently constructed tuple. This sub-tuple will be
// treated as the currently constructed tuple and all subsequent
// method calls will be applied to this sub-tuple. When the
// sub-tuple is completed, EndTuple can be called and
// construction of the previous tuple will be resumed.
func (x *TupBuilder) StartSubTuple() {
	_ = x.mutateTuple(func(tup keyval.Tuple) (keyval.Tuple, error) {
		return append(tup, keyval.Tuple{}), nil
	})
	x.depth++
}

// EndTuple completes the currently constructed tuple and
// assigns it as a sub-tuple of the previously constructed
// tuple. The previously constructed tuple becomes the
// currently constructed tuple and all subsequent method
// calls will be applied to it.
func (x *TupBuilder) EndTuple() bool {
	x.depth--
	return x.depth == -1
}

// Append appends the given element to the keyval.Tuple.
func (x *TupBuilder) Append(e keyval.TupElement) {
	_ = x.mutateTuple(func(tup keyval.Tuple) (keyval.Tuple, error) {
		return append(tup, e), nil
	})
}

// AppendToLastElemStr appends the given string to the keyval.String
// assigned as the last element of the currently constructed tuple.
// If the last element is not a keyval.String, then this method panics.
func (x *TupBuilder) AppendToLastElemStr(token string) error {
	return x.mutateTuple(func(tup keyval.Tuple) (keyval.Tuple, error) {
		i := len(tup) - 1
		str, ok := tup[i].(keyval.String)
		if !ok {
			return nil, errors.Errorf("expected element %d to be string, actually is %T", i, tup[i])
		}
		tup[i] = keyval.String(string(str) + token)
		return tup, nil
	})
}

// AppendToLastElemVar appends the given keyval.ValueType to the keyval.Variable
// assigned as the last element of the currently constructed tuple. If the last
// element is not a keyval.Variable then this method panics.
func (x *TupBuilder) AppendToLastElemVar(typ keyval.ValueType) error {
	return x.mutateTuple(func(tup keyval.Tuple) (keyval.Tuple, error) {
		i := len(tup) - 1
		v, ok := tup[i].(keyval.Variable)
		if !ok {
			return nil, errors.Errorf("expected element %d to be variable, actually is %T", i, tup[i])
		}
		tup[i] = append(v, typ)
		return tup, nil
	})
}

// TODO: Don't assign tuple into parent until EndTuple is called.
func (x *TupBuilder) mutateTuple(f func(keyval.Tuple) (keyval.Tuple, error)) error {
	tuples := []keyval.Tuple{x.root}
	tup := tuples[0]

	for i := 0; i < x.depth; i++ {
		tup = tup[len(tup)-1].(keyval.Tuple)
		tuples = append(tuples, tup)
	}

	tup, err := f(tup)
	if err != nil {
		return err
	}

	tuples[len(tuples)-1] = tup

	for i := len(tuples) - 1; i > 0; i-- {
		tuples[i-1][len(tuples[i-1])-1] = tuples[i]
	}

	x.root = tuples[0]
	return nil
}

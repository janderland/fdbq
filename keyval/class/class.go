// Package class classifies a key-value by the kind of operation it represents.
package class

import q "github.com/janderland/fql/keyval"

// Class categorizes a KeyValue.
type Class string

const (
	// Constant specifies that the KeyValue has no Variable,
	// MaybeMore, or Clear. This kind of KeyValue can be used to
	// perform a set operation or is returned by a get operation.
	Constant Class = "constant"

	// Clear specifies that the KeyValue has no Variable or
	// MaybeMore and has a Clear Value. This kind of KeyValue can
	// be used to perform a clear operation.
	Clear Class = "clear"

	// ReadSingle specifies that the KeyValue has a Variable
	// Value and doesn't have a Variable or MaybeMore in its Key.
	// This kind of KeyValue can be used to perform a get operation
	// that returns a single KeyValue.
	ReadSingle Class = "single"

	// ReadRange specifies that the KeyValue has a Variable
	// or MaybeMore in its Key and doesn't have a Clear Value.
	// This kind of KeyValue can be used to perform a get
	// operation that returns multiple KeyValue.
	ReadRange Class = "range"

	// VariableClear specifies that the KeyValue has a
	// Variable or MaybeMore in its Key and has a Clear for
	// its value. This is an invalid class of KeyValue.
	VariableClear Class = "variable clear"

	// Nil specifies that the KeyValue contains a nil (not keyval.Nil).
	// This is an invalid class of KeyValue.
	Nil Class = "nil"
)

// subClass categorizes the Key, Directory,
// Tuple, and Value within a KeyValue.
type subClass int

const (
	// constantSubClass specifies that the component contains no
	// Variable, MaybeMore, or Clear.
	constantSubClass subClass = iota

	// variableSubClass specifies that the component contains a
	// Variable or MaybeMore.
	variableSubClass

	// clearSubClass specifies that the component contains a Clear.
	clearSubClass

	// nilSubClass specifies that the component contains a nil, which
	// isn't allowed in any part of the key-value. This shouldn't be
	// confused with an instance of the Nil type.
	nilSubClass
)

// Classify returns the Class of the given KeyValue.
func Classify(kv q.KeyValue) Class {
	keyClass := classifyKey(kv.Key)
	valClass := classifyValue(kv.Value)

	// If a nil is present in any part of the key, the Nil class
	// takes precedence.
	if keyClass == nilSubClass || valClass == nilSubClass {
		return Nil
	}

	// If the key is constant, then this query will only affect
	// a single key and the value will dictate what kind of
	// single-key query it will be.
	if keyClass == constantSubClass {
		switch valClass {
		case clearSubClass:
			return Clear
		case variableSubClass:
			return ReadSingle
		default:
			return Constant
		}
	}

	// If the key is not constant then the query should be a
	// range read, unless it has a Clear instance for its
	// value.
	if valClass == clearSubClass {
		return VariableClear
	}
	return ReadRange
}

func classifyKey(key q.Key) subClass {
	dirClass := classifyDir(key.Directory)
	tupClass := classifyTuple(key.Tuple)

	if dirClass == nilSubClass || tupClass == nilSubClass {
		return nilSubClass
	}
	if dirClass == variableSubClass || tupClass == variableSubClass {
		return variableSubClass
	}
	return constantSubClass
}

func classifyDir(dir q.Directory) subClass {
	class := dirClassification{}
	for _, element := range dir {
		if element == nil {
			return nilSubClass
		}
		element.DirElement(&class)
	}
	return class.out
}

func classifyTuple(tup q.Tuple) subClass {
	class := tupClassification{}
	for _, element := range tup {
		if element == nil {
			return nilSubClass
		}
		element.TupElement(&class)
	}
	return class.out
}

func classifyValue(val q.Value) subClass {
	if val == nil {
		return nilSubClass
	}
	class := valClassification{}
	val.Value(&class)
	return class.out
}

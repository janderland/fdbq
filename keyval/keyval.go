// Package keyval contains types representing key-values and related
// utilities. These types model both queries and the data returned by
// queries. They can be constructed from query strings using [parser.Parser]
// but are also designed to be easily constructed directly in Go source
// code.
//
// # Embedded Query Strings
//
// When working with SQL, programmers will often embed SQL strings in
// the application. This requires extra tooling to catch syntax errors
// at build time. Instead of using string literals, FQL allows the
// programmer to directly construct the queries using the types in this
// package, allowing some syntax errors to be caught at build time.
//
// This package does not prevent all kinds of syntax errors, only
// "structural" ones. For instance, the type system ensures tuples only
// contain valid elements and limits what kinds of objects can be used
// as the value of a key-value. In spite of this, invalid queries can
// still be constructed (see package class).
//
// # Operations (Visitor Pattern)
//
// The Directory, Tuple, & Value types would be best represented by tagged
// unions. While Go does not natively support tagged unions, this codebase
// implements equivalent functionality using the visitor pattern. See
// package [operation] which generates the boilerplate code associated with
// this. See DirectoryOperation, TupleOperation, or ValueOperation as
// examples of the generated code.
//
// # Primitive Types
//
// There are a special group of types defined in this package named the
// "primitive" types. These include Nil, Int, Uint, Bool, Float, String,
// UUID, and Bytes. All of these types can be used as a TupElement or as
// a Value. When used as a TupElement, they are serialized by FDB tuple
// packing. When used as a Value, they are known as a "primitive" values
// and are serialized by FQL.
package keyval

// TODO: Add BigInt to Tuple and Value.
//go:generate go run ./operation -op-name Query     -param-name query      -types Directory,Key,KeyValue
//go:generate go run ./operation -op-name Directory -param-name DirElement -types String,Variable
//go:generate go run ./operation -op-name Tuple     -param-name TupElement -types Tuple,Nil,Int,Uint,Bool,Float,String,UUID,Bytes,Variable,MaybeMore
//go:generate go run ./operation -op-name Value     -param-name value      -types Tuple,Nil,Int,Uint,Bool,Float,String,UUID,Bytes,Variable,Clear

type (
	// Query is an interface implemented by the types which can
	// be passed to [engine.Engine] as a query. This includes
	// KeyValue, Key, & Directory.
	Query = query

	// KeyValue can be passed to [engine.Engine] as a query or be
	// returned from [engine.Engine] as a query's result. When
	// returned as a result, KeyValue will not contain a Variable,
	// Clear, or MaybeMore.
	KeyValue struct {
		Key   Key
		Value Value
	}

	// Key can be passed to [engine.Engine] as a query. When used
	// as a query, it is equivalent to a KeyValue whose Value is
	// an empty Variable.
	Key struct {
		Directory Directory
		Tuple     Tuple
	}

	// Directory can be passed to [engine.Engine] as a directory
	// query. These kinds of queries define a directory path
	// schema. When executed, all directories matching the
	// schema are returned.
	Directory []DirElement

	// Tuple may contain a Tuple, Variable, MaybeMore, or any
	// of the "primitive" types.
	Tuple []TupElement

	// Value may contain Tuple, Variable, Clear, or any
	// of the "primitive" types.
	Value = value

	// Variable is a placeholder which implements the DirElement,
	// TupElement, & Value interfaces. A Query containing a Variable
	// defines a schema. When the Query is executed, all key-values
	// (or directories) matching the schema are returned.
	Variable []ValueType

	// MaybeMore is a special kind of TupElement. It may only
	// appear as the last element of the Tuple. A Query containing
	// a MaybeMore defines a schema which allows all keys prefixed
	// by the key in the schema.
	// TODO: Implement as a flag on Tuple.
	MaybeMore struct{}

	// Clear is a special kind of Value which designates
	// a KeyValue as a clear query. When executed, the
	// provided key is cleared from the DB. Clear may
	// not be used in a query containing Variable.
	Clear struct{}
)

// These are the "primitive" types.
type (
	// Nil is a "primitive" type representing an empty element
	// when used as a TupElement. It's equivalent to an empty
	// Bytes when used as a Value. Go's nil value is never
	// allowed in a Key or KeyValue. Instead, use this type.
	Nil struct{}

	// Int is a "primitive" type implementing an int64 as either
	// a TupElement or Value. When used as a Value, it's serialized
	// as a 2-compliment 8-byte string. Endianness depends on how
	// the [engine.Engine] is configured.
	Int int64

	// Uint is a "primitive" type implementing a uint64 as either
	// a TupElement or Value. When used as a Value, it's serialized
	// as an 8-byte array. Endianness depends on how the
	// [engine.Engine] is configured.
	Uint uint64

	// Bool is a "primitive" type implementing a bool as either a
	// TupElement or Value. When used as a Value, it's serialized
	// as a single byte (0 for false, 1 for true).
	Bool bool

	// Float is a "primitive" type implementing a float64 as either
	// a TupElement or Value. When used as a Value, it's serialized
	// as an 8-byte array in accordance with IEEE 754. Endianness
	// depends on how the [engine.Engine] is configured.
	Float float64

	// TODO: Add support for BigInt.
	/*
		// BigInt is a "primitive" type implementing a big.Int as either
		// a TupElement or Value.
		// TODO: Document how BigInt is serialized.
		BigInt big.Int
	*/

	// String is a "primitive" type implementing a string as either
	// a TupElement or Value. When used as a Value, it's serialized
	// as a UTF-8 encoded byte string.
	String string

	// UUID is a "primitive" type implementing a 16-byte string as
	// either a TupElement or Value. When used as a Value, it's
	// serialized as is.
	UUID [16]byte

	// Bytes is a "primitive" type implementing a byte string as
	// either a TupElement or Value. When used as a Value, it's
	// serialized as is.
	Bytes []byte
)

// ValueType defines the expected types of a Variable.
type ValueType string

const (
	// AnyType designates a Variable to allow any value.
	// A Variable containing AnyType and an empty Variable
	// are equivalent.
	AnyType ValueType = ""

	// IntType designates a Variable to allow Int values.
	IntType ValueType = "int"

	// UintType designates a Variable to allow Uint values.
	UintType ValueType = "uint"

	// BoolType designates a Variable to allow Bool values.
	BoolType ValueType = "bool"

	// FloatType designates a Variable to allow Float values.
	FloatType ValueType = "float"

	// TODO: Add support for BigInt.
	/*
		// BigIntType designates a Variable to allow BigInt values.
		BigIntType ValueType = "bigint"
	*/

	// StringType designates a Variable to allow String values.
	StringType ValueType = "string"

	// BytesType designates a Variable to allow Bytes values.
	BytesType ValueType = "bytes"

	// UUIDType designates a Variable to allow UUID values.
	UUIDType ValueType = "uuid"

	// TupleType designates a Variable to allow Tuple values.
	TupleType ValueType = "tuple"
)

// AllTypes returns all valid values for ValueType.
func AllTypes() []ValueType {
	return []ValueType{
		AnyType,
		IntType,
		UintType,
		BoolType,
		FloatType,
		// TODO: Add support for BigInt.
		// BigIntType,
		StringType,
		BytesType,
		UUIDType,
		TupleType,
	}
}

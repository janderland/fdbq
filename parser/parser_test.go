package parser

import (
	"testing"

	tup "github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	q "github.com/janderland/fdbq/keyval"
	"github.com/stretchr/testify/assert"
)

func TestKeyValue(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.KeyValue
	}{
		{name: "full", str: "/hi/there{54,nil}={33.8}",
			ast: q.KeyValue{Key: q.Key{Directory: q.Directory{"hi", "there"}, Tuple: q.Tuple{int64(54), nil}}, Value: q.Tuple{33.8}}},
	}

	for _, test := range roundTrips {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseKeyValue(test.str)
			assert.NoError(t, err)
			assert.Equal(t, test.ast, *ast)

			str, err := FormatKeyValue(test.ast)
			assert.NoError(t, err)
			assert.Equal(t, test.str, str)
		})
	}

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "empty tup", str: "{}"},
		{name: "double sep", str: "{}={}={}"},
		{name: "bad key", str: "badkey={}"},
		{name: "bad value", str: "{}=badvalue"},
	}

	for _, test := range parseFailures {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseKeyValue(test.str)
			assert.Error(t, err)
			assert.Nil(t, ast)
		})
	}
}

func TestKey(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Key
	}{
		{name: "dir", str: "/my/dir",
			ast: q.Key{Directory: q.Directory{"my", "dir"}}},
		{name: "tup", str: "{\"str\",-13,{1.2e+13}}",
			ast: q.Key{Tuple: q.Tuple{"str", int64(-13), q.Tuple{1.2e13}}}},
		{name: "full", str: "/my/dir{\"str\",-13,{1.2e+13}}",
			ast: q.Key{Directory: q.Directory{"my", "dir"}, Tuple: q.Tuple{"str", int64(-13), q.Tuple{1.2e13}}}},
	}

	for _, test := range roundTrips {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseKey(test.str)
			assert.NoError(t, err)
			assert.Equal(t, &test.ast, ast)

			str, err := FormatKey(*ast)
			assert.NoError(t, err)
			assert.Equal(t, test.str, str)
		})
	}

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "bad dir", str: "baddir"},
		{name: "bad tup", str: "/dir{badtup"},
	}

	for _, fail := range parseFailures {
		t.Run(fail.name, func(t *testing.T) {
			key, err := ParseKey(fail.str)
			assert.Error(t, err)
			assert.Nil(t, key)
		})
	}
}

func TestValue(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Value
	}{
		{name: "clear", str: "clear", ast: q.Clear{}},
		{name: "tuple", str: "{-16,13.2,\"hi\"}", ast: q.Tuple{int64(-16), 13.2, "hi"}},
		{name: "raw", str: "-16", ast: int64(-16)},
	}

	for _, test := range roundTrips {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseValue(test.str)
			assert.NoError(t, err)
			assert.Equal(t, test.ast, ast)

			str, err := FormatValue(test.ast)
			assert.NoError(t, err)
			assert.Equal(t, test.str, str)
		})
	}

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
	}

	for _, test := range parseFailures {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseValue(test.str)
			assert.Error(t, err)
			assert.Nil(t, ast)
		})
	}
}

func TestDirectory(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Directory
	}{
		{name: "single", str: "/hello", ast: q.Directory{"hello"}},
		{name: "multi", str: "/hello/world", ast: q.Directory{"hello", "world"}},
		{name: "variable", str: "/hello/<int>/thing", ast: q.Directory{"hello", q.Variable{q.IntType}, "thing"}},
	}

	for _, test := range roundTrips {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseDirectory(test.str)
			assert.NoError(t, err)
			assert.Equal(t, test.ast, ast)

			str, err := FormatDirectory(test.ast)
			assert.NoError(t, err)
			assert.Equal(t, test.str, str)
		})
	}

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "no paths", str: "/"},
		{name: "no slash", str: "hello"},
		{name: "empty path", str: "/ /path"},
		{name: "trailing slash", str: "/hello/world/"},
		{name: "invalid var", str: "/hello/</thing"},
	}

	for _, test := range parseFailures {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseDirectory(test.str)
			assert.Error(t, err)
			assert.Nil(t, ast)
		})
	}
}

func TestTuple(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Tuple
	}{
		{name: "empty", str: "{}", ast: q.Tuple{}},
		{name: "one", str: "{17}", ast: q.Tuple{int64(17)}},
		{name: "two", str: "{17,\"hello world\"}", ast: q.Tuple{int64(17), "hello world"}},
		{name: "sub tuple", str: "{\"hello\",23.3,{-3}}", ast: q.Tuple{"hello", 23.3, q.Tuple{int64(-3)}}},
		{name: "uuid", str: "{{bcefd2ec-4df5-43b6-8c79-81b70b886af9}}",
			ast: q.Tuple{q.Tuple{tup.UUID{0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}}}},
	}

	for _, test := range roundTrips {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseTuple(test.str)
			assert.NoError(t, err)
			assert.Equal(t, test.ast, ast)

			str, err := FormatTuple(test.ast)
			assert.NoError(t, err)
			assert.Equal(t, test.str, str)
		})
	}

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "no close", str: "{"},
		{name: "no open", str: "}"},
		{name: "bad element", str: "{bad}"},
		{name: "empty element", str: "{\"hello\",, -3}"},
	}

	for _, test := range parseFailures {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseTuple(test.str)
			assert.Error(t, err)
			assert.Nil(t, ast)
		})
	}
}

func TestData(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  interface{}
	}{
		{name: "nil", str: "nil", ast: nil},
		{name: "true", str: "true", ast: true},
		{name: "false", str: "false", ast: false},
		{name: "variable", str: "<int>", ast: q.Variable{q.IntType}},
		{name: "string", str: "\"hello world\"", ast: "hello world"},
		{name: "uuid", str: "bcefd2ec-4df5-43b6-8c79-81b70b886af9", ast: tup.UUID{0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}},
		{name: "int", str: "123", ast: int64(123)},
		{name: "float", str: "-94.2", ast: -94.2},
		{name: "scientific", str: "3.47e-08", ast: 3.47e-8},
	}

	for _, test := range roundTrips {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseData(test.str)
			assert.NoError(t, err)
			assert.Equal(t, test.ast, ast)

			str, err := FormatData(test.ast)
			assert.NoError(t, err)
			assert.Equal(t, test.str, str)
		})
	}

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "invalid", str: "invalid"},
	}

	for _, test := range parseFailures {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseData(test.str)
			assert.Error(t, err)
			assert.Nil(t, ast)
		})
	}
}

func TestVariable(t *testing.T) {
	tests := []struct {
		name string
		str  string
		ast  q.Variable
	}{
		{name: "empty", str: "<>", ast: nil},
		{name: "single", str: "<int>", ast: q.Variable{q.IntType}},
		{name: "multiple", str: "<int|float|tuple>", ast: q.Variable{q.IntType, q.FloatType, q.TupleType}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseVariable(test.str)
			assert.Equal(t, test.ast, ast)
			assert.NoError(t, err)

			str := FormatVariable(test.ast)
			assert.Equal(t, test.str, str)
		})
	}

	fails := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "unclosed", str: "<"},
		{name: "unopened", str: ">"},
		{name: "invalid", str: "<invalid>"},
	}

	for _, test := range fails {
		t.Run(test.name, func(t *testing.T) {
			v, err := ParseVariable(test.str)
			assert.Error(t, err)
			assert.Nil(t, v)
		})
	}
}

func TestString(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  interface{}
	}{
		{name: "regular", str: "\"hello world\"", ast: "hello world"},
	}

	for _, test := range roundTrips {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseString(test.str)
			assert.NoError(t, err)
			assert.Equal(t, test.ast, ast)

			str, err := FormatString(test.ast)
			assert.NoError(t, err)
			assert.Equal(t, test.str, str)
		})
	}

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "no close", str: "\"hello"},
		{name: "no open", str: "hello\""},
		{name: "single quote", str: "'hello'"},
	}

	for _, test := range parseFailures {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseString(test.str)
			assert.Error(t, err)
			assert.Empty(t, ast)
		})
	}
}

func TestUUID(t *testing.T) {
	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "bad group 1", str: "cefd2ec-4df5-43b6-8c79-81b70b886af9"},
		{name: "bad group 2", str: "bcefd2ec-df5-43b6-8c79-81b70b886af9"},
		{name: "bad group 3", str: "bcefd2ec-4df5-3b6-8c79-81b70b886af9"},
		{name: "bad group 4", str: "bcefd2ec-4df5-43b6-c79-81b70b886af9"},
		{name: "bad group 5", str: "bcefd2ec-4df5-43b6-8c79-1b70b886af9"},
		{name: "long", str: "bcefdyec-4df5-43%6-8c79-81b70bg86af9"},
	}

	for _, test := range parseFailures {
		ast, err := ParseUUID(test.str)
		assert.Error(t, err)
		assert.Equal(t, tup.UUID{}, ast)
	}

	roundTrips := []struct {
		name string
		str  string
		ast  tup.UUID
	}{
		{name: "normal", str: "bcefd2ec-4df5-43b6-8c79-81b70b886af9",
			ast: tup.UUID{0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}},
	}

	for _, test := range roundTrips {
		ast, err := ParseUUID(test.str)
		assert.NoError(t, err)
		assert.Equal(t, test.ast, ast)

		str := FormatUUID(test.ast)
		assert.Equal(t, test.str, str)
	}
}

func TestNumber(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  interface{}
	}{
		{name: "int", str: "-34000", ast: int64(-34000)},
		{name: "uint", str: "18446744073709551610", ast: uint64(18446744073709551610)},
		{name: "float", str: "94.33", ast: 94.33},
		{name: "scientific", str: "1.254e-07", ast: 1.254e-7},
	}

	for _, test := range roundTrips {
		ast, err := ParseNumber(test.str)
		assert.NoError(t, err)
		assert.Equal(t, test.ast, ast)

		str, err := FormatNumber(test.ast)
		assert.NoError(t, err)
		assert.Equal(t, test.str, str)
	}

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
	}

	for _, test := range parseFailures {
		ast, err := ParseNumber(test.str)
		assert.Error(t, err)
		assert.Nil(t, ast)
	}
}

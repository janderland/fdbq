package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	q "github.com/janderland/fql/keyval"
	"github.com/janderland/fql/parser/format"
	"github.com/janderland/fql/parser/scanner"
)

func TestDirectory(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Directory
	}{
		{name: "single", str: "/hello", ast: q.Directory{q.String("hello")}},
		{name: "multi", str: "/hello/world", ast: q.Directory{q.String("hello"), q.String("world")}},
		{name: "variable", str: "/hello/<>/thing", ast: q.Directory{q.String("hello"), q.Variable{}, q.String("thing")}},
		{name: "multiline", str: "/hi/\"you\nwow\"/<>", ast: q.Directory{q.String("hi"), q.String("you\nwow"), q.Variable{}}},
	}

	t.Run("key round trip", func(t *testing.T) {
		for _, test := range roundTrips {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))

				ast, err := p.Parse()
				require.NoError(t, err)
				require.Equal(t, test.ast, ast)

				f := newFormat()
				f.Directory(test.ast)
				require.Equal(t, test.str, f.String())
			})
		}
	})

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
		{name: "no paths", str: "/"},
		{name: "no slash", str: "hello"},
		{name: "trailing slash", str: "/hello/world/"},
		{name: "invalid var", str: "/hello/</thing"},
	}

	t.Run("parse failures", func(t *testing.T) {
		for _, test := range parseFailures {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				ast, err := p.Parse()
				require.Error(t, err)
				require.Nil(t, ast)
			})
		}
	})
}

func TestTuple(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Tuple
	}{
		{name: "empty", str: "()", ast: q.Tuple(nil)},
		{name: "one", str: "(17)", ast: q.Tuple{q.Int(17)}},
		{name: "two", str: "(17,\"hello world\")", ast: q.Tuple{q.Int(17), q.String("hello world")}},
		{name: "sub tuple", str: "(\"hello\",23.3,(-3))", ast: q.Tuple{q.String("hello"), q.Float(23.3), q.Tuple{q.Int(-3)}}},
		{name: "uuid", str: "((bcefd2ec-4df5-43b6-8c79-81b70b886af9))", ast: q.Tuple{q.Tuple{q.UUID{0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}}}},
		{name: "maybe more", str: "(18.2,0xffaa,...)", ast: q.Tuple{q.Float(18.2), q.Bytes{0xFF, 0xAA}, q.MaybeMore{}}},
		{name: "escape", str: "(\"i want to say \\\"yo\\\"\")", ast: q.Tuple{q.String("i want to say \"yo\"")}},
	}

	t.Run("key round trip", func(t *testing.T) {
		for _, test := range roundTrips {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateDirTail

				ast, err := p.Parse()
				require.NoError(t, err)
				require.Equal(t, test.ast, ast.(q.Key).Tuple)

				f := newFormat()
				f.Tuple(test.ast)
				require.Equal(t, test.str, f.String())
			})
		}
	})

	t.Run("value round trip", func(t *testing.T) {
		for _, test := range roundTrips {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateValue

				ast, err := p.Parse()
				require.NoError(t, err)
				require.Equal(t, test.ast, ast.(q.KeyValue).Value)

				f := newFormat()
				f.Tuple(test.ast)
				require.Equal(t, test.str, f.String())
			})
		}
	})

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "no close", str: "("},
		{name: "no open", str: ")"},
		{name: "bad element", str: "(bad)"},
		{name: "empty element", str: "(\"hello\",, -3)"},
	}

	t.Run("key parse failures", func(t *testing.T) {
		for _, test := range parseFailures {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateDirTail

				ast, err := p.Parse()
				require.Error(t, err)
				require.Nil(t, ast)
			})
		}
	})

	t.Run("value parse failures", func(t *testing.T) {
		for _, test := range parseFailures {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateValue

				ast, err := p.Parse()
				require.Error(t, err)
				require.Nil(t, ast)
			})
		}
	})
}

func TestValue(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Value
	}{
		{name: "clear", str: "clear", ast: q.Clear{}},
		{name: "tuple", str: "(-16,13.2,\"hi\")", ast: q.Tuple{q.Int(-16), q.Float(13.2), q.String("hi")}},
		{name: "raw", str: "-16", ast: q.Int(-16)},
		{name: "string", str: "\"he said \\\"wowee\\\"\"", ast: q.String("he said \"wowee\"")},
	}

	t.Run("round trip", func(t *testing.T) {
		for _, test := range roundTrips {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateValue

				ast, err := p.Parse()
				require.NoError(t, err)
				require.Equal(t, test.ast, ast.(q.KeyValue).Value)

				f := newFormat()
				f.Value(test.ast)
				require.Equal(t, test.str, f.String())
			})
		}
	})

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "empty", str: ""},
	}

	t.Run("value parse failures", func(t *testing.T) {
		for _, test := range parseFailures {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateValue

				ast, err := p.Parse()
				require.Error(t, err)
				require.Nil(t, ast)
			})
		}
	})
}

func TestString(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.String
		val  bool
	}{
		{name: "empty", str: "\"\"", ast: q.String("")},
		{name: "simple", str: "\"hi\"", ast: q.String("hi")},
		{name: "escapes", str: "\"\\\\ \\\" \"", ast: q.String("\\ \" ")},
	}

	t.Run("value round trip", func(t *testing.T) {
		for _, test := range roundTrips {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateValue

				ast, err := p.Parse()
				require.NoError(t, err)
				require.Equal(t, test.ast, ast.(q.KeyValue).Value)

				f := newFormat()
				f.Value(test.ast)
				require.Equal(t, test.str, f.String())
			})
		}
	})

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "illegal escape", str: "\" \\d \""},
	}

	t.Run("value parse failures", func(t *testing.T) {
		for _, test := range parseFailures {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateValue

				ast, err := p.Parse()
				require.Error(t, err)
				require.Nil(t, ast)
			})
		}
	})
}

func TestVariable(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Variable
		val  bool
	}{
		{name: "empty", str: "<>", ast: q.Variable{}},
		{name: "single", str: "<int>", ast: q.Variable{q.IntType}},
		{name: "multiple", str: "<int|float|tuple>", ast: q.Variable{q.IntType, q.FloatType, q.TupleType}},
		{name: "value", str: "<int|string>", ast: q.Variable{q.IntType, q.StringType}, val: true},
	}

	t.Run("value round trip", func(t *testing.T) {
		for _, test := range roundTrips {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateValue

				ast, err := p.Parse()
				require.NoError(t, err)
				require.Equal(t, test.ast, ast.(q.KeyValue).Value)

				f := newFormat()
				f.Value(test.ast)
				require.Equal(t, test.str, f.String())
			})
		}
	})

	parseFailures := []struct {
		name string
		str  string
	}{
		{name: "unclosed", str: "<"},
		{name: "unopened", str: ">"},
		{name: "invalid", str: "<invalid>"},
	}

	t.Run("value parse failures", func(t *testing.T) {
		for _, test := range parseFailures {
			t.Run(test.name, func(t *testing.T) {
				p := New(scanner.New(strings.NewReader(test.str)))
				p.state = stateValue

				ast, err := p.Parse()
				require.Error(t, err)
				require.Nil(t, ast)
			})
		}
	})
}

func TestData(t *testing.T) {
	roundTrips := []struct {
		name string
		str  string
		ast  q.Value
	}{
		{name: "nil", str: "nil", ast: q.Nil{}},
		{name: "true", str: "true", ast: q.Bool(true)},
		{name: "false", str: "false", ast: q.Bool(false)},
		{name: "hex", str: "0xabc032", ast: q.Bytes{0xab, 0xc0, 0x32}},
		{name: "uuid", str: "bcefd2ec-4df5-43b6-8c79-81b70b886af9", ast: q.UUID{0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}},
		{name: "int", str: "123", ast: q.Int(123)},
		{name: "float", str: "-94.2", ast: q.Float(-94.2)},
		{name: "scientific", str: "3.47e-08", ast: q.Float(3.47e-8)},
	}

	for _, test := range roundTrips {
		t.Run(test.name, func(t *testing.T) {
			ast, err := parseData(test.str)
			require.NoError(t, err)
			require.Equal(t, test.ast, ast)

			f := newFormat()
			f.Value(test.ast)
			require.Equal(t, test.str, f.String())
		})
	}

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
		ast, err := parseData(test.str)
		require.Error(t, err)
		require.Nil(t, ast)
	}
}

func newFormat() format.Format {
	return format.New(format.WithPrintBytes())
}

package internal

import (
	"encoding/binary"
	"testing"

	q "github.com/janderland/fql/keyval"
	"github.com/stretchr/testify/assert"
)

func TestNoFilter(t *testing.T) {
	tests := []struct {
		name  string
		query q.Value
		val   []byte
		out   q.Value
		err   bool
	}{
		{name: "empty variable", query: q.Variable{}, val: []byte{0xAE, 0xBC}, out: q.Bytes{0xAE, 0xBC}},
		{name: "variable match", query: q.Variable{q.IntType, q.StringType}, val: []byte("hi"), out: q.String("hi")},
		{name: "variable mismatch", query: q.Variable{q.IntType}, val: []byte("hi"), err: true},
		{name: "packed match", query: q.String("you"), val: []byte("you"), out: q.String("you")},
		{name: "packed mismatch", query: q.Int(22), val: []byte("you"), err: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewValueHandler(test.query, binary.BigEndian, false)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			val, err := handler.Handle(test.val)
			assert.Equal(t, test.out, val)

			if test.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name  string
		query q.Value
		val   []byte
		out   q.Value
	}{
		{name: "empty variable", query: q.Variable{}, val: []byte{0xAE, 0xBC}, out: q.Bytes{0xAE, 0xBC}},
		{name: "variable match", query: q.Variable{q.IntType, q.StringType}, val: []byte("hi"), out: q.String("hi")},
		{name: "variable mismatch", query: q.Variable{q.IntType}, val: []byte("hi"), out: nil},
		{name: "packed match", query: q.String("you"), val: []byte("you"), out: q.String("you")},
		{name: "packed mismatch", query: q.Int(22), val: []byte("you"), out: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewValueHandler(test.query, binary.BigEndian, true)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			val, err := handler.Handle(test.val)
			assert.Equal(t, test.out, val)
			assert.NoError(t, err)
		})
	}
}

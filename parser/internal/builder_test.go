package internal

import (
	"testing"

	q "github.com/janderland/fql/keyval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTupBuilder_startSubTuple(t *testing.T) {
	var b TupBuilder

	b.StartSubTuple()
	require.Equal(t, q.Tuple{q.Tuple{}}, b.Get())

	b.StartSubTuple()
	require.Equal(t, q.Tuple{q.Tuple{q.Tuple{}}}, b.Get())
}

func TestTupBuilder_appendStringToTuple(t *testing.T) {
	var b TupBuilder

	b.Append(q.String(""))
	require.Equal(t, q.Tuple{q.String("")}, b.Get())

	b.StartSubTuple()
	b.Append(q.String(""))
	require.Equal(t, q.Tuple{q.String(""), q.Tuple{q.String("")}}, b.Get())
}

func TestTupBuilder_appendToLastTupElem(t *testing.T) {
	var b TupBuilder

	b.StartSubTuple()
	b.StartSubTuple()
	b.Append(q.String(""))
	err := b.AppendToLastElemStr("hello")
	assert.NoError(t, err)
	require.Equal(t, q.Tuple{q.Tuple{q.Tuple{q.String("hello")}}}, b.Get())
}

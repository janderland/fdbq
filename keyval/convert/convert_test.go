package convert

import (
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	q "github.com/janderland/fql/keyval"
	"github.com/stretchr/testify/require"
)

func TestToStringArray(t *testing.T) {
	dir := q.Directory{q.String("my"), q.String("dir"), q.String("path")}
	arr, err := ToStringArray(dir)
	require.NoError(t, err)
	require.Equal(t, []string{"my", "dir", "path"}, arr)

	dir[1] = q.Variable{}
	_, err = ToStringArray(dir)
	require.Error(t, err)
}

func TestFromStringArray(t *testing.T) {
	dir := FromStringArray([]string{"my", "dir", "path"})
	require.Equal(t, q.Directory{q.String("my"), q.String("dir"), q.String("path")}, dir)
}

func TestToFDBTuple(t *testing.T) {
	tup, err := ToFDBTuple(q.Tuple{q.Nil{}, q.Int(22), q.Bool(false)})
	require.NoError(t, err)
	require.Equal(t, tuple.Tuple{nil, int64(22), false}, tup)

	tup, err = ToFDBTuple(q.Tuple{q.Bool(true), q.Tuple{q.Float(32.8), q.String("hi")}})
	require.NoError(t, err)
	require.Equal(t, tuple.Tuple{true, tuple.Tuple{32.8, "hi"}}, tup)
}

func TestFromFDBTuple(t *testing.T) {
	tup := FromFDBTuple(tuple.Tuple{nil, int64(22), false})
	require.Equal(t, q.Tuple{q.Nil{}, q.Int(22), q.Bool(false)}, tup)

	tup = FromFDBTuple(tuple.Tuple{true, tuple.Tuple{32.8, "hi"}})
	require.Equal(t, q.Tuple{q.Bool(true), q.Tuple{q.Float(32.8), q.String("hi")}}, tup)
}

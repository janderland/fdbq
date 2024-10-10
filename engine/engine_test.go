package engine

import (
	"context"
	"flag"
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/janderland/fql/engine/facade"
	"github.com/janderland/fql/engine/internal"
	q "github.com/janderland/fql/keyval"
)

var (
	force bool
)

func init() {
	fdb.MustAPIVersion(620)
	flag.BoolVar(&force, "force", false, "remove test directory if it exists")
}

func TestEngine_SetReadSingle(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			query := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi"), q.String("there")}, Tuple: q.Tuple{q.Float(33.3)}}, Value: q.Int(33)}

			err := e.Set(query)
			require.NoError(t, err)

			expected := query
			query.Value = q.Variable{q.IntType}

			result, err := e.ReadSingle(query, SingleOpts{})
			require.NoError(t, err)
			require.Equal(t, &expected, result)
		})
	})

	t.Run("set and get empty value", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			query := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi"), q.String("there")}, Tuple: q.Tuple{q.Float(33.3)}}, Value: q.Bytes{}}

			err := e.Set(query)
			require.NoError(t, err)

			expected := query
			query.Value = q.Variable{}

			result, err := e.ReadSingle(query, SingleOpts{})
			require.NoError(t, err)
			require.Equal(t, &expected, result)
		})
	})

	t.Run("get nothing", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			query := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("nothing"), q.String("here")}}, Value: q.Variable{}}

			result, err := e.ReadSingle(query, SingleOpts{})
			require.NoError(t, err)
			require.Nil(t, result)
		})
	})

	t.Run("set errors", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			query := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi")}, Tuple: q.Tuple{q.Float(32.33), q.Variable{}}}, Value: q.Nil{}}
			err := e.Set(query)
			require.Error(t, err)

			query = q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi")}, Tuple: q.Tuple{q.Float(32.33)}}, Value: q.Clear{}}
			err = e.Set(query)
			require.Error(t, err)
		})
	})

	t.Run("get errors", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			query := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi")}, Tuple: q.Tuple{q.Float(32.33), q.Variable{}}}, Value: q.Nil{}}
			result, err := e.ReadSingle(query, SingleOpts{})
			require.Error(t, err)
			require.Nil(t, result)

			query = q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi")}, Tuple: q.Tuple{q.Float(32.33)}}, Value: q.Clear{}}
			result, err = e.ReadSingle(query, SingleOpts{})
			require.Error(t, err)
			require.Nil(t, result)
		})
	})
}

func TestEngine_Clear(t *testing.T) {
	t.Run("set clear get", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			set := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("place")}, Tuple: q.Tuple{q.Float(32.33)}}, Value: q.Bytes{}}
			err := e.Set(set)
			require.NoError(t, err)

			get := set
			get.Value = q.Variable{}
			result, err := e.ReadSingle(get, SingleOpts{})
			require.NoError(t, err)
			require.Equal(t, &set, result)

			clear := set
			clear.Value = q.Clear{}
			err = e.Clear(clear)
			require.NoError(t, err)

			result, err = e.ReadSingle(get, SingleOpts{})
			require.NoError(t, err)
			require.Nil(t, result)
		})
	})

	t.Run("errors", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			query := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi")}, Tuple: q.Tuple{q.Float(32.33), q.Variable{}}}, Value: q.Clear{}}
			err := e.Clear(query)
			require.Error(t, err)

			query = q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi")}, Tuple: q.Tuple{q.Float(32.33)}}, Value: q.Nil{}}
			err = e.Clear(query)
			require.Error(t, err)
		})
	})
}

func TestEngine_ReadRange(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			var expected []q.KeyValue

			query := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("place")}, Tuple: q.Tuple{q.String("hi you")}}, Value: q.Bytes{}}
			expected = append(expected, query)
			err := e.Set(query)
			require.NoError(t, err)

			query.Key.Tuple = q.Tuple{q.Float(11.3)}
			expected = append(expected, query)
			err = e.Set(query)
			require.NoError(t, err)

			query.Key.Tuple = q.Tuple{q.Float(55.3)}
			expected = append(expected, query)
			err = e.Set(query)
			require.NoError(t, err)

			var results []q.KeyValue
			query.Key.Tuple = q.Tuple{q.Variable{}}
			for kve := range e.ReadRange(context.Background(), query, RangeOpts{}) {
				require.NoError(t, kve.Err)

				// The first element of the dir path is dropped because it
				// should be a random dir created by the test framework.
				kve.KV.Key.Directory = kve.KV.Key.Directory[1:]

				results = append(results, kve.KV)
			}
			require.Equal(t, expected, results)
		})
	})

	t.Run("errors", func(t *testing.T) {
		testEnv(t, func(e Engine) {
			query := q.KeyValue{Key: q.Key{Directory: q.Directory{q.String("hi")}, Tuple: q.Tuple{q.Float(32.33)}}, Value: q.Clear{}}
			out := e.ReadRange(context.Background(), query, RangeOpts{})

			msg := <-out
			require.Error(t, msg.Err)
			_, open := <-out
			require.False(t, open)
		})
	})
}

func TestEngine_Directories(t *testing.T) {
	t.Run("created and open", func(t *testing.T) {
		internal.TestEnv(t, force, func(tr facade.Transactor, log zerolog.Logger) {
			query := q.Directory{q.String("my"), q.Variable{}}
			paths := [][]string{
				{"my", "path"},
				{"my", "somewhere"},
			}

			var expected []directory.DirectorySubspace
			for _, p := range paths {
				dir, err := tr.DirCreateOrOpen(p)
				require.NoError(t, err)
				expected = append(expected, dir)
			}

			e := New(tr, Logger(log))

			var result []directory.DirectorySubspace
			for msg := range e.Directories(context.Background(), query) {
				require.NoError(t, msg.Err)
				result = append(result, msg.Dir)
			}
			require.Equal(t, expected, result)
		})
	})
}

func testEnv(t *testing.T, f func(Engine)) {
	internal.TestEnv(t, force, func(tr facade.Transactor, log zerolog.Logger) {
		f(New(tr, Logger(log)))
	})
}

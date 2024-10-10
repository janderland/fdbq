package stream

import (
	"context"
	"encoding/binary"
	"flag"
	"strings"
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/janderland/fql/engine/facade"
	"github.com/janderland/fql/engine/internal"
	q "github.com/janderland/fql/keyval"
	"github.com/janderland/fql/keyval/convert"
	"github.com/janderland/fql/keyval/values"
)

var (
	byteOrder binary.ByteOrder
	force     bool
)

func init() {
	fdb.MustAPIVersion(620)
	byteOrder = binary.BigEndian
	flag.BoolVar(&force, "force", false, "remove test directory if it exists")
}

func TestStream_OpenDirectories(t *testing.T) {
	var tests = []struct {
		name     string
		query    q.Directory
		initial  [][]string
		expected [][]string
	}{
		{
			name:  "no exist one",
			query: q.Directory{q.String("hello")},
		},
		{
			name:     "exist one",
			query:    q.Directory{q.String("hello")},
			initial:  [][]string{{"hello"}},
			expected: [][]string{{"hello"}},
		},
		{
			name:  "no exist many",
			query: q.Directory{q.String("people"), q.Variable{}},
		},
		{
			name:  "exist many",
			query: q.Directory{q.String("people"), q.Variable{}, q.String("job"), q.Variable{}},
			initial: [][]string{
				{"people", "billy", "job", "dancer"},
				{"people", "billy", "job", "tailor"},
				{"people", "jon", "job", "programmer"},
				{"people", "sally", "job", "designer"},
			},
			expected: [][]string{
				{"people", "billy", "job", "dancer"},
				{"people", "billy", "job", "tailor"},
				{"people", "jon", "job", "programmer"},
				{"people", "sally", "job", "designer"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEnv(t, func(tr facade.Transaction, s Stream) {
				for _, path := range test.initial {
					_, err := tr.DirCreateOrOpen(path)
					require.NoError(t, err, "failed to create directory")
				}

				ch := s.OpenDirectories(tr, test.query)
				dirs, err := collectDirs(ch)
				require.NoError(t, err)

				var actual [][]string
				for _, dir := range dirs {
					// The first element of the dir path is dropped because it
					// should be a random dir created by the test framework.
					actual = append(actual, dir.GetPath()[1:])
				}

				require.Equal(t, len(test.expected), len(actual), "unexpected number of directories")
				for i, expected := range test.expected {
					require.Equalf(t, expected, actual[i], "unexpected directory at index %d", i)
				}
			})
		})
	}
}

func TestStream_ReadRange(t *testing.T) {
	var tests = []struct {
		name     string
		query    q.Tuple
		initial  []q.KeyValue
		expected []q.KeyValue
	}{
		{
			name:  "no variable",
			query: q.Tuple{q.Int(123), q.String("hello"), q.Float(-50.6)},
			initial: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("first")}, Tuple: q.Tuple{q.Int(123), q.String("hello"), q.Float(-50.6)}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("first")}, Tuple: q.Tuple{q.Int(321), q.String("goodbye"), q.Float(50.6)}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("second")}, Tuple: q.Tuple{q.Int(-69), q.Int(-55), q.Tuple{q.String("world")}}}, Value: q.Nil{}},
			},
			expected: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("first")}, Tuple: q.Tuple{q.Int(123), q.String("hello"), q.Float(-50.6)}}, Value: q.Bytes{}},
			},
		},
		{
			name:  "variable",
			query: q.Tuple{q.Int(123), q.Variable{}, q.String("sing")},
			initial: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("thing")}, Tuple: q.Tuple{q.Int(123), q.String("song"), q.String("sing")}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("that"), q.String("there")}, Tuple: q.Tuple{q.Int(123), q.Float(13.45), q.String("sing")}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("iam")}, Tuple: q.Tuple{q.UUID{
					0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}}}, Value: q.Nil{}},
			},
			expected: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("thing")}, Tuple: q.Tuple{q.Int(123), q.String("song"), q.String("sing")}}, Value: q.Bytes{}},
				{Key: q.Key{Directory: q.Directory{q.String("that"), q.String("there")}, Tuple: q.Tuple{q.Int(123), q.Float(13.45), q.String("sing")}}, Value: q.Bytes{}},
			},
		},
		{
			name:  "read everything",
			query: q.Tuple{},
			initial: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("thing")}, Tuple: q.Tuple{q.Int(123), q.String("song"), q.String("sing")}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("that"), q.String("there")}, Tuple: q.Tuple{q.Int(123), q.Float(13.45), q.String("sing")}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("iam")}, Tuple: q.Tuple{
					q.UUID{0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}}}, Value: q.Nil{}},
			},
			expected: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("thing")}, Tuple: q.Tuple{q.Int(123), q.String("song"), q.String("sing")}}, Value: q.Bytes{}},
				{Key: q.Key{Directory: q.Directory{q.String("that"), q.String("there")}, Tuple: q.Tuple{q.Int(123), q.Float(13.45), q.String("sing")}}, Value: q.Bytes{}},
				{Key: q.Key{Directory: q.Directory{q.String("iam")}, Tuple: q.Tuple{
					q.UUID{0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}}}, Value: q.Bytes{}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEnv(t, func(tr facade.Transaction, s Stream) {
				dirsByPath, uniqueDirs := openDirs(t, tr, test.initial)

				initial := buildDirKVs(t, dirsByPath, test.initial)
				for _, dirKV := range initial {
					tr.Set(dirKV.kv.Key, dirKV.kv.Value)
				}

				expected := buildDirKVs(t, dirsByPath, test.expected)

				ch := s.ReadRange(tr, test.query, RangeOpts{}, sendDirs(t, s, uniqueDirs))
				actual, err := collectDirKVs(ch)
				require.NoError(t, err, "failed to read range")

				require.Equal(t, len(expected), len(actual), "unexpected number of results")
				for i := range expected {
					require.Equalf(t, expected[i], actual[i], "unexpected directory at index %d", i)
				}
			})
		})
	}
}

func TestStream_UnpackKeys(t *testing.T) {
	var tests = []struct {
		name     string
		filter   bool
		query    q.Tuple
		initial  []q.KeyValue
		expected []q.KeyValue
		err      bool
	}{
		{
			name:   "no variable",
			filter: true,
			query:  q.Tuple{q.Int(123), q.String("hello"), q.Float(-50.6)},
			initial: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("first")}, Tuple: q.Tuple{q.Int(123), q.String("hello"), q.Float(-50.6)}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("first")}, Tuple: q.Tuple{q.Int(321), q.String("goodbye"), q.Float(50.6)}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("second")}, Tuple: q.Tuple{q.Int(-69), q.Int(-55), q.Tuple{q.String("world")}}}, Value: q.Nil{}},
			},
			expected: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("first")}, Tuple: q.Tuple{q.Int(123), q.String("hello"), q.Float(-50.6)}}, Value: q.Bytes(nil)},
			},
		},
		{
			name:   "variable",
			filter: true,
			query:  q.Tuple{q.Int(123), q.Variable{}, q.String("sing")},
			initial: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("thing")}, Tuple: q.Tuple{q.Int(123), q.String("song"), q.String("sing")}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("that"), q.String("there")}, Tuple: q.Tuple{q.Int(123), q.Float(13.45), q.String("sing")}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("iam")}, Tuple: q.Tuple{q.UUID{
					0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}}}, Value: q.Nil{}},
			},
			expected: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("thing")}, Tuple: q.Tuple{q.Int(123), q.String("song"), q.String("sing")}}, Value: q.Bytes(nil)},
				{Key: q.Key{Directory: q.Directory{q.String("that"), q.String("there")}, Tuple: q.Tuple{q.Int(123), q.Float(13.45), q.String("sing")}}, Value: q.Bytes(nil)},
			},
		},
		{
			name:   "read everything",
			filter: true,
			query:  q.Tuple{},
			initial: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("thing")}, Tuple: q.Tuple{q.Int(123), q.String("song"), q.String("sing")}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("that"), q.String("there")}, Tuple: q.Tuple{q.Int(123), q.Float(13.45), q.String("sing")}}, Value: q.Nil{}},
				{Key: q.Key{Directory: q.Directory{q.String("iam")}, Tuple: q.Tuple{
					q.UUID{0xbc, 0xef, 0xd2, 0xec, 0x4d, 0xf5, 0x43, 0xb6, 0x8c, 0x79, 0x81, 0xb7, 0x0b, 0x88, 0x6a, 0xf9}}}, Value: q.Nil{}},
			},
		},
		{
			name:  "non-filter err",
			query: q.Tuple{q.Int(123), q.Variable{q.IntType}, q.String("sing")},
			initial: []q.KeyValue{
				{Key: q.Key{Directory: q.Directory{q.String("this"), q.String("thing")}, Tuple: q.Tuple{q.Int(123), q.String("song"), q.String("sing")}}, Value: q.Nil{}},
			},
			err: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEnv(t, func(tr facade.Transaction, s Stream) {
				dirsByPath, _ := openDirs(t, tr, test.initial)
				dirKVs := buildDirKVs(t, dirsByPath, test.initial)

				ch := s.UnpackKeys(test.query, test.filter, sendDirKVs(t, s, dirKVs))
				actual, err := collectKVs(ch)
				if test.err {
					require.Error(t, err, "failed to unpack keys")
				} else {
					require.NoError(t, err, "successfully unpacked keys")
				}

				for i := range actual {
					// The first element of the dir path is dropped because it
					// should be a random dir created by the test framework.
					actual[i].Key.Directory = actual[i].Key.Directory[1:]
				}

				require.Equal(t, len(test.expected), len(actual), "unexpected number of key-values")
				for i, expected := range test.expected {
					require.Equalf(t, expected, actual[i], "unexpected key-value at index %d", i)
				}
			})
		})
	}
}

func TestStream_UnpackValues(t *testing.T) {
	var tests = []struct {
		name     string
		query    q.Value
		initial  []q.KeyValue
		expected []q.KeyValue
	}{
		{
			name:  "no variable",
			query: q.Int(123),
			initial: []q.KeyValue{
				{Value: packWithPanic(q.Int(123))},
				{Value: packWithPanic(q.String("hello world"))},
				{Value: q.Bytes{}},
			},
			expected: []q.KeyValue{
				{Value: q.Int(123)},
			},
		},
		{
			name:  "variable",
			query: q.Variable{q.IntType, q.UUIDType, q.TupleType},
			initial: []q.KeyValue{
				{Value: packWithPanic(q.String("hello world"))},
				{Value: packWithPanic(q.Int(55))},
				{Value: packWithPanic(q.Float(23.9))},
				{Value: packWithPanic(q.Tuple{q.String("there we go"), q.Nil{}})},
			},
			expected: []q.KeyValue{
				{Value: q.Int(55)},
				{Value: unpackWithPanic(q.IntType, packWithPanic(q.Float(23.9)))},
				{Value: q.Tuple{q.String("there we go"), q.Nil{}}},
			},
		},
		{
			name:  "empty variable",
			query: q.Variable{},
			initial: []q.KeyValue{
				{Value: packWithPanic(q.Int(55))},
				{Value: packWithPanic(q.Float(23.9))},
				{Value: packWithPanic(q.Tuple{q.String("there we go"), q.Nil{}})},
			},
			expected: []q.KeyValue{
				{Value: packWithPanic(q.Int(55))},
				{Value: packWithPanic(q.Float(23.9))},
				{Value: packWithPanic(q.Tuple{q.String("there we go"), q.Nil{}})},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEnv(t, func(_ facade.Transaction, s Stream) {
				ch := s.UnpackValues(test.query, true, sendKVs(t, s, test.initial))
				kvs, err := collectKVs(ch)
				require.NoError(t, err, "failed to unpack values")

				require.Equal(t, len(test.expected), len(kvs), "unexpected number of key-values")
				for i, expected := range test.expected {
					require.Equalf(t, expected, kvs[i], "unexpected key-value at index %d", i)
				}
			})
		})
	}
}

func TestSplitAtFirstVariable(t *testing.T) {
	prefix, variable, suffix := splitAtFirstVariable(q.Directory{
		q.String("one"), q.Variable{q.FloatType}, q.String("-39.9"),
	})
	require.Equal(t, q.Directory{q.String("one")}, prefix)
	require.Equal(t, &q.Variable{q.FloatType}, variable)
	require.Equal(t, q.Directory{q.String("-39.9")}, suffix)
}

func TestToTuplePrefix(t *testing.T) {
	prefix := toTuplePrefix(q.Tuple{
		q.String("one"), q.Int(55), q.Variable{q.FloatType}, q.Tuple{q.Float(-39.9)},
	})
	require.Equal(t, q.Tuple{q.String("one"), q.Int(55)}, prefix)
}

func testEnv(t *testing.T, f func(facade.Transaction, Stream)) {
	internal.TestEnv(t, force, func(tr facade.Transactor, log zerolog.Logger) {
		_, err := tr.Transact(func(tr facade.Transaction) (interface{}, error) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			f(tr, New(ctx, Logger(log)))
			return nil, nil
		})
		if err != nil {
			t.Fatal(errors.Wrap(err, "transaction failed"))
		}
	})
}

func packWithPanic(val q.Value) q.Bytes {
	packed, err := values.Pack(val, byteOrder)
	if err != nil {
		panic(err)
	}
	return packed
}

func unpackWithPanic(typ q.ValueType, bytes q.Bytes) q.Value {
	unpacked, err := values.Unpack(bytes, typ, byteOrder)
	if err != nil {
		panic(err)
	}
	return unpacked
}

type DirKV struct {
	dir directory.DirectorySubspace
	kv  fdb.KeyValue
}

func openDirs(t *testing.T, tr facade.Transaction, kvs []q.KeyValue) (map[string]directory.DirectorySubspace, []directory.DirectorySubspace) {
	var (
		dirsByPath = make(map[string]directory.DirectorySubspace)
		uniqueDirs []directory.DirectorySubspace
	)

	for _, kv := range kvs {
		path, err := convert.ToStringArray(kv.Key.Directory)
		require.NoError(t, err, "failed to convert to string array")

		pathStr := strings.Join(path, "/")

		if _, exists := dirsByPath[pathStr]; !exists {
			t.Logf("adding to dir list: %v", path)

			dir, err := tr.DirCreateOrOpen(path)
			require.NoError(t, err, "failed to create or open directory")

			dirsByPath[pathStr] = dir
			uniqueDirs = append(uniqueDirs, dir)
		}
	}

	return dirsByPath, uniqueDirs
}

func buildDirKVs(t *testing.T, dirs map[string]directory.DirectorySubspace, kvs []q.KeyValue) []DirKV {
	var out []DirKV

	for _, kv := range kvs {
		path, err := convert.ToStringArray(kv.Key.Directory)
		require.NoError(t, err, "failed to convert to string array")

		pathStr := strings.Join(path, "/")
		dir, ok := dirs[pathStr]
		require.Truef(t, ok, "%s wasn't provided", pathStr)

		tup, err := convert.ToFDBTuple(kv.Key.Tuple)
		require.NoError(t, err, "failed to convert to FDB tuple")

		val, err := values.Pack(kv.Value, byteOrder)
		require.NoError(t, err, "failed to pack value")

		out = append(out, DirKV{
			dir: dir,
			kv:  fdb.KeyValue{Key: dir.Pack(tup), Value: val},
		})
	}

	return out
}

func collectDirs(in chan DirErr) ([]directory.DirectorySubspace, error) {
	var out []directory.DirectorySubspace

	for msg := range in {
		if msg.Err != nil {
			return nil, msg.Err
		}
		out = append(out, msg.Dir)
	}

	return out, nil
}

func collectDirKVs(in chan DirKVErr) ([]DirKV, error) {
	var out []DirKV

	for msg := range in {
		if msg.Err != nil {
			return nil, msg.Err
		}
		out = append(out, DirKV{
			dir: msg.Dir,
			kv:  msg.KV,
		})
	}

	return out, nil
}

func collectKVs(in chan KeyValErr) ([]q.KeyValue, error) {
	var out []q.KeyValue

	for msg := range in {
		if msg.Err != nil {
			return nil, msg.Err
		}
		out = append(out, msg.KV)
	}

	return out, nil
}

func sendDirs(t *testing.T, s Stream, in []directory.DirectorySubspace) chan DirErr {
	out := make(chan DirErr)

	go func() {
		defer close(out)
		for _, dir := range in {
			if !s.SendDir(out, DirErr{Dir: dir}) {
				return
			}
			t.Logf("sent dir: %s", dir.GetPath())
		}
	}()

	return out
}

func sendDirKVs(t *testing.T, s Stream, in []DirKV) chan DirKVErr {
	out := make(chan DirKVErr)

	go func() {
		defer close(out)
		for _, dirKV := range in {
			if !s.SendDirKV(out, DirKVErr{Dir: dirKV.dir, KV: dirKV.kv}) {
				return
			}
			t.Logf("sent dir-kv: %+v", dirKV.dir.GetPath())
		}
	}()

	return out
}

func sendKVs(t *testing.T, s Stream, in []q.KeyValue) chan KeyValErr {
	out := make(chan KeyValErr)

	go func() {
		defer close(out)
		for _, kv := range in {
			if !s.SendKV(out, KeyValErr{KV: kv}) {
				return
			}
			t.Logf("sent kv: %+v", kv)
		}
	}()

	return out
}

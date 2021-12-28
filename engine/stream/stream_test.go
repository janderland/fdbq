package stream

import (
	"context"
	"encoding/binary"
	"flag"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/janderland/fdbq/engine/facade"
	"github.com/janderland/fdbq/engine/internal"
	q "github.com/janderland/fdbq/keyval"
	"github.com/janderland/fdbq/keyval/convert"
	"github.com/janderland/fdbq/keyval/values"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const root = "root"

var (
	db        fdb.Database
	byteOrder binary.ByteOrder

	flags struct {
		force bool
	}
)

func init() {
	fdb.MustAPIVersion(620)
	db = fdb.MustOpenDefault()
	byteOrder = binary.BigEndian

	flag.BoolVar(&flags.force, "force", false, "remove test directory if it exists")
}

func TestStream_OpenDirectories(t *testing.T) {
	var tests = []struct {
		name     string
		query    q.Directory
		initial  [][]string
		expected [][]string
		error    bool
	}{
		{
			name:  "no exist one",
			query: q.Directory{q.String("hello")},
			error: true,
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
			error: true,
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
			testEnv(t, func(tr fdb.Transaction, rootDir directory.DirectorySubspace, s Stream) {
				for _, path := range test.initial {
					_, err := rootDir.Create(tr, path, nil)
					if !assert.NoError(t, err) {
						t.FailNow()
					}
				}

				out := s.OpenDirectories(facade.NewReadTransaction(tr), append(convert.FromStringArray(rootDir.GetPath()), test.query...))
				directories, err := collectDirs(out)
				if test.error {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				if !assert.Equalf(t, len(test.expected), len(directories), "unexpected number of directories") {
					t.FailNow()
				}
				for i, expected := range test.expected {
					expected = append(rootDir.GetPath(), expected...)
					if !assert.Equalf(t, expected, directories[i].GetPath(), "unexpected directory at index %d", i) {
						t.FailNow()
					}
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
				{Key: q.Key{Directory: q.Directory{q.String("second")}, Tuple: q.Tuple{q.Int(-69), q.BigInt(*big.NewInt(-55)), q.Tuple{q.String("world")}}}, Value: q.Nil{}},
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
			testEnv(t, func(tr fdb.Transaction, rootDir directory.DirectorySubspace, s Stream) {
				var (
					byPath = make(map[string]directory.DirectorySubspace)
					toSend []directory.DirectorySubspace
				)
				for _, kv := range test.initial {
					path, err := convert.ToStringArray(kv.Key.Directory)
					if !assert.NoError(t, err) {
						t.FailNow()
					}
					tup, err := convert.ToFDBTuple(kv.Key.Tuple)
					if !assert.NoError(t, err) {
						t.FailNow()
					}
					dir, err := rootDir.CreateOrOpen(tr, path, nil)
					if !assert.NoError(t, err) {
						t.FailNow()
					}
					tr.Set(dir.Pack(tup), nil)

					pathStr := strings.Join(path, "/")
					if _, exists := byPath[pathStr]; !exists {
						t.Logf("adding to dir list: %v", path)
						byPath[pathStr] = dir
						toSend = append(toSend, dir)
					}
				}

				var (
					expectedDirs []directory.DirectorySubspace
					expectedKVs  []fdb.KeyValue
				)
				for _, kv := range test.expected {
					path, err := convert.ToStringArray(kv.Key.Directory)
					require.NoError(t, err)

					dir, exists := byPath[strings.Join(path, "/")]
					require.Truef(t, exists, "dir missing for path %v", path)

					tup, err := convert.ToFDBTuple(kv.Key.Tuple)
					require.NoError(t, err)

					val, err := values.Pack(kv.Value, byteOrder)
					require.NoError(t, err)

					expectedDirs = append(expectedDirs, dir)
					expectedKVs = append(expectedKVs, fdb.KeyValue{
						Key:   dir.Pack(tup),
						Value: val,
					})
				}

				out := s.ReadRange(facade.NewReadTransaction(tr), test.query, RangeOpts{}, sendDirs(t, s, toSend))
				dirs, kvs, err := collectDirKVs(out)
				assert.NoError(t, err)

				require.Equal(t, len(expectedDirs), len(dirs), "unexpected number of results")
				for i := range expectedDirs {
					require.Equalf(t, expectedDirs[i], dirs[i], "unexpected directory at index %d", i)
					require.Equalf(t, expectedKVs[i], kvs[i], "unexpected key-value at index %d", i)
				}
			})
		})
	}
}

func TestStream_FilterKeys(t *testing.T) {
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
				{Key: q.Key{Directory: q.Directory{q.String("second")}, Tuple: q.Tuple{q.Int(-69), q.BigInt(*big.NewInt(-55)), q.Tuple{q.String("world")}}}, Value: q.Nil{}},
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
			testEnv(t, func(tr fdb.Transaction, rootDir directory.DirectorySubspace, s Stream) {
				var (
					dirsToSend []directory.DirectorySubspace
					kvsToSend  []fdb.KeyValue
				)
				for _, kv := range test.initial {
					path, err := convert.ToStringArray(kv.Key.Directory)
					require.NoError(t, err)

					dir, err := rootDir.CreateOrOpen(tr, path, nil)
					require.NoError(t, err)

					tup, err := convert.ToFDBTuple(kv.Key.Tuple)
					require.NoError(t, err)

					val, err := values.Pack(kv.Value, byteOrder)
					require.NoError(t, err)

					dirsToSend = append(dirsToSend, dir)
					kvsToSend = append(kvsToSend, fdb.KeyValue{Key: dir.Pack(tup), Value: val})
				}

				out := s.FilterKeys(test.query, test.filter, sendDirKVs(t, s, dirsToSend, kvsToSend))
				kvs, err := collectKVs(out)
				if test.err {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}

				rootPath := convert.FromStringArray(rootDir.GetPath())
				require.Equal(t, len(test.expected), len(kvs), "unexpected number of key-values")
				for i, expected := range test.expected {
					expected.Key.Directory = append(rootPath, expected.Key.Directory...)
					require.Equalf(t, expected, kvs[i], "unexpected key-value at index %d", i)
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
			query: q.Variable{q.IntType, q.BigIntType, q.TupleType},
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
			testEnv(t, func(tr fdb.Transaction, rootDir directory.DirectorySubspace, s Stream) {
				valHandler, err := internal.NewValueHandler(test.query, byteOrder, true)
				require.NoError(t, err)

				out := s.UnpackValues(test.query, valHandler, sendKVs(t, s, test.initial))
				kvs, err := collectKVs(out)
				require.NoError(t, err)

				require.Equal(t, len(test.expected), len(kvs), "unexpected number of key-values")
				for i, expected := range test.expected {
					require.Equalf(t, expected, kvs[i], "unexpected key-value at index %d", i)
				}
			})
		})
	}
}

func testEnv(t *testing.T, f func(fdb.Transaction, directory.DirectorySubspace, Stream)) {
	exists, err := directory.Exists(db, []string{root})
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to check if root directory exists"))
	}
	if exists {
		if !flags.force {
			t.Fatal(errors.New("test directory already exists, use '-force' flag to remove"))
		}
		if _, err := directory.Root().Remove(db, []string{root}); err != nil {
			t.Fatal(errors.Wrap(err, "failed to remove directory"))
		}
	}

	dir, err := directory.Create(db, []string{root}, nil)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to create test directory"))
	}
	defer func() {
		_, err := directory.Root().Remove(db, []string{root})
		if err != nil {
			t.Error(errors.Wrap(err, "failed to clean root directory"))
		}
	}()

	writer := zerolog.ConsoleWriter{Out: os.Stdout}
	writer.FormatLevel = func(_ interface{}) string { return "" }
	writer.FormatTimestamp = func(_ interface{}) string { return "" }
	log := zerolog.New(writer)

	_, err = db.Transact(func(tr fdb.Transaction) (interface{}, error) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		f(tr, dir, Stream{Ctx: ctx, Log: log})
		return nil, nil
	})
	if err != nil {
		t.Fatal(errors.Wrap(err, "transaction failed"))
	}
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

func collectDirKVs(in chan DirKVErr) ([]directory.DirectorySubspace, []fdb.KeyValue, error) {
	var (
		dirs []directory.DirectorySubspace
		kvs  []fdb.KeyValue
	)
	for msg := range in {
		if msg.Err != nil {
			return nil, nil, msg.Err
		}
		dirs = append(dirs, msg.Dir)
		kvs = append(kvs, msg.KV)
	}
	return dirs, kvs, nil
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

func sendDirKVs(t *testing.T, s Stream, dirs []directory.DirectorySubspace, kvs []fdb.KeyValue) chan DirKVErr {
	out := make(chan DirKVErr)

	go func() {
		defer close(out)
		for i, dir := range dirs {
			if !s.SendDirKV(out, DirKVErr{Dir: dir, KV: kvs[i]}) {
				return
			}
			t.Logf("sent dir-kv: %+v", dir.GetPath())
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

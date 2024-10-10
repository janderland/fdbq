// Package engine executes queries.
package engine

import (
	"context"
	"encoding/binary"

	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/janderland/fql/engine/facade"
	"github.com/janderland/fql/engine/internal"
	"github.com/janderland/fql/engine/stream"
	"github.com/janderland/fql/keyval"
	"github.com/janderland/fql/keyval/class"
	"github.com/janderland/fql/keyval/convert"
	"github.com/janderland/fql/keyval/values"
)

// SingleOpts configures how an [Engine.ReadSingle] call is executed.
type SingleOpts struct {
	Filter bool
}

// RangeOpts configures how an [Engine.ReadRange] call is executed.
type RangeOpts struct {
	Reverse bool
	Filter  bool
	Limit   int
}

func (x RangeOpts) forStream() stream.RangeOpts {
	return stream.RangeOpts{
		Reverse: x.Reverse,
		Limit:   x.Limit,
	}
}

// Option can be passed as a trailing argument to the New function
// to modify properties of the created Engine.
type Option func(*Engine)

// Engine provides methods which execute queries. Each valid [class.Class]
// has a corresponding method for executing that class of query. The methods
// will fail if a query of the wrong class in provided. Unless [Engine.Transact]
// is used, each query is executed in its own transaction.
type Engine struct {
	tr    facade.Transactor
	log   zerolog.Logger
	order binary.ByteOrder
}

func New(tr facade.Transactor, opts ...Option) Engine {
	eg := Engine{
		tr:    tr,
		log:   zerolog.Nop(),
		order: binary.BigEndian,
	}
	for _, option := range opts {
		option(&eg)
	}
	return eg
}

// Logger enables debug logging using the provided logger. This method
// must not be called concurrently with other methods.
func Logger(log zerolog.Logger) Option {
	return func(eg *Engine) {
		eg.log = log
	}
}

// ByteOrder sets the endianness used for encoding/decoding values. This
// method must not be called concurrently with other methods.
func ByteOrder(order binary.ByteOrder) Option {
	return func(eg *Engine) {
		eg.order = order
	}
}

// Transact wraps a group of Engine method calls under a single transaction. The newly
// created Engine inherits the logger & byte order of the parent engine. Any changes to
// the logger or byte order of the new Engine has no effect on the parent Engine.
func (x *Engine) Transact(f func(Engine) (interface{}, error)) (interface{}, error) {
	return x.tr.Transact(func(tr facade.Transaction) (interface{}, error) {
		return f(Engine{
			tr:    tr,
			log:   x.log,
			order: x.order,
		})
	})
}

// Set preforms a write operation for a single key-value. The given query must
// belong to [class.Constant].
func (x *Engine) Set(query keyval.KeyValue) error {
	if class.Classify(query) != class.Constant {
		return errors.New("query not constant class")
	}

	path, err := convert.ToStringArray(query.Key.Directory)
	if err != nil {
		return errors.Wrap(err, "failed to convert directory to string array")
	}

	valueBytes, err := values.Pack(query.Value, x.order)
	if err != nil {
		return errors.Wrap(err, "failed to pack value")
	}

	_, err = x.tr.Transact(func(tr facade.Transaction) (interface{}, error) {
		x.log.Log().Interface("query", query).Msg("setting")

		dir, err := tr.DirCreateOrOpen(path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open directory")
		}

		tup, err := convert.ToFDBTuple(query.Key.Tuple)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert to FDB tuple")
		}

		tr.Set(dir.Pack(tup), valueBytes)
		return nil, nil
	})
	return errors.Wrap(err, "transaction failed")
}

// Clear performs a clear operation for a single key-value. The given query
// must belong to [class.Clear].
func (x *Engine) Clear(query keyval.KeyValue) error {
	if class.Classify(query) != class.Clear {
		return errors.New("query not clear class")
	}

	path, err := convert.ToStringArray(query.Key.Directory)
	if err != nil {
		return errors.Wrap(err, "failed to convert directory to string array")
	}

	_, err = x.tr.Transact(func(tr facade.Transaction) (interface{}, error) {
		x.log.Log().Interface("query", query).Msg("clearing")

		dir, err := tr.DirOpen(path)
		if err != nil {
			if errors.Is(err, directory.ErrDirNotExists) {
				return nil, nil
			}
			return nil, errors.Wrap(err, "failed to open directory")
		}

		tup, err := convert.ToFDBTuple(query.Key.Tuple)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert to FDB tuple")
		}

		tr.Clear(dir.Pack(tup))
		return nil, nil
	})
	return errors.Wrap(err, "transaction failed")
}

// ReadSingle performs a read operation for a single key-value. The given query must
// belong to [class.ReadSingle].
func (x *Engine) ReadSingle(query keyval.KeyValue, opts SingleOpts) (*keyval.KeyValue, error) {
	if class.Classify(query) != class.ReadSingle {
		return nil, errors.New("query not single-read class")
	}

	path, err := convert.ToStringArray(query.Key.Directory)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert directory to string array")
	}

	valHandler, err := internal.NewValueHandler(query.Value, x.order, opts.Filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init value handler")
	}

	var valBytes []byte
	_, err = x.tr.Transact(func(tr facade.Transaction) (interface{}, error) {
		x.log.Log().Interface("query", query).Msg("single reading")

		dir, err := tr.DirOpen(path)
		if err != nil {
			if errors.Is(err, directory.ErrDirNotExists) {
				return nil, nil
			}
			return nil, errors.Wrap(err, "failed to open directory")
		}

		tup, err := convert.ToFDBTuple(query.Key.Tuple)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert to FDB tuple")
		}

		valBytes = tr.Get(dir.Pack(tup)).MustGet()
		return nil, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "transaction failed")
	}

	value, err := valHandler.Handle(valBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack value")
	}
	if value == nil {
		return nil, nil
	}
	return &keyval.KeyValue{
		Key:   query.Key,
		Value: value,
	}, nil
}

// ReadRange performs a read across a range of key-values. The given query must belong to [class.ReadRange].
// After an error occurs or the entire range is read, the returned channel is closed. If the provided context
// is canceled, then the read operation will be stopped after the latest FDB call finishes.
func (x *Engine) ReadRange(ctx context.Context, query keyval.KeyValue, opts RangeOpts) chan stream.KeyValErr {
	out := make(chan stream.KeyValErr)

	go func() {
		defer close(out)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		s := stream.New(ctx, stream.Logger(x.log), stream.ByteOrder(x.order))

		if class.Classify(query) != class.ReadRange {
			s.SendKV(out, stream.KeyValErr{Err: errors.New("query not range-read class")})
			return
		}

		_, err := x.tr.ReadTransact(func(tr facade.ReadTransaction) (interface{}, error) {
			stage1 := s.OpenDirectories(tr, query.Key.Directory)
			stage2 := s.ReadRange(tr, query.Key.Tuple, opts.forStream(), stage1)
			stage3 := s.UnpackKeys(query.Key.Tuple, opts.Filter, stage2)
			for kve := range s.UnpackValues(query.Value, opts.Filter, stage3) {
				s.SendKV(out, kve)
			}
			return nil, nil
		})
		if err != nil {
			s.SendKV(out, stream.KeyValErr{Err: errors.Wrap(err, "transaction failed")})
		}
	}()

	return out
}

// Directories reads directories from the directory layer. If the query contains a [keyval.Variable],
// multiple directories may be returned. If the query doesn't contain a [keyval.Variable], at most a
// single directory will be returned. After an error occurs or all directories have been read, the
// returned channel is closed. If the provided context is canceled, then the read operation will
// be stopped after the latest FDB call finishes.
func (x *Engine) Directories(ctx context.Context, query keyval.Directory) chan stream.DirErr {
	out := make(chan stream.DirErr)

	go func() {
		defer close(out)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		s := stream.New(ctx, stream.Logger(x.log))

		_, err := x.tr.ReadTransact(func(tr facade.ReadTransaction) (interface{}, error) {
			for dir := range s.OpenDirectories(tr, query) {
				s.SendDir(out, dir)
			}
			return nil, nil
		})
		if err != nil {
			s.SendDir(out, stream.DirErr{Err: errors.Wrap(err, "transaction failed")})
		}
	}()

	return out
}

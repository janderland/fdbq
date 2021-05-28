package stream

import (
	"bytes"
	"context"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/janderland/fdbq/keyval"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type (
	Stream struct {
		ctx    context.Context
		cancel context.CancelFunc
		log    *zerolog.Logger
	}

	DirErr struct {
		Dir directory.DirectorySubspace
		Err error
	}

	KeyValErr struct {
		KV  keyval.KeyValue
		Err error
	}
)

func New(ctx context.Context) Stream {
	ctx, cancel := context.WithCancel(ctx)

	return Stream{
		ctx:    ctx,
		cancel: cancel,
		log:    zerolog.Ctx(ctx),
	}
}

func (r *Stream) OpenDirectories(tr fdb.ReadTransactor, query keyval.KeyValue) chan DirErr {
	out := make(chan DirErr)

	go func() {
		defer close(out)
		r.doOpenDirectories(tr, query.Key.Directory, out)
	}()

	return out
}

func (r *Stream) ReadRange(tr fdb.ReadTransaction, query keyval.KeyValue, in chan DirErr) chan KeyValErr {
	out := make(chan KeyValErr)

	go func() {
		defer close(out)
		r.doReadRange(tr, query.Key.Tuple, in, out)
	}()

	return out
}

func (r *Stream) FilterKeys(query keyval.KeyValue, in chan KeyValErr) chan KeyValErr {
	out := make(chan KeyValErr)

	go func() {
		defer close(out)
		r.doFilterKeys(query.Key.Tuple, in, out)
	}()

	return out
}

func (r *Stream) UnpackValues(query keyval.KeyValue, in chan KeyValErr) chan KeyValErr {
	out := make(chan KeyValErr)

	go func() {
		defer close(out)
		r.doUnpackValues(query.Value, in, out)
	}()

	return out
}

func (r *Stream) doOpenDirectories(tr fdb.ReadTransactor, query keyval.Directory, out chan DirErr) {
	log := r.log.With().Str("stage", "open directories").Interface("query", query).Logger()

	prefix, variable, suffix := keyval.SplitAtFirstVariable(query)
	prefixStr, err := keyval.ToStringArray(prefix)
	if err != nil {
		r.sendDir(out, DirErr{Err: errors.Wrapf(err, "failed to convert directory prefix to string array")})
		return
	}

	if variable != nil {
		subDirs, err := directory.List(tr, prefixStr)
		if err != nil {
			r.sendDir(out, DirErr{Err: errors.Wrap(err, "failed to list directories")})
			return
		}
		if len(subDirs) == 0 {
			r.sendDir(out, DirErr{Err: errors.Errorf("no subdirectories for %v", prefixStr)})
			return
		}

		log.Trace().Strs("sub dirs", subDirs).Msg("found subdirectories")

		for _, subDir := range subDirs {
			var dir keyval.Directory
			dir = append(dir, prefix...)
			dir = append(dir, subDir)
			dir = append(dir, suffix...)
			r.doOpenDirectories(tr, dir, out)
		}
	} else {
		dir, err := directory.Open(tr, prefixStr, nil)
		if err != nil {
			r.sendDir(out, DirErr{Err: errors.Wrapf(err, "failed to open directory %v", prefixStr)})
			return
		}

		log.Debug().Strs("dir", dir.GetPath()).Msg("sending directory")
		r.sendDir(out, DirErr{Dir: dir})
	}
}

func (r *Stream) doReadRange(tr fdb.ReadTransaction, query keyval.Tuple, in chan DirErr, out chan KeyValErr) {
	log := r.log.With().Str("stage", "read range").Interface("query", query).Logger()

	prefix, _, _ := keyval.SplitAtFirstVariable(query)
	fdbPrefix := keyval.ToFDBTuple(prefix)

	for msg := r.recvDir(in); msg != nil; msg = r.recvDir(in) {
		if msg.Err != nil {
			r.sendKV(out, KeyValErr{Err: errors.Wrap(msg.Err, "read range input closed")})
			return
		}

		dir := msg.Dir
		log := log.With().Strs("dir", dir.GetPath()).Logger()
		log.Debug().Msg("received directory")

		rng, err := fdb.PrefixRange(dir.Pack(fdbPrefix))
		if err != nil {
			r.sendKV(out, KeyValErr{Err: errors.Wrap(err, "failed to create prefix range")})
			return
		}

		iter := tr.GetRange(rng, fdb.RangeOptions{}).Iterator()
		for iter.Advance() {
			fromDB, err := iter.Get()
			if err != nil {
				r.sendKV(out, KeyValErr{Err: errors.Wrap(err, "failed to get key-value")})
				return
			}

			tup, err := dir.Unpack(fromDB.Key)
			if err != nil {
				r.sendKV(out, KeyValErr{Err: errors.Wrap(err, "failed to unpack key")})
				return
			}

			kv := keyval.KeyValue{
				Key: keyval.Key{
					Directory: keyval.FromStringArray(dir.GetPath()),
					Tuple:     keyval.FromFDBTuple(tup),
				},
				Value: fromDB.Value,
			}

			log.Debug().Interface("kv", kv).Msg("sending key-value")
			r.sendKV(out, KeyValErr{KV: kv})
		}
	}
}

func (r *Stream) doFilterKeys(query keyval.Tuple, in chan KeyValErr, out chan KeyValErr) {
	log := r.log.With().Str("stage", "filter keys").Interface("query", query).Logger()

	for msg := r.recvKV(in); msg != nil; msg = r.recvKV(in) {
		if msg.Err != nil {
			r.sendKV(out, KeyValErr{Err: errors.Wrap(msg.Err, "filter keys input closed")})
			return
		}

		kv := msg.KV
		log := log.With().Interface("kv", kv).Logger()
		log.Debug().Msg("received key-value")

		if keyval.CompareTuples(query, kv.Key.Tuple) == nil {
			log.Debug().Msg("sending key-value")
			r.sendKV(out, KeyValErr{KV: kv})
		}
	}
}

func (r *Stream) doUnpackValues(query keyval.Value, in chan KeyValErr, out chan KeyValErr) {
	log := r.log.With().Str("stage", "unpack values").Interface("query", query).Logger()

	if variable, isVar := query.(keyval.Variable); isVar {
		for msg := r.recvKV(in); msg != nil; msg = r.recvKV(in) {
			if msg.Err != nil {
				r.sendKV(out, KeyValErr{Err: msg.Err})
				return
			}

			kv := msg.KV
			log := log.With().Interface("kv", kv).Logger()
			log.Debug().Msg("received key-value")

			for _, typ := range variable {
				outVal, err := keyval.UnpackValue(typ, kv.Value.([]byte))
				if err != nil {
					continue
				}

				kv.Value = outVal
				log.Debug().Interface("kv", kv).Msg("sending key-value")
				r.sendKV(out, KeyValErr{KV: kv})
				break
			}
		}
	} else {
		queryBytes, err := keyval.PackValue(query)
		if err != nil {
			r.sendKV(out, KeyValErr{Err: errors.Wrap(err, "failed to pack query value")})
			return
		}

		for msg := r.recvKV(in); msg != nil; msg = r.recvKV(in) {
			if msg.Err != nil {
				r.sendKV(out, KeyValErr{Err: msg.Err})
				return
			}

			kv := msg.KV
			log := log.With().Interface("kv", kv).Logger()
			log.Debug().Msg("received key-value")

			if bytes.Equal(queryBytes, kv.Value.([]byte)) {
				kv.Value = query
				log.Debug().Interface("kv", kv).Msg("sending key-value")
				r.sendKV(out, KeyValErr{KV: kv})
			}
		}
	}
}

func (r *Stream) sendDir(ch chan<- DirErr, dir DirErr) {
	select {
	case <-r.ctx.Done():
	case ch <- dir:
	}
	if dir.Err != nil {
		r.cancel()
	}
}

func (r *Stream) sendKV(ch chan<- KeyValErr, kv KeyValErr) {
	select {
	case <-r.ctx.Done():
	case ch <- kv:
	}
	if kv.Err != nil {
		r.cancel()
	}
}

func (r *Stream) recvDir(ch <-chan DirErr) *DirErr {
	dir, open := <-ch
	if open {
		return &dir
	}
	return nil
}

func (r *Stream) recvKV(ch <-chan KeyValErr) *KeyValErr {
	kv, open := <-ch
	if open {
		return &kv
	}
	return nil
}
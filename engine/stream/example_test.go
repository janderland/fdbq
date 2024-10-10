package stream_test

import (
	"context"
	"fmt"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"

	"github.com/janderland/fql/engine/facade"
	"github.com/janderland/fql/engine/stream"
	"github.com/janderland/fql/keyval"
)

func Example() {
	db := facade.NewTransactor(fdb.MustOpenDefault(), directory.Root())
	s := stream.New(context.Background())

	query := keyval.KeyValue{
		Key: keyval.Key{
			Directory: keyval.Directory{keyval.String("my"), keyval.String("dir")},
			Tuple:     keyval.Tuple{keyval.Float(22.9), keyval.MaybeMore{}},
		},
		Value: keyval.Variable{},
	}

	_, err := db.ReadTransact(func(tr facade.ReadTransaction) (interface{}, error) {
		ch1 := s.OpenDirectories(tr, query.Key.Directory)
		ch2 := s.ReadRange(tr, query.Key.Tuple, stream.RangeOpts{}, ch1)
		ch3 := s.UnpackKeys(query.Key.Tuple, true, ch2)
		for msg := range s.UnpackValues(query.Value, true, ch3) {
			if msg.Err != nil {
				return nil, msg.Err
			}
			fmt.Printf("read kv: %v\n", msg.KV)
		}
		return nil, nil
	})
	if err != nil {
		panic(err)
	}
}

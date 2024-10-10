package engine_test

import (
	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"

	"github.com/janderland/fql/engine"
	"github.com/janderland/fql/engine/facade"
	"github.com/janderland/fql/keyval"
)

func Example() {
	eg := engine.New(facade.NewTransactor(fdb.MustOpenDefault(), directory.Root()))

	key := keyval.Key{
		Directory: keyval.Directory{keyval.String("hello"), keyval.String("there")},
		Tuple:     keyval.Tuple{keyval.Float(33.3)},
	}

	// /hello/there{33.3}=10
	query := keyval.KeyValue{Key: key, Value: keyval.Int(10)}
	if err := eg.Set(query); err != nil {
		panic(err)
	}

	didWrite, err := eg.Transact(func(e engine.Engine) (interface{}, error) {
		// /hello/there{33.3}=<>
		query = keyval.KeyValue{Key: key, Value: keyval.Variable{}}
		result, err := eg.ReadSingle(query, engine.SingleOpts{})
		if err != nil {
			return nil, err
		}
		if result != nil {
			return false, nil
		}

		// /hello/there{33.3}=15
		query = keyval.KeyValue{Key: key, Value: keyval.Int(15)}
		if err := eg.Set(query); err != nil {
			return nil, err
		}
		return true, nil
	})
	if err != nil {
		panic(err)
	}

	if didWrite.(bool) {
		panic("didWrite should be false")
	}
}

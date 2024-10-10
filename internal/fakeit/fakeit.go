package main

import (
	"math/rand"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/brianvoe/gofakeit/v6"

	"github.com/janderland/fql/engine"
	"github.com/janderland/fql/engine/facade"
	kv "github.com/janderland/fql/keyval"
)

func main() {
	fdb.MustAPIVersion(620)
	db := fdb.MustOpenDefault()
	eg := engine.New(facade.NewTransactor(db, directory.Root()))

	for _, path := range [][]string{{"user"}, {"status"}} {
		if _, err := directory.Root().Remove(db, path); err != nil {
			panic(err)
		}
	}

	for i := 0; i < 3; i++ {
		query := kv.KeyValue{
			Key: kv.Key{
				Directory: kv.Directory{kv.String("user")},
				Tuple: kv.Tuple{
					kv.Int(rand.Int63()),
					kv.String("Goodwin"),
					kv.String(gofakeit.FirstName()),
				},
			},
			Value: kv.Nil{},
		}
		if err := eg.Set(query); err != nil {
			panic(err)
		}
	}

	for i := 0; i < 320; i++ {
		query := kv.KeyValue{
			Key: kv.Key{
				Directory: kv.Directory{kv.String("user")},
				Tuple: kv.Tuple{
					kv.Int(rand.Int63()),
					kv.String(gofakeit.LastName()),
					kv.String(gofakeit.FirstName()),
				},
			},
			Value: kv.Nil{},
		}
		if err := eg.Set(query); err != nil {
			panic(err)
		}
	}

	for i := 0; i < 10; i++ {
		query := kv.KeyValue{
			Key: kv.Key{
				Directory: kv.Directory{kv.String("status"), kv.String("service")},
				Tuple:     kv.Tuple{kv.Int(rand.Int63())},
			},
			Value: func() kv.Value {
				switch rand.Int() % 3 {
				case 0:
					return kv.String("healthy")
				case 1:
					return kv.String("warning")
				default:
					return kv.String("failed")
				}
			}(),
		}
		if err := eg.Set(query); err != nil {
			panic(err)
		}
	}

	for i := 0; i < 10; i++ {
		query := kv.KeyValue{
			Key: kv.Key{
				Directory: kv.Directory{kv.String("status"), kv.String("queue")},
				Tuple:     kv.Tuple{kv.Int(rand.Int63())},
			},
			Value: func() kv.Value {
				switch rand.Int() % 3 {
				case 0:
					return kv.String("healthy")
				case 1:
					return kv.String("overflow")
				default:
					return kv.String("empty")
				}
			}(),
		}
		if err := eg.Set(query); err != nil {
			panic(err)
		}
	}
}

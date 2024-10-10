package internal

import (
	"crypto/rand"
	"encoding/binary"
	"os"
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/janderland/fql/engine/facade"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func TestEnv(t *testing.T, force bool, f func(facade.Transactor, zerolog.Logger)) {
	db := fdb.MustOpenDefault()
	rootPath := genRootPath()

	exists, err := directory.Exists(db, rootPath)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to check if root directory exists"))
	}
	if exists {
		if !force {
			t.Fatal(errors.Errorf("test directory '%v' already exists, use '-force' flag to remove", rootPath))
		}
		if _, err := directory.Root().Remove(db, rootPath); err != nil {
			t.Fatal(errors.Wrap(err, "failed to remove directory"))
		}
	}

	dir, err := directory.Create(db, rootPath, nil)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to create test directory"))
	}
	defer func() {
		_, err := directory.Root().Remove(db, rootPath)
		if err != nil {
			t.Fatal(errors.Wrap(err, "failed to clean root directory"))
		}
	}()

	writer := zerolog.ConsoleWriter{Out: os.Stdout}
	writer.FormatLevel = func(_ interface{}) string { return "" }
	writer.FormatTimestamp = func(_ interface{}) string { return "" }
	log := zerolog.New(writer)

	f(facade.NewTransactor(db, dir), log)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func genRootPath() []string {
	root := make([]rune, 6)
	for i := range root {
		b := make([]byte, 8)
		_, err := rand.Read(b)
		if err != nil {
			panic(err)
		}

		n := int(binary.BigEndian.Uint64(b))
		if n < 0 {
			n = -n
		}
		root[i] = letters[n%len(letters)]
	}
	return []string{string(root)}
}

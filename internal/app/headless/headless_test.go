package headless

import (
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/janderland/fql/engine"
	"github.com/janderland/fql/engine/facade"
)

func TestHeadless_Query(t *testing.T) {
	tests := []struct {
		name    string
		write   bool
		queries []string
		err     bool
	}{
		{
			name:    "set",
			write:   true,
			queries: []string{"/my/dir(\"hi\",\"there\")=33.9"},
			err:     false,
		},
		{
			name:    "set error",
			write:   false,
			queries: []string{"/my/dir(\"hi\",\"there\")=33.9"},
			err:     true,
		},
		{
			name:    "clear",
			write:   true,
			queries: []string{"/my/dir(\"hi\",\"there\")=clear"},
			err:     false,
		},
		{
			name:    "clear error",
			write:   false,
			queries: []string{"/my/dir(\"hi\",\"there\")=clear"},
			err:     true,
		},
		{
			name:    "get nothing",
			write:   false,
			queries: []string{"/nothing/is/here(\"wont\",\"match\")=<>"},
			err:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEnv(t, func(app App) {
				app.Write = test.write

				err := app.Run(context.Background(), test.queries)
				if test.err {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func testEnv(t *testing.T, f func(App)) {
	writer := zerolog.ConsoleWriter{Out: os.Stdout}
	writer.FormatLevel = func(_ interface{}) string { return "" }
	writer.FormatTimestamp = func(_ interface{}) string { return "" }
	log := zerolog.New(writer)

	dv, closeDV := devnull(t)
	defer closeDV()

	f(App{
		Engine: engine.New(facade.NewNilTransactor(), engine.Logger(log)),
		Out:    dv,
	})
}

func devnull(t *testing.T) (*os.File, func()) {
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to open devnull"))
	}
	return devnull, func() {
		if err := devnull.Close(); err != nil {
			t.Fatal(errors.Wrap(err, "failed to close devnull"))
		}
	}
}

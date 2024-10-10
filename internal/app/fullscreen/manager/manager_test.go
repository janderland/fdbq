package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/janderland/fql/engine"
	"github.com/janderland/fql/engine/facade"
)

func TestWriteFlag(t *testing.T) {
	tests := []struct {
		name  string
		write bool
		query string
		err   bool
	}{
		{
			name:  "set",
			write: true,
			query: "/my/dir(\"hi\",\"there\")=33.9",
			err:   false,
		},
		{
			name:  "set error",
			write: false,
			query: "/my/dir(\"hi\",\"there\")=33.9",
			err:   true,
		},
		{
			name:  "clear",
			write: true,
			query: "/my/dir(\"hi\",\"there\")=clear",
			err:   false,
		},
		{
			name:  "clear error",
			write: false,
			query: "/my/dir(\"hi\",\"there\")=clear",
			err:   true,
		},
		{
			name:  "get nothing",
			write: false,
			query: "/nothing/is/here(\"wont\",\"match\")=<>",
			err:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			qm := New(
				context.Background(),
				engine.New(facade.NewNilTransactor()),
				WithWrite(test.write))

			out := qm.Query(test.query)
			err, ok := out().(error)
			if test.err {
				require.True(t, ok)
				require.Error(t, err)
			} else {
				require.False(t, ok)
			}
		})
	}
}

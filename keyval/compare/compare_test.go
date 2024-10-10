package compare

import (
	"testing"

	"github.com/stretchr/testify/require"

	q "github.com/janderland/fql/keyval"
)

func TestTuples(t *testing.T) {
	t.Run("equal", func(t *testing.T) {
		candidate := q.Tuple{
			q.Int(-8742),
			q.Uint(12342),
			q.String("goodbye world"),
			q.Float(-55.93),
			q.Bool(true),
			q.Nil{},
		}
		pattern := make(q.Tuple, len(candidate))
		require.Equal(t, copy(pattern, candidate), len(candidate))

		mismatch := Tuples(pattern, candidate)
		require.Empty(t, mismatch)
	})

	t.Run("not equal", func(t *testing.T) {
		tests := []struct {
			name      string
			candidate q.Tuple
			pattern   q.Tuple
		}{
			{
				name:      "conversion error",
				candidate: q.Tuple{q.Int(-8742), q.Uint(12342)},
				pattern:   q.Tuple{q.Float(-55.93), q.Bool(true)},
			},
			{
				name:      "not equal",
				candidate: q.Tuple{q.Int(-8742)},
				pattern:   q.Tuple{q.Int(-55)},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				mismatch := Tuples(test.pattern, test.candidate)
				require.NotEmpty(t, mismatch)
			})
		}
	})

	t.Run("variable", func(t *testing.T) {
		candidate1 := q.Tuple{
			q.Int(-8742),
			q.Uint(12342),
			q.String("goodbye world"),
			q.Nil{},
		}
		candidate2 := q.Tuple{
			q.Int(-8742),
			q.Tuple{q.String("nowhere"), q.Int(55)},
			q.String("goodbye world"),
			q.Int(555),
		}
		pattern := q.Tuple{
			q.Int(-8742),
			q.Variable{},
			q.String("goodbye world"),
			q.Variable{},
		}

		mismatch := Tuples(pattern, candidate1)
		require.Empty(t, mismatch)

		mismatch = Tuples(pattern, candidate2)
		require.Empty(t, mismatch)
	})

	t.Run("multi type", func(t *testing.T) {
		candidate := q.Tuple{q.String("where am i?")}
		pattern := q.Tuple{q.Variable{q.IntType, q.TupleType, q.StringType}}

		mismatch := Tuples(pattern, candidate)
		require.Empty(t, mismatch)
	})

	t.Run("too long", func(t *testing.T) {
		candidate := q.Tuple{
			q.Int(-8742),
			q.Uint(12342),
			q.String("goodbye world"),
			q.Nil{},
			q.Tuple{q.Float(-55.1428)},
		}
		pattern := q.Tuple{
			q.Int(-8742),
			q.Variable{},
			q.String("goodbye world"),
			q.Variable{},
		}

		mismatch := Tuples(pattern, candidate)
		require.NotEmpty(t, mismatch)
	})

	t.Run("too short", func(t *testing.T) {
		candidate := q.Tuple{
			q.Int(-8742),
			q.Uint(12342),
			q.String("goodbye world"),
		}
		pattern := q.Tuple{
			q.Int(-8742),
			q.Variable{},
			q.String("goodbye world"),
			q.Variable{},
		}

		mismatch := Tuples(pattern, candidate)
		require.NotEmpty(t, mismatch)
	})

	t.Run("maybe more", func(t *testing.T) {
		candidate := q.Tuple{
			q.Int(-8742),
			q.Uint(12342),
			q.String("goodbye world"),
			q.Nil{},
			q.Tuple{q.Float(-55.1428)},
		}
		pattern := q.Tuple{
			q.Int(-8742),
			q.Variable{},
			q.String("goodbye world"),
			q.MaybeMore{},
		}

		mismatch := Tuples(pattern, candidate)
		require.Empty(t, mismatch)
	})
}

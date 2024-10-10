package stack

import (
	"github.com/janderland/fql/internal/app/fullscreen/results"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestResultsStack(t *testing.T) {
	var (
		x  ResultsStack
		r1 = results.New()
		r2 = results.New()
	)

	require.Nil(t, x.Top())

	x.Push(r1)
	require.Equal(t, &r1, x.Top())

	x.Push(r2)
	require.Equal(t, &r2, x.Top())

	x.Pop()
	require.Equal(t, &r1, x.Top())
}

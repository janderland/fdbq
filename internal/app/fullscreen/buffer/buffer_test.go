package buffer

import (
	"container/list"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/janderland/fql/engine/stream"
	"github.com/janderland/fql/keyval"
)

func TestStreamBuffer(t *testing.T) {
	ch := make(chan stream.KeyValErr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := []stream.KeyValErr{
		{KV: keyval.KeyValue{
			Key: keyval.Key{
				Directory: keyval.Directory{keyval.String("hi")},
				Tuple:     keyval.Tuple{keyval.Int(22)},
			},
			Value: keyval.Int(99),
		}},
		{KV: keyval.KeyValue{
			Key: keyval.Key{
				Directory: keyval.Directory{keyval.String("you"), keyval.String("there")},
				Tuple:     keyval.Tuple{keyval.String("person"), keyval.Bytes{0xFF, 0x22}},
			},
			Value: keyval.Nil{},
		}},
	}

	go func() {
		defer close(ch)

		for _, kve := range in {
			select {
			case <-ctx.Done():
				return
			case ch <- kve:
			}
		}
	}()

	var out []stream.KeyValErr
	done := false
	sb := New(ch)

	for !done {
		var buffer *list.List
		buffer, done = sb.Get()

		for item := buffer.Front(); item != nil; item = item.Next() {
			out = append(out, item.Value.(stream.KeyValErr))
		}
	}

	require.ElementsMatch(t, in, out)
}

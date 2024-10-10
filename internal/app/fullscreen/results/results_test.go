package results

import (
	"fmt"
	"strings"
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/janderland/fql/engine/facade"
	"github.com/janderland/fql/engine/stream"
	"github.com/janderland/fql/keyval"
)

func TestHeight(t *testing.T) {
	x := setup()

	require.Empty(t, x.View())

	x.Height(50)
	require.Equal(t, 50, lineCount(x.View()))
}

func TestWrapWidth(t *testing.T) {
	x := New()
	x.Push("xxx xxx xxx")
	x.Push("xxx xxx")
	x.Push("xxx xxx xxx")

	x.WrapWidth(12)
	x.Height(4)

	// The [1:] gets rid of the leading newline.
	expected := `
   xxx
2  # xxx xxx
3  # xxx xxx
   xxx`[1:]

	require.Equal(t, expected, x.View())

	x.Reset()
	x.Push("xxxxxxxxxx xxx")
	x.WrapWidth(8)

	expected = `
1  # 
   xxxxx
   xxxxx
   xxx`[1:]

	require.Equal(t, expected, x.View())
}

func TestReset(t *testing.T) {
	x := setup()
	x.Height(50)

	x.Reset()
	require.Empty(t, x.View())

	for i := 0; i < 10; i++ {
		x.Push("")
	}
	require.Equal(t, 10, lineCount(x.View()))
}

func TestSingleLine(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{
			errors.New("error"),
			"1  ERR! error",
		},
		{
			"string",
			"1  # string",
		},
		{
			keyval.KeyValue{
				Key: keyval.Key{
					Directory: keyval.Directory{keyval.String("dir")},
					Tuple:     keyval.Tuple{keyval.Int(23)},
				},
				Value: keyval.Int(10),
			},
			"1  /dir(23)=10",
		},
		{
			dir([]string{"dir"}),
			"1  /dir",
		},
		{
			stream.KeyValErr{
				KV: keyval.KeyValue{
					Key: keyval.Key{
						Directory: keyval.Directory{keyval.String("dir")},
						Tuple:     keyval.Tuple{keyval.Int(23)},
					},
					Value: keyval.Int(10),
				},
			},
			"1  /dir(23)=10",
		},
		{
			stream.KeyValErr{
				Err: errors.New("error"),
			},
			"1  ERR! error",
		},
		{
			stream.DirErr{
				Dir: dir([]string{"dir"}),
			},
			"1  /dir",
		},
		{
			stream.DirErr{
				Err: errors.New("error"),
			},
			"1  ERR! error",
		},
		{
			[]uint8{},
			"1  ERR! unexpected []uint8",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T", test.input), func(t *testing.T) {
			x := New()
			x.Height(1)

			x.Push(test.input)
			require.Equal(t, test.expected, x.View())
		})
	}
}

func TestSpaced(t *testing.T) {
	x := New(WithSpaced(true))
	x.Height(2)

	x.Push("1")
	x.Push("2")

	require.Equal(t, "   \n2  # 2", x.View())
}

func TestItemScroll(t *testing.T) {
	x := setup()

	const height = 5
	x.Height(height)

	expected := func(start int) string {
		var str strings.Builder
		for i := start; i < start+height; i++ {
			if i != start {
				str.WriteRune('\n')
			}
			str.WriteString(fmt.Sprintf("%d  # %d", i, i))
		}
		return str.String()
	}

	require.Nil(t, x.cursor)
	require.Equal(t, expected(96), x.View())

	x.scrollUpItems(10)
	require.Equal(t, expected(86), x.View())

	x.scrollDownItems(9)
	require.Equal(t, expected(95), x.View())

	x.scrollUpItems(95)
	require.Equal(t, expected(1), x.View())
}

func TestLineScroll(t *testing.T) {
	x := New()
	x.Height(3)
	x.WrapWidth(10)

	x.Push("xxx xxx")
	require.Equal(t, "1  # xxx \n   xxx", x.View())

	x.scrollUpLines(1)
	require.Equal(t, "1  # xxx \n   xxx", x.View())
}

func setup() Model {
	x := New()
	for i := 1; i <= 100; i++ {
		x.Push(fmt.Sprintf("%d", i))
	}
	return x
}

func lineCount(str string) int {
	return len(strings.Split(str, "\n"))
}

func dir(path []string) directory.DirectorySubspace {
	return &mockDir{facade.NewNilDirectorySubspace(), path}
}

type mockDir struct {
	directory.DirectorySubspace
	path []string
}

func (x *mockDir) GetPath() []string {
	return x.path
}

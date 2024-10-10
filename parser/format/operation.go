package format

import (
	"strings"

	q "github.com/janderland/fql/keyval"
	"github.com/janderland/fql/parser/internal"
)

// formatDirElement is a keyval.DirectoryOperation
// which calls the appropriate format method for the
// given keyval.DirElement.
type formatDirElement struct {
	format *Format
}

var _ q.DirectoryOperation = &formatDirElement{}

// quotedRunes is a string containing a set of special
// runes. If any of these runes are found in a directory
// element, then that element must be formatted with
// surrounding quotes.
var quotedRunes = internal.AllSingleRuneTokens() +
	internal.Newline +
	internal.Whitespace

func (x *formatDirElement) ForString(in q.String) {
	needsQuotes := strings.ContainsAny(string(in), quotedRunes)
	if needsQuotes {
		x.format.builder.WriteRune(internal.StrMark)
	}
	x.format.builder.WriteString(escapeString(string(in)))
	if needsQuotes {
		x.format.builder.WriteRune(internal.StrMark)
	}
}

func (x *formatDirElement) ForVariable(in q.Variable) {
	x.format.Variable(in)
}

// formatQuery is a keyval.QueryOperation which calls the
// appropriate Format method for the given keyval.Query.
type formatQuery struct {
	format *Format
}

var _ q.QueryOperation = &formatQuery{}

func (x *formatQuery) ForDirectory(in q.Directory) {
	x.format.Directory(in)
}

func (x *formatQuery) ForKey(in q.Key) {
	x.format.Key(in)
}

func (x *formatQuery) ForKeyValue(in q.KeyValue) {
	x.format.KeyValue(in)
}

// formatData is both a keyval.TupleOperation and a
// keyval.ValueOperation which calls the appropriate
// Format method for the given data element.
type formatData struct {
	format *Format
}

var (
	_ q.TupleOperation = &formatData{}
	_ q.ValueOperation = &formatData{}
)

func (x *formatData) ForVariable(in q.Variable) {
	x.format.Variable(in)
}

func (x *formatData) ForString(in q.String) {
	x.format.Str(in)
}

func (x *formatData) ForNil(in q.Nil) {
	x.format.Nil(in)
}

func (x *formatData) ForMaybeMore(in q.MaybeMore) {
	x.format.MaybeMore(in)
}

func (x *formatData) ForTuple(in q.Tuple) {
	x.format.Tuple(in)
}

func (x *formatData) ForInt(in q.Int) {
	x.format.Int(in)
}

func (x *formatData) ForUint(in q.Uint) {
	x.format.Uint(in)
}

func (x *formatData) ForBool(in q.Bool) {
	x.format.Bool(in)
}

func (x *formatData) ForFloat(in q.Float) {
	x.format.Float(in)
}

func (x *formatData) ForUUID(in q.UUID) {
	x.format.UUID(in)
}

func (x *formatData) ForBytes(in q.Bytes) {
	x.format.Bytes(in)
}

func (x *formatData) ForClear(in q.Clear) {
	x.format.Clear(in)
}

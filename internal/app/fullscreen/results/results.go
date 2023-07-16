package results

import (
	"container/list"
	"fmt"
	"strings"

	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"

	"github.com/janderland/fdbq/engine/stream"
	"github.com/janderland/fdbq/keyval"
	"github.com/janderland/fdbq/keyval/convert"
	"github.com/janderland/fdbq/parser/format"
)

type keyMap struct {
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Down         key.Binding
	Up           key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", " ", "f"),
			key.WithHelp("f/pgdn", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("b/pgup", "page up"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
			key.WithHelp("u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("d", "½ page down"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
	}
}

type result struct {
	i     int
	value any
}

type Model struct {
	keyMap keyMap
	format format.Format

	// height is the max number of lines
	// that will be rendered.
	height int

	// wrapWidth is the width at which each
	// line is wrapped. 0 disables wrapping.
	wrapWidth int

	// builder is used by the View method
	// to construct the final output.
	builder *strings.Builder

	// list contains all the scrollable items.
	// Newer items are placed at the front.
	list *list.List

	// cursor points at the newest item
	// that will be displayed.
	cursor *list.Element

	// endCursor points at the oldest item
	// which cursor is allowed to scroll to.
	// This prevents scrolling past the
	// final page.
	endCursor *list.Element
}

func New(kvFmt format.Format) Model {
	return Model{
		keyMap:  defaultKeyMap(),
		format:  kvFmt,
		builder: &strings.Builder{},
		list:    list.New(),
	}
}

func (x *Model) Reset() {
	x.list = list.New()
	x.cursor = nil
	x.endCursor = nil
}

func (x *Model) Height(height int) {
	x.height = height
	x.updateCursors()
}

func (x *Model) WrapWidth(width int) {
	x.wrapWidth = width
	x.updateCursors()
}

func (x *Model) PushMany(list *list.List) {
	for cursor := list.Front(); cursor != nil; cursor = cursor.Next() {
		x.push(cursor.Value)
	}
	x.updateCursors()
}

func (x *Model) Push(val any) {
	x.push(val)
	x.updateCursors()
}

func (x *Model) push(val any) {
	x.list.PushFront(result{
		i:     x.list.Len() + 1,
		value: val,
	})
}

func (x *Model) updateCursors() {
	if x.list.Len() == 0 {
		return
	}

	// We only move height-1 elements in the
	// for-loop below to ensure the subset of
	// elements from endCursor to list.Back()
	// is "height" elements long, inclusive.
	x.endCursor = x.list.Back()
	for i := 0; i < x.height-1; i++ {
		if x.endCursor.Prev() == nil {
			break
		}

		// As we move the end cursor back through
		// the list, if we encounter the start
		// cursor then move it along with us.
		if x.cursor == x.endCursor {
			x.cursor = x.endCursor.Prev()
		}
		x.endCursor = x.endCursor.Prev()
	}
}

func (x *Model) View() string {
	if x.height == 0 || x.list.Len() == 0 {
		return ""
	}

	// If we have scrolled back through
	// the list then start our local
	// cursor there. Otherwise, start
	// at the front of the list.
	cursor := x.cursor
	if cursor == nil {
		cursor = x.list.Front()
	}

	var lines []string
	for len(lines) < x.height && cursor != nil {
		lines = append(lines, x.render(cursor.Value.(result))...)
		cursor = cursor.Next()
	}

	start := x.height - 1
	if start > len(lines)-1 {
		start = len(lines) - 1
	}

	x.builder.Reset()
	for i := start; i >= 0; i-- {
		if i != start {
			x.builder.WriteRune('\n')
		}
		x.builder.WriteString(lines[i])
	}
	return x.builder.String()
}

func (x *Model) render(res result) []string {
	prefix := fmt.Sprintf("%d  ", res.i)
	indent := strings.Repeat(" ", len(prefix))

	str := x.value(res.value)
	str = wordwrap.String(str, x.wrapWidth-len(prefix))
	str = wrap.String(str, x.wrapWidth-len(prefix))
	lines := strings.Split(str, "\n")

	var reversed []string
	for i := len(lines) - 1; i >= 0; i-- {
		var line string
		if i == 0 {
			line = prefix + lines[i]
		} else {
			line = indent + lines[i]
		}
		reversed = append(reversed, line)
	}
	return reversed
}

func (x *Model) value(item any) string {
	switch val := item.(type) {
	case error:
		return fmt.Sprintf("ERR! %s", val)

	case string:
		return fmt.Sprintf("# %s", val)

	case keyval.KeyValue:
		x.format.Reset()
		x.format.KeyValue(val)
		return x.format.String()

	case directory.DirectorySubspace:
		x.format.Reset()
		x.format.Directory(convert.FromStringArray(val.GetPath()))
		return x.format.String()

	case stream.KeyValErr:
		if val.Err != nil {
			return x.value(val.Err)
		}
		return x.value(val.KV)

	case stream.DirErr:
		if val.Err != nil {
			return x.value(val.Err)
		}
		return x.value(val.Dir)

	default:
		return fmt.Sprintf("ERR! unexpected %T", val)
	}
}

func (x *Model) Update(msg tea.Msg) Model {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, x.keyMap.PageDown):
			x.scrollDown(x.height - 1)

		case key.Matches(msg, x.keyMap.PageUp):
			x.scrollUp(x.height - 1)

		case key.Matches(msg, x.keyMap.HalfPageDown):
			x.scrollDown(x.height / 2)

		case key.Matches(msg, x.keyMap.HalfPageUp):
			x.scrollUp(x.height / 2)

		case key.Matches(msg, x.keyMap.Down):
			x.scrollDown(1)

		case key.Matches(msg, x.keyMap.Up):
			x.scrollUp(1)
		}

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelDown:
			x.scrollDown(1)

		case tea.MouseWheelUp:
			x.scrollUp(1)
		}
	}

	return *x
}

func (x *Model) scrollDown(lines int) {
	if x.cursor == nil {
		return
	}
	for i := 0; i < lines; i++ {
		x.cursor = x.cursor.Prev()
		if x.cursor == nil {
			break
		}
	}
}

func (x *Model) scrollUp(lines int) {
	if x.list.Len() == 0 {
		return
	}
	if x.cursor == nil {
		x.cursor = x.list.Front()
	}
	for i := 0; i < lines; i++ {
		if x.cursor == x.endCursor {
			break
		}
		newCursor := x.cursor.Next()
		if newCursor == nil {
			break
		}
		x.cursor = newCursor
	}
}

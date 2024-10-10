package results

import (
	"container/list"
	"fmt"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderland/fql/internal/app/fullscreen/results/wrap"
	"github.com/rs/zerolog"
	"math"
	"strings"

	"github.com/janderland/fql/engine/stream"
	"github.com/janderland/fql/keyval"
	"github.com/janderland/fql/keyval/convert"
	"github.com/janderland/fql/parser/format"
)

type keyMap struct {
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	DownLine     key.Binding
	UpLine       key.Binding
	DownItem     key.Binding
	UpItem       key.Binding
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
		UpLine: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		DownLine: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		UpItem: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "up line"),
		),
		DownItem: key.NewBinding(
			key.WithKeys("J"),
			key.WithHelp("J", "down line"),
		),
	}
}

type result struct {
	i     int
	value any
}

type Option func(*Model)

type Model struct {
	// log traces major events.
	log zerolog.Logger

	// keyMap specifies the key bindings.
	keyMap keyMap

	// format is used to stringify
	// key-values.
	format format.Format

	// height is the max number of lines
	// that will be rendered.
	height int

	// wrapWidth is the width at which each
	// line is wrapped. 0 disables wrapping.
	wrapWidth int

	// maxWrapWidth caps the wrapWidth value.
	// If a higher value is set the maxWrapWidth
	// is used instead.
	maxWrapWidth int

	// spaced determines if a blank line
	// appears between each item.
	spaced bool

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

	// subCursor is the number of lines
	// above the last line of the rendered
	// cursor which aligns with the
	// bottom of the screen.
	subCursor int

	// endSubCursor is the maximum value
	// allowed for subCursor. This prevents
	// scrolling past the final page.
	endSubCursor int
}

func New(opts ...Option) Model {
	x := Model{
		log:          zerolog.Nop(),
		keyMap:       defaultKeyMap(),
		format:       format.New(),
		maxWrapWidth: math.MaxInt,
		builder:      &strings.Builder{},
		list:         list.New(),
	}
	for _, option := range opts {
		option(&x)
	}
	return x
}

func WithFormat(f format.Format) Option {
	return func(x *Model) {
		x.format = f
	}
}

func WithSpaced(spaced bool) Option {
	return func(x *Model) {
		x.spaced = spaced
	}
}

func WithLogger(log zerolog.Logger) Option {
	return func(x *Model) {
		x.log = log
	}
}

func (x *Model) Reset() {
	x.log.Log().Msg("resetting")
	x.list = list.New()
	x.cursor = nil
	x.endCursor = nil
}

func (x *Model) Height(height int) {
	x.log.Log().Int("height", height).Msg("setting")
	x.height = height
	x.updateCursors()
}

func (x *Model) WrapWidth(width int) {
	if width > x.maxWrapWidth {
		width = x.maxWrapWidth
	}
	x.log.Log().Int("wrapWidth", width).Msg("setting")
	x.wrapWidth = width
	x.subCursor = 0
	x.updateCursors()
}

func (x *Model) MaxWrapWidth(width int) {
	x.maxWrapWidth = width
}

func (x *Model) PushMany(list *list.List) {
	x.log.Log().Int("n", list.Len()).Msg("pushing many")
	for cursor := list.Front(); cursor != nil; cursor = cursor.Next() {
		x.push(cursor.Value)
	}
	x.updateCursors()
}

func (x *Model) Push(val any) {
	x.log.Log().Msg("pushing")
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
	if x.list.Len() == 0 || x.height == 0 {
		return
	}

	x.endCursor = x.list.Back()
	lines := len(x.render(x.endCursor))
	fromEnd := 0

	for x.endCursor.Prev() != nil && lines < x.height {
		// As we move the end cursor back through
		// the list, if we encounter the start
		// cursor then move it along with us.
		if x.cursor == x.endCursor {
			x.cursor = x.endCursor.Prev()
		}
		x.endCursor = x.endCursor.Prev()

		lines += len(x.render(x.endCursor))
		fromEnd++
	}

	x.endSubCursor = 0
	if lines > x.height {
		x.endSubCursor = lines % x.height
	}

	x.log.Log().
		Int("fromEnd", fromEnd).
		Int("subEnd", x.endSubCursor).
		Msg("end cursors")
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

	lines := x.render(cursor)[x.subCursor:]
	cursor = cursor.Next()

	for len(lines) < x.height && cursor != nil {
		lines = append(lines, x.render(cursor)...)
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

func (x *Model) render(e *list.Element) []string {
	res := e.Value.(result)
	prefix := fmt.Sprintf("%d  ", res.i)
	indent := strings.Repeat(" ", len(prefix))
	lines := wrap.Wrap(x.str(res.value), x.wrapWidth-len(prefix))

	// If spaced is enabled, add an extra blank
	// line after each item except the newest.
	if x.spaced && e != x.list.Front() {
		lines = append(lines, "")
	}

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

func (x *Model) str(item any) string {
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
			return x.str(val.Err)
		}
		return x.str(val.KV)

	case stream.DirErr:
		if val.Err != nil {
			return x.str(val.Err)
		}
		return x.str(val.Dir)

	default:
		return fmt.Sprintf("ERR! unexpected %T", val)
	}
}

func (x *Model) Scroll(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, x.keyMap.PageDown):
			x.scrollDownLines(x.height - 1)

		case key.Matches(msg, x.keyMap.PageUp):
			x.scrollUpLines(x.height - 1)

		case key.Matches(msg, x.keyMap.HalfPageDown):
			x.scrollDownLines(x.height / 2)

		case key.Matches(msg, x.keyMap.HalfPageUp):
			x.scrollUpLines(x.height / 2)

		case key.Matches(msg, x.keyMap.DownLine):
			x.scrollDownLines(1)

		case key.Matches(msg, x.keyMap.UpLine):
			x.scrollUpLines(1)

		case key.Matches(msg, x.keyMap.DownItem):
			x.scrollDownItems(1)

		case key.Matches(msg, x.keyMap.UpItem):
			x.scrollUpItems(1)
		}

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelDown:
			x.scrollDownItems(1)

		case tea.MouseWheelUp:
			x.scrollUpItems(1)
		}
	}
}

func (x *Model) scrollDownItems(n int) bool {
	log := x.log.With().Int("n", n).Logger()
	log.Log().Msg("down items")

	if x.cursor == nil {
		log.Log().Msg("down items ignored")
		return false
	}
	for i := 0; i < n; i++ {
		x.cursor = x.cursor.Prev()
		if x.cursor == nil {
			log.Log().Int("i", i).Msg("down items stopped")
			break
		}
	}
	x.subCursor = 0
	return true
}

func (x *Model) scrollUpItems(n int) bool {
	log := x.log.With().Int("n", n).Logger()
	log.Log().Msg("up items")

	if x.list.Len() == 0 || x.cursor == x.endCursor {
		log.Log().Msg("up items ignored")
		return false
	}
	if x.cursor == nil {
		x.cursor = x.list.Front()
	}
	for i := 0; i < n; i++ {
		if x.cursor == x.endCursor {
			log.Log().Int("i", i).Msg("up items stopped")
			break
		}
		newCursor := x.cursor.Next()
		// This check is for detecting bugs.
		// endCursor should always be set,
		// so we should never encounter
		// this case.
		if newCursor == nil {
			log.Error().Int("i", i).Msg("up items unreachable?")
			break
		}
		x.cursor = newCursor
	}
	x.subCursor = 0
	return true
}

func (x *Model) scrollDownLines(n int) {
	log := x.log.With().Int("n", n).Logger()
	log.Log().Msg("down lines")

	for i := 0; i < n; i++ {
		if x.subCursor == 0 {
			if x.scrollDownItems(1) {
				if x.cursor == nil {
					log.Log().Int("i", i).Msg("down lines stopped")
					return
				}
				x.subCursor = len(x.render(x.cursor)) - 1
				continue
			}
			return
		}
		x.subCursor--
	}
}

func (x *Model) scrollUpLines(n int) {
	log := x.log.With().Int("n", n).Logger()
	log.Log().Msg("up lines")

	if x.list.Len() == 0 {
		log.Log().Msg("up lines ignored")
		return
	}
	if x.cursor == nil {
		x.cursor = x.list.Front()
	}
	for i := 0; i < n; i++ {
		if x.subCursor+1 >= len(x.render(x.cursor)) {
			if !x.scrollUpItems(1) {
				return
			}
			continue
		}
		if x.cursor == x.endCursor && x.subCursor == x.endSubCursor {
			log.Log().Int("i", i).Msg("up lines stopped")
			return
		}
		x.subCursor++
	}
}

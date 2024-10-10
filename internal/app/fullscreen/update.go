package fullscreen

import (
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"

	"github.com/janderland/fql/internal/app/fullscreen/manager"
	"github.com/janderland/fql/keyval"
)

func (x Model) Init() tea.Cmd {
	return func() tea.Msg {
		return "Press '?' to see the help menu."
	}
}

func (x Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(cursor.BlinkMsg); !ok {
		x.log.Log().Msgf("msg: %T %v", msg, msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return x.updateKey(msg)

	case tea.MouseMsg:
		return x.updateMouse(msg)

	case manager.AsyncQueryMsg:
		return x.updateAsyncQuery(msg)

	case error, string, keyval.KeyValue:
		return x.updateSingle(msg)

	case tea.WindowSizeMsg:
		return x.updateSize(msg), nil

	default:
		return x.updateBlink(msg)
	}
}

func (x Model) updateKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch x.mode {
	case modeScroll:
		switch msg.Type {
		case tea.KeyEnter:
			return x, x.qm.Query(x.input.Value())

		case tea.KeyCtrlC:
			x.qm.Cancel()
			return x, nil

		case tea.KeyRunes:
			switch msg.String() {
			case "i":
				x.mode = modeInput
				x.input.Focus()
				return x, textinput.Blink

			case "?":
				x.mode = modeHelp
				x.results.Push(newHelp())
				return x, nil

			case "q":
				x.mode = modeQuit
				x.results.Push(newQuit())
				return x, nil
			}
		}

		x.results.Top().Scroll(msg)
		return x, nil

	case modeInput:
		switch msg.Type {
		case tea.KeyEnter:
			return x, x.qm.Query(x.input.Value())

		case tea.KeyCtrlC:
			x.qm.Cancel()
			return x, nil

		case tea.KeyEscape:
			x.mode = modeScroll
			x.input.Blur()
			return x, nil

		case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
			x.results.Top().Scroll(msg)
		}

		var cmd tea.Cmd
		x.input, cmd = x.input.Update(msg)
		return x, cmd

	case modeHelp:
		switch msg.Type {
		case tea.KeyEscape:
			x.mode = modeScroll
			x.results.Pop()
			return x, nil
		}

		x.results.Top().Scroll(msg)
		return x, nil

	case modeQuit:
		switch msg.Type {
		case tea.KeyEscape:
			x.mode = modeScroll
			x.results.Pop()
			return x, nil

		case tea.KeyRunes:
			switch msg.String() {
			case "n", "N":
				x.mode = modeScroll
				x.results.Pop()
				return x, nil

			case "y", "Y":
				return x, tea.Quit
			}
		}

		x.results.Top().Scroll(msg)
		return x, nil

	default:
		panic(errors.Errorf("unexpected mode '%v'", x.mode))
	}
}

func (x Model) updateMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	x.input, cmd = x.input.Update(msg)
	x.results.Top().Scroll(msg)
	return x, cmd
}

func (x Model) updateAsyncQuery(msg manager.AsyncQueryMsg) (Model, tea.Cmd) {
	if x.latest.After(msg.StartedAt) {
		return x, nil
	}
	if x.latest.Before(msg.StartedAt) {
		x.results.Top().Reset()
		x.latest = msg.StartedAt
	}

	buf, done := msg.Buffer.Get()
	x.results.Top().PushMany(buf)
	if !done {
		return x, tea.Tick(50*time.Millisecond, func(_ time.Time) tea.Msg {
			return msg
		})
	}
	return x, nil
}

func (x Model) updateSingle(msg any) (Model, tea.Cmd) {
	x.results.Top().Reset()
	x.results.Top().Push(msg)
	return x, nil
}

func (x Model) updateSize(msg tea.WindowSizeMsg) Model {
	const inputLine = 1
	const cursorChar = 1
	inputHeight := x.style.input.GetVerticalFrameSize() + inputLine

	x.style.results.Height(msg.Height - x.style.results.GetVerticalFrameSize() - inputHeight)
	x.style.results.Width(msg.Width - x.style.results.GetHorizontalFrameSize())

	x.results.Height(x.style.results.GetHeight() - x.style.results.GetVerticalFrameSize())
	x.results.WrapWidth(x.style.results.GetWidth() - x.style.results.GetHorizontalFrameSize())

	x.input.Width = msg.Width - x.style.input.GetHorizontalFrameSize() - len(x.input.Prompt) - cursorChar - 2
	x.style.input.Width(msg.Width - x.style.input.GetHorizontalFrameSize())

	return x
}

func (x Model) updateBlink(msg any) (Model, tea.Cmd) {
	var cmd tea.Cmd
	x.input, cmd = x.input.Update(msg)
	return x, cmd
}

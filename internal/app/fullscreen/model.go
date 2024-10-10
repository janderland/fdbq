package fullscreen

import (
	"context"
	"github.com/janderland/fql/internal/app/fullscreen/results"
	"github.com/janderland/fql/internal/app/fullscreen/stack"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lip "github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"

	"github.com/janderland/fql/engine"
	"github.com/janderland/fql/internal/app/fullscreen/manager"
	"github.com/janderland/fql/parser/format"
)

type App struct {
	Engine engine.Engine
	Format format.Format
	Log    zerolog.Logger
	Out    io.Writer

	Write      bool
	SingleOpts engine.SingleOpts
	RangeOpts  engine.RangeOpts
}

func (x *App) Run(ctx context.Context) error {
	input := textinput.New()
	input.Placeholder = "Query"

	resultsStack := stack.ResultsStack{}
	resultsStack.Push(results.New(
		results.WithFormat(x.Format),
		results.WithLogger(x.Log)))

	model := Model{
		mode: modeScroll,
		log:  x.Log,

		style: Style{
			results: lip.NewStyle().
				Padding(0, 1),

			input: lip.NewStyle().
				Border(lip.RoundedBorder()).
				Padding(0, 1),
		},

		results: resultsStack,
		input:   input,

		qm: manager.New(
			ctx,
			x.Engine,
			manager.WithSingleOpts(x.SingleOpts),
			manager.WithRangeOpts(x.RangeOpts),
			manager.WithWrite(x.Write)),
	}

	_, err := tea.NewProgram(
		model,
		tea.WithContext(ctx),
		tea.WithOutput(x.Out),
		tea.WithAltScreen(),
	).Run()
	return err
}

type Mode int

const (
	modeScroll Mode = iota
	modeInput
	modeHelp
	modeQuit
)

type Style struct {
	results lip.Style
	input   lip.Style
}

type Model struct {
	mode   Mode
	latest time.Time
	log    zerolog.Logger

	style   Style
	input   textinput.Model
	results stack.ResultsStack
	qm      manager.QueryManager
}

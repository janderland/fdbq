package fullscreen

import (
	"regexp"
	"strings"

	"github.com/janderland/fql/internal/app/fullscreen/results"
)

var helpMsg string

func init() {
	const str = `
FQL provides an environment for querying and mutating
data of a Foundation DB cluster. The environment has
3 modes: input, scroll, & help. The environment starts
in input mode. 

During input mode, the user can type queries into the
input box at the bottom of the screen. Pressing "enter"
cancels the currently executing query, clears the on
screen results, and executes a new query defined by
the input box. Pressing "up" or "down" scrolls by line.
Pressing "page up" or "page down" scrolls by page.
Pressing "escape" switches to scroll mode.

During scroll mode, the user can scroll through the
results of the previously executed query. Pressing
"up", "down", "page up", or "page down" scrolls as in
input mode. Pressing "j" or "k" scrolls by line.
Pressing "J" or "K" scrolls by item. Pressing "ctrl+d"
or "ctrl+u" scrolls by half page. Pressing "i"
switches back to input mode. Pressing "?" switches to
help mode. Pressing "q" quits the application after
confirmation.

During help mode, this help screen is displayed.
Scrolling works the same as in scroll mode. Pressing
"escape" switches to scroll mode.
`

	// Remove leading & trailing whitespace.
	helpMsg = strings.TrimSpace(str)

	// Concat paragraphs into single lines.
	helpMsg = regexp.MustCompile(`([^\n])\n([^\n])`).ReplaceAllString(helpMsg, "$1 $2")

	// Remove empty lines.
	helpMsg = regexp.MustCompile(`\n\n([^\n])`).ReplaceAllString(helpMsg, "\n$1")
}

func newHelp() results.Model {
	x := results.New(results.WithSpaced(true))
	for _, str := range strings.Split(helpMsg, "\n") {
		x.Push(str)
	}
	x.MaxWrapWidth(65)
	return x
}

func newQuit() results.Model {
	x := results.New()
	x.Push("Are you sure you want to quit? (y/n)")
	return x
}

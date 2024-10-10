package stack

import "github.com/janderland/fql/internal/app/fullscreen/results"

type ResultsStack struct {
	stack []results.Model

	height    int
	wrapWidth int
}

func (x *ResultsStack) Push(model results.Model) {
	model.Height(x.height)
	model.WrapWidth(x.wrapWidth)
	x.stack = append(x.stack, model)
}

func (x *ResultsStack) Pop() {
	if len(x.stack) != 0 {
		x.stack = x.stack[:len(x.stack)-1]
	}
}

func (x *ResultsStack) Top() *results.Model {
	if len(x.stack) == 0 {
		return nil
	}
	return &x.stack[len(x.stack)-1]
}

func (x *ResultsStack) Height(height int) {
	x.height = height
	for i := range x.stack {
		x.stack[i].Height(height)
	}
}

func (x *ResultsStack) WrapWidth(width int) {
	x.wrapWidth = width
	for i := range x.stack {
		x.stack[i].WrapWidth(width)
	}
}

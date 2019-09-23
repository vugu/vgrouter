package vgrouter

import "net/url"

// NavigatorOpt is a marker interface to ensure that options to Navigator are passed intentionally.
type NavigatorOpt interface {
	IsNavigatorOpt()
}

type intNavigatorOpt int

func (i intNavigatorOpt) IsNavigatorOpt() {}

var (
	// NavReplace will cause this navigation to replace the
	// current history entry rather than pushing to the stack.
	// Implemented using window.history.replaceState()
	NavReplace NavigatorOpt = intNavigatorOpt(1)

	// NavSkipRender will cause this navigation to not re-render
	// the current component state.  It can be used when a component
	// has already accounted for the render in some other way and
	// just wants to inform the Navigator of the current logical path and query.
	NavSkipRender NavigatorOpt = intNavigatorOpt(2)
)

type Navigator interface {
	Navigate(path string, query url.Values, opts ...NavigatorOpt)
}

// NOTE: here's an idea of how to support injection of values during component creation without much hassle:
type NavigatorRef struct {
	Navigator // embed Navigator
}

func (h *NavigatorRef) NavigatorSet(o Navigator) {
	h.Navigator = o
}

type NavigatorSetter interface {
	NavigatorSet(Navigator)
}

// TODO: should we make NavQuery interface {Navigator;QueryUpdater} or similar?

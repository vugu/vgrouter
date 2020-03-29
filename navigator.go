package vgrouter

import "net/url"

// Navigator interface has the methods commonly needed throughout the application
// to go to different pages and deal with routing.
type Navigator interface {

	// MustNavigate is like Navigate but panics upon error.
	MustNavigate(path string, query url.Values, opts ...NavigatorOpt)

	// Navigate will go the specified path and query.
	Navigate(path string, query url.Values, opts ...NavigatorOpt) error

	// Push will take any bound parameters and put them into the URL in the appropriate place.
	// Only works in wasm environment otherwise has no effect.
	Push(opts ...NavigatorOpt) error
}

// NavigatorRef embeds a reference to a Navigator and provides a NavigatorSet method.
type NavigatorRef struct {
	Navigator
}

// NavigatorSet implements NavigatorSetter interface.
func (nr *NavigatorRef) NavigatorSet(v Navigator) {
	nr.Navigator = v
}

// NavigatorSetter is implemented by things that can accept a Navigator.
type NavigatorSetter interface {
	NavigatorSet(v Navigator)
}

// NavigatorOpt is a marker interface to ensure that options to Navigator are passed intentionally.
type NavigatorOpt interface {
	IsNavigatorOpt()
}

type intNavigatorOpt int

// IsNavigatorOpt implements NavigatorOpt.
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

type navOpts []NavigatorOpt

func (no navOpts) has(o NavigatorOpt) bool {
	for _, o2 := range no {
		if o == o2 {
			return true
		}
	}
	return false
}

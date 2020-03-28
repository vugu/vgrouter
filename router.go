package vgrouter

import (
	"net/url"
)

// TODO:
// * more tests
// * implplement js stuff and hash more
// * do tests in wasm test suite
// * make codegen directory router

// EventEnv is our view of a Vugu EventEnv
type EventEnv interface {
	Lock()         // acquire write lock
	UnlockOnly()   // release write lock
	UnlockRender() // release write lock and request re-render

	// RLock()   // acquire read lock
	// RUnlock() // release read lock
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

// New returns a new Router.
func New(eventEnv EventEnv) *Router {
	// FIXME: how do we account for NavSkipRender?  Is it even needed?  The render loop is controlled outside of
	// this so maybe just leave rendering outside of the router altogether.

	// TODO: WE NEED TO THINK ABOUT SYNCHRONIZATION FOR ALL THIS STUFF - WHAT HAPPENS IF A
	// GOROUTINE TRIES TO NAVIGATE WHILE SOMETHING ELSE IS HAPPENING?

	return &Router{
		eventEnv:     eventEnv,
		bindParamMap: make(map[string]BindParam),
	}
}

// Router handles URL routing.
type Router struct {
	useFragment bool

	eventEnv EventEnv

	rlist           []routeEntry
	notFoundHandler RouteHandler

	// bindRoutePath string // the route (with :param stuff in it) that matches the bind params, so we can reconstruct it
	bindRouteMPath mpath
	bindParamMap   map[string]BindParam
}

type routeEntry struct {
	mpath mpath
	rh    RouteHandler
}

// UseFragment sets the fragment flag which if set means the fragment part of the URL (after the "#")
// is used as the path and query string.  This can be useful for compatibility in applications which are
// served statically and do not have the ability to handle URL routing on the server side.
// This option is disabled by default.  If used it should be set immediately after creation.  Changing it
// after navigation may have undefined results.
func (r *Router) UseFragment(v bool) {
	r.useFragment = v
	// TODO: if avail, register/unregister for fragment change event
}

// MustNavigate is like Navigate but panics upon error.
func (r *Router) MustNavigate(path string, query url.Values, opts ...NavigatorOpt) {
	err := r.Navigate(path, query, opts...)
	if err != nil {
		panic(err)
	}
}

// Navigate will go the specified path and query.
func (r *Router) Navigate(path string, query url.Values, opts ...NavigatorOpt) error {

	pq := path
	q := query.Encode()
	if len(q) > 0 {
		pq = pq + "?" + q
	}

	if navOpts(opts).has(NavReplace) {
		r.replacePathAndQuery(pq)
	} else {
		r.pushPathAndQuery(pq)
	}

	return nil
}

// Pull will read the current browser URL and navigate to it.  This is generally called
// once at application startup.
// Only works in wasm environment otherwise has no effect and will return error.
func (r *Router) Pull() error {

	u, err := r.readBrowserURL()
	if err != nil {
		return err
	}

	r.process(u.Path, u.Query())

	return nil
}

// Push will take any bound parameters and put them into the URL in the appropriate place.
// Only works in wasm environment otherwise has no effect.
func (r *Router) Push(opts ...NavigatorOpt) error {

	params := make(url.Values, len(r.bindParamMap))
	for k, v := range r.bindParamMap {
		params[k] = v.BindParamRead()
	}

	outPath, outParams, err := r.bindRouteMPath.merge(params)
	if err != nil {
		return err
	}

	q := outParams.Encode()
	pq := outPath
	if len(q) > 0 {
		pq = pq + "?" + q
	}

	if navOpts(opts).has(NavReplace) {
		r.replacePathAndQuery(pq)
	} else {
		r.pushPathAndQuery(pq)
	}

	return nil
}

// UnbindParams will remove any previous parameter bindings.
// Note that this is called implicitly when navigiation occurs since that involves re-binding newly based on the
// path being navigated to.
func (r *Router) UnbindParams() {
	for k := range r.bindParamMap {
		delete(r.bindParamMap, k)
	}
}

// MustAddRoute is like AddRoute but panics upon error.
func (r *Router) MustAddRoute(path string, rh RouteHandler) {
	err := r.AddRoute(path, rh)
	if err != nil {
		panic(err)
	}
}

// AddRoute adds a route to the list.
func (r *Router) AddRoute(path string, rh RouteHandler) error {

	mp, err := parseMpath(path)
	if err != nil {
		return err
	}

	r.rlist = append(r.rlist, routeEntry{
		mpath: mp,
		rh:    rh,
	})

	return nil
}

// SetNotFound assigns the handler for the case of no exact match reoute.
func (r *Router) SetNotFound(rh RouteHandler) {
	r.notFoundHandler = rh
}

// process is used interally to run through the routes and call appropriate handlers.
// It will set bindRouteMPath and unbind the params and allow them to be reset.
func (r *Router) process(path string, query url.Values) {

	// TODO: ideally we would improve the performance here with some fancy trie stuff, but for the moment
	// I'm much more concerned with getting things functional.

	for k := range r.bindParamMap {
		delete(r.bindParamMap, k)
	}
	r.bindRouteMPath = nil
	foundExact := false

	for _, re := range r.rlist {

		pvals, exact, ok := re.mpath.match(path)
		if !ok {
			continue
		}

		if !foundExact && exact {
			foundExact = true
			r.bindRouteMPath = re.mpath
		}

		// merge any other values from query into pvals
		for k, v := range query {
			if pvals[k] == nil {
				pvals[k] = v
			}
		}

		req := &RouteMatch{
			router:    r,
			Path:      path,
			RoutePath: re.mpath.String(),
			Params:    pvals,
			Exact:     exact,
		}

		re.rh.RouteHandle(req)

	}

	if !foundExact && r.notFoundHandler != nil {
		r.notFoundHandler.RouteHandle(&RouteMatch{
			router: r,
			Path:   path,
		})
	}

}

// RouteHandler implementations are called in response to a route matching (being navigated to).
type RouteHandler interface {
	RouteHandle(rm *RouteMatch)
}

// RouteHandlerFunc implements RouteHandler as a function.
type RouteHandlerFunc func(rm *RouteMatch)

// RouteHandle implements the RouteHandler interface.
func (f RouteHandlerFunc) RouteHandle(rm *RouteMatch) { f(rm) }

// RouteMatch describes a request to navigate to a route.
type RouteMatch struct {
	Path      string     // path input (with any params interpolated)
	RoutePath string     // route path pattern with params as :param
	Params    url.Values // parameters (combined query and route params)
	Exact     bool       // true if the path is an exact match or false if just the prefix

	router *Router
}

// Bind adds a BindParam to the list of bound parameters.
// Later calls to Bind with the same name will replace the bind
// from earlier calls.
func (r *RouteMatch) Bind(name string, param BindParam) {
	if r.router.bindParamMap == nil {
		r.router.bindParamMap = make(map[string]BindParam)
	}
	r.router.bindParamMap[name] = param
}

// BindParam is implemented by something that can be read and written as a URL param.
type BindParam interface {
	BindParamRead() []string
	BindParamWrite(v []string)
}

// StringParam implements BindParam on a string.
type StringParam string

// BindParamRead implements BindParam.
func (s *StringParam) BindParamRead() []string { return []string{string(*s)} }

// BindParamWrite implements BindParam.
func (s *StringParam) BindParamWrite(v []string) {
	if len(*s) == 0 {
		*s = ""
		return
	}
	*s = StringParam(v[0])
}

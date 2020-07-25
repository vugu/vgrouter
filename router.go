package vgrouter

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/vugu/vugu/js"
)

// TODO:
// * more tests DONE
// * need path prefix support - wasm test suite needs it plus other
// * need a method to just say "process this" and a variation of that which accepts an http.Request and sets it on the RouteMatch
// * implement js stuff and fragment
// * do tests in wasm test suite
//   - one test can cover with and without fragement by detecting "#" upon page load
// * do the change to generate to _gen.go, allow MixedCase.vugu, and put in a banner
//   at the top of the generated file and detect before clobbering it
// * make codegen directory router
//   also make it output a list of files, so static generator can use it
//   need index functionanlity plus see what we do about parameters if we can support

// EventEnv is our view of a Vugu EventEnv
type EventEnv interface {
	Lock()         // acquire write lock
	UnlockOnly()   // release write lock
	UnlockRender() // release write lock and request re-render

	// RLock()   // acquire read lock
	// RUnlock() // release read lock
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
	pathPrefix  string

	popStateFunc js.Func

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

// SetUseFragment sets the fragment flag which if set means the fragment part of the URL (after the "#")
// is used as the path and query string.  This can be useful for compatibility in applications which are
// served statically and do not have the ability to handle URL routing on the server side.
// This option is disabled by default.  If used it should be set immediately after creation.  Changing it
// after navigation may have undefined results.
func (r *Router) SetUseFragment(v bool) {
	r.useFragment = v
}

// SetPathPrefix sets the path prefix to use prepend or stripe when iteracting with the browser or external requests.
// Internally paths do not use this prefix.
// For example, calling `r.SetPrefix("/pfx"); r.Navigate("/a", nil)` will result in /pfx/a in the browser, but the
// path will be treated as just /a during processing.  The prefix is stripped from the URL when calling Pull()
// and also when http.Requests are processed server-side.
func (r *Router) SetPathPrefix(pfx string) {
	r.pathPrefix = pfx
}

// ListenForPopState registers an event listener so the user navigating with
// forward/back/history or fragment changes will be detected and handled by this router.
// Any call to SetUseFragment or SetPathPrefix should occur before calling
// ListenForPopState.
//
// Only works in wasm environment and if called outside it will have no effect and return error.
func (r *Router) ListenForPopState() error {
	return r.addPopStateListener(func(this js.Value, args []js.Value) interface{} {

		// TODO: see if we need something better for error handling

		// log.Printf("addPopStateListener callack")

		u, err := r.readBrowserURL()
		// log.Printf("addPopStateListener callack: u=%#v, err=%v", u, err)
		if err != nil {
			log.Printf("ListenForPopState: error from readBrowserURL: %v", err)
			return nil
		}

		p := u.Path
		if !strings.HasPrefix(p, r.pathPrefix) {
			log.Printf("ListenForPopState: prefix error: %v",
				ErrMissingPrefix{Path: p, Message: fmt.Sprintf("path %q does not begin with prefix %q", p, r.pathPrefix)})
			return nil
		}

		tp := strings.TrimPrefix(p, r.pathPrefix)
		q := u.Query()

		// log.Printf("addPopStateListener calling process: tp=%q, q=%#v", tp, q)

		r.eventEnv.Lock()
		defer r.eventEnv.UnlockRender()
		r.process(tp, q)

		return nil

	})
}

// UnlistenForPopState removes the listener created by ListenForPopState.
func (r *Router) UnlistenForPopState() error {
	return r.removePopStateListener()
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

	r.process(path, query)

	pq := r.pathPrefix + path
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

// BrowserAvail returns true if in browser mode.
func (r *Router) BrowserAvail() bool {
	// this is really just so otehr packages don't have to import `js` just to figure out if they should do extra browser setup
	return js.Global().Truthy()
}

// ErrMissingPrefix is returned when a prefix was expected but not found.
type ErrMissingPrefix struct {
	Message string // error message
	Path    string // path which is missing the prefix
}

// Error implements error.
func (e ErrMissingPrefix) Error() string { return e.Message }

// Pull will read the current browser URL and navigate to it.  This is generally called
// once at application startup.
// Only works in wasm environment otherwise has no effect and will return error.
// If a path prefix has been set and the path read does not start with prefix
// then an error of type *ErrMissingPrefix will be returned.
func (r *Router) Pull() error {

	u, err := r.readBrowserURL()
	if err != nil {
		return err
	}

	p := u.Path
	if !strings.HasPrefix(p, r.pathPrefix) {
		return ErrMissingPrefix{Path: p, Message: fmt.Sprintf("path %q does not begin with prefix %q", p, r.pathPrefix)}
	}

	r.process(strings.TrimPrefix(p, r.pathPrefix), u.Query())

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
	pq := r.pathPrefix + outPath
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

// MustAddRouteExact is like AddRouteExact but panic's upon error.
func (r *Router) MustAddRouteExact(path string, rh RouteHandler) {
	err := r.AddRouteExact(path, rh)
	if err != nil {
		panic(err)
	}
}

// AddRouteExact adds a route but only calls the handler if the path
// provided matches exactly. E.g. an exact route for "/a" will not fire
// when "/a/b" is navigated to (whereas AddRoute would do this).
func (r *Router) AddRouteExact(path string, rh RouteHandler) error {
	return r.AddRoute(path, RouteHandlerFunc(func(rm *RouteMatch) {
		if rm.Exact {
			rh.RouteHandle(rm)
		}
	}))
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

// GetNotFound returns what was set by SetNotFound.  Provided to facilitate code that needs
// to wrap an existing not found behavior with another one.
func (r *Router) GetNotFound() RouteHandler {
	return r.notFoundHandler
}

// ProcessRequest processes the route contained in request. This is meant for server-side use with static rendering.
func (r *Router) ProcessRequest(req *http.Request) {

	p := req.URL.Path
	q := req.URL.Query()

	r.process2(p, q, req)

}

// process is used interally to run through the routes and call appropriate handlers.
// It will set bindRouteMPath and unbind the params and allow them to be reset.
func (r *Router) process(path string, query url.Values) {
	r.process2(path, query, nil)
}

func (r *Router) process2(path string, query url.Values, req *http.Request) {

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
		if pvals == nil {
			pvals = make(url.Values)
		}
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
			Request:   req,
		}

		re.rh.RouteHandle(req)

	}

	if !foundExact && r.notFoundHandler != nil {
		r.notFoundHandler.RouteHandle(&RouteMatch{
			router:  r,
			Path:    path,
			Request: req,
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

	Request *http.Request // if ProcessRequest is used, this will be set to Request instance passed to it; server-side only

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

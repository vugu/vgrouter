package vgrouter

import "regexp"

// func New() *Router {
// 	return &Router{}
// }

// type Router interface {
// }

type RouteList struct {
	routeEntryList []routeEntry
}

type routeEntry struct {
	pathPattern *regexp.Regexp
}

func (r *RouteList) AddRoute(path string, h RouteHandler) {

}

type RouteHandler interface {
	HandleRoute(path string)
}

type RouteHandlerFunc func(path string)

func (f RouteHandlerFunc) HandleRoute(path string) { f(path) }

// // BrowseTo is like BrowseToURL but accepts a string instead.
// // The pathEtc param is the portion of a URL from the path part onward to the right
// // (including any URL params). It must not include a scheme, hostname, etc.
// func (r *Router) BrowseTo(pathEtc string) {
// }

// // BrowseToURL causes the logical app URL of the browser to be changed to the following path.
// // The applicable route handling code will then be called to update as appropriate and re-render.
// // Portions of the URL before the path (i.e. scheme, hostname, etc.) are ignored.
// func (r *Router) BrowseToURL(u *url.URL) {
// }

// // UpdateParams causes the logical URL params of the app URL to be updated based on current
// // param bindings.  It does not cause a re-render.
// // If addToHistory is true then the change is added to the browser history, otherwise not.
// func (r *Router) UpdateParams(addToHistory bool) {
// }

// REQUIREMENTS:
// - two modes/impls for url path vs url fragment
// - param binding mechanism

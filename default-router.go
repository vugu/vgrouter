package vgrouter

import "net/url"

type DefaultRouter struct {
	RouteList
	QueryBind
	UseFragment bool
}

// // BrowseTo is like BrowseToURL but accepts a string instead.
// // The pathEtc param is the portion of a URL from the path part onward to the right
// // (including any URL params). It must not include a scheme, hostname, etc.
// func (r *DefaultRouter) BrowseTo(pathEtc string) {
// }

// // BrowseToURL causes the logical app URL of the browser to be changed to the following path.
// // The applicable route handling code will then be called to update as appropriate and re-render.
// // Portions of the URL before the path (i.e. scheme, hostname, etc.) are ignored.
// func (r *DefaultRouter) BrowseToURL(u *url.URL) {
// }

func (r *DefaultRouter) Navigate(path string, query url.Values, opts ...NavigatorOpt) {
}

func (r *DefaultRouter) QueryUpdate() {

}

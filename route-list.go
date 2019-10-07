package vgrouter

import (
	"net/url"
	"sort"
	"strings"
)

// func New() *Router {
// 	return &Router{}
// }

// type Router interface {
// }

// RouteList has a list of routes and their handlers, with functionality to index and process (call the router handlers) them.
// Internally a radix tree is used to combine common static prefixes and reduce the number of lookups required for large numbers of routes.
type RouteList struct {
	routeMatcherList []routeMatcher
	lastPriority     float64
	trie             *node
}

type routeMatcherFunc func(path string) (PathParamList, bool)

type routeMatcher struct {
	priority    float64 // lower is processed earlier
	prefix      string  // static prefix for radix tree
	prefixExact bool    // if true then prefix is an exact string, else it is a prefix and applies to children
	matcherFunc routeMatcherFunc
	handler     RouteHandler
}

type node struct {
	data             string
	children         []*node
	routeMatcherList []*routeMatcher
}

func (rl *RouteList) addStatic(path string, routeHandler RouteHandler) {
	rl.trie = nil // force reindex
	rl.lastPriority += 1.0
	rl.routeMatcherList = append(rl.routeMatcherList, routeMatcher{
		priority: rl.lastPriority,
		prefix:   path,
		matcherFunc: func(cpath string) (pp PathParamList, ok bool) {
			return nil, path == cpath
		},
		handler: routeHandler,
	})
}

func (rl *RouteList) addPrefix(path string, routeHandler RouteHandler) {
	rl.trie = nil // force reindex
	rl.lastPriority += 1.0
	rl.routeMatcherList = append(rl.routeMatcherList, routeMatcher{
		priority: rl.lastPriority,
		prefix:   path,
		matcherFunc: func(cpath string) (pp PathParamList, ok bool) {
			return nil, strings.HasPrefix(cpath, path)
		},
		handler: routeHandler,
	})
}

// func runeSplit(s string) (ret []rune) {
// 	ret = make([]rune, 0, len(s))
// 	for _, r := range s {
// 		ret = append(ret, r)
// 	}
// 	return
// }

// index builds the radix tree (trie)
func (rl *RouteList) index() error {

	// // carve up the prefix strings into rune slices
	// prefixes := make([][]rune, len(rl.routeMatcherList))
	// for i, rm := range rl.routeMatcherList {
	// 	p := runeSplit(rm.prefix)
	// 	if len(p) == 0 {
	// 		return fmt.Errorf("cannot index empty prefix")
	// 	}
	// 	prefixes[i] = p
	// }

	var buildNode func(depth int, rmlIn []*routeMatcher, rmlExtra []*routeMatcher) *node
	buildNode = func(depth int, rmlIn []*routeMatcher, rmlExtra []*routeMatcher) *node {

		// TODO:
		// spin over with delta 0 and make a node,
		// flag if this is at the end of any prefix
		// then spin over with delta+1 and see if all prefixes are the same,
		// if so then update output node with new delta,
		// if not then for each difference call buildNode for child and add children to output node
		// return

		// nready := false

		var n node
		// d2 := depth
		// for {

		// 	pfx := ""

		// 	for i := range rmlIn {
		// 		rm := rmlIn[i]

		// 		rmpfx := rm.prefix[depth:d2]

		// 		if pfx == "" {
		// 			pfx = rmpfx
		// 		}

		// 		if rmpfx != pfx {

		// 		}

		// 		// pfx

		// 		// if strings.HasPrefix(rm.prefix, pfx) {
		// 		// }

		// 	}
		// }

		return &n
	}

	rml := make([]*routeMatcher, 0, len(rl.routeMatcherList))
	for i := range rl.routeMatcherList {
		rml = append(rml, &rl.routeMatcherList[i])
	}

	// buildNode expects these to be sorted by prefix
	sort.SliceStable(rml, func(i, j int) bool {
		return rml[i].prefix < rml[j].prefix
	})

	rl.trie = buildNode(0, rml, nil)

	return nil
}

func longestPrefix(start int, rmlIn []*routeMatcher) string {
	if len(rmlIn) == 0 {
		panic("longestPrefix called with empty list")
	}
	pfx := rmlIn[0].prefix[:start]
	foundEnd := false
	_ = foundEnd
	for end := start; ; end++ {

		for i := range rmlIn {

			// can't go past a prefix that ends
			if len(rmlIn[i].prefix) == end {
				foundEnd = true
			}

			pfx2 := rmlIn[i].prefix[:end]
			if pfx2 != pfx {
				foundEnd = true
			}
		}
	}
	return pfx
}

// Process matches against the route list and calls all match RouteHandlers.
// Static prefixes are looked up in a radix tree as an optimization to avoid unnecessary comparisons.
func (rl *RouteList) Process(u *url.URL) {
	if rl.trie == nil {
		rl.index()
	}

}

// func (r *RouteList) AddRoute(path string, h RouteHandler) {

// }

type RouteHandler interface {
	HandleRoute(u *url.URL, pp PathParamList)
}

type RouteHandlerFunc func(u *url.URL, pp PathParamList)

func (f RouteHandlerFunc) HandleRoute(u *url.URL, pp PathParamList) { f(u, pp) }

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

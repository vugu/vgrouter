package vgrouter

import (
	"fmt"
	"log"
	"net/url"
	"testing"
)

func TestRouter(t *testing.T) {

	type appRouter struct {
		*Router
		out map[string]RouteMatch
	}

	type tcase struct {
		path   string // the path to request
		params string
		rp     []string // route paths for which we AddRoute
		check  func(ar *appRouter) bool
	}

	tclist := []tcase{

		{
			"/",
			"",
			[]string{"/"},
			func(ar *appRouter) bool { return ar.out["/"].Path == "/" },
		},

		{
			"/nothing",
			"",
			[]string{"/"},
			func(ar *appRouter) bool { return ar.out["_not_found"].Path == "/nothing" },
		},

		{
			"/a",
			"",
			[]string{"/a"},
			func(ar *appRouter) bool {
				return ar.out["/a"].Path == "/a" &&
					ar.out["/a"].Exact
			},
		},

		{
			"/a",
			"",
			[]string{"/", "/a"},
			func(ar *appRouter) bool {
				return ar.out["/"].Path == "/a" &&
					!ar.out["/"].Exact &&
					ar.out["/a"].Path == "/a" &&
					ar.out["/a"].Exact
			},
		},

		{
			"/a/v1",
			"",
			[]string{"/", "/a", "/a/:id"},
			func(ar *appRouter) bool {
				// log.Printf("len(ar.out): %#v", len(ar.out))
				// log.Printf("ar.out: %#v", ar.out)
				return ar.out["/a"].Path == "/a/v1" &&
					!ar.out["/a"].Exact &&
					ar.out["/a/:id"].Exact &&
					ar.out["/a/:id"].Path == "/a/v1"
			},
		},

		{
			"/",
			"p=1",
			[]string{"/"},
			func(ar *appRouter) bool {
				return ar.out["/"].Path == "/" &&
					ar.out["/"].Exact &&
					ar.out["/"].Params.Get("p") == "1"
			},
		},

		{
			"/",
			"p=1&q=2",
			[]string{"/"},
			func(ar *appRouter) bool {
				return ar.out["/"].Path == "/" &&
					ar.out["/"].Exact &&
					ar.out["/"].Params.Get("p") == "1" &&
					ar.out["/"].Params.Get("q") == "2"
			},
		},

		{
			"/a",
			"p=1",
			[]string{"/", "/a"},
			func(ar *appRouter) bool {
				return ar.out["/"].Path == "/a" &&
					!ar.out["/"].Exact &&
					ar.out["/a"].Params.Get("p") == "1" &&
					ar.out["/a"].Path == "/a" &&
					ar.out["/a"].Exact &&
					ar.out["/a"].Params.Get("p") == "1"
			},
		},

		{
			"/a/v1",
			"p=1",
			[]string{"/", "/a", "/a/:id"},
			func(ar *appRouter) bool {
				return ar.out["/"].Path == "/a/v1" &&
					!ar.out["/"].Exact &&
					ar.out["/a"].Params.Get("p") == "1" &&
					ar.out["/a"].Path == "/a/v1" &&
					!ar.out["/a"].Exact &&
					ar.out["/a"].Params.Get("p") == "1" &&
					ar.out["/a/:id"].Path == "/a/v1" &&
					ar.out["/a/:id"].Exact &&
					ar.out["/a/:id"].Params.Get("p") == "1" &&
					ar.out["/a/:id"].Params.Get("id") == "v1"
			},
		},
	}

	for i, tc := range tclist {
		t.Run(fmt.Sprint(i), func(t *testing.T) {

			ar := appRouter{Router: New(nil), out: make(map[string]RouteMatch)}
			for _, p := range tc.rp {
				p := p
				// log.Printf("adding route for %q", p)
				ar.MustAddRoute(p, RouteHandlerFunc(func(rm *RouteMatch) {
					// log.Printf("got route handle for %q", p)
					ar.out[p] = *rm
				}))
			}
			ar.SetNotFound(RouteHandlerFunc(func(rm *RouteMatch) {
				ar.out["_not_found"] = *rm
			}))

			params, err := url.ParseQuery(tc.params)
			if err != nil {
				log.Printf("Error parsing supplied test case parameters - %v\n", err)
				t.Fail()
			}

			ar.process(tc.path, params)

			if !tc.check(&ar) {
				t.Fail()
			}

		})
	}

}

package vgrouter

import (
	"fmt"
	"testing"
)

func TestRouter(t *testing.T) {

	type appRouter struct {
		*Router
		out map[string]RouteMatch
	}

	type tcase struct {
		path  string   // the path to request
		rp    []string // route paths for which we AddRoute
		check func(ar *appRouter) bool
	}

	tclist := []tcase{

		{
			"/",
			[]string{"/"},
			func(ar *appRouter) bool { return ar.out["/"].Path == "/" },
		},

		{
			"/nothing",
			[]string{"/"},
			func(ar *appRouter) bool { return ar.out["_not_found"].Path == "/nothing" },
		},

		{
			"/a",
			[]string{"/", "/a"},
			func(ar *appRouter) bool { return ar.out["/a"].Path == "/a" },
		},

		{
			"/a/v1",
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
	}

	for i, tc := range tclist {
		t.Run(fmt.Sprint(i), func(t *testing.T) {

			ar := appRouter{Router: New(nil), out: make(map[string]RouteMatch)}
			for _, p := range tc.rp {
				ar.MustAddRoute(p, RouteHandlerFunc(func(rm *RouteMatch) {
					ar.out[p] = *rm
				}))
			}
			ar.SetNotFound(RouteHandlerFunc(func(rm *RouteMatch) {
				ar.out["_not_found"] = *rm
			}))

			ar.process(tc.path, nil)

			if !tc.check(&ar) {
				t.Fail()
			}

		})
	}

}

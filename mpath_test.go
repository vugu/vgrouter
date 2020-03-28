package vgrouter

import (
	"net/url"
	"reflect"
	"testing"
)

func TestMPathParse(t *testing.T) {

	var tlist = []struct {
		in  string
		out mpath
	}{
		{"/", mpath{"/"}},
		{"/:p1", mpath{"/", ":p1"}},
		{"/:p1/", mpath{"/", ":p1"}},
		{"/:p1/test", mpath{"/", ":p1", "/test"}},
		{"/:p1/test/:p2", mpath{"/", ":p1", "/test/", ":p2"}},
		{"/:p1/:p2", mpath{"/", ":p1", "/", ":p2"}},
		{"/a/b", mpath{"/a", "/b"}},
	}

	for _, ti := range tlist {
		t.Run(ti.in, func(t *testing.T) {
			mp, err := parseMpath(ti.in)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(ti.out, mp) {
				t.Errorf("expected %#v, got %#v", ti.out, mp)
			}
		})
	}

}

func TestMPathMergeMatch(t *testing.T) {

	var tlist = []struct {
		inpath string
		mpath  mpath
		pvals  url.Values
	}{
		{"/", mpath{"/"}, nil},
		{"/somewhere", mpath{"/", ":id"}, url.Values{"id": []string{"somewhere"}}},
		{"/blah/somewhere", mpath{"/blah/", ":id"}, url.Values{"id": []string{"somewhere"}}},
		{"/blah/somewhere/something", mpath{"/blah/", ":id", "/", ":id2"}, url.Values{"id": []string{"somewhere"}, "id2": []string{"something"}}},
	}

	for _, ti := range tlist {
		t.Run(ti.inpath, func(t *testing.T) {
			pv, _, ok := ti.mpath.match(ti.inpath)
			if !ok {
				t.Errorf("got ok false")
			}
			if !reflect.DeepEqual(ti.pvals, pv) {
				t.Errorf("expected params %#v, got %#v", ti.pvals, pv)
			}
			p2, _, err := ti.mpath.merge(pv)
			if err != nil {
				t.Errorf("merge error: %w", err)
			}
			if p2 != ti.inpath {
				t.Errorf("expected p2 %#v, got %#v", ti.inpath, p2)
			}
		})
	}

}

func TestMPathMatchExact(t *testing.T) {

	var tlist = []struct {
		inpath string
		mpath  mpath
		exact  bool
		ok     bool
	}{
		{"/", mpath{"/"}, true, true},
		{"/somewhere", mpath{"/"}, false, true},
		{"/somewhere/here", mpath{"/somewhere"}, false, true},
		{"/somewhere", mpath{"/somewhere"}, true, true},
		{"/somewhere/1", mpath{"/somewhere/", ":id"}, true, true},
		{"/somewhere/1/2", mpath{"/somewhere/", ":id"}, false, true},
	}

	for _, ti := range tlist {
		t.Run(ti.inpath, func(t *testing.T) {
			_, exact, ok := ti.mpath.match(ti.inpath)
			if !(ok == ti.ok) {
				t.Errorf("expected ok %#v, got %#v", ti.ok, ok)
			}
			if !(exact == ti.exact) {
				t.Errorf("expected exact %#v, got %#v", ti.exact, exact)
			}
		})
	}

}

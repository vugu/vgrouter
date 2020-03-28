package vgrouter

import (
	"errors"
	"net/url"
	"strings"

	"github.com/vugu/vugu/js"
)

func (r *Router) pushPathAndQuery(pathAndQuery string) {

	g := js.Global()
	if g.Truthy() {
		pqv := pathAndQuery
		if r.useFragment {
			pqv = "#" + pathAndQuery
		}
		g.Get("window").Get("history").Call("pushState", nil, "", pqv)
	}

}

func (r *Router) replacePathAndQuery(pathAndQuery string) {

	g := js.Global()
	if g.Truthy() {
		pqv := pathAndQuery
		if r.useFragment {
			pqv = "#" + pathAndQuery
		}
		g.Get("window").Get("history").Call("replaceState", nil, "", pqv)
	}

}

func (r *Router) readBrowserURL() (*url.URL, error) {

	g := js.Global()
	if !g.Truthy() {
		return nil, errors.New("not in browser (js) environment")
	}

	var locstr string
	if r.useFragment {
		locstr = strings.TrimPrefix(js.Global().Get("window").Get("location").Get("hash").Call("toString").String(), "#")
	} else {
		locstr = js.Global().Get("window").Get("location").Call("toString").String()
	}

	u, err := url.Parse(locstr)
	if err != nil {
		return u, err
	}

	return u, nil

}

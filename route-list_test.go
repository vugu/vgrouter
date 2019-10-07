package vgrouter

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteList(t *testing.T) {

	assert := assert.New(t)

	var outBuf bytes.Buffer

	outBuf.Reset()

	var rl RouteList
	rl.addPrefix("/", RouteHandlerFunc(func(u *url.URL, pp PathParamList) {
		outBuf.WriteString("/ [before]\n")
	}))
	rl.addPrefix("/test", RouteHandlerFunc(func(u *url.URL, pp PathParamList) {
		outBuf.WriteString("/test\n")
	}))
	rl.addPrefix("/testing", RouteHandlerFunc(func(u *url.URL, pp PathParamList) {
		outBuf.WriteString("/testing\n")
	}))
	rl.addPrefix("/testing/123", RouteHandlerFunc(func(u *url.URL, pp PathParamList) {
		outBuf.WriteString("/testing/123\n")
	}))
	rl.addPrefix("/testing/124", RouteHandlerFunc(func(u *url.URL, pp PathParamList) {
		outBuf.WriteString("/testing/124\n")
	}))
	rl.addPrefix("/tester", RouteHandlerFunc(func(u *url.URL, pp PathParamList) {
		outBuf.WriteString("/tester\n")
	}))
	rl.addPrefix("/", RouteHandlerFunc(func(u *url.URL, pp PathParamList) {
		outBuf.WriteString("/ [after]\n")
	}))

	u, err := url.Parse("/blah")
	assert.NoError(err)

	rl.Process(u)

}

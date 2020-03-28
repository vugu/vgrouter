package vgrouter

import (
	"bytes"
	"errors"
	"net/url"
	"path"
	"strings"
)

// parseMpath will split p into appropriate parts for an mpath.
// After parsing each element of mpath will either be a static
// string or a parameter starting with ":".
func parseMpath(p string) (mpath, error) {
	ret := make(mpath, 0, 2)
	p = path.Clean("/" + p)

	lastWasSlash := false
	inParam := false
	startIdx := 0

	for i := range p {

		c := p[i]

		if c == '/' {
			if inParam {
				ret = append(ret, p[startIdx:i])
				inParam = false
				startIdx = i
				continue
			}
			lastWasSlash = true
			continue
		}

		if lastWasSlash && c == ':' {
			ret = append(ret, p[startIdx:i])
			inParam = true
			startIdx = i
			continue
		}

	}

	// append last part if needed
	if startIdx < len(p) {
		ret = append(ret, p[startIdx:len(p)])
	}

	return ret, nil
}

// mpath is a matchable-path.  It's basically just a path split by parameter values.
type mpath []string

func (mp mpath) prefix() string {
	if len(mp) > 0 {
		return mp[0]
	}
	return ""
}

// paramNames will return the parameter names
// without the preceding colon, i.e. the path "/somewhere/:p1/:p2"
// will return []string{"p1","p2"}
func (mp mpath) paramNames() []string {
	var ret []string
	for _, p := range mp {
		if strings.HasPrefix(p, ":") {
			ret = append(ret, p[1:])
		}
	}
	return ret
}

// String returns the re-assembled path pattern
func (mp mpath) String() string {
	return strings.Join(mp, "")
}

var errMissingParam = errors.New("missing param")

// merge will use any values provided for the appropriate path params
// and return the constructed path.  A missing param value will cause
// errMissingParam to be returned but will still return the path with
// the missing param(s) replaced with "_".  The otherValues will
// be populated with all values not merged into the output path.
func (mp mpath) merge(v url.Values) (outPath string, otherValues url.Values, reterr error) {

	if len(v) > 0 {
		otherValues = make(url.Values, len(v))
		for k, val := range v {
			otherValues[k] = val
		}
	}

	var buf bytes.Buffer
	buf.Grow(64)

	for _, p := range mp {
		if strings.HasPrefix(p, ":") {
			pname := p[1:]
			vlist := v[pname]
			if len(vlist) == 0 { // it's only an error if no value provided, we want "?param=" to not error
				reterr = errMissingParam
				buf.WriteString("_")
				continue
			}
			buf.WriteString(vlist[0])
			otherValues.Del(pname)
			continue
		}
		buf.WriteString(p)
		continue
	}

	if len(otherValues) == 0 {
		otherValues = nil
	}

	return buf.String(), otherValues, reterr
}

// match compares our mpath to the path provided and returns the parameter
// values plus ok true if match.  If !exact it means the path matched but there is more after
func (mp mpath) match(p string) (paramValues url.Values, exact, ok bool) {

	prest := path.Clean("/" + p)

	readParam := func(pin string) (pr, pv string) {
		// log.Printf("readParam called with %q", pin)
		for i := range pin {
			if pin[i] == '/' {
				return pin[i:], pin[:i]
			}
		}
		// no slash means the entire input is the param value
		return "", pin
	}

	for _, mpart := range mp {

		// log.Printf("mp=%#v, mpart=%v, prest=%v", mp, mpart, prest)

		// log.Printf("mpart=%q", mpart)
		// read param
		if strings.HasPrefix(mpart, ":") {
			pname := mpart[1:]
			var pval string
			prest, pval = readParam(prest)
			if paramValues == nil {
				paramValues = make(url.Values, 2)
			}
			paramValues.Set(pname, pval)
			// log.Printf("GOT TO Set %q=%q", pname, pval)
			continue
		}
		// // check for exact match
		// if !strings.HasPrefix(prest, mpart) {
		// 	return
		// }
		// move past this part
		prest = prest[len(mpart):]
	}

	exact = prest == ""

	ok = true
	return
}

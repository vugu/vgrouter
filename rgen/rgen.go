package rgen

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// New returns a new Generator instance.
func New() *Generator {
	return &Generator{}
}

// Generator performs route generation on a given directory (and optionally sub-directories)
type Generator struct {
	dir         string                           // starting directory
	recursive   bool                             // if true we will descend into directories
	packageName string                           // fully qualified package name corresponding to dir
	pathFunc    func(fileName string) string     // function derive path from file or struct name
	includeFunc func(path, fileName string) bool // function to determine if a file should be included
}

// SetDir assigns the directory to start generating in.
func (g *Generator) SetDir(dir string) *Generator {
	g.dir = dir
	return g
}

// SetRecursive if passed true will enable the generator recursing
// into sub-directories.
func (g *Generator) SetRecursive(recursive bool) *Generator {
	g.recursive = recursive
	return g
}

// SetPackageName sets the fully qualified package name that corresponds
// with the directory set with SetDir.
func (g *Generator) SetPackageName(packageName string) *Generator {
	g.packageName = packageName
	return g
}

// SetPathFunc sets a function which transforms.
// If not set, DefaultPathFunc will be used.
func (g *Generator) SetPathFunc(f func(fileName string) string) *Generator {
	g.pathFunc = f
	return g
}

// SetIncludeFunc sets the function which determines which files are included in the route map.
// The include function will be passed the path relative to the dir set by SetDir (and will be empty
// for files in that directory) and fileName will contain the base file name.  E.g. given SetDir("/a")
// "/a/b.vugu" will result in a call with ("", "b.vugu"), and "/a/b/c.vugu" will result in a call
// with ("b", "c.vugu"), "/a/b/c/d.vugu" with ("b/c", "d.vugu") and so on.
func (g *Generator) SetIncludeFunc(f func(path, fileName string) bool) *Generator {
	g.includeFunc = f
	return g
}

// DefaultPathFunc will return the fileName with any suffix removed and a slash prepended.
// E.g. file name "example.vugu" will return "/example".  The special case of index.vugu
// will return "/".
func DefaultPathFunc(fileName string) string {
	if fileName == "index.vugu" {
		return "/"
	}
	return "/" + strings.TrimSuffix(fileName, path.Ext(fileName))
}

// DefaultIncludeFunc will return true for any file which ends with .vugu.
func DefaultIncludeFunc(path, fileName string) bool {
	return strings.HasSuffix(fileName, ".vugu")
}

// Generate does the route generation.
func (g *Generator) Generate() error {

	// to keep our sanity we need to guarantee that g.dir is absolute
	dir, err := filepath.Abs(g.dir)
	if err != nil {
		return err
	}
	g.dir = dir

	df, err := g.readDirf(g.dir)
	if err != nil {
		return err
	}

	// TODO: prune branches that have nothing to be generated underneath them

	err = g.writeRoutes(df)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) readDirf(dirPath string) (*dirf, error) {

	includeFunc := g.includeFunc
	if includeFunc == nil {
		includeFunc = DefaultIncludeFunc
	}

	f, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fis, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	rel, err := filepath.Rel(g.dir, dirPath)
	if err != nil {
		return nil, fmt.Errorf("relative path conversion failed: %w", err)
	}
	rel = strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(rel)), "/")

	ret := &dirf{
		path: rel,
	}

	for _, fi := range fis {

		if fi.IsDir() {
			if !g.recursive {
				continue
			}
			subdirf, err := g.readDirf(filepath.Join(dirPath, fi.Name()))
			if err != nil {
				return nil, err
			}
			if ret.subdirs == nil {
				ret.subdirs = make(map[string]*dirf)
			}
			ret.subdirs[fi.Name()] = subdirf
			continue
		}

		if includeFunc(rel, fi.Name()) {
			ret.fileNames = append(ret.fileNames, fi.Name())
		}
	}

	return ret, nil

}

type dirf struct {
	path      string           // path relative to g.dir
	fileNames []string         // list of included files
	subdirs   map[string]*dirf // children
}

func (g *Generator) writeRoutes(df *dirf) error {

	// TODO: we can try to auto-detect the import path by running `go list -json` and looking for ImportPath.

	_, localPackage := path.Split(df.path)
	if localPackage == "" {
		_, localPackage = filepath.Split(g.dir)
	}

	cm := map[string]interface{}{
		"LocalPackage": localPackage,
		"FileNameList": df.fileNames,
		"Recursive":    g.recursive,
		"G":            g,
	}

	fm := template.FuncMap{
		"PathName": func(s string) string {
			pf := g.pathFunc
			if pf == nil {
				pf = DefaultPathFunc
			}
			return pf(s)
		},
		"StructName": func(s string) string {
			return structName(s)
		},
	}

	t := template.New("route_map_vgen.go")
	t.Funcs(fm)
	t, err := t.Parse(`package {{.LocalPackage}}

// WARNING: This file was generated by vgrouter. Do not modify.

// routeMap is the generated route mappings for this package.
// The key is the path and the value is an instance of the component
// that should be used for it.
var vgRouteMap = map[string]interface{}{
	{{range $k, $v := .FileNameList}}
		"{{PathName $v}}": &{{StructName $v}}{},
	{{end}}
}

type vgroutes struct {
	prefix string
	recursive bool
}

func (r vgroutes) Recursive(v bool) vgroutes {
	r.recursive = v
	return r
}

func (r vgroutes) Prefix(v string) vgroutes {
	r.prefix = v
	return r
}

func (r vgroutes) Map() map[string]interface{} {
	ret := make(map[string]interface{}, len(vgRouteMap))
	for k, v := range vgRouteMap {
		ret[r.prefix+k] = v
	}

	{{if .Recursive}}
	if r.recursive {
		// TODO: implement recursively calling Map()
	}
	{{end}}

	return ret
}

// MakeRoutes returns the routes for this package and an sub-packages as applicable.
func MakeRoutes() vgroutes {
	return vgroutes{}
}
`)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, cm)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(g.dir, df.path, "route_map_vgen.go"), buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	// TODO: go fmt

	return nil
}

func structName(s string) string {

	// // trim file extension
	// s = strings.TrimSuffix(s, path.Ext(s))

	// // if any upper case letters we use the file name as-is
	// for _, c := range s {
	// 	if unicode.IsUpper(c) {
	// 		return s
	// 	}
	// }

	// otherwise we transform it the same way vugu does
	return fnameToGoTypeName(s)

}

func fnameToGoTypeName(s string) string {
	s = strings.Split(s, ".")[0] // remove file extension if present
	parts := strings.Split(s, "-")
	for i := range parts {
		p := parts[i]
		if len(p) > 0 {
			p = strings.ToUpper(p[:1]) + p[1:]
		}
		parts[i] = p
	}
	return strings.Join(parts, "")
}

/*

// routeMap is the generated route mappings for this package.
// The key is the path and the value is an instance of the component
// that should be used for it.
var routeMap = map[string]interface{}{
    "/": &Index{},
    "/:argv": &IndexArgv{},
    "/test1": &Test1{},
}

type routes struct {
    prefix string
    recursive bool
}

func (r routes) Recursive(v bool) routes {
    r.recursive = v
    return r
}

func (r routes) Prefix(v string) routes {
    r.prefix = v
    return r
}

func (r routes) Map() map[string]interface{} {
}

func MakeRoutes() routes {

}
*/

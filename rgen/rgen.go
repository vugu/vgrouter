package rgen

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
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

	// auto-detect g.packageName as needed
	if g.packageName == "" {
		g.packageName, err = guessImportPath(dir)
		// cmd := exec.Command("go", "list", "-json")
		// cmd.Dir = g.dir
		// b, err := cmd.CombinedOutput()
		// if err != nil {
		// 	return fmt.Errorf("error running `go list -json` to detect import path: %w; full output:\n%s", err, b)
		// }
		// var listData struct {
		// 	ImportPath string `json:"ImportPath"`
		// }
		// err = json.Unmarshal(b, &listData)
		// if err != nil {
		// 	return fmt.Errorf("error unmarshaling `go list -json` output: %w", err)
		// }
		// g.packageName = listData.ImportPath
	}

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

func (df *dirf) Path() string { return df.path }

func (g *Generator) writeRoutes(df *dirf) error {

	_, localPackage := path.Split(df.path)
	if localPackage == "" {
		_, localPackage = filepath.Split(g.dir)
	}

	cm := map[string]interface{}{
		"LocalPackage": localPackage,
		"PackageName":  g.packageName,
		"FileNameList": df.fileNames,
		"Subdirs":      df.subdirs,
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
		"HashIdent": func(s string) string {
			return fmt.Sprintf("ident%x", md5.Sum([]byte(s)))
		},
		"PathBase": path.Base,
	}

	t := template.New("0_routes_vgen.go")
	t.Funcs(fm)
	t, err := t.Parse(`package {{.LocalPackage}}

// WARNING: This file was generated by vgrouter/rgen. Do not modify.

import "path"

{{if .Recursive}}{{range $k, $subdir := .Subdirs}}import {{HashIdent (printf "%s%s" $.PackageName $subdir.Path)}} "{{$.PackageName}}/{{$subdir.Path}}"
{{end}}{{end}}

// routeMap is the generated route mappings for this package.
// The key is the path and the value is an instance of the component
// that should be used for it.
var vgRouteMap = map[string]interface{}{
{{range $k, $v := .FileNameList}}	"{{PathName $v}}": &{{StructName $v}}{},
{{end}}
}

type vgroutes struct {
	prefix string
	recursive bool
	clean bool
}

func (r vgroutes) WithRecursive(v bool) vgroutes {
	r.recursive = v
	return r
}

func (r vgroutes) WithPrefix(v string) vgroutes {
	r.prefix = v
	return r
}

func (r vgroutes) WithClean(v bool) vgroutes {
	r.clean = v
	return r
}

func (r vgroutes) Map() map[string]interface{} {
	ret := make(map[string]interface{}, len(vgRouteMap))
	for k, v := range vgRouteMap {
		key := r.prefix+k
		if r.clean {
			key = path.Clean(key)
		}
		ret[key] = v
	}

	{{if .Recursive}}
	if r.recursive {
		{{range $k, $subdir := .Subdirs}}
		for k, v := range {{HashIdent (printf "%s%s" $.PackageName $subdir.Path)}}.
				MakeRoutes().
				WithClean(r.clean).
				WithRecursive(true).
				WithPrefix(r.prefix+"/{{PathBase $subdir.Path}}").
				Map() {
			if r.clean {
				k = path.Clean(k)
			}
			ret[k] = v
		}
		{{end}}
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

	fullRouteMapPath := filepath.Join(g.dir, df.path, "0_routes_vgen.go")

	err = ioutil.WriteFile(fullRouteMapPath, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	b, err := exec.Command("go", "fmt", fullRouteMapPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running go fmt on %q: %w; full output: %s", fullRouteMapPath, err, b)
	}

	if g.recursive {
		// recurse into subdirs
		for _, subdf := range df.subdirs {
			err := g.writeRoutes(subdf)
			if err != nil {
				return fmt.Errorf("error in writeRoutes for %q: %w", subdf.path, err)
			}
		}
	}

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

func guessImportPath(dir string) (string, error) {

	after := ""
	lastDir := dir

	for {
		f, err := os.Open(filepath.Join(dir, "go.mod"))
		if err == nil {
			defer f.Close()
			ret, err := readModuleEntry(f)
			return ret + after, err
		}

		after = "/" + filepath.Base(dir) + after

		dir, err = filepath.Abs(filepath.Join(dir, ".."))
		if err != nil {
			return "", err
		}

		if dir == lastDir { // we hit the root dir
			return "", fmt.Errorf("no go.mod file found, cannot guess import path")
		}
	}

}

func readModuleEntry(r io.Reader) (string, error) {

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	ret := modulePath(b)
	if ret == "" {
		return "", errors.New("unable to determine module path from go.mod")
	}

	return ret, nil
}

// shamelessly stolen from: https://github.com/golang/vgo/blob/master/vendor/cmd/go/internal/modfile/read.go#L837
// ModulePath returns the module path from the gomod file text.
// If it cannot find a module path, it returns an empty string.
// It is tolerant of unrelated problems in the go.mod file.
func modulePath(mod []byte) string {
	for len(mod) > 0 {
		line := mod
		mod = nil
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, mod = line[:i], line[i+1:]
		}
		if i := bytes.Index(line, slashSlash); i >= 0 {
			line = line[:i]
		}
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, moduleStr) {
			continue
		}
		line = line[len(moduleStr):]
		n := len(line)
		line = bytes.TrimSpace(line)
		if len(line) == n || len(line) == 0 {
			continue
		}

		if line[0] == '"' || line[0] == '`' {
			p, err := strconv.Unquote(string(line))
			if err != nil {
				return "" // malformed quoted string or multiline module path
			}
			return p
		}

		return string(line)
	}
	return "" // missing module path
}

var (
	slashSlash = []byte("//")
	moduleStr  = []byte("module")
)

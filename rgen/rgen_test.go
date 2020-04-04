package rgen

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/vugu/vugu/gen"
)

func TestFull(t *testing.T) {

	tmpDir, err := ioutil.TempDir("", "rgen")
	if err != nil {
		t.Fatal(err)
	}
	// defer os.RemoveAll(tmpDir)
	log.Printf("tmpDir: %s", tmpDir)

	must(ioutil.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module rgentestfull
	
require github.com/vugu/vugu master
`), 0644))
	must(ioutil.WriteFile(filepath.Join(tmpDir, "index.vugu"), []byte("<div></div>"), 0644))
	must(ioutil.WriteFile(filepath.Join(tmpDir, "page1.vugu"), []byte("<div></div>"), 0644))
	must(os.MkdirAll(filepath.Join(tmpDir, "section1"), 0755))
	must(ioutil.WriteFile(filepath.Join(tmpDir, "section1", "index.vugu"), []byte("<div></div>"), 0644))
	must(ioutil.WriteFile(filepath.Join(tmpDir, "section1", "page-a.vugu"), []byte("<div></div>"), 0644))
	must(ioutil.WriteFile(filepath.Join(tmpDir, "section1", "page-b.vugu"), []byte("<div></div>"), 0644))

	err = New().SetDir(tmpDir).SetRecursive(true).Generate()
	if err != nil {
		t.Fatal(err)
	}

	// run vugugen stuff so the expected Go structs exist
	parser := gen.NewParserGoPkg(tmpDir, nil)
	err = parser.Run()
	if err != nil {
		t.Fatal(err)
	}
	parser = gen.NewParserGoPkg(filepath.Join(tmpDir, "section1"), nil)
	err = parser.Run()
	if err != nil {
		t.Fatal(err)
	}

	// write a go test file that we can use as our entry point
	must(ioutil.WriteFile(filepath.Join(tmpDir, "run_test.go"), []byte(`package `+filepath.Base(tmpDir)+`

import (
	"fmt"
	"sort"
	"testing"
)

func TestOutput(t *testing.T) {

	m := MakeRoutes().WithRecursive(true).WithClean(true).Map()
	plist := make([]string, 0, len(m))
	for p := range m {
		plist = append(plist, p)
	}
	sort.Strings(plist)
	for _, p := range plist {
		fmt.Printf("ROUTE: %s -> %T\n", p, m[p])
	}

}

`), 0644))

	// run it and get it's output (ensures both compilation and expected result)
	cmd := exec.Command("go", "test", "-v")
	cmd.Dir = tmpDir
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Error executing go test, OUTPUT:\n%s", b)
		t.Fatal(err)
	}

	// make sure it's what we expect
	t.Logf("OUTPUT:\n%s", b)

	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	routeLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "ROUTE:") {
			routeLines = append(routeLines, line)
		}
	}

	if !regexp.MustCompile(`ROUTE: / -> \*.*\.Index`).MatchString(routeLines[0]) {
		t.Fail()
	}
	if !regexp.MustCompile(`ROUTE: /page1 -> \*.*\.Page1`).MatchString(routeLines[1]) {
		t.Fail()
	}
	// if !regexp.MustCompile(`ROUTE: /section1/ -> \*section1\.Index`).MatchString(routeLines[2]) {
	if !regexp.MustCompile(`ROUTE: /section1 -> \*section1\.Index`).MatchString(routeLines[2]) {
		t.Fail()
	}
	if !regexp.MustCompile(`ROUTE: /section1/page-a -> \*section1\.PageA`).MatchString(routeLines[3]) {
		t.Fail()
	}
	if !regexp.MustCompile(`ROUTE: /section1/page-b -> \*section1\.PageB`).MatchString(routeLines[4]) {
		t.Fail()
	}

}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

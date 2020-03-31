package rgen

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestFull(t *testing.T) {

	tmpDir, err := ioutil.TempDir("", "rgen")
	if err != nil {
		t.Fatal(err)
	}
	// defer os.RemoveAll(tmpDir)
	log.Printf("tmpDir: %s", tmpDir)

	must(ioutil.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module rgentestfull\n"), 0644))
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

}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/vugu/vgrouter/rgen"
)

func main() {

	packageName := flag.String("p", "", "The full package name to use.  If unspecified auto-detection will be attempted using go.mod")
	recursive := flag.Bool("r", false, "Specify to recursively process subdirectories")
	q := flag.Bool("q", false, "Only print information upon error (quiet mode)")

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."} // default to current dir
	}

	if *packageName != "" && len(args) > 1 {
		log.Fatalf("-p is only valid with a single directory, either don't use -p or only specify one dir")
	}

	for _, arg := range args {

		dir, err := filepath.Abs(arg)
		if err != nil {
			log.Fatalf("Error converting %q to absolute path: %v", arg, err)
		}

		if !*q {
			log.Printf("Processing routes for dir: %s", arg)
		}

		err = rgen.New().
			SetDir(dir).
			SetPackageName(*packageName).
			SetRecursive(*recursive).
			Generate()
		if err != nil {
			log.Fatal(err)
		}

	}

}

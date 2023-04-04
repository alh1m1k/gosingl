package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"

	"io"

	"github.com/jawher/mow.cli"
)

func main() {

	app := cli.App("gosingl", "generate reset method")
	chosenPackage := app.StringArg("PKG", "", "package to walk to")
	chosenStruct := app.StringArg("STRUCTURE", "", "structure will be use as module singleton")
	chosenInstance := app.StringOpt("instance", "Instance", "singleton instance (module variable)")

	// writeType
	write := app.BoolOpt("w write", false, "writes the result in file")

	exitOnError := func(err error) {
		log.Println(err)
		os.Exit(1)
	}
	var err error
	app.Action = func() {
		err = parsePackage(chosenPackage, chosenStruct, chosenInstance, write, nil)
		if err != nil {
			exitOnError(err)
		}
	}
	err = app.Run(os.Args)
	if err != nil {
		exitOnError(err)
	}

}

// parsePackage launchs the generation
func parsePackage(pkg *string, structure *string, variable *string, write *bool, customWriter io.Writer) error {

	if pkg == nil {
		return errors.New("no directory submitted")
	}

	if strings.TrimSpace(*pkg) == "" {
		return errors.New("directory empty submitted")
	}

	if strings.TrimSpace(*variable) == "" {
		return errors.New("instance empty submitted")
	}

	var writeToFile bool
	if write != nil && *write {
		writeToFile = true
	}

	// get the path of the package
	if strings.TrimSpace(os.Getenv("GOPATH")) == "" {
		log.Println("WARNING: OS ENV GOPATH NOT SET!")
	}
	if strings.TrimSpace(os.Getenv("GOROOT")) == "" {
		log.Println("WARNING: OS ENV GOROOT NOT SET!")
	}

	pkgdir := os.Getenv("GOPATH") + "/src/" + *pkg
	// reinstall package to be sure that we are uptodate
	/*	log.Println("install", *pkg)
		c := exec.Command(runtime.GOROOT()+"/bin/go", []string{"install", *pkg}...)
		c.Stderr = os.Stderr
		err := c.Run()
		if err != nil {
			return err
		}*/

	fset := token.NewFileSet()
	var f map[string]*ast.Package
	log.Println("parse directory", pkgdir)
	f, err := parser.ParseDir(fset, pkgdir, nil, 0)

	if err != nil {
		return err
	}

	for i := range f {
		var files []*ast.File
		for j := range f[i].Files {
			files = append(files, f[i].Files[j])

		}

		for j := range f[i].Files {
			log.Println("file", j)
			if !strings.Contains(j, "_singleton.go") {
				err = generate(fset, f[i].Files[j], files, pkgdir, i, j, *structure, *variable, writeToFile, customWriter)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

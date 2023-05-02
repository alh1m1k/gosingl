package main

import (
	"context"
	cli "github.com/jawher/mow.cli"
	"os"
	"time"
)

func main() {

	cfg := Config{}

	delay := 0

	app := cli.App("gosingl", "generate module level singleton")
	app.StringArgPtr(&cfg.Package, "PKG", "", "package to walk to")
	app.StringArgPtr(&cfg.Target, "TARGET", "", "Structure will be use as module singleton")
	app.StringOptPtr(&cfg.Variable, "variable", "Instance", "singleton instance (module variable).\n *Instance declare var as real,\n "+
		"&Instance declare var as ref, Instance[T,K,Z] resolves generic")
	app.StringOptPtr(&cfg.Comment, "comment", "Code generated by <git repo>. DO NOT EDIT.", "file header comment")
	app.StringOptPtr(&cfg.Suffix, "suffix", "_singleton.go", "suffix of generated file")
	app.StringOptPtr(&cfg.Path, "filepath", "", "path override")
	app.IntOptPtr(&cfg.Deep, "deep", 0, "recursive deep")
	app.IntOptPtr(&delay, "delay", 0, "debug only")
	app.BoolOptPtr(&cfg.Write, "w write", false, "writes the result in file")

	exitOnError := func(err error) {
		critical(err)
		os.Exit(1)
	}

	var err error
	app.Action = func() {
		if delay > 0 {
			time.Sleep(time.Second * time.Duration(delay))
		}
		ctx := context.Background()
		ctx = SetupCtx(ctx, //as reference
			nil, //global output target //"write" and "file" flags will be ignored if set
			nil, //shared buffer, mostly internal
			nil, //helper to generate name for unnamed function input parameters
			nil, //checker which excludes duplicates from output
			nil, //blacklisted file suffix ([]string{"_test.go", cfg.Suffix},)
		)
		err = ParsePackage(ctx, cfg)
		if err != nil {
			exitOnError(err)
		}
	}

	err = app.Run(os.Args)
	if err != nil {
		exitOnError(err)
	}

}

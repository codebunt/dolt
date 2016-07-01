// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/samples/go/csv"
)

var (
	// Actually the delimiter uses runes, which can be multiple characters long.
	// https://blog.golang.org/strings
	delimiter = flag.String("delimiter", ",", "field delimiter for csv file, must be exactly one character long.")
)

func main() {
	spec.RegisterDatabaseFlags()

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: csv-export [options] dataset > filename")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		d.CheckError(errors.New("expected dataset arg"))
	}

	ds, err := spec.GetDataset(flag.Arg(0))
	d.CheckError(err)

	defer ds.Database().Close()

	comma, err := csv.StringToRune(*delimiter)
	d.CheckError(err)

	err = d.Try(func() {
		defer profile.MaybeStartProfile().Stop()

		nomsList, structDesc := csv.ValueToListAndElemDesc(ds.HeadValue(), ds.Database())
		csv.Write(nomsList, structDesc, comma, os.Stdout)
	})
	if err != nil {
		fmt.Println("Failed to export dataset as CSV:")
		fmt.Println(err)
	}
}

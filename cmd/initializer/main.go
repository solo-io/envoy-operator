package main

import (
	"flag"

	"github.com/solo-io/envoy-operator/pkg/downward"
)

func main() {

	inputfile := flag.String("input", "", "input file")
	outfile := flag.String("output", "", "output file")
	flag.Parse()
	err := downward.TransformFiles(*inputfile, *outfile)

	if err != nil {
		panic(err)
	}
}

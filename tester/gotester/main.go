package main

import (
	"encoding/xml"
	"flag"
	"log"
	"os"

	"github.com/shabbyrobe/xmlwriter"
	"github.com/shabbyrobe/xmlwriter/tester/xwrunner"
)

func main() {
	var err error
	var script xwrunner.Script
	var indent bool
	var options []xmlwriter.Option

	flag.BoolVar(&indent, "indent", false, "Use default indenter")
	flag.Parse()

	if indent {
		options = append(options, xmlwriter.WithIndent())
	}

	reader := os.Stdin
	err = xml.NewDecoder(reader).Decode(&script)
	if err != nil {
		log.Fatal(err)
	}

	s := &script
	if err = s.Run(os.Stdout, options...); err != nil {
		log.Fatal(err)
	}
}

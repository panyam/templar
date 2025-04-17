package main

import (
	"flag"
	"strings"

	tu "github.com/panyam/templar/utils"
)

type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ", ")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

var (
	addr = flag.String("addr", ":7777", "Address where the http server will be running")
)

func main() {
	var templateDirs multiStringFlag
	var staticDirs multiStringFlag
	flag.Var(&templateDirs, "t", "List of template directories to load templates from")
	flag.Var(&staticDirs, "s", "List of static directores and http static prefixes in the form <http prefix>:<local folder>")
	flag.Parse()

	b := tu.BasicServer{TemplateDirs: templateDirs, StaticDirs: staticDirs}
	b.Serve(nil, *addr)
}

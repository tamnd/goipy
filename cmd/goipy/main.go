package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tamnd/goipy/marshal"
	"github.com/tamnd/goipy/object"
	"github.com/tamnd/goipy/vm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: goipy <file.pyc>")
		os.Exit(2)
	}
	code, err := marshal.LoadPyc(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "load error:", err)
		os.Exit(1)
	}
	i := vm.New()
	if dir, derr := filepath.Abs(filepath.Dir(os.Args[1])); derr == nil {
		i.SearchPath = []string{dir}
	}
	i.Argv = append([]string{os.Args[1]}, os.Args[2:]...)
	if _, err := i.Run(code); err != nil {
		if code, ok := i.SystemExitCode(err); ok {
			os.Exit(code)
		}
		if e, ok := err.(*object.Exception); ok {
			fmt.Fprint(os.Stderr, vm.FormatException(e))
		} else {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}
}

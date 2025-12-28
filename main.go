package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args, NewConversionService()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

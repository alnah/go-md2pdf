package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args, &CommonMarkToPandocPreprocessor{}, NewPandocConverter(), &CSSInjection{}, NewChromeConverter()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

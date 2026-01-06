package main

import (
	"fmt"
	"os"

	md2pdf "github.com/alnah/go-md2pdf"
)

func main() {
	svc := md2pdf.New()
	defer svc.Close()

	if err := run(os.Args, svc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

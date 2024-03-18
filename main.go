package main

import (
	"log"
	"os"

	"github.com/speedata/fixxref/scanner"
)

func writePDFFile(filename string, contents string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	if err = f.Truncate(0); err != nil {
		return err
	}
	if _, err = f.WriteString(contents); err != nil {
		return err
	}
	return f.Close()
}

func fixXRefForFile(fn string) error {
	pdffile, err := os.Open(fn)
	if err != nil {
		return err
	}

	out, err := scanner.Scan(pdffile)
	if err != nil {
		return err
	}
	pdffile.Close()
	if err = writePDFFile(fn, out); err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("fixxref: expect file name of PDF file")
	}
	if err := fixXRefForFile(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}

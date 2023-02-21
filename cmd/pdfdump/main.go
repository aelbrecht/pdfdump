package main

import (
	"fmt"
	"log"
	"os"
	"pdfdump/internal/pdf"
	"pdfdump/internal/token"
)

func main() {

	if len(os.Args) != 2 {
		log.Fatalln("expected only one argument")
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
	}

	parser := pdf.NewParser(token.NewScanner(f))
	parser.Parse()
	_ = f.Close()

	fmt.Print(parser.PDF().String())
}

package main

import (
	"fmt"
	"github.com/aelbrecht/pdfdump/external/pdf"
	"github.com/aelbrecht/pdfdump/internal/token"
	"log"
	"os"
)

func main() {

	if len(os.Args) != 2 {
		log.Println("error: expected one argument")
		os.Exit(1)
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

package main

import (
	"fmt"
	"github.com/aelbrecht/pdfdump/external/pdf"
	"github.com/aelbrecht/pdfdump/internal/token"
	"log"
	"os"
	"path"
	"strings"
)

func parsePDF(filePath string) {

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
	}

	scanner := token.NewScanner(f)
	parser := pdf.NewParser(scanner)
	parser.Parse()

	_ = f.Close()

	dirName, fileName := path.Split(filePath)
	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	o, err := os.Create(path.Join(dirName, fileName+".txt"))
	if err != nil {
		log.Fatalln(err)
	}
	parser.Dump(o)
	_ = o.Close()
}

func main() {
	for _, arg := range os.Args[1:] {
		parsePDF(arg)
	}
}

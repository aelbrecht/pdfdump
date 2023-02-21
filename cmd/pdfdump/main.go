package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"pdfdump/internal/pdf"
	"pdfdump/internal/scan"
	"strconv"
	"strings"
)

func NewPDFParser(scanner *scan.TokenScanner) *Parser {
	return &Parser{scanner: scanner}
}

type Parser struct {
	scanner *scan.TokenScanner
}

func (p *Parser) ParseDict() (pdf.ObjectType, bool) {
	if !p.scanner.Pop("<<") {
		return nil, false
	}
	dict := make([]pdf.KeyValuePair, 0)
	for true {
		if p.scanner.Pop(">>") {
			return pdf.NewDictionary(dict), true
		} else {
			dict = append(dict, pdf.KeyValuePair{
				Key:   p.ParseNext(),
				Value: p.ParseNext(),
			})
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseStream() (pdf.ObjectType, bool) {
	if !strings.HasPrefix(p.scanner.Peek(), "stream") {
		return nil, false
	}
	buffer := make([]byte, 0)
	for true {
		token := p.scanner.Next()
		buffer = append(buffer, []byte(token)...)
		if token == "endstream" {
			return pdf.NewStream(buffer), true
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseNumber() (pdf.ObjectType, bool) {
	token := p.scanner.Peek()
	if strings.Contains(token, ".") {
		v, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return nil, false
		}
		p.scanner.Next() // consume token
		return pdf.NewFloatingNumber(v), true
	} else {
		v, err := strconv.ParseInt(token, 10, 64)
		if err != nil {
			return nil, false
		}
		p.scanner.Next() // consume token
		return pdf.NewIntegerNumber(v), true
	}
}

func (p *Parser) ParseString() (pdf.ObjectType, bool) {
	if p.scanner.Peek()[0] != '(' {
		return nil, false
	}
	buffer := ""
	for true {
		token := p.scanner.Next()
		buffer += token
		if token == ")" || token[len(token)-1] == ')' {
			return pdf.NewString(buffer), true
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseArray() (pdf.ObjectType, bool) {
	if !p.scanner.Pop("[") {
		return nil, false
	}
	arr := make([]pdf.ObjectType, 0)
	for true {
		if p.scanner.Pop("]") {
			return pdf.NewArray(arr), true
		} else {
			arr = append(arr, p.ParseNext())
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseBoolean() (pdf.ObjectType, bool) {
	if p.scanner.Pop("true") {
		return pdf.NewBoolean(true), true
	} else if p.scanner.Pop("false") {
		return pdf.NewBoolean(false), true
	} else {
		return nil, false
	}
}

func (p *Parser) ParseNull() (pdf.ObjectType, bool) {
	if p.scanner.Pop("null") {
		return pdf.NewNull(), true
	} else {
		return nil, false
	}
}

func (p *Parser) ParseReference() (pdf.ObjectType, bool) {
	if p.scanner.PeekAhead(2) != "R" {
		return nil, false
	}
	objNum, err := strconv.Atoi(p.scanner.Next())
	check(err)
	genNum, err := strconv.Atoi(p.scanner.Next())
	check(err)
	if !p.scanner.Pop("R") {
		log.Fatalln("failed to parse indirect ref")
	}
	return pdf.NewReference(pdf.ObjectIdentifier{
		ObjectNumber:     objNum,
		ObjectGeneration: genNum,
	}), true
}

func (p *Parser) ParseLabel() (pdf.ObjectType, bool) {
	if p.scanner.Peek()[0] != '/' {
		return nil, false
	}
	token := p.scanner.Next()
	if token[0] != '/' {
		log.Fatalln("invalid label start")
	}
	return pdf.NewLabel(token), true
}

func (p *Parser) ParseNext() pdf.ObjectType {

	if v, ok := p.ParseDict(); ok {
		return v
	}

	if v, ok := p.ParseArray(); ok {
		return v
	}

	if v, ok := p.ParseStream(); ok {
		return v
	}

	if v, ok := p.ParseBoolean(); ok {
		return v
	}

	if v, ok := p.ParseNull(); ok {
		return v
	}

	if v, ok := p.ParseReference(); ok {
		return v
	}

	if v, ok := p.ParseLabel(); ok {
		return v
	}

	if v, ok := p.ParseString(); ok {
		return v
	}

	if v, ok := p.ParseNumber(); ok {
		return v
	}

	token := p.scanner.Next()
	log.Printf("failed to parse: %s\n", token)
	p.scanner.Dump()
	os.Exit(1)
	return nil
}

func (p *Parser) ParseObject() (*pdf.Object, bool) {
	if p.scanner.PeekAhead(2) != "obj" {
		return nil, false
	}
	objNum, err := strconv.Atoi(p.scanner.Next())
	check(err)
	objGen, err := strconv.Atoi(p.scanner.Next())
	check(err)
	p.scanner.Next() // obj
	id := pdf.ObjectIdentifier{
		ObjectNumber:     objNum,
		ObjectGeneration: objGen,
	}
	children := make([]pdf.ObjectType, 0)
	for true {
		if p.scanner.Pop("endobj") {
			return pdf.NewObject(id, children), true
		}
		children = append(children, p.ParseNext())
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseTrailer() bool {
	if !p.scanner.Pop("xref") {
		return false
	}
	entries := 0
	for true {
		if p.scanner.Pop("%%EOF") {
			return true
		}
		entries++
		p.scanner.Next()
	}
	log.Fatalln("unexpected EOF")
	return false
}

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func parsePDF(filePath string) {

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
	}

	scanner := scan.NewTokenScanner(f)
	parser := NewPDFParser(scanner)
	objects := make([]*pdf.Object, 0)
	version := ""
	for scanner.HasToken() {

		if version == "" {
			version = scanner.Next()
			if len(version) == 0 || version[0] != '%' {
				log.Fatalln("invalid header")
			}
			version = version[1:]
			scanner.Next()
			continue
		}

		if scanner.Peek() == "" {
			scanner.Next()
			continue
		}

		if v, ok := parser.ParseObject(); ok {
			fmt.Println(v.String())
			objects = append(objects, v)
			continue
		}

		if parser.ParseTrailer() {
			continue
		}

		fmt.Println("unknown prefix")
		scanner.Dump()
		os.Exit(1)
	}

	err = f.Close()
	if err != nil {
		fmt.Println("failed to close file")
	}

	result := pdf.PDF{
		Version:  version,
		Children: objects,
	}

	dirName, fileName := path.Split(filePath)
	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	o, err := os.Create(path.Join(dirName, fileName+".txt"))
	if err != nil {
		log.Fatalln(err)
	}

	for _, child := range result.Children {
		o.WriteString(child.String())
	}

	_ = o.Close()
	//err = json.NewEncoder(o).Encode(result)
	//if err != nil {
	//	log.Println("could not encode json")
	//	log.Fatalln(err)
	//}

}

func main() {
	for _, arg := range os.Args[1:] {
		parsePDF(arg)
	}
}

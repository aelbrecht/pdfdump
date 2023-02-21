package pdf

import (
	"fmt"
	"log"
	"os"
	"pdfdump/internal/token"
	"strconv"
	"strings"
)

func NewParser(scanner *token.Scanner) *Parser {
	return &Parser{
		scanner:    scanner,
		objects:    make(map[string]*Object),
		version:    "",
		references: make([]*ObjectReference, 0),
	}
}

func (p *Parser) Parse() {
	for p.scanner.HasToken() {

		if p.ParseHeader() {
			continue
		}

		if p.scanner.Peek() == "" {
			p.scanner.Next()
			continue
		}

		if v, ok := p.ParseObject(); ok {
			p.objects[v.Identifier.Hash()] = v
			continue
		}

		if p.ParseTrailer() {
			continue
		}

		fmt.Println("unknown prefix")
		p.scanner.Dump()
		os.Exit(1)
	}

	for _, ref := range p.references {
		o, ok := p.objects[ref.Link.Hash()]
		if !ok {
			log.Fatalln("unresolved reference")
		}
		ref.Value = o
		o.References = append(o.References, ref)
	}
}

type Parser struct {
	scanner    *token.Scanner
	objects    map[string]*Object
	version    string
	references []*ObjectReference
}

func (p *Parser) Objects() map[string]*Object {
	return p.objects
}

func (p *Parser) Dump(f *os.File) {
	for _, child := range p.objects {
		_, _ = f.WriteString(child.String())
	}
}

func (p *Parser) ParseHeader() bool {
	if p.version != "" {
		return false
	}
	p.version = p.scanner.Next()
	if len(p.version) == 0 || p.version[0] != '%' {
		log.Fatalln("invalid header")
	}
	p.version = p.version[1:]
	p.scanner.Next()
	return true
}

func (p *Parser) ParseDict() (ObjectType, bool) {
	if !p.scanner.Pop("<<") {
		return nil, false
	}
	dict := make([]KeyValuePair, 0)
	for true {
		if p.scanner.Pop(">>") {
			return NewDictionary(dict), true
		} else {
			dict = append(dict, KeyValuePair{
				Key:   p.ParseNext(),
				Value: p.ParseNext(),
			})
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseStream() (ObjectType, bool) {
	if !strings.HasPrefix(p.scanner.Peek(), "stream") {
		return nil, false
	}
	buffer := make([]byte, 0)
	for true {
		t := p.scanner.Next()
		buffer = append(buffer, []byte(t)...)
		if t == "endstream" {
			return NewStream(buffer), true
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseNumber() (ObjectType, bool) {
	t := p.scanner.Peek()
	if strings.Contains(t, ".") {
		v, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return nil, false
		}
		p.scanner.Next() // consume token
		return NewFloatingNumber(v), true
	} else {
		v, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return nil, false
		}
		p.scanner.Next() // consume token
		return NewIntegerNumber(v), true
	}
}

func (p *Parser) ParseString() (ObjectType, bool) {
	if p.scanner.Peek()[0] != '(' {
		return nil, false
	}
	buffer := ""
	for true {
		t := p.scanner.Next()
		buffer += t
		if t == ")" || t[len(t)-1] == ')' {
			return NewString(buffer), true
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseArray() (ObjectType, bool) {
	if !p.scanner.Pop("[") {
		return nil, false
	}
	arr := make([]ObjectType, 0)
	for true {
		if p.scanner.Pop("]") {
			return NewArray(arr), true
		} else {
			arr = append(arr, p.ParseNext())
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseBoolean() (ObjectType, bool) {
	if p.scanner.Pop("true") {
		return NewBoolean(true), true
	} else if p.scanner.Pop("false") {
		return NewBoolean(false), true
	} else {
		return nil, false
	}
}

func (p *Parser) ParseNull() (ObjectType, bool) {
	if p.scanner.Pop("null") {
		return NewNull(), true
	} else {
		return nil, false
	}
}

func (p *Parser) ParseReference() (ObjectType, bool) {
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
	ref := NewReference(ObjectIdentifier{
		ObjectNumber:     objNum,
		ObjectGeneration: genNum,
	})
	p.references = append(p.references, ref)
	return ref, true
}

func (p *Parser) ParseLabel() (ObjectType, bool) {
	if p.scanner.Peek()[0] != '/' {
		return nil, false
	}
	t := p.scanner.Next()
	if t[0] != '/' {
		log.Fatalln("invalid label start")
	}
	return NewLabel(t), true
}

func (p *Parser) ParseNext() ObjectType {

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

	t := p.scanner.Next()
	log.Printf("failed to parse: %s\n", t)
	p.scanner.Dump()
	os.Exit(1)
	return nil
}

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func (p *Parser) ParseObject() (*Object, bool) {
	if p.scanner.PeekAhead(2) != "obj" {
		return nil, false
	}
	objNum, err := strconv.Atoi(p.scanner.Next())
	check(err)
	objGen, err := strconv.Atoi(p.scanner.Next())
	check(err)
	p.scanner.Next() // obj
	id := ObjectIdentifier{
		ObjectNumber:     objNum,
		ObjectGeneration: objGen,
	}
	children := make([]ObjectType, 0)
	for true {
		if p.scanner.Pop("endobj") {
			return NewObject(id, children), true
		}
		child := p.ParseNext()
		children = append(children, child)
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

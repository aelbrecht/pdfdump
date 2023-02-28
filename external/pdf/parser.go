package pdf

import (
	"fmt"
	"github.com/aelbrecht/pdfdump/internal/token"
	"log"
	"os"
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

	visited := make(map[string]bool)
	uniques := make(map[string]*Object)
	redirected := make(map[string]*Object)
	dupes := 0
	opts := MatchOptions{MatchReferences: true, MatchStream: true}
	for k1, o1 := range p.objects {
		if visited[k1] {
			continue
		}
		visited[k1] = true
		uniques[k1] = o1
		for k2, o2 := range p.objects {
			if k1 == k2 {
				continue
			}
			if visited[k2] {
				continue
			}
			score := MatchTypes(o1, o2, &opts)
			if score == 1 {
				visited[k2] = true
				redirected[k2] = o1
				dupes++
				break
			}
		}
	}
	p.objects = uniques

	for _, ref := range p.references {

		if o, ok := p.objects[ref.Link.Hash()]; !ok {
			redirect, ok := redirected[ref.Link.Hash()]
			if !ok {
				log.Fatalln("unresolved reference")
			}
			ref.Link = redirect.Identifier
			ref.Value = redirect
			redirect.References = append(redirect.References, ref)
		} else {
			ref.Value = o
			o.References = append(o.References, ref)
		}
	}

	for _, o := range p.objects {
		if len(o.References) == 0 {
			assignMinimalDepth(o, 0)
		}
	}
}

func assignMinimalDepth(root ObjectType, depth int) {
	switch root.(type) {
	case *Object:
		for _, child := range root.(*Object).Children {
			assignMinimalDepth(child, depth+1)
		}
	case *Dictionary:
		for _, child := range root.(*Dictionary).Value {
			assignMinimalDepth(child.V, depth+1)
		}
	case *Array:
		for _, child := range root.(*Array).Value {
			assignMinimalDepth(child, depth+1)
		}
	case *ObjectReference:
		object := root.(*ObjectReference).Value
		if object.Depth == 0 || depth <= object.Depth {
			object.Depth = depth
			assignMinimalDepth(object, depth+1)
		}
	default:
		// pass
	}

}

type Parser struct {
	scanner    *token.Scanner
	objects    map[string]*Object
	version    string
	references []*ObjectReference
}

func (p *Parser) PDF() *PDF {
	return &PDF{
		Version: p.version,
		Objects: p.objects,
	}
}

func (p *Parser) Dump(f *os.File) {
	for _, child := range p.objects {
		_, _ = f.WriteString(child.String())
	}
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
				K: p.ParseNext(),
				V: p.ParseNext(),
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
	v, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return nil, false
	}
	p.scanner.Next() // consume token
	return NewNumber(v), true
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
			return NewText(buffer), true
		}
	}
	log.Fatalln("unreachable statement")
	return nil, false
}

func (p *Parser) ParseHexString() (ObjectType, bool) {
	if p.scanner.Peek()[0] != '<' {
		return nil, false
	}
	buffer := ""
	for true {
		t := p.scanner.Next()
		buffer += t
		if t == ">" || t[len(t)-1] == '>' {
			return NewText(buffer), true
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

	if v, ok := p.ParseHexString(); ok {
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

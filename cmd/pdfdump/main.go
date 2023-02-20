package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"pdfdump/internal/pdf"
	"strconv"
	"strings"
)

func ScanCarriage(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

const historySize = 6

type TokenScanner struct {
	scanner      *bufio.Scanner
	tokens       []string
	index        int
	history      [historySize]string
	historyIndex int
	end          bool
}

func NewTokenScanner(r io.Reader) *TokenScanner {
	scanner := bufio.NewScanner(r)
	scanner.Split(ScanCarriage)
	t := &TokenScanner{
		scanner: scanner,
	}
	if !t.scan() {
		log.Fatalln("EOF")
	}
	return t
}

func (t *TokenScanner) HasToken() bool {
	return !t.end
}

func (t *TokenScanner) Dump() {
	i := t.historyIndex
	for true {
		offset := 0
		if t.index == 0 {
			offset = 1
		}
		if i != (t.historyIndex+historySize-1-offset)%historySize {
			fmt.Print("#####")
		} else {
			fmt.Print("---->")
		}
		fmt.Println("   " + t.history[i])
		i = (i + 1) % historySize
		if t.historyIndex == i {
			break
		}
	}
	for j := 0; j < 2; j++ {
		if !t.scanner.Scan() {
			return
		}
		fmt.Print("#####")
		fmt.Println("   " + t.scanner.Text())
	}
}

func (t *TokenScanner) Next() string {
	next := t.tokens[t.index]
	t.index++
	if t.index >= len(t.tokens) {
		if !t.scan() {
			if t.end {
				log.Fatalln("unexpected EOF")
			}
			t.end = true
		}
	}
	return next
}

func (t *TokenScanner) HasSuffix(suffix string) bool {
	return t.tokens[len(t.tokens)-1] == suffix
}

func (t *TokenScanner) HasPrefix(prefix string) bool {
	return t.tokens[0] == prefix
}

func (t *TokenScanner) Pop(token string) bool {
	if t.Peek() == token {
		t.Next()
		return true
	}
	return false
}

func (t *TokenScanner) Peek() string {
	return t.tokens[t.index]
}

func (t *TokenScanner) PeekAhead(offset int) string {
	if t.index+offset >= len(t.tokens) {
		return ""
	} else {
		return t.tokens[t.index+offset]
	}
}

func (t *TokenScanner) scan() bool {
	if !t.scanner.Scan() {
		return false
	}
	line := t.scanner.Text()
	t.history[t.historyIndex] = line
	t.historyIndex = (t.historyIndex + 1) % historySize
	ts := strings.Split(line, " ")
	t.index = 0
	t.tokens = make([]string, len(ts))
	for i, s := range ts {
		t.tokens[i] = strings.TrimSpace(s)
	}
	return true
}

func parseDict(scanner *TokenScanner) bool {
	if !scanner.Pop("<<") {
		return false
	}
	fmt.Println("[dict]")
	for true {
		if scanner.Pop(">>") {
			fmt.Println("[end-dict]")
			return true
		} else {
			parseAny(scanner)
		}
	}
	return false
}

func parseStream(scanner *TokenScanner) bool {
	if !strings.HasPrefix(scanner.Peek(), "stream") {
		return false
	}
	buffer := ""
	for true {
		token := scanner.Next()
		buffer += token
		if token == "endstream" {
			fmt.Printf("[stream] %d characters\n", len(buffer))
			return true
		}
	}
	return false
}

func parseNumber(scanner *TokenScanner) bool {
	token := scanner.Peek()
	if strings.Contains(token, ".") {
		v, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return false
		}
		fmt.Printf("[float] %f\n", v)
	} else {
		v, err := strconv.ParseInt(token, 10, 64)
		if err != nil {
			return false
		}
		fmt.Printf("[int] %d\n", v)
	}
	scanner.Next()
	return true
}

func parseString(scanner *TokenScanner) bool {
	if scanner.Peek()[0] != '(' {
		return false
	}
	buffer := ""
	for true {
		token := scanner.Next()
		buffer += token
		if token == ")" || token[len(token)-1] == ')' {
			fmt.Printf("[string] %s\n", buffer)
			return true
		}
	}
	return false
}

func exitUnknown(s string) {
	fmt.Printf("[unknown] %s\n", s)
	os.Exit(0)
}

func parseArray(scanner *TokenScanner) bool {
	if !scanner.Pop("[") {
		return false
	}
	fmt.Println("[array]")
	for true {
		if scanner.Pop("]") {
			fmt.Println("[end-array]")
			return true
		} else {
			parseAny(scanner)
		}
	}
	return false
}

func parseBoolean(scanner *TokenScanner) bool {
	if scanner.Pop("true") {
		fmt.Println("[boolean] true")
		return true
	} else if scanner.Pop("false") {
		fmt.Println("[boolean] false")
		return true
	} else {
		return false
	}
}

func parseReference(scanner *TokenScanner) bool {
	if scanner.PeekAhead(2) != "R" {
		return false
	}
	objNum, err := strconv.Atoi(scanner.Next())
	check(err)
	genNum, err := strconv.Atoi(scanner.Next())
	check(err)
	if !scanner.Pop("R") {
		log.Fatalln("failed to parse indirect ref")
	}
	fmt.Printf("[reference] %d %d\n", objNum, genNum)
	return true
}

func parseType(scanner *TokenScanner) bool {
	if scanner.Peek()[0] != '/' {
		return false
	}
	token := scanner.Next()
	if token[0] != '/' {
		log.Fatalln("invalid type start")
	}
	fmt.Printf("[type] %s\n", token)
	return true
}

func parseAny(scanner *TokenScanner) {

	if parseDict(scanner) {
		return
	}

	if parseArray(scanner) {
		return
	}

	if parseStream(scanner) {
		return
	}

	if parseBoolean(scanner) {
		return
	}

	if parseReference(scanner) {
		return
	}

	if parseType(scanner) {
		return
	}

	if parseString(scanner) {
		return
	}

	if parseNumber(scanner) {
		return
	}

	token := scanner.Next()
	log.Printf("failed to parse: %s\n", token)
	scanner.Dump()
	os.Exit(1)
}

func parseObject(scanner *TokenScanner, o *pdf.Object) {
	fmt.Printf("[object %d %d]\n", o.Identifier.ObjectNumber, o.Identifier.ObjectGeneration)
	for true {
		if scanner.Pop("endobj") {
			fmt.Println("[end-obj]")
			return
		}
		parseAny(scanner)
	}
	log.Fatalln("unexpected EOF")
}

func parseTrailer(scanner *TokenScanner) {
	entries := 0
	for true {
		if scanner.Pop("%%EOF") {
			fmt.Printf("[trailer] %d items\n", entries)
			return
		}
		entries++
		scanner.Next()
	}
	log.Fatalln("unexpected EOF")
}

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {

	f, err := os.Open("./input.pdf")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	scanner := NewTokenScanner(f)
	objects := make([]*pdf.Object, 0)
	for scanner.HasToken() {
		if scanner.HasSuffix("obj") {
			o := new(pdf.Object)
			objects = append(objects, o)
			objNum, err := strconv.Atoi(scanner.Next())
			check(err)
			objGen, err := strconv.Atoi(scanner.Next())
			check(err)
			scanner.Next() // obj
			o.Identifier = pdf.ObjectIdentifier{
				ObjectNumber:     objNum,
				ObjectGeneration: objGen,
			}
			parseObject(scanner, o)
		} else if scanner.Pop("xref") {
			parseTrailer(scanner)
		} else {
			token := scanner.Next()
			if token != "" {
				fmt.Printf("[unknown prefix] %s\n", token)
			}
		}
	}

	fmt.Println("parsing successful!")
}

package scan

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
)

const historySize = 6

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

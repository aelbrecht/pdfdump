package pdfdiff

import (
	"bytes"
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"strings"
)

type comparer struct {
	buffer               bytes.Buffer
	currentLine          string
	currentRemoveLine    string
	currentAddLine       string
	currentLineHasRemove bool
	currentLineHasAdd    bool
}

func (f *comparer) emit() {
	if !f.currentLineHasAdd && !f.currentLineHasRemove {
		f.buffer.WriteString(fmt.Sprintf("= %s\n", f.currentLine))
	}
	if f.currentLineHasAdd {
		xs := strings.Split(f.currentAddLine, "\n")
		for _, x := range xs {
			f.buffer.WriteString(fmt.Sprintf("+ %s\n", x))
		}
	}
	if f.currentLineHasRemove {
		xs := strings.Split(f.currentRemoveLine, "\n")
		for _, x := range xs {
			f.buffer.WriteString(fmt.Sprintf("- %s\n", x))
		}
	}
}

func (c *Comparison) String() string {
	cmp := comparer{}
	return cmp.diff(c.LeftOutput, c.RightOutput)
}

func (f *comparer) diff(left string, right string) string {
	differ := diffmatchpatch.New()
	output := differ.DiffMain(left, right, false)
	for _, d := range output {
		f.parse(&d)
	}
	f.emit()
	return f.buffer.String()
}

func (f *comparer) parse(d *diffmatchpatch.Diff) {
	switch d.Type {
	case diffmatchpatch.DiffEqual:
		for _, c := range d.Text {
			if c == '\n' {
				f.emit()
				f.currentLineHasRemove = false
				f.currentLineHasAdd = false
				f.currentRemoveLine = ""
				f.currentAddLine = ""
				f.currentLine = ""
			} else {
				f.currentRemoveLine += string(c)
				f.currentAddLine += string(c)
				f.currentLine += string(c)
			}
		}
	case diffmatchpatch.DiffDelete:
		f.currentLineHasRemove = true
		f.currentRemoveLine += d.Text
	case diffmatchpatch.DiffInsert:
		f.currentLineHasAdd = true
		f.currentAddLine += d.Text
	}
}

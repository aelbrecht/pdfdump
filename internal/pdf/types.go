package pdf

import (
	"fmt"
	"strconv"
	"strings"
)

type PDF struct {
	Version  string    `json:"version"`
	Children []*Object `json:"children"`
}

type ObjectIdentifier struct {
	ObjectNumber     int `json:"number"`
	ObjectGeneration int `json:"generation"`
}

func (o *ObjectIdentifier) String() string {
	return fmt.Sprintf("num -> %d, gen -> %d", o.ObjectNumber, o.ObjectGeneration)
}

func (o *ObjectIdentifier) Hash() string {
	return strconv.Itoa(o.ObjectNumber) + "," + strconv.Itoa(o.ObjectGeneration)
}

type Object struct {
	Identifier ObjectIdentifier   `json:"identifier"`
	Children   []ObjectType       `json:"children"`
	References []*ObjectReference `json:"references"`
}

var indent = 0

func padding() string {
	output := ""
	for i := 0; i < indent; i++ {
		output += "\t"
	}
	return output
}

func (o *Object) String() string {
	indent++
	items := make([]string, 0)
	for _, child := range o.Children {
		items = append(items, padding()+child.String())
	}
	indent--
	return fmt.Sprintf("Object[ %s, refs -> %d ]\n(\n%s\n)\n\n", o.Identifier.String(), len(o.References), strings.Join(items, "\n"))
}

func NewObject(id ObjectIdentifier, children []ObjectType) *Object {
	return &Object{
		Identifier: id,
		Children:   children,
	}
}

type ObjectType interface {
	String() string
}

type Boolean struct {
	Type  string `json:"type"`
	Value bool   `json:"value"`
}

func (b *Boolean) String() string {
	if b.Value {
		return "true"
	} else {
		return "false"
	}
}

func NewBoolean(b bool) *Boolean {
	return &Boolean{
		Type:  "boolean",
		Value: b,
	}
}

type String struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (s *String) String() string {
	v := s.Value[1 : len(s.Value)-1]
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "\n", "\\n")
	v = strings.ReplaceAll(v, "\t", "\\t")
	v = strings.ReplaceAll(v, "\"", "\\\"")
	return fmt.Sprintf("\"%s\"", v)
}

func NewString(s string) *String {
	return &String{
		Type:  "string",
		Value: s,
	}
}

type Null struct {
	Type string `json:"type"`
}

func (n *Null) String() string {
	return "null"
}

func NewNull() *Null {
	return &Null{
		Type: "null",
	}
}

type FloatingNumber struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

func (f *FloatingNumber) String() string {
	return fmt.Sprintf("%f", f.Value)
}

func NewFloatingNumber(f float64) *FloatingNumber {
	return &FloatingNumber{
		Type:  "float",
		Value: f,
	}
}

type IntegerNumber struct {
	Type  string `json:"type"`
	Value int64  `json:"value"`
}

func (i *IntegerNumber) String() string {
	return fmt.Sprintf("%d", i.Value)
}

func NewIntegerNumber(i int64) *IntegerNumber {
	return &IntegerNumber{
		Type:  "int",
		Value: i,
	}
}

type ObjectReference struct {
	Type  string           `json:"type"`
	Link  ObjectIdentifier `json:"link"`
	Value *Object          `json:"value"`
}

func (o *ObjectReference) String() string {
	return fmt.Sprintf("Object[ %s ]", o.Link.String())
}

func NewReference(ref ObjectIdentifier) *ObjectReference {
	return &ObjectReference{
		Type: "reference",
		Link: ref,
	}
}

type Label struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (l *Label) String() string {
	return l.Value
}

func NewLabel(t string) *Label {
	return &Label{
		Type:  "label",
		Value: t,
	}
}

type Stream struct {
	Type  string `json:"type"`
	Value []byte `json:"value"`
}

func (s *Stream) String() string {
	return fmt.Sprintf("stream_%d", len(s.Value))
}

func NewStream(b []byte) *Stream {
	return &Stream{
		Type:  "stream",
		Value: b,
	}
}

type KeyValuePair struct {
	Key   ObjectType `json:"key"`
	Value ObjectType `json:"value"`
}

func (k *KeyValuePair) String() string {
	return fmt.Sprintf("%s -> %s", k.Key.String(), k.Value.String())
}

type Dictionary struct {
	Type  string         `json:"type"`
	Value []KeyValuePair `json:"value"`
}

func (d *Dictionary) String() string {
	indent++
	items := make([]string, 0)
	for _, pair := range d.Value {
		items = append(items, padding()+pair.String())
	}
	indent--
	return "{\n" + strings.Join(items, ",\n") + "\n" + padding() + "}"
}

func NewDictionary(dict []KeyValuePair) *Dictionary {
	return &Dictionary{
		Type:  "dict",
		Value: dict,
	}
}

type Array struct {
	Type  string       `json:"type"`
	Value []ObjectType `json:"value"`
}

func (a *Array) String() string {
	indent++
	items := make([]string, 0)
	for _, o := range a.Value {
		items = append(items, padding()+o.String())
	}
	indent--
	return "[\n" + strings.Join(items, ",\n") + "\n" + padding() + "]"
}

func NewArray(arr []ObjectType) *Array {
	return &Array{
		Type:  "array",
		Value: arr,
	}
}

package pdf

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var HideIdentifiers = false
var NoIndents = false
var HideStreamLength = false
var HideVariableData = false
var HideRandomKeys = false
var TrimFontPrefix = false

type PDF struct {
	Version string             `json:"version"`
	Objects map[string]*Object `json:"objects"`
}

type ObjectType interface {
	String() string
}

func (p *PDF) String() string {
	buffer := ""
	for _, child := range p.Objects {
		buffer += child.String()
	}
	return buffer
}

type ObjectIdentifier struct {
	ObjectNumber     int `json:"number"`
	ObjectGeneration int `json:"generation"`
}

func (o *ObjectIdentifier) String() string {
	return fmt.Sprintf("num:%d, gen:%d", o.ObjectNumber, o.ObjectGeneration)
}

func (o *ObjectIdentifier) Hash() string {
	return strconv.Itoa(o.ObjectNumber) + "," + strconv.Itoa(o.ObjectGeneration)
}

type Object struct {
	Identifier ObjectIdentifier   `json:"identifier"`
	Children   []ObjectType       `json:"children"`
	References []*ObjectReference `json:"references"`
	Depth      int                `json:"depth"`
}

var indent = 0

func padding() string {
	if NoIndents {
		return ""
	}
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
	header := ""
	if !HideIdentifiers {
		header = fmt.Sprintf(" %s, refs:%d ", o.Identifier.String(), len(o.References))
	}
	return fmt.Sprintf("Object(%s) {\n%s\n}\n\n", header, strings.Join(items, "\n"))
}

func NewObject(id ObjectIdentifier, children []ObjectType) *Object {
	return &Object{
		Identifier: id,
		Children:   children,
	}
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

type Text struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func toHash(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Text) String() string {
	if len(s.Value) > 240 {
		hash := toHash(s.Value)
		return fmt.Sprintf("String( length: %d, hash: %s )", len(s.Value), hash)
	}
	v := s.Value[1 : len(s.Value)-1]
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "\n", "\\n")
	v = strings.ReplaceAll(v, "\t", "\\t")
	v = strings.ReplaceAll(v, "\"", "\\\"")
	return fmt.Sprintf("\"%s\"", v)
}

func NewText(s string) *Text {
	return &Text{
		Type:  "string",
		Value: s,
	}
}

type HexString struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (s *HexString) String() string {
	return s.Value
}

func NewHexString(s string) *HexString {
	return &HexString{
		Type:  "hex",
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

type Number struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

func (f *Number) String() string {
	return fmt.Sprintf("%f", f.Value)
}

func NewNumber(f float64) *Number {
	return &Number{
		Type:  "number",
		Value: f,
	}
}

type ObjectReference struct {
	Type  string           `json:"type"`
	Link  ObjectIdentifier `json:"link"`
	Value *Object          `json:"value"`
}

func (o *ObjectReference) String() string {
	if HideIdentifiers {
		return fmt.Sprintf("Ref()")
	} else {
		return fmt.Sprintf("Ref( %s )", o.Link.String())
	}
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
	return l.Value[1:]
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
	if HideStreamLength {
		return fmt.Sprintf("Stream()")
	} else {
		return fmt.Sprintf("Stream( size:%d )", len(s.Value))
	}
}

func NewStream(b []byte) *Stream {
	return &Stream{
		Type:  "stream",
		Value: b,
	}
}

type KeyValuePair struct {
	K ObjectType `json:"key"`
	V ObjectType `json:"value"`
}

var variableDictKeys = []string{
	"LastModified",
	"ModDate",
	"Length",
	"CreationDate",
}

var fontDictKeys = []string{
	"BaseFont",
	"FontName",
}

var reRandomDictKeys = regexp.MustCompile("([A-Za-z]{1,4})([0-9]+)")

func (k *KeyValuePair) String() string {
	return fmt.Sprintf("%s -> %s", k.Key(), k.Value())
}

func (k *KeyValuePair) Value() string {
	key := k.K.String()
	if HideVariableData {
		for _, vk := range variableDictKeys {
			if strings.HasPrefix(key, vk) {
				return "String()"
			}
		}
	}
	if TrimFontPrefix {
		for _, vk := range fontDictKeys {
			if strings.HasPrefix(key, vk) {
				xs := strings.Split(k.V.String(), "+")
				if len(xs) == 1 || len(xs[0]) != 6 {
					return k.V.String()
				} else {
					return strings.Join(xs[1:], "+")
				}
			}
		}
	}
	return k.V.String()
}

func (k *KeyValuePair) Key() string {
	key := k.K.String()
	if HideRandomKeys {
		lastChar := key[len(key)-1]
		if !(lastChar >= '0' && lastChar <= '9') {
			return key
		}
		parsePrefix := true
		hasPrefix := false
		prefix := bytes.Buffer{}
		for _, c := range key {
			if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
				if !parsePrefix {
					return key
				}
				hasPrefix = true
				prefix.WriteRune(c)
			} else if c >= '0' && c <= '9' {
				if !hasPrefix {
					return key
				}
				parsePrefix = false
			}
		}
		if !parsePrefix {
			return fmt.Sprintf("Key( prefix:%s )", prefix.String())
		}
	}
	return key
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
	if len(items) == 0 {
		return "Dict( size:0 ) {}"
	}
	return fmt.Sprintf("Dict( size:%d ) {\n%s\n"+padding()+"}", len(items), strings.Join(items, ",\n"))
}

func NewDictionary(dict []KeyValuePair) *Dictionary {
	sort.Slice(dict, func(i, j int) bool {
		k1 := dict[i].Key()
		k2 := dict[j].Key()
		if k1 != k2 {
			return k1 < k2
		}
		v1 := dict[i].Value()
		v2 := dict[j].Value()
		return v1 < v2
	})
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
	if len(items) == 0 {
		return "Array( size:0 ) []"
	}
	return fmt.Sprintf("Array( size:%d ) [\n%s\n"+padding()+"]", len(items), strings.Join(items, ",\n"))
}

func NewArray(arr []ObjectType) *Array {
	return &Array{
		Type:  "array",
		Value: arr,
	}
}

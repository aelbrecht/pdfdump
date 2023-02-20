package pdf

type PDF struct {
	Version  string    `json:"version"`
	Children []*Object `json:"children"`
}

type ObjectIdentifier struct {
	ObjectNumber     int `json:"number"`
	ObjectGeneration int `json:"generation"`
}

type Object struct {
	Identifier ObjectIdentifier `json:"identifier"`
	Children   []ObjectType     `json:"children"`
}

func NewObject(id ObjectIdentifier, children []ObjectType) *Object {
	return &Object{
		Identifier: id,
		Children:   children,
	}
}

type ObjectType interface {
}

type Boolean struct {
	Type  string `json:"type"`
	Value bool   `json:"value"`
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

func NewString(s string) *String {
	return &String{
		Type:  "string",
		Value: s,
	}
}

type Null struct {
	Type string `json:"type"`
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

func NewIntegerNumber(i int64) *IntegerNumber {
	return &IntegerNumber{
		Type:  "int",
		Value: i,
	}
}

type ObjectReference struct {
	Type  string           `json:"type"`
	Value ObjectIdentifier `json:"value"`
}

func NewReference(ref ObjectIdentifier) *ObjectReference {
	return &ObjectReference{
		Type:  "reference",
		Value: ref,
	}
}

type Label struct {
	Type  string `json:"type"`
	Value string `json:"value"`
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

type Dictionary struct {
	Type  string         `json:"type"`
	Value []KeyValuePair `json:"value"`
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

func NewArray(arr []ObjectType) *Array {
	return &Array{
		Type:  "array",
		Value: arr,
	}
}

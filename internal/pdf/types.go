package pdf

type ObjectIdentifier struct {
	ObjectNumber     int
	ObjectGeneration int
}

type Object struct {
	Identifier ObjectIdentifier
}

type Dictionary struct {
	Data map[string]string
}

type Array struct {
}

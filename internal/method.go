package codeStructure

type Method struct {
	Name       string
	Params     []Parameter
	ReturnType Type
}

type Parameter struct {
	Name      string
	Type      Type
	IsPointer bool
}

package metadata

type Type struct {
	Name       string
	Properties []Property
	IsPointer  bool
	IsArray    bool
	IsBuiltIn  bool
}

type Property struct {
	Name string
	Type Type
}

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

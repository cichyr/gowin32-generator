package codeStructure

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/microsoft/go-winmd"
	"github.com/microsoft/go-winmd/flags"
)

type Type struct {
	Name       string
	Properties []Property
	IsPointer  bool
	IsArray    bool
	IsBuiltIn  bool
}

func (t *Type) GenerateStruct(statement *jen.Statement) {
	if t.IsBuiltIn {
		return
	}

	statement.
		Type().
		Id(t.Name).
		StructFunc(func(g *jen.Group) {
			for _, prop := range t.Properties {
				prop.WriteProperty(g)
			}
		}).
		Line()
}

//lint:ignore U1000 Ignore unused function temporarily for debugging
func GetType(metadata winmd.Metadata, sigType winmd.SigType) (Type, error) {
	builtInType, found := builtInElementTypes[sigType.Kind]
	if found {
		return Type{Name: builtInType, IsBuiltIn: true}, nil
	}

	if sigType.Kind == flags.ElementType_PTR {
		innerSigType, _ := sigType.Value.(winmd.SigType)
		innerType, err := GetType(metadata, innerSigType)
		innerType.IsPointer = true
		return innerType, err
	}

	if sigType.Kind == flags.ElementType_ARRAY {
		innerSigType, _ := sigType.Value.(winmd.SigType)
		innerType, err := GetType(metadata, innerSigType)
		innerType.IsArray = true
		return innerType, err
	}

	typeDef, err := getTypeDef(metadata, sigType)
	if err != nil {
		return Type{}, fmt.Errorf("no matching type definition for type was found: %w", err)
	}

	builtInType, found = builtInTypes[typeDef.Name.String()]
	if found {
		return Type{Name: builtInType, IsBuiltIn: true}, nil
	}

	retType := Type{Name: typeDef.Name.String(), Properties: make([]Property, 0)}
	for i := typeDef.FieldList.Start; i < typeDef.FieldList.End; i++ {
		field, err := metadata.Tables.Field.Record(i)
		if err != nil {
			return Type{}, fmt.Errorf("no matching field was found: %w", err)
		}
		property, err := GetProperty(metadata, *field)
		if err != nil {
			return Type{}, fmt.Errorf("no matching type definition for type was found: %w", err)
		}
		retType.Properties = append(retType.Properties, property)
	}

	return retType, nil
}

func GetProperty(metadata winmd.Metadata, field winmd.Field) (Property, error) {
	fieldSignature, err := metadata.FieldSignature(field.Signature)
	if err != nil {
		return Property{}, fmt.Errorf("no matching field signature for field '%s' was found: %w", field.Name.String(), err)
	}
	propertyType, err := GetType(metadata, fieldSignature.Type)
	if err != nil {
		return Property{}, fmt.Errorf("could not determine property type: %w", err)
	}

	return Property{Name: field.Name.String(), Type: propertyType}, nil
}

func getTypeDef(metadata winmd.Metadata, sigType winmd.SigType) (winmd.TypeDef, error) {
	sigTypeIndex := sigType.Value.(winmd.CodedIndex)
	retTypeRef, err := metadata.Tables.TypeRef.Record(sigTypeIndex.Index)
	if err != nil {
		return winmd.TypeDef{}, fmt.Errorf("did not found matching type reference: %w", err)
	}

	var typeDef *winmd.TypeDef = nil
	for i := uint32(0); i < metadata.Tables.TypeDef.Len && typeDef == nil; i++ {
		x, _ := metadata.Tables.TypeDef.Record(winmd.Index(i))
		if x.Name.String() == retTypeRef.Name.String() && x.Namespace.String() == retTypeRef.Namespace.String() {
			typeDef = x
		}
	}

	if typeDef == nil {
		return winmd.TypeDef{}, fmt.Errorf("did not found matching type definition: %w", err)
	}

	return *typeDef, nil
}

type Property struct {
	Name string
	Type Type
}

func (p *Property) WriteProperty(group *jen.Group) {
	group.Id(p.Name).Id(p.Type.Name)
}

var builtInElementTypes map[flags.ElementType]string = map[flags.ElementType]string{
	flags.ElementType_BOOLEAN: "bool",
	flags.ElementType_CHAR:    "rune",
	flags.ElementType_STRING:  "string",
	flags.ElementType_I1:      "int8",
	flags.ElementType_I2:      "int16",
	flags.ElementType_I4:      "int32",
	flags.ElementType_I8:      "int64",
	flags.ElementType_U1:      "uint8",
	flags.ElementType_U2:      "uint16",
	flags.ElementType_U4:      "uint32",
	flags.ElementType_U8:      "uint64",
	flags.ElementType_R4:      "float32",
	flags.ElementType_R8:      "float64",
}

var builtInTypes map[string]string = map[string]string{
	"BOOL": "uint32",
}

// The package used for operating on and describing Windows Metadata.
package metadata

import (
	"debug/pe"
	"fmt"
	"gowin32/internal"
	"runtime"

	"github.com/microsoft/go-winmd"
	"github.com/microsoft/go-winmd/flags"
)

type WinMdReader struct {
	metadata winmd.Metadata
}

// The map of basic C types to Go equivalents
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

// The map of types created by `typedef` in C code to Go types
var builtInTypeDefs map[string]string = map[string]string{
	"BOOL": "uint32",
}

// Generates a new metadata reader based WinMd file under given path
func NewReader(winMdPath string) WinMdReader {
	peFile, err := pe.Open(winMdPath)
	internal.PanicOnError(err)
	defer peFile.Close()

	winmdMetadata, err := winmd.New(peFile)
	internal.PanicOnError(err)

	return WinMdReader{
		*winmdMetadata,
	}
}

// Tries to get method with given name
func (reader *WinMdReader) TryGetMethod(name string) (element Method, found bool) {
	methodDef := reader.tryGetMethodDef(name)
	if methodDef == nil {
		return Method{}, false
	}

	return reader.getMethod(methodDef), true
}

// Tries to get type with given name
func (reader *WinMdReader) TryGetType(name string) (element Type, found bool) {
	typeDef := reader.tryGetTypeDef(name)
	if typeDef == nil {
		return Type{}, false
	}

	// return reader.getMethod(methodDef), true
	return Type{}, false
}

func (reader *WinMdReader) getType(sigType winmd.SigType) (Type, error) {
	builtInType, found := builtInElementTypes[sigType.Kind]
	if found {
		return Type{Name: builtInType, IsBuiltIn: true}, nil
	}

	if sigType.Kind == flags.ElementType_PTR {
		innerSigType, _ := sigType.Value.(winmd.SigType)
		innerType, err := reader.getType(innerSigType)
		innerType.IsPointer = true
		return innerType, err
	}

	if sigType.Kind == flags.ElementType_ARRAY {
		innerSigType, _ := sigType.Value.(winmd.SigType)
		innerType, err := reader.getType(innerSigType)
		innerType.IsArray = true
		return innerType, err
	}

	typeDef, err := reader.getTypeDef(sigType)
	if err != nil {
		return Type{}, fmt.Errorf("no matching type definition for type was found: %w", err)
	}

	builtInType, found = builtInTypeDefs[typeDef.Name.String()]
	if found {
		return Type{Name: builtInType, IsBuiltIn: true}, nil
	}

	retType := Type{Name: typeDef.Name.String(), Properties: make([]Property, 0)}
	for i := typeDef.FieldList.Start; i < typeDef.FieldList.End; i++ {
		field, err := reader.metadata.Tables.Field.Record(i)
		if err != nil {
			return Type{}, fmt.Errorf("no matching field was found: %w", err)
		}
		property, err := reader.getProperty(*field)
		if err != nil {
			return Type{}, fmt.Errorf("no matching type definition for type was found: %w", err)
		}
		retType.Properties = append(retType.Properties, property)
	}

	return retType, nil
}

func (reader *WinMdReader) getProperty(field winmd.Field) (Property, error) {
	fieldSignature, err := reader.metadata.FieldSignature(field.Signature)
	if err != nil {
		return Property{}, fmt.Errorf("no matching field signature for field '%s' was found: %w", field.Name.String(), err)
	}
	propertyType, err := reader.getType(fieldSignature.Type)
	if err != nil {
		return Property{}, fmt.Errorf("could not determine property type: %w", err)
	}

	return Property{Name: field.Name.String(), Type: propertyType}, nil
}

func (reader *WinMdReader) getTypeDef(sigType winmd.SigType) (winmd.TypeDef, error) {
	sigTypeIndex := sigType.Value.(winmd.CodedIndex)
	retTypeRef, err := reader.metadata.Tables.TypeRef.Record(sigTypeIndex.Index)
	if err != nil {
		return winmd.TypeDef{}, fmt.Errorf("did not found matching type reference: %w", err)
	}

	var typeDef *winmd.TypeDef = nil
	for i := uint32(0); i < reader.metadata.Tables.TypeDef.Len && typeDef == nil; i++ {
		x, _ := reader.metadata.Tables.TypeDef.Record(winmd.Index(i))
		if x.Name.String() == retTypeRef.Name.String() && x.Namespace.String() == retTypeRef.Namespace.String() {
			typeDef = x
		}
	}

	if typeDef == nil {
		return winmd.TypeDef{}, fmt.Errorf("did not found matching type definition: %w", err)
	}

	return *typeDef, nil
}

func (reader *WinMdReader) tryGetTypeDef(name string) *winmd.TypeDef {
	table := reader.metadata.Tables.TypeDef
	for idx := uint32(0); idx < table.Len; idx++ {
		typeDef, err := table.Record(winmd.Index(idx))
		internal.PanicOnError(err) // It returns an error only when creating return value and for out of scope file
		if typeDef.Name.String() == name {
			return typeDef
		}
	}

	return nil
}

// Gets the name of the *.dll file that implements given member.
// ToDo: Reconsider returning `found bool`.
func (reader *WinMdReader) getImportingDll(memberName string) (importingDll string, found bool) {
	for i := uint32(0); i < reader.metadata.Tables.ImplMap.Len; i++ {
		implMap, err := reader.metadata.Tables.ImplMap.Record(winmd.Index(i))
		internal.PanicOnError(err)
		if implMap.ImportName.String() == memberName {
			dllImport, err := reader.metadata.Tables.ModuleRef.Record(implMap.ImportScope)
			internal.PanicOnError(err)
			return dllImport.Name.String(), true
		}
	}

	return "", false
}

func (reader *WinMdReader) getMethod(methodDef *winmd.MethodDef) Method {
	methodSignature, err := reader.metadata.MethodDefSignature(methodDef.Signature)
	internal.PanicOnError(err)

	returnType, err := reader.getType(methodSignature.RetType.Type)
	internal.PanicOnError(err)

	dllName, _ := reader.getImportingDll(methodDef.Name.String())
	method := Method{
		Name:       methodDef.Name.String(),
		ReturnType: returnType,
		DllImport:  dllName,
	}

	methodParamListValues := make([]winmd.Param, 0)
	for idx := uint32(methodDef.ParamList.Start + 1); idx < uint32(methodDef.ParamList.End); idx++ {
		param, err := reader.metadata.Tables.Param.Record(winmd.Index(idx))
		internal.PanicOnError(err)
		methodParamListValues = append(methodParamListValues, *param)
	}

	for i := 0; i < len(methodSignature.Param); i++ {
		methodParam := methodSignature.Param[i]
		methodParamSignature := methodParam.Type.Value.(winmd.SigType)
		paramType, err := reader.getType(methodParamSignature)
		internal.PanicOnError(err)
		method.Params = append(
			method.Params,
			Parameter{
				Name:      methodParamListValues[0].Name.String(),
				Type:      paramType,
				IsPointer: methodParam.Type.Kind == flags.ElementType_PTR,
			})
	}

	return method
}

// ToDo: Consider inlining it
func (reader *WinMdReader) tryGetMethodDef(name string) *winmd.MethodDef {
	return findElementInTable(
		reader.metadata.Tables.MethodDef,
		func(methodDef *winmd.MethodDef) bool { return methodDef.Name.String() == name })

	// table := reader.metadata.Tables.MethodDef
	// for idx := uint32(0); idx < table.Len; idx++ {
	// 	methodDef, err := table.Record(winmd.Index(idx))
	// 	internal.PanicOnError(err) // It returns an error only when creating return value and for out of scope file
	// 	if methodDef.Name.String() == name {
	// 		return methodDef
	// 	}
	// }

	// return nil
}

//lint:ignore U1000 In development
func iterateOverTable[T any, TP winmd.Record[T]](table winmd.Table[T, TP], action func(TP)) {
	for idx := uint32(0); idx < table.Len; idx++ {
		element, err := table.Record(winmd.Index(idx))
		internal.PanicOnError(err) // It returns an error only when creating return value and for out of scope file
		action(element)
	}
}

// Finds element in given table and returns it. If element is not found then `nil` is returned.
func findElementInTable[T any, TP winmd.Record[T]](table winmd.Table[T, TP], action func(TP) bool) TP {
	for idx := uint32(0); idx < table.Len; idx++ {
		element, err := table.Record(winmd.Index(idx))
		internal.PanicOnError(err) // It returns an error only when creating return value and for out of scope file
		if action(element) {
			return element
		}
	}
}

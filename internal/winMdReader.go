package codeStructure

import (
	"github.com/microsoft/go-winmd"
	"github.com/microsoft/go-winmd/flags"
)

type WinMdReader struct {
	Metadata winmd.Metadata
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func (reader *WinMdReader) GetMethod(name string) Method {
	methodDef, _ := reader.getMethodDef(name)
	methodSignature, err := reader.Metadata.MethodDefSignature(methodDef.Signature)
	panicOnError(err)
	returnType, err := GetType(reader.Metadata, methodSignature.RetType.Type)
	panicOnError(err)
	method := Method{
		Name:       methodDef.Name.String(),
		ReturnType: returnType}

	methodParamListVals := make([]winmd.Param, 0)
	for idx := uint32(methodDef.ParamList.Start + 1); idx < uint32(methodDef.ParamList.End); idx++ {
		param, err := reader.Metadata.Tables.Param.Record(winmd.Index(idx))
		panicOnError(err)
		methodParamListVals = append(methodParamListVals, *param)
	}

	for i := 0; i < len(methodSignature.Param); i++ {
		methodParam := methodSignature.Param[i]
		methodParamSignature := methodParam.Type.Value.(winmd.SigType)
		paramType, err := GetType(reader.Metadata, methodParamSignature)
		panicOnError(err)
		method.Params = append(
			method.Params,
			Parameter{
				Name:      methodParamListVals[0].Name.String(),
				Type:      paramType,
				IsPointer: methodParam.Type.Kind == flags.ElementType_PTR,
			})
	}

	return method
}

func (reader *WinMdReader) getMethodDef(name string) (winmd.MethodDef, bool) {
	for idx := uint32(0); idx < reader.Metadata.Tables.MethodDef.Len; idx++ {
		methodDef, err := reader.Metadata.Tables.MethodDef.Record(winmd.Index(idx))
		panicOnError(err) // It returns an error only when creating return value and for out of scope file
		if methodDef.Name.String() == name {
			return *methodDef, true
		}
	}

	return winmd.MethodDef{}, false
}

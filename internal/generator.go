package codeStructure

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

type Generator struct {
	Methods []Method
	Types   map[string]Type
}

func (generator *Generator) GenerateMethods(statement *jen.Statement) {
	for _, method := range generator.Methods {
		generator.registerType(method.ReturnType)
		for _, param := range method.Params {
			generator.registerType(param.Type)
		}

		statement.Line().Func().Id(method.Name).ParamsFunc(func(g *jen.Group) {
			for _, param := range method.Params {
				if param.IsPointer {
					g.Id(param.Name).Add(jen.Op("*")).Qual("", param.Type.Name)
				} else {
					g.Id(param.Name).Id(param.Type.Name)
				}
			}
		})

		if method.ReturnType.IsPointer {
			statement.Add(jen.Op("*")).Id(method.ReturnType.Name)
		} else {
			statement.Id(method.ReturnType.Name)
		}

		statement.
			BlockFunc(func(g *jen.Group) {
				g.Id("dll").Op(":=").Qual("syscall", "NewLazyDLL").Call(jen.Lit("user32.dll"))
				g.Id("proc").Op(":=").Qual("dll", "NewProc").Call(jen.Lit(method.Name))
				for _, param := range method.Params {
					g.Id(fmt.Sprintf("%sPtr", param.Name)).Op(":=").Qual("unsafe", "Pointer").Call(jen.Id(param.Name))
				}
				g.List(jen.Id("r1"), jen.Id("_"), jen.Id("_")).Op(":=").Qual("proc", "Call").CallFunc(func(g *jen.Group) {
					for _, param := range method.Params {
						g.Id("uintptr").Call(jen.Id(fmt.Sprintf("%sPtr", param.Name)))
					}
				})
				g.Return().Id("r1")
			}).
			Line().
			Line()
	}
}

func (generator *Generator) registerType(typeToRegister Type) {
	if !typeToRegister.IsBuiltIn {
		generator.Types[typeToRegister.Name] = typeToRegister
	}
}

package generation

import (
	"fmt"
	"gowin32/internal/metadata"

	"github.com/dave/jennifer/jen"
)

type Generator struct {
	Methods []metadata.Method
	Types   map[string]metadata.Type
}

func (generator *Generator) RegisterMethod(element metadata.Method) {
	generator.Methods = append(generator.Methods, element)
	for _, param := range element.Params {
		generator.RegisterType(param.Type)
	}
}

func (generator *Generator) RegisterType(element metadata.Type) {
	if !element.IsBuiltIn {
		generator.Types[element.Name] = element
	}
}

func (generator *Generator) Generate(statement *jen.Statement) {
	generator.addTypesFromMethods()

	// ToDo: Should probably group it somehow xD
	// ToDo: Handle files here
	generator.GenerateTypes(statement)
	generator.GenerateMethods(statement)
}

func (generator *Generator) WriteProperty(p metadata.Property, group *jen.Group) {
	group.Id(p.Name).Id(p.Type.Name)
}

func (generator *Generator) GenerateStruct(t metadata.Type, statement *jen.Statement) {
	if t.IsBuiltIn {
		return
	}

	statement.
		Type().
		Id(t.Name).
		StructFunc(func(g *jen.Group) {
			for _, prop := range t.Properties {
				generator.WriteProperty(prop, g)
			}
		}).
		Line()
}

func (generator *Generator) GenerateTypes(statement *jen.Statement) {
	for _, typeToGenerate := range generator.Types {
		statement.Type().Id(typeToGenerate.Name).StructFunc(func(g *jen.Group) {
			for _, prop := range typeToGenerate.Properties {
				g.Id(prop.Name).Id(prop.Type.Name)
			}
		}).Line()
	}
}

func (generator *Generator) addTypesFromMethods() {
	for _, method := range generator.Methods {
		generator.registerType(method.ReturnType)
		for _, param := range method.Params {
			generator.registerType(param.Type)
		}
	}
}

func (generator *Generator) GenerateMethods(statement *jen.Statement) {
	for _, method := range generator.Methods {
		statement.Line().Func().Id(method.Name).ParamsFunc(func(g *jen.Group) {
			for _, param := range method.Params {
				if param.IsPointer {
					g.Id(param.Name).Add(jen.Op("*")).Id(param.Type.Name)
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
				g.Id("dll").Op(":=").Qual("syscall", "NewLazyDLL").Call(jen.Lit(method.DllImport))
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

func (generator *Generator) registerType(typeToRegister metadata.Type) {
	if !typeToRegister.IsBuiltIn {
		generator.Types[typeToRegister.Name] = typeToRegister
	}
}

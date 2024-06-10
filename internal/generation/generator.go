package generation

import (
	"errors"
	"fmt"
	"gowin32/internal"
	"gowin32/internal/metadata"
	"io/fs"
	"os"

	"github.com/dave/jennifer/jen"
)

type Generator struct {
	Methods     []metadata.Method
	Types       map[string]metadata.Type
	PackageName string
	OutputPath  string
}

func NewGenerator(packageName string, outputPath string) Generator {
	return Generator{
		make([]metadata.Method, 0),
		make(map[string]metadata.Type, 0),
		packageName,
		outputPath,
	}
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

func (generator *Generator) Generate(path string) {
	generator.addTypesFromMethods()

	err := os.Mkdir(path, os.ModePerm)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		panic(err)
	}

	// ToDo: Should probably group it somehow xD
	// ToDo: Handle files here
	generator.generateTypes()
	generator.generateMethods()
}

func (generator *Generator) writeProperty(p metadata.Property, group *jen.Group) {
	group.Id(p.Name).Id(p.Type.Name)
}

func (generator *Generator) generateStruct(t metadata.Type, statement *jen.Statement) {
	if t.IsBuiltIn {
		return
	}

	statement.
		Type().
		Id(t.Name).
		StructFunc(func(g *jen.Group) {
			for _, prop := range t.Properties {
				generator.writeProperty(prop, g)
			}
		}).
		Line()
}

func (generator *Generator) generateTypes() {
	for _, typeToGenerate := range generator.Types {
		file := jen.NewFile(generator.PackageName)

		file.Type().Id(typeToGenerate.Name).StructFunc(func(g *jen.Group) {
			for _, prop := range typeToGenerate.Properties {
				g.Id(prop.Name).Id(prop.Type.Name)
			}
		})

		err := file.Save(fmt.Sprintf("%s/%s.go", generator.OutputPath, typeToGenerate.Name))
		internal.PanicOnError(err)
	}
}

func (generator *Generator) addTypesFromMethods() {
	registerType := func(typeToRegister metadata.Type) {
		if !typeToRegister.IsBuiltIn {
			generator.Types[typeToRegister.Name] = typeToRegister
		}
	}

	for _, method := range generator.Methods {
		registerType(method.ReturnType)
		for _, param := range method.Params {
			registerType(param.Type)
		}
	}
}

func (generator *Generator) generateMethods() {
	file := jen.NewFile(generator.PackageName)

	for _, method := range generator.Methods {
		funcHeader := file.Func().Id(method.Name).ParamsFunc(func(g *jen.Group) {
			for _, param := range method.Params {
				if param.IsPointer {
					g.Id(param.Name).Add(jen.Op("*")).Id(param.Type.Name)
				} else {
					g.Id(param.Name).Id(param.Type.Name)
				}
			}
		})

		if method.ReturnType.IsPointer {
			funcHeader.Op("*")
		}

		funcHeader.Id(method.ReturnType.Name)

		funcHeader.
			BlockFunc(func(g *jen.Group) {
				g.Id("dll").Op(":=").Qual("syscall", "NewLazyDLL").Call(jen.Lit(method.DllImport))
				g.Id("proc").Op(":=").Id("dll").Dot("NewProc").Call(jen.Lit(method.Name))
				for _, param := range method.Params {
					if !param.Type.IsBuiltIn {
						g.Id(fmt.Sprintf("%sPtr", param.Name)).Op(":=").Qual("unsafe", "Pointer").Call(jen.Id(param.Name))
					}
				}
				g.List(jen.Id("r1"), jen.Id("_"), jen.Id("_")).Op(":=").Id("proc").Dot("Call").CallFunc(func(g *jen.Group) {
					for _, param := range method.Params {
						g.Id("uintptr").CallFunc(func(g *jen.Group) {
							if param.Type.IsBuiltIn {
								g.Id(param.Name)
							} else {
								g.Id(fmt.Sprintf("%sPtr", param.Name))
							}
						})
					}
				})

				// ToDo: handle return type (and lack of it) correctly
				g.Return().Id(method.ReturnType.Name).Block(jen.Id("int32").Call(jen.Id("r1")).Op(","))
			}).
			Line()
	}

	err := file.Save(fmt.Sprintf("%s/%s.go", generator.OutputPath, generator.PackageName))
	internal.PanicOnError(err)
}

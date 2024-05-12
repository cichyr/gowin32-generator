package main

import (
	"debug/pe"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/microsoft/go-winmd"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	// metadata := loadWinMdFile()
	// method := codeStructure.GetMethod(metadata, "GetCursorPos")
	// typesToGenerate := make([]codeStructure.Type, 1)
	// typesToGenerate[0] = method.ReturnType
	// for _, param := range method.Params {
	// 	if !param.Type.IsBuiltIn {
	// 		typesToGenerate = append(typesToGenerate, param.Type)
	// 	}
	// }

	// file := jen.Line()

	// for _, typeToGenerate := range typesToGenerate {
	// 	file.Type().Id(typeToGenerate.Name).StructFunc(func(g *jen.Group) {
	// 		for _, prop := range typeToGenerate.Properties {
	// 			g.Id(prop.Name).Id(prop.Type.Name)
	// 		}
	// 	}).Line()
	// }

	// file.Line().Func().Id(method.Name).ParamsFunc(func(g *jen.Group) {
	// 	for _, param := range method.Params {
	// 		if param.IsPointer {
	// 			g.Id(param.Name).Add(jen.Op("*")).Qual("", param.Type.Name)
	// 		} else {
	// 			g.Id(param.Name).Id(param.Type.Name)
	// 		}
	// 	}
	// })
	// if method.ReturnType.IsPointer {
	// 	file.Add(jen.Op("*")).Id(method.ReturnType.Name)
	// } else {
	// 	file.Id(method.ReturnType.Name)
	// }
	// file.BlockFunc(func(g *jen.Group) {
	// 	g.Id("dll").Op(":=").Qual("syscall", "NewLazyDLL").Call(jen.Lit("user32.dll"))
	// 	g.Id("proc").Op(":=").Qual("dll", "NewProc").Call(jen.Lit(method.Name))
	// 	for _, param := range method.Params {
	// 		g.Id(fmt.Sprintf("%sPtr", param.Name)).Op(":=").Qual("unsafe", "Pointer").Call(jen.Id(param.Name))
	// 	}
	// 	g.List(jen.Id("r1"), jen.Id("_"), jen.Id("_")).Op(":=").Qual("proc", "Call").CallFunc(func(g *jen.Group) {
	// 		for _, param := range method.Params {
	// 			g.Id("uintptr").Call(jen.Id(fmt.Sprintf("%sPtr", param.Name)))
	// 		}
	// 	})
	// 	g.Return().Id("r1")
	// })

	// fmt.Printf("%#v", file)
	// runtime.KeepAlive(file)
	// runtime.KeepAlive(loadWinMdFile())

	user32 := syscall.NewLazyDLL("user32.dll")
	getCursorPos := user32.NewProc("GetCursorPos")
	point := POINT{}
	lpPoint := unsafe.Pointer(&point)
	r1, r2, error := getCursorPos.Call(uintptr(lpPoint))
	runtime.KeepAlive(error)
	runtime.KeepAlive(r1)
	runtime.KeepAlive(r2)
	fmt.Printf("X: %d\nY: %d", point.X, point.Y)
}

//lint:ignore U1000 Ignore unused function temporarily for debugging
func loadWinMdFile() winmd.Metadata {
	peFile, error := pe.Open("Windows.Win32.winmd")
	check(error)
	defer peFile.Close()

	winmdMetadata, error := winmd.New(peFile)
	check(error)

	return *winmdMetadata
}

type POINT struct {
	X int32
	Y int32
}

// [SupportedOSPlatform("windows5.0")]
// [Documentation("https://learn.microsoft.com/windows/win32/api/winuser/nf-winuser-getcursorpos")]
// [DllImport("USER32.dll", SetLastError = true, PreserveSig = false)]
// public static extern BOOL GetCursorPos([Out] POINT* lpPoint);
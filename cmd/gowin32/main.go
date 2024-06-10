// Some text
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"gowin32/internal"
	"gowin32/internal/generation"
	"gowin32/internal/metadata"
	"io"
	"io/fs"
	"log"
	"os"
	"runtime"
	"strings"
)

func main() {
	var metadataFilePath = flag.String("metadataPath", "Windows.Win32.winmd", "The path to the metadata file to read. Default: Windows.Win32.winmd")
	var inputFilePath = flag.String("input", "", "The path to the file containing information what types and/or methods should be read from the metadata file.")
	var packageName = flag.String("packageName", "PInvoke", "The name of the package with generated code. Default: PInvoke")
	var outputPath = flag.String("outputPath", "./output/", "The path where all generated files will be placed.")
	var forceClean = flag.Bool("forceCleanOutput", false, "If given forces cleaning output file before generation.")
	flag.Usage = func() {
		fmt.Println("App that helps generate PInvoke calls.")
		flag.PrintDefaults()
	}

	/*
		ToDo: Params to handle:
		- EmitSingleFile (default -> false)
		- PackageName    (default -> PInvoke)
		- FileName       (default -> test_input.txt)
		- MetadataPath   (default -> Windows.Win32.winmd)
	*/

	flag.Parse()

	if _, err := os.Stat(*metadataFilePath); errors.Is(err, os.ErrNotExist) {
		*metadataFilePath = "Windows.Win32.winmd"
		metadata.DownloadMetadata(*metadataFilePath)
	}

	if *inputFilePath == "" {
		log.Fatal("Input file path is missing!")
	} else if _, err := os.Stat(*inputFilePath); errors.Is(err, os.ErrNotExist) {
		log.Fatal("Input file does not exist!")
	}

	err := os.Mkdir(*outputPath, os.ModePerm)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		panic(err)
	}

	err = ClearDirectoryIfNotEmpty(*outputPath, *forceClean)
	internal.PanicOnError(err)

	metadataReader := metadata.NewReader("Windows.Win32.winmd")
	generator := generation.NewGenerator(*packageName, *outputPath)

	file, err := os.Open(*inputFilePath)
	internal.PanicOnError(err)
	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		if err := fileScanner.Err(); err != nil {
			log.Fatal(err)
		}

		methodElement, found := metadataReader.TryGetMethod(fileScanner.Text())
		if found {
			generator.RegisterMethod(methodElement)
			continue
		}

		typeElement, found := metadataReader.TryGetType(fileScanner.Text())
		if found {
			generator.RegisterType(typeElement)
			continue
		}
	}

	generator.Generate(*outputPath)

	runtime.KeepAlive(packageName)
}

func ClearDirectoryIfNotEmpty(path string, silent bool) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()

	_, err = directory.Readdirnames(1)
	if err == io.EOF {
		return nil
	}

	if err != nil {
		return err
	}

	var response string
	if !silent {
		fmt.Print("Output directory is not empty. Continuation will result in removing all output file. Proceed? [Y/n]")
		fmt.Scan(&response)
		if strings.ToUpper(response) != "Y" {
			log.Fatal("Explicit agreement was not given. Exiting.")
		}
	}

	fmt.Println("Cleaning output directory.")
	// return os.RemoveAll(path) - Suppressing error for now
	os.RemoveAll(path)
	return nil
}

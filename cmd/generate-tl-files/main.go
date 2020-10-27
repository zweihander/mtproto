package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/dave/jennifer/jen"
	"github.com/k0kubun/pp"
)

func normalizeID(s string, isType bool) string {
	isVector := strings.HasPrefix(s, "Vector<")
	s = strings.TrimPrefix(s, "Vector<")
	s = strings.TrimSuffix(s, ">")
	prefix := ""
	constructorName := s
	index := strings.Index(s, ".")
	if index != -1 {
		prefix = s[:index]
		constructorName = s[index+1:]
		if !unicode.IsUpper([]rune(constructorName)[0]) && isType {
			pp.Println(s)
			panic("something strange, object doesn't look like interface")
		}
	}

	if !unicode.IsUpper([]rune(constructorName)[0]) {
		newOne := []rune(constructorName)
		newOne[0] = unicode.ToUpper(newOne[0])
		constructorName = string(newOne)
	}

	s = prefix + constructorName
	if isVector {
		s = "[]" + s
	}

	if !unicode.IsUpper([]rune(s)[0]) {
		newOne := []rune(s)
		newOne[0] = unicode.ToUpper(newOne[0])
		s = string(newOne)
	}

	switch s {
	case "type",
		"default",
		"range":
		return "_" + s
	default:
		return s
	}
}

const helpMsg = `generate-tl-files
usage: generate-tl-files input_file.tl output_dir/

THIS TOOL IS USING ONLY FOR AUTOMATIC CODE
GENERATION, DO NOT GENERATE FILES BY HAND!

No, seriously. Don't. go generate is amazing. You
are amazing too, but lesser 😏
`

func main() {
	for _, s := range os.Args {
		if s == "--help" {
			fmt.Println(helpMsg)
			os.Exit(0)
		}
	}

	if len(os.Args) < 2 {
		fmt.Println(helpMsg)
		os.Exit(1)
	}

	inputFilePath := os.Args[1]
	outputDir := os.Args[2]

	if err := rootCmd(inputFilePath, outputDir); err != nil {
		panic(err)
	}
}

func rootCmd(inputFilePath, outputDir string) error {
	data, err := ioutil.ReadFile(inputFilePath)
	if err != nil {
		return err
	}

	res, err := ParseTL(string(data))
	if err != nil {
		return fmt.Errorf("parse tl: %w", err)
	}

	s, err := FileFromTlSchema(res)
	if err != nil {
		return fmt.Errorf("create file from parsed tl: %w", err)
	}

	err = GenerateAndWirteTo(GenerateEnumDefinitions, s, filepath.Join(outputDir, "enums.go"))
	if err != nil {
		return fmt.Errorf("GenerateEnumDefinitionss: %w", err)
	}

	err = GenerateAndWirteTo(GenerateSpecificStructs, s, filepath.Join(outputDir, "types.go"))
	if err != nil {
		return fmt.Errorf("GenerateSpecificStructs: %w", err)
	}

	err = GenerateAndWirteTo(GenerateInterfaces, s, filepath.Join(outputDir, "interfaces.go"))
	if err != nil {
		return fmt.Errorf("GenerateInterfaces: %w", err)
	}
	err = GenerateAndWirteTo(GenerateMethods, s, filepath.Join(outputDir, "methods.go"))
	if err != nil {
		return fmt.Errorf("GenerateMethods: %w", err)
	}

	err = GenerateAndWirteTo(GenerateConstructorRouter, s, filepath.Join(outputDir, "constructor.go"))
	if err != nil {
		return fmt.Errorf("GenerateConstructorRouter: %w", err)
	}

	return nil
}

func GenerateAndWirteTo(f func(file *jen.File, data *FileStructure) error, data *FileStructure, storeTo string) error {
	file := jen.NewFile("telegram")
	file.HeaderComment("Code generated by generate-tl-files; DO NOT EDIT.")

	file.ImportAlias("github.com/xelaj/go-dry", "dry")

	if err := f(file, data); err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte{})
	if err := file.Render(buf); err != nil {
		return err
	}

	return ioutil.WriteFile(storeTo, buf.Bytes(), 0644)
}

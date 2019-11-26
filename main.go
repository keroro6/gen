package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	controllerPath string
	filterPath     string
	lowerFlag      bool
	snakeFlag      bool
	shortFlag      bool
	buf            = new(bytes.Buffer)
)

func getGoFiles(path string) (files []string, err error) {
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) (reterr error) {
		if err != nil {
			reterr = err
			return
		}
		if info.IsDir() {
			return
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return
		}
		if filepath.Ext(path) != ".go" {
			return
		}
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			reterr = err
			return
		}
		files = append(files, absPath)
		return
	})
	if err != nil {
		return
	}

	return
}

func parseControllerFiles(path string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return
	}

	for _, decl := range f.Decls {
		mdecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !mdecl.Name.IsExported() {
			continue
		}

		str := new(bytes.Buffer)
		printer.Fprint(str, fset, mdecl)
		//log.Println(str.String())
		mathes := regexp.MustCompile(`func.+?\(.+?\*gin.Engine\) {(.|\n)*}`).FindStringSubmatch(str.String())
		//for _, math2 := range mathes {
		//	log.Println(math2)
		//	log.Println("111111")
		//}
		if len(mathes) <= 1 {
			//log.Println(mathes)
			continue
		}
		fmt.Println(mathes[0])
		mathAll := regexp.MustCompile(`,(.+?)\)`).FindAllStringSubmatch(mathes[0], -1)
		fmt.Println(mathAll)
	}

	return
}
func main() {
	flag.StringVar(&controllerPath, "c", "controller", "Specify the directory of controller path")
	paths, err := getGoFiles(controllerPath)
	if err != nil {
		log.Printf("get go files err:%v", err)
		return
	}

	for _, path := range paths {
		parseControllerFiles(path)
	}
	//log.Println("dddd")
}

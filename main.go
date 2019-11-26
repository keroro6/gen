package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/ioutil"
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
func IsExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}
func parseControllerFiles(path, moduleName string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
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
		dir := filepath.Dir(path)
		idx := strings.LastIndex(dir, "\\")
		packageName := ""
		if idx != -1 {
			packageName = dir[idx+1:]
		}

		for _, math1 := range mathAll {
			//去空格
			tmp := strings.Trim(math1[1], " ")
			bts := []byte(tmp)
			firstLower := string(byte(bts[0]+32)) + tmp[1:]

			file1 := dir + "\\" + firstLower + ".go"
			fmt.Println(file1)
			if IsExist(file1) {
				continue
			}
			var s string
			s += "//gen\n\n"
			s += fmt.Sprintf("package %s\n\n", packageName)
			s += "import (\n"
			s += "\t\"github.com/gin-gonic/gin\"\n"
			fmt.Println(moduleName)
			s += fmt.Sprintf("\t\"%s/conf\"\n", moduleName)
			s += "\t\"xgit.xiaoniangao.cn/xngo/lib/sdk/lib\"\n"
			s += "\t\"xgit.xiaoniangao.cn/xngo/lib/sdk/xng\"\n)\n\n"

			s += fmt.Sprintf("type %sReq struct {\n}\n\n", tmp)
			s += fmt.Sprintf("type %sResp struct {\n}\n\n", tmp)
			s += fmt.Sprintf("func (req *%sReq)checkParam() (ok bool) {\n\treturn\n}\n\n", tmp)
			s += fmt.Sprintf("func %s(c *gin.Context) {\n", tmp)
			s += "\txc := xng.NewXContext(c)\n"
			s += fmt.Sprintf("\tvar req *%sReq\n", tmp)
			s += "\tif !xc.GetReqObject(&req) {\n"
			s += "\t\treturn\n\t}\n"
			s += "\tif !req.checkParam() {\n"
			s += "\t\txc.ReplyFail(lib.CodePara)\n"
			s += "\t\tconf.Logger.Error(\"fail to check param\", \"req\", req)\n"
			s += "\t\treturn\n\t}\n\n"
			s += "\txc.ReplyOK(nil)\n}\n"
			//math1[1]
			err = ioutil.WriteFile(file1, []byte(s), 0644)
			//break
		}
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

	file, _ := os.Open("go.mod")

	defer file.Close()
	rd := bufio.NewReader(file)
	var line string
	for {
		line, err = rd.ReadString('\n') //以'\n'为结束符读入一行
		if err != nil || io.EOF == err {
			break
		}
		idx := strings.Index(line, "module")
		if idx == -1 {
			continue
		}

		line = strings.Replace(line, "module", "", -1)
		line = strings.Trim(line, " \t\n")

		break
	}
	fmt.Println(line)
	_ = paths

	for _, path := range paths {
		parseControllerFiles(path, line)
	}
	//log.Println("dddd")
}

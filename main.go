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

	//"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	controllerFlag bool
	filterPath     string
	serviceFlag    bool
	daoFlag        bool
	redisFlag      bool
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
		mathes := regexp.MustCompile(`func.+?\(.+?\*gin.Engine\) {(.|\n)*}`).FindStringSubmatch(str.String())
		if len(mathes) <= 1 {
			continue
		}
		fmt.Println(mathes[0])
		mathAll := regexp.MustCompile(`,(.+?)\)`).FindAllStringSubmatch(mathes[0], -1)
		fmt.Println(mathAll)
		dir := filepath.Dir(path)
		idx := strings.LastIndex(dir, "/")
		packageName := ""
		if idx != -1 {
			packageName = dir[idx+1:]
		}

		for _, math1 := range mathAll {
			//去空格
			tmp := strings.Trim(math1[1], " ")
			bts := []byte(tmp)
			firstLower := string(byte(bts[0]+32)) + tmp[1:]

			file1 := dir + "/" + firstLower + ".go"
			fmt.Println(file1)

			var s string
			s += "//gen\n\n"
			s += fmt.Sprintf("package %s\n\n", packageName)
			s += "import (\n"
			s += "\t\"github.com/gin-gonic/gin\"\n"
			s += fmt.Sprintf("\t\"%s/api\"\n", moduleName)
			s += fmt.Sprintf("\t\"%s/conf\"\n", moduleName)
			if serviceFlag {
				s += fmt.Sprintf("\t\"%s/service/%s\"\n", moduleName, packageName)
			}
			s += "\t\"xgit.xiaoniangao.cn/xngo/lib/sdk/lib\"\n"
			s += "\t\"xgit.xiaoniangao.cn/xngo/lib/sdk/xng\"\n)\n\n"

			s += fmt.Sprintf("func (req *api.%sReq)checkParam() (ok bool) {\n\treturn\n}\n\n", tmp)
			s += fmt.Sprintf("func %s(c *gin.Context) {\n", tmp)
			s += "\txc := xng.NewXContext(c)\n"
			s += fmt.Sprintf("\tvar req *%sReq\n", tmp)
			s += "\tif !xc.GetReqObject(&req) {\n"
			s += "\t\treturn\n\t}\n"
			s += "\tif !req.checkParam() {\n"
			s += "\t\txc.ReplyFail(lib.CodePara)\n"
			s += "\t\tconf.Logger.Error(\"fail to check param\", \"req\", req)\n"
			s += "\t\treturn\n\t}\n\n"
			s += "\t//some operations\n\n\n"
			if serviceFlag {
				serviceName := fmt.Sprintf("%sService", tmp)
				s += fmt.Sprintf("\tif err := %s.%s(req); err != nil {\n", packageName, serviceName)
				s += fmt.Sprintf("\t\tconf.Logger.Error(\"%s\", \"req\", req, \"err\", err)\n", serviceName)
				s += fmt.Sprintf("\t\txc.ReplyFail(lib.CodeSrv)\n")
				s += fmt.Sprintf("\t\treturn\n\t}\n")
			}
			s += "\txc.ReplyOK(nil)\n}\n"
			//math1[1]
			if !IsExist(file1) {
				err = ioutil.WriteFile(file1, []byte(s), 0644)
			}
			var r string
			r += "//gen\n\n"
			r += fmt.Sprintf("package api\n\n")
			r += fmt.Sprintf("type %sReq struct {\n\n}\n\n", tmp)
			r += fmt.Sprintf("type %sResp struct {\n\n}\n\n", tmp)

			baseDir := filepath.Dir(filepath.Dir(dir))
			apiPath := baseDir + "/api/" + firstLower + "Api.go"

			_ = apiPath
			if !IsExist(apiPath) {
				err = ioutil.WriteFile(apiPath, []byte(r), 0644)
			}

			if serviceFlag {
				servicePath := fmt.Sprintf(baseDir+"/service/%s/%sService.go", packageName, firstLower)
				if !IsExist(servicePath) {
					fmt.Println(servicePath)
					var a string
					a += fmt.Sprintf("package %s\n\n", packageName)
					a += "import (\n"
					a += fmt.Sprintf("\t\"%s/api\"\n)\n", moduleName)
					//a += fmt.Sprintf("\t\"%s/dao\"\n", moduleName)
					//a += "\t\"xgit.xiaoniangao.cn/xngo/lib/sdk/lib\"\n"
					//a += "\t\"xgit.xiaoniangao.cn/xngo/lib/sdk/xng\"\n)\n\n"
					a += fmt.Sprintf("func %sService(req *api.%sReq) {\n\n", tmp, tmp)
					a += "}\n"
					err = ioutil.WriteFile(servicePath, []byte(a), 0644)
				}
			}
			//if serviceFlag {
			//	parseServiceFiles(path, )
			//}
			break
		}
	}

	return
}

func parseDaoFiles(path, moduleName string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return
	}
	fmt.Println(path)
	for _, decl := range f.Decls {
		mdecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		//fmt.Println(mdecl.Tok)
		if mdecl.Doc != nil {
			fmt.Println(mdecl.Specs)
			for _, comment := range (*mdecl.Doc).List {
				//fmt.Println()
				if comment != nil {
					fmt.Println(*comment)
				}
			}

		}

	}
}
func main() {
	flag.BoolVar(&controllerFlag, "c", false, "Specify the directory of controller path")
	flag.BoolVar(&serviceFlag, "s", false, "generate service")
	flag.BoolVar(&daoFlag, "d", false, "generate dao")
	flag.BoolVar(&redisFlag, "r", false, "generate redis")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "A partner for golang webserver\n")
		fmt.Fprintf(os.Stderr, "version: 1.12, Created by wgy [11/2019]\n\n")
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if daoFlag == true || redisFlag == true {
		serviceFlag = true
	}
	paths, err := getGoFiles(".")
	if err != nil {
		log.Printf("get go files err:%v", err)
		return
	}

	log.Printf("s:%v,d:%v,r:%v\n", serviceFlag, daoFlag, redisFlag)
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

	daoFlag = true
	if daoFlag {
		for _, path := range paths {
			parseDaoFiles(path, line)
		}
	}

	if redisFlag {

	}
	//log.Println("dddd")
}

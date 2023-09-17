package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var (
	rootDirF  = flag.String("root", "", "root dir of project")
	dataF     = flag.StringToString("data", nil, "comma-separated key=value pairs")
	templateF = flag.String("template", "", "path to a yaml template to execute")
)

type FileTemplate struct {
	Path     string
	Contents string
}

type Templeton struct {
	root string
	data map[string]string
}

func (ttn *Templeton) Process(ft *FileTemplate) error {
	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
	}

	pathTpl, err := template.New(ft.Path).Funcs(funcMap).Parse(ft.Path)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = pathTpl.Execute(&buf, ttn.data)
	if err != nil {
		return err
	}
	path := buf.String()

	tpl, err := template.New(path).Funcs(funcMap).Parse(ft.Contents)
	if err != nil {
		return err
	}
	dir := filepath.Join(ttn.root, filepath.Dir(path))
	err = os.MkdirAll(dir, 0770)
	if err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(ttn.root, path))
	if err != nil {
		return err
	}
	defer file.Close()
	return tpl.Execute(file, ttn.data)
}

func main() {
	flag.Parse()

	ttn := Templeton{
		root: *rootDirF,
		data: *dataF,
	}

	file, err := os.ReadFile(*templateF)
	if err != nil {
		log.Fatal(err)
	}

	var fts []*FileTemplate
	err = yaml.Unmarshal(file, &fts)

	if err != nil {
		log.Fatal(err)
	}

	for _, ft := range fts {
		err = ttn.Process(ft)
		if err != nil {
			log.Fatal(err)
		}
	}
}
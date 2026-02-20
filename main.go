package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"

	"text/template/parse"

	"github.com/manifoldco/promptui"
	flag "github.com/spf13/pflag"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

var (
	rootDirF  = flag.String("root", "", "root dir of project")
	dataF     = flag.StringToString("data", nil, "comma-separated key=value pairs")
	templateF = flag.String("template", "", "path to a yaml template to execute")
	projectF  = flag.String("project", "", "looks for a template in ~/.templeton/<project>.yaml")
)

type FileTemplate struct {
	Path     string
	Contents string
	Delims   []string
}

type Templeton struct {
	root string
	data map[string]string
}

func ToTile(word string) string {
	caser := cases.Title(language.English)

	return caser.String(word)
}

func (ttn *Templeton) Process(ft *FileTemplate) error {
	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
		"ToTitle": ToTile,
	}

	pathTpl, err := template.New(ft.Path).Funcs(funcMap).Delims(ft.Delims[0], ft.Delims[1]).Parse(ft.Path)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = pathTpl.Execute(&buf, ttn.data)
	if err != nil {
		return err
	}
	path := buf.String()

	tpl, err := template.New(path).Funcs(funcMap).Delims(ft.Delims[0], ft.Delims[1]).Parse(ft.Contents)
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

	if *templateF == "" && *projectF == "" {
		log.Fatal(errors.New("no yaml template specified. use --template or --project to specify one"))
	}

	ttn := Templeton{
		root: *rootDirF,
		data: *dataF,
	}

	templateToUse := *templateF
	if templateToUse == "" {
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		homedir := filepath.Join(usr.HomeDir, ".templeton")

		templateToUse = filepath.Join(homedir, *projectF+".yaml")
	}

	file, err := os.ReadFile(templateToUse)
	if err != nil {
		log.Fatal(err)
	}

	var fts []*FileTemplate
	err = yaml.Unmarshal(file, &fts)

	if err != nil {
		log.Fatal(err)
	}

	for _, ft := range fts {
		if ft.Delims == nil {
			ft.Delims = []string{"{{", "}}"}
		}
	}

	if len(*dataF) == 0 {
		allKeys := make(map[string]bool)
		for _, ft := range fts {
			keys, err := ExtractKeys(ft.Contents, ft.Delims)
			if err != nil {
				log.Fatal(err)
			}
			for _, k := range keys {
				allKeys[k] = true
			}
			keys, err = ExtractKeys(ft.Path, ft.Delims)
			if err != nil {
				log.Fatal(err)
			}
			for _, k := range keys {
				allKeys[k] = true
			}
		}

		if len(allKeys) > 0 {
			ttn.data = make(map[string]string)
			for k := range allKeys {
				prompt := promptui.Prompt{
					Label: "Value for " + k,
				}
				result, err := prompt.Run()
				if err != nil {
					log.Fatal(err)
				}
				ttn.data[k] = result
			}
		}
	}

	for _, ft := range fts {
		err = ttn.Process(ft)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func ExtractKeys(tplContent string, delims []string) ([]string, error) {
	tpl, err := template.New("temp").Delims(delims[0], delims[1]).Parse(tplContent)
	if err != nil {
		return nil, err
	}

	keys := make(map[string]bool)
	collectKeys(tpl.Tree.Root, keys)

	var result []string
	for k := range keys {
		result = append(result, k)
	}
	return result, nil
}

func collectKeys(node parse.Node, keys map[string]bool) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *parse.ListNode:
		for _, next := range n.Nodes {
			collectKeys(next, keys)
		}
	case *parse.ActionNode:
		collectKeys(n.Pipe, keys)
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			collectKeys(cmd, keys)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			collectKeys(arg, keys)
		}
	case *parse.FieldNode:
		if len(n.Ident) > 0 {
			keys[n.Ident[0]] = true
		}
	case *parse.IfNode:
		collectKeys(n.Pipe, keys)
		collectKeys(n.List, keys)
		collectKeys(n.ElseList, keys)
	case *parse.RangeNode:
		collectKeys(n.Pipe, keys)
		collectKeys(n.List, keys)
		collectKeys(n.ElseList, keys)
	case *parse.WithNode:
		collectKeys(n.Pipe, keys)
		collectKeys(n.List, keys)
		collectKeys(n.ElseList, keys)
	}
}

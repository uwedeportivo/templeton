package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"text/template/parse"

	"github.com/charmbracelet/huh"
	flag "github.com/spf13/pflag"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"
)

var (
	rootDirF  = flag.String("root", "", "root dir of project")
	dataF     = flag.StringToString("data", nil, "comma-separated key=value pairs")
	templateF = flag.String("template", "", "path to a yaml template to execute")
	projectF  = flag.String("project", "", "looks for a template in ~/.templeton/<project>.yaml")
)

type FileTemplate struct {
	Path     string   `yaml:"path"`
	Contents string   `yaml:"contents"`
	Delims   []string `yaml:"delims"`
}

type Variable struct {
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
	Validate    string `yaml:"validate"`
}

type Config struct {
	Variables map[string]Variable `yaml:"variables"`
	Templates []*FileTemplate     `yaml:"templates"`
}

func (v Variable) Validators() []func(string) error {
	var validators []func(string) error
	specs := strings.Split(v.Validate, ",")
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		switch spec {
		case "required":
			validators = append(validators, func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errors.New("this field is required")
				}
				return nil
			})
		case "number":
			validators = append(validators, func(s string) error {
				if s == "" {
					return nil
				}
				_, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return errors.New("must be a valid number")
				}
				return nil
			})
		case "date":
			validators = append(validators, func(s string) error {
				if s == "" {
					return nil
				}
				layouts := []string{"2006-01-02", "01/02/2006", "02/01/2006"}
				var err error
				for _, l := range layouts {
					_, err = time.Parse(l, s)
					if err == nil {
						return nil
					}
				}
				return errors.New("must be a valid date (YYYY-MM-DD)")
			})
		}
	}
	return validators
}

type Templeton struct {
	root string
	data map[string]string
}

func ToTile(word string) string {
	caser := cases.Title(language.English)

	return caser.String(word)
}

func FormatCurrency(v any) string {
	var i int64
	switch val := v.(type) {
	case string:
		// Parse as float first to handle cases like "2500.00"
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return val
		}
		i = int64(f)
	case float64:
		i = int64(val)
	case int:
		i = int64(val)
	case int64:
		i = val
	default:
		return fmt.Sprintf("%v", v)
	}
	p := message.NewPrinter(language.English)
	return p.Sprintf("$%d", i)
}

func FormatDate(layout string, v any) string {
	var t time.Time
	var err error
	switch val := v.(type) {
	case string:
		// Try a few common formats
		layouts := []string{"2006-01-02", "01/02/2006", "02/01/2006", time.RFC3339}
		for _, l := range layouts {
			t, err = time.Parse(l, val)
			if err == nil {
				break
			}
		}
		if err != nil {
			return val
		}
	case time.Time:
		t = val
	default:
		return fmt.Sprintf("%v", v)
	}
	return t.Format(layout)
}

func (ttn *Templeton) funcMap() template.FuncMap {
	return template.FuncMap{
		"ToUpper":  strings.ToUpper,
		"ToLower":  strings.ToLower,
		"ToTitle":  ToTile,
		"split":    strings.Split,
		"currency": FormatCurrency,
		"date":     FormatDate,
	}
}

func (ttn *Templeton) Process(ft *FileTemplate) error {
	funcMap := ttn.funcMap()

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

	assetDir := strings.TrimSuffix(templateToUse, ".yaml")
	if info, err := os.Stat(assetDir); err == nil && info.IsDir() {
		if err := copyAssets(assetDir, ttn.root); err != nil {
			log.Fatal(err)
		}
	}

	file, err := os.ReadFile(templateToUse)
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil || len(config.Templates) == 0 {
		// Fallback to old format
		var oldTemplates []*FileTemplate
		err = yaml.Unmarshal(file, &oldTemplates)
		if err != nil {
			log.Fatal(err)
		}
		config.Templates = oldTemplates
	}

	fts := config.Templates
	for _, ft := range fts {
		if ft.Delims == nil {
			ft.Delims = []string{"{{", "}}"}
		}
	}

	if len(*dataF) == 0 {
		var orderedKeys []string
		seenKeys := make(map[string]bool)
		for _, ft := range fts {
			contentsKeys, err := ExtractKeys(ft.Contents, ft.Delims)
			if err != nil {
				log.Fatal(err)
			}
			pathKeys, err := ExtractKeys(ft.Path, ft.Delims)
			if err != nil {
				log.Fatal(err)
			}

			allKeysForFile := append(contentsKeys, pathKeys...)
			for _, k := range allKeysForFile {
				if !seenKeys[k] {
					seenKeys[k] = true
					orderedKeys = append(orderedKeys, k)
				}
			}
		}

		if len(orderedKeys) > 0 {
			ttn.data = make(map[string]string)
			var fields []huh.Field
			for _, k := range orderedKeys {
				input := huh.NewInput().
					Title("Value for " + k).
					Key(k)

				if varMeta, ok := config.Variables[k]; ok {
					if varMeta.Description != "" {
						input.Description(varMeta.Description)
					}
					if varMeta.Default != "" {
						input.Value(&varMeta.Default)
					}
					validators := varMeta.Validators()
					if len(validators) > 0 {
						input.Validate(func(s string) error {
							for _, v := range validators {
								if err := v(s); err != nil {
									return err
								}
							}
							return nil
						})
					}
				} else {
					input.Description(k)
				}
				fields = append(fields, input)
			}

			form := huh.NewForm(huh.NewGroup(fields...).Title("Template Variables"))
			err = form.Run()
			if err != nil {
				log.Fatal(err)
			}

			for _, k := range orderedKeys {
				ttn.data[k] = form.GetString(k)
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
	ttn := &Templeton{}
	tpl, err := template.New("temp").Funcs(ttn.funcMap()).Delims(delims[0], delims[1]).Parse(tplContent)
	if err != nil {
		return nil, err
	}

	var orderedKeys []string
	seen := make(map[string]bool)
	fn := func(k string) {
		if !seen[k] {
			seen[k] = true
			orderedKeys = append(orderedKeys, k)
		}
	}

	if tpl.Tree != nil && tpl.Tree.Root != nil {
		collectKeys(tpl.Tree.Root, fn)
	}

	return orderedKeys, nil
}

func collectKeys(node parse.Node, addKey func(string)) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, next := range n.Nodes {
			collectKeys(next, addKey)
		}
	case *parse.ActionNode:
		if n == nil {
			return
		}
		collectKeys(n.Pipe, addKey)
	case *parse.PipeNode:
		if n == nil {
			return
		}
		for _, cmd := range n.Cmds {
			collectKeys(cmd, addKey)
		}
	case *parse.CommandNode:
		if n == nil {
			return
		}
		for _, arg := range n.Args {
			collectKeys(arg, addKey)
		}
	case *parse.FieldNode:
		if n == nil {
			return
		}
		if len(n.Ident) > 0 {
			addKey(n.Ident[0])
		}
	case *parse.IfNode:
		if n == nil {
			return
		}
		collectKeys(n.Pipe, addKey)
		collectKeys(n.List, addKey)
		collectKeys(n.ElseList, addKey)
	case *parse.RangeNode:
		if n == nil {
			return
		}
		collectKeys(n.Pipe, addKey)
		collectKeys(n.List, addKey)
		collectKeys(n.ElseList, addKey)
	case *parse.WithNode:
		if n == nil {
			return
		}
		collectKeys(n.Pipe, addKey)
		collectKeys(n.List, addKey)
		collectKeys(n.ElseList, addKey)
	}
}

func copyAssets(src, dst string) error {
	if err := os.MkdirAll(dst, 0770); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

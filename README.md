# 🏗️ Templeton

**Templeton** is a lightweight project scaffolding tool that generates directory structures and files from YAML-based Go templates. It supports both command-line data injection and a smart interactive mode that prompts for missing variables.

## ✨ Features

- **YAML-based Specifications**: Define your project structure in a single, readable YAML file.
- **Dynamic Templating**: Use Go's `text/template` syntax in both file paths and contents.
- **Smart Variable Extraction**: Automatically detects required keys from your templates.
- **Interactive Scaffolding**: Prompts you for missing values if they aren't provided via flags.
- **Global Templates**: Store common templates in `~/.templeton/` for easy access.

## 🚀 Quick Start

### 1. Define a Template

Create a file named `jumping.yaml`:

```yaml
- path: "src/{{.Hurdle}}Prose.txt"
  contents: |
    The quick brown fox jumped over {{.Hurdle}}
- path: "LICENSE"
  contents: |
    This is a license for the {{.Hurdle}} project.
```

### 2. Run Templeton

#### Interactive Mode (Recommended)
Simply specify the template. Templeton will find `Hurdle` and ask you for its value:

```bash
templeton --template jumping.yaml --root my-project
# ✔ Value for Hurdle: fence
```

#### Command Line Mode
Provide the data directly via the `--data` flag:

```bash
templeton --template jumping.yaml --root my-project --data Hurdle=Fence
```

## 📂 Global Templates

You can store templates in `~/.templeton/` and access them using the `--project` flag:

```bash
# Looks for ~/.templeton/go-api.yaml
templeton --project go-api --root my-new-api
```

## 🛠️ Installation & Building

### Using Bazel

```bash
bazel build //:templeton
./bazel-bin/templeton_/templeton --help
```

### Using Go

```bash
go build -o templeton .
./templeton --help
```

---

> [!TIP]
> Both path names and file contents support full Go template functionality, including functions like `ToUpper`, `ToLower`, and `ToTitle`.
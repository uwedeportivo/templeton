![Templeton Hero](./header.png)

# 🏗️ Templeton

**Templeton** is a lightweight project scaffolding tool that generates directory structures and files from YAML-based Go templates. It supports both command-line data injection and a smart interactive mode that prompts for missing variables.

## ✨ Features

- **YAML-based Specifications**: Define your project structure in a single, readable YAML file.
- **Dynamic Templating**: Use Go's `text/template` syntax in both file paths and contents.
- **Smart Variable Extraction**: Automatically detects required keys from your templates.
- **Interactive Scaffolding**: Prompts you for missing values if they aren't provided via flags.
- **Global Templates**: Store common templates in `~/.templeton/` for easy access.
- **Additional Assets**: If a directory exists next to a template with the same name, its contents are automatically copied to the project root.

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

## 🏗️ Advanced Template Format

Templeton supports a more advanced YAML format that allows you to define variable metadata, input validation, and template-wide settings.

### Enhanced `template.yaml`

```yaml
variables:
  ProjectName:
    description: "The name of the new project"
    default: "my-cool-app"
    validate: "required"
  Rent:
    description: "Monthly rent amount"
    validate: "required,number"
  StartDate:
    description: "Lease start date (YYYY-MM-DD)"
    validate: "required,date"

templates:
  - path: "src/{{.ProjectName}}/config.txt"
    contents: |
      Project: {{.ProjectName}}
      Monthly Rent: {{.Rent | currency}}
      Starts on: {{.StartDate | date "January 2, 2006"}}
```

### 🛠️ Formatters

You can use formatters in your templates using the pipe (`|`) syntax:

| Formatter | Description | Example |
| :--- | :--- | :--- |
| `currency` | Formats a number as a currency string with commas. | `{{.Price \| currency}}` → `$1,250` |
| `date` | Formats a date string using Go's date layout. Use `2nd` for ordinal days. | `{{.Day \| date "January 2nd"}}` → `August 31st` |

### 📋 Variable Types

| Type | Description |
| :--- | :--- |
| `list` | Automatically parses comma-separated input into a Typst-formatted list. |
| `ToUpper` | Converts string to uppercase. | `{{.Name \| ToUpper}}` |
| `ToLower` | Converts string to lowercase. | `{{.Name \| ToLower}}` |
| `ToTitle` | Converts string to Title Case. | `{{.Name \| ToTitle}}` |
| `split` | Splits a string into an array. | `{{range (split .List ",")}}...{{end}}` |

### ✅ Validators

Validators can be added to the `validate` field in the `variables` section (comma-separated):

| Validator | Description |
| :--- | :--- |
| `required` | Ensures the field is not empty. |
| `number` | Ensures the input is a valid number. |
| `date` | Ensures the input is a valid date (YYYY-MM-DD, MM/DD/YYYY). |

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
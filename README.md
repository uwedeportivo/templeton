# templeton

Little utility that generates a project scaffold from a template specification
in a YAML file.

Here is an example specification:

```yaml
- path: "src/{{.Hurdle}}Prose.txt"
  contents: |
    The quick brown fox jumped over {{.Hurdle}}
- path: "LICENSE"
  contents: |
    This is a license.
```

Let's say the above contents are in a file called `jumping.yaml`. We can then invoke:

```shell
templeton --root fence --data Hurdle=Fence --template jumping.yaml
```

and `templeton` will create the following directory structure:

```txt
fence
├── src
│  └── FenceProse.txt
└── LICENSE
```

The contents of `FenceProse.txt` have `{{.Hurdle}}` replaced with `Fence` as expected.

Both path names and file contents can be templated.

For a more useful example, check out the 
[Haskell project template](https://github.com/uwedeportivo/project-templates/blob/main/haskell.yaml) I use.
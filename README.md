# Gogitix (Go Git Index Checks)

Gogitix is a tool for writing git pre-commit checks for golang.  It allows you to run a sequence of commands on the changes in your git index by checking out those files to a separate workarea.

Install it with:

```
go get -u github.com/launchdarkly/gogitix
```

Run it with:

```
gogitix [<config file name>.yml]
```

The config file must be a YAML file with a syntax similar that used by CircleCI.
The config file is passed through the golang text template processor.  By default, it will run a file that looks like:

```
- parallel:
{{- if gt (len .packages) 0 }}
    - run:
        name: build
        command: go build {{ ._packages_ }}
    - run:
        name: vet
        command: go vet {{ ._packages_ }}
{{- end }}
{{- if gt (len .files) 0 }}
    - run:
        name: fmt
        command: gofmt {{ ._files_ }}
{{- end }}
{{- if gt (len .packages) 0 }}
- run:
    name: test compile
    description: Compiling and initializing tests (but not running them)
    command: |
      go test -run non-existent-test-name-!!! {{ ._packages_ }}
{{- end }}`
```

Several variables are provided by default:

```
.files - an array of files that have been updated (and still exist)
.packages - an array of packages that have been updated (and still exist)
.dirs - an array of directories that have been updated (and still exist). Paths are relative
```

For your convenience there are also versions of these that are space separated:

```
_files_
_packages_
_dirs_
```

The commands are:

  * "run" - Run a single command (if value is a string or object) or a sequence of commands (if value is a sequence)
  * "parallel" - Run a sequence of commands in parallel

If the value of "run" is an object, it may have the following keys:
  * "name" - a name of the job to use as the prefix for output
  * "description" - a text description of the job
  * "command" - a BASH shell command to run.  It is run in the context of "/bin/bash -e".

There is also a special interactive command called "reformat".  Reformat takes two keys:
  * "check" - a single (non-sequence) command used to check (typically `gofmt -l` or `goimports -l`).
  * "reformat" - a single (non-sequence) command used to format files (typically `gofmt -l` or `goimports -l`).

`reformat` will update the files in the workarea and copy them back to your git directory.  It will abort this operation if you have local changes that differ from what is in the git index.


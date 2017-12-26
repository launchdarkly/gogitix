# Gogitix (Go Git Index Checks)

[![Build Status](https://travis-ci.org/launchdarkly/gogitix.svg?branch=master)](https://travis-ci.org/launchdarkly/gogitix)

**Stable release:** [gopkg.in/launchdarkly/gogitix.v1](http://gopkg.in/launchdarkly/gogitix.v1)

Gogitix is a tool for writing git pre-commit checks for golang.  It allows you to run a sequence of commands on the changes in your git index by checking out those files to a separate workarea.

If `-lndir` is specified, gogitix will use [`go-lndir`](https://github.com/launchdarkly/go-lndir) or `lndir` to create a create a git workspace populated only by links.

![gogitix in action](gogitix.gif?raw=true    "gogitix in action")

## Installation

Install it with:

```
go get -u http://github.com/launchdarkly/gogitix/cmd/...
```

Run it on your git index with:

```
gogitix [<config file name>.yml]
```

## Configuration

The config file must be a YAML file with a syntax similar that used by CircleCI.
The config file is passed through the golang text template processor.  By default, it will run a file that looks like:

```
- parallel:
{{ if gt (len .packages) 0 }}
    - run:
        name: build
        command: go build {{ ._packages_ }}
    - run:
        name: vet
        command: go vet {{ ._packages_ }}
{{ end }}
{{ if gt (len .files) 0 }}
    - run:
        name: fmt
        command: gofmt {{ ._files_ }}
{{ end }}
{{ if gt (len .packages) 0 }}
- run:
    name: test compile
    description: Compiling and initializing tests (but not running them)
    command: |
      go test -run non-existent-test-name-!!! {{ ._packages_ }}
{{ end }}
```

Several variables are provided by default:

```
.files - an array of files that have been updated (and still exist). Sorted alphabetically.
.packages - an array of packages that have been updated (and still exist).  e.g. "github.com/launchdarkly/gogitix"
.dirs - an array of directories that have been updated (and still exist). Paths are relative. Sorted alphabetically.
.trees - an array of subtrees that have been updated (and still exist). Paths are relative. Sorted alphabetically.
.root - root directory for your git repository in the temporary workarea
.gitRoot - root directory of your go source
.workRoot -- root directory of the temporary workarea
```

For your convenience there are also versions of the arrays that are space separated:

```
_files_
_packages_
_dirs_
_trees_
```

The commands are:

  * "run" - Run a single command (if value is a string or object) or a sequence of commands (if value is a sequence)
  * "parallel" - Run a sequence of commands in parallel

If the value of "run" is an object, it may have the following keys:
  * "name" - a name of the job to use as the prefix for output
  * "description" - a text description of the job
  * "command" - a BASH shell command to run.  It is run in the context of `/bin/bash -e`.

There is also a special interactive command called "reformat".  Reformat takes two keys:
  * "check" - a single (non-sequence) command used to check (typically `gofmt -l` or `goimports -l`).
  * "reformat" - a single (non-sequence) command used to format files (typically `gofmt -l` or `goimports -l`).

`reformat` will update the files in the workarea and copy them back to your git directory.  It will abort this operation if you have local changes that differ from what is in the git index.


## Setting up your pre-commit hook

```
#!/usr/bin/env bash

# Skip this if no non-vendor go files have changed
changed_files=$(git diff --cached --name-only --diff-filter=ACDMR -- '*.go')
[ -n "$changed_files" ] || exit 0

# Include this if you want to use "reformat" 
# exec < /dev/tty

exec gogitix gogitix.yml
``` 

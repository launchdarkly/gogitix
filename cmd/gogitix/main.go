package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"

	"gopkg.in/launchdarkly/gogitix.v2/lib"
)

var debug = false
var dryRun = false
var staging = false

var defaultFlow = `
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
{{ end }}`

var DefaultPathSpec = []string{"*.go", ":(exclude)vendor/"}

var pathSpec FlagSlice

var revisionRangeRegexp = regexp.MustCompile(`\^[@!-]`)

func main() {
	var configFilePath string
	flag.BoolVar(&debug, "d", false, "debug")
	flag.BoolVar(&dryRun, "n", false, "dry run")
	flag.BoolVar(&staging, "s", false, "run changes on staging area")
	flag.StringVar(&configFilePath, "c", "", "config file path")
	useLndir := flag.Bool("lndir", false, "Use go-lndir or lndir if available")
	flag.Var(&pathSpec, "path-spec", fmt.Sprintf("git path spec (default: %v)", DefaultPathSpec))

	if len(pathSpec) == 0 {
		pathSpec = make([]string, len(DefaultPathSpec))
		copy(pathSpec, DefaultPathSpec)
	}

	flag.Parse()

	var gitRevSpec string
	if len(flag.Args()) > 0 {
		gitRevSpec = flag.Arg(0)

		// Convert a single sha into a range with just that sha
		if gitRevSpec != "" && !strings.Contains(gitRevSpec, "..") && !revisionRangeRegexp.MatchString(gitRevSpec) {
			gitRevSpec = gitRevSpec + "^!"
		}
	}

	lib.SetDebug(debug)

	gitRoot := strings.TrimSpace(lib.MustRunCmd("git", "rev-parse", "--show-toplevel"))

	ws, wsErr := lib.Start(gitRoot, pathSpec, *useLndir, gitRevSpec, staging)
	if wsErr != nil {
		lib.Failf(wsErr.Error())
	}

	defer ws.Close()

	configFileRaw := []byte(defaultFlow)
	if configFilePath != "" {
		var err error
		if err != nil {
			lib.Failf(`Unable to read config file "%s": %s`, flag.Arg(0), err.Error())
		}
	}

	templateData := map[string]interface{}{
		"files":      ws.UpdatedFiles,
		"_files_":    strings.Join(ws.UpdatedFiles, " "),
		"dirs":       ws.UpdatedDirs,
		"_dirs_":     strings.Join(ws.UpdatedDirs, " "),
		"trees":      ws.UpdatedTrees,
		"_trees_":    strings.Join(ws.UpdatedTrees, " "),
		"topDirs":    ws.UpdatedTrees, // Old names for trees
		"_topDirs_":  strings.Join(ws.UpdatedTrees, " "),
		"packages":   ws.UpdatedPackages,
		"_packages_": strings.Join(ws.UpdatedPackages, " "),
		"gitRoot":    gitRoot,
		"workRoot":   ws.WorkDir,
		"root":       ws.RootDir,
	}

	if debug {
		data, _ := json.MarshalIndent(templateData, "", "  ")
		fmt.Printf("Template data: %s\n", data)
	}

	var configFile bytes.Buffer
	template.Must(template.New("config").Parse(string(configFileRaw))).Execute(&configFile, templateData)

	var checks interface{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic while parsing config file:\n=======\n%s\n=======\n", configFile.Bytes())
				panic(r)
			}
		}()
		if err := yaml.Unmarshal(configFile.Bytes(), &checks); err != nil {
			lib.Failf(fmt.Sprintf("Unable to parse config file:\n=======\n%s\n=======\n", configFile.Bytes()))
		}
	}()

	parsedCheck, parseError := lib.NewParser().Parse(checks, "")
	if parseError != nil {
		lib.Failf("Unable to parse config file: %s", parseError.Error())
	}

	color.Yellow("Running checks...")

	// Don't do reformat unless we're just checking the index
	skipReformat := gitRevSpec != ""

	errResult := make(chan error)
	go lib.RunCheck(ws, lib.CommandExecutor{DryRun: dryRun}, parsedCheck, skipReformat, errResult)

	for {
		if err, ok := <-errResult; !ok {
			return
		} else if err != nil {
			lib.Failf(err.Error())
		}
	}
}

type FlagSlice []string

func (p *FlagSlice) String() string {
	return strings.Join(pathSpec, " ")
}

func (p *FlagSlice) Set(s string) error {
	*p = append(*p, s)
	return nil
}

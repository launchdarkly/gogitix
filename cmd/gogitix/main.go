package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"

	"github.com/launchdarkly/gogitix/lib"
)

var debug = false
var dryRun = false

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

func main() {
	flag.BoolVar(&debug, "d", false, "debug")
	flag.BoolVar(&dryRun, "n", false, "dry run")
	flag.Parse()

	lib.SetDebug(debug)

	gitRoot, err := os.Getwd()
	if err != nil {
		lib.Failf(err.Error())
	}

	ws, wsErr := lib.Start(gitRoot)
	if wsErr != nil {
		lib.Failf(err.Error())
	}

	defer ws.Close()

	configFileRaw := []byte(defaultFlow)
	if len(flag.Args()) > 0 {
		configFilePath := flag.Arg(0)
		configFileRaw, err = ioutil.ReadFile(configFilePath)
		if err != nil {
			lib.Failf(`Unable to read config file "%s": %s`, flag.Arg(0), err.Error())
		}
	}

	templateData := map[string]interface{}{
		"files":      ws.UpdatedFiles,
		"_files_":    strings.Join(ws.UpdatedFiles, " "),
		"dirs":       ws.UpdatedDirs,
		"_dirs_":     strings.Join(ws.UpdatedDirs, " "),
		"topDirs":    ws.TopUpdatedDirs,
		"_topDirs_":  strings.Join(ws.TopUpdatedDirs, " "),
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

	errResult := make(chan error)
	go lib.RunCheck(ws, lib.CommandExecutor{DryRun: dryRun}, parsedCheck, errResult)

	for {
		if err, ok := <-errResult; !ok {
			return
		} else if err != nil {
			lib.Failf(err.Error())
		}
	}
}

package lib

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

type Parser struct {
	nextNumberForName map[string]int
}

func NewParser() Parser {
	return Parser{
		nextNumberForName: map[string]int{},
	}
}

func (p Parser) Parse(check interface{}, path string) (Check, error) {
	switch check := check.(type) {
	case map[interface{}]interface{}: // Object
		if reformat, isReformat := check["reformat"].(map[interface{}]interface{}); isReformat {
			var reformatCheck, reformatCommand SingleCheck
			var ok bool

			if reformatCheckRaw, err := p.Parse(reformat["check"], path+"/check"); err != nil {
				return nil, fmt.Errorf("could not parse reformat 'check' at %s: %s", orRoot(path), err)
			} else if reformatCheck, ok = reformatCheckRaw.(SingleCheck); !ok {
				return nil, fmt.Errorf("expected simple command for reformat 'check' at %s", orRoot(path))
			}

			if reformatCommandRaw, err := p.Parse(reformat["format"], path+"/format"); err != nil {
				return nil, fmt.Errorf("could not parse reformat 'format' at %s: %s", orRoot(path), err)
			} else if reformatCommand, ok = reformatCommandRaw.(SingleCheck); !ok {
				return nil, fmt.Errorf("expected simple command for reformat 'format' at %s", orRoot(path))
			}

			return ReformatCheck{
				Check:  reformatCheck,
				Format: reformatCommand,
			}, nil
		} else if check["reformat"] != nil {
			return nil, fmt.Errorf("reformat must be key for an object at %s", orRoot(path))
		}

		if check["parallel"] != nil && len(check) > 1 {
			return nil, fmt.Errorf("'parallel' must be the only key at %s", orRoot(path))
		}

		if check["run"] != nil && len(check) > 1 {
			return nil, fmt.Errorf("'run' must be the only key at %s", orRoot(path))
		}

		switch checkParallel, found := check["parallel"]; checkParallel := checkParallel.(type) {
		case nil: // ignore
			if found {
				return p.parseCheckArray([]interface{}{}, path+"/parallel", true)
			}
		case []interface{}:
			return p.parseCheckArray(checkParallel, path+"/parallel", true)
		default:
			return nil, fmt.Errorf("value for key 'parallel' must be an array")
		}

		var cmd Command
		switch checkRun := check["run"].(type) {
		case nil: // ignore
			// Remarshal the check into a command object
			if data, err := yaml.Marshal(check); err != nil {
				return nil, fmt.Errorf("unable to parse command at %s: %s", orRoot(path), err)
			} else {
				yaml.Unmarshal(data, &cmd)
			}
		case string:
			cmd.Command = checkRun
		case []interface{}:
			return p.parseCheckArray(checkRun, path, false)
		case map[interface{}]interface{}:
			return p.Parse(checkRun, path+"/run")
		default:
			return nil, fmt.Errorf("unexpected type for 'run' at %s: %v", orRoot(path), checkRun)
		}

		cmd.Name = p.makeNumberedName(cmd.Name, cmd.Command)
		cmd.Path = path

		return SingleCheck{
			Command: cmd,
		}, nil

	case string: // String
		cmd := Command{
			Command: check,
			Name:    p.makeNumberedName("", check),
		}
		return SingleCheck{
			Command: cmd,
		}, nil

	case []interface{}: // Array
		childChecks := make([]Check, len(check))
		for i, c := range check {
			var err error
			if childChecks[i], err = p.Parse(c, path+fmt.Sprintf("/%d", i+1)); err != nil {
				return nil, err
			}
		}
		return ManyChecks{
			Checks: childChecks,
		}, nil
	default:
		return nil, fmt.Errorf("unexpected type: %v at %s", check, orRoot(path))
	}
}

func (p Parser) makeNumberedName(name string, cmd string) string {
	if name == "" {
		if strings.TrimSpace(cmd) == "" {
			name = "<empty command>"
		} else {
			name = strings.Fields(cmd)[0]
		}
	}
	if number, found := p.nextNumberForName[name]; found {
		p.nextNumberForName[name] = number + 1
		return fmt.Sprintf("%s:%d", name, number)
	} else {
		p.nextNumberForName[name] = 2
		return name
	}
}

func (p Parser) parseCheckArray(checkArray []interface{}, path string, parallel bool) (Check, error) {
	if childChecksIf, err := p.Parse(checkArray, path); err != nil {
		return nil, err
	} else {
		childChecks := childChecksIf.(ManyChecks)
		childChecks.Parallel = parallel
		return childChecks, nil
	}
}

func orRoot(str string) string {
	if str == "" {
		return "/"
	} else {
		return str
	}
}

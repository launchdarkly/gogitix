package lib

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
)

func parse(t *testing.T, str string) (Check, error) {
	var check interface{}
	if !assert.NoError(t, yaml.Unmarshal([]byte(str), &check)) {
		t.FailNow()
	}
	return NewParser().Parse(check, "")
}

func TestParseCheck(t *testing.T) {
	specs := []struct {
		data          string
		expectedCheck Check
		expectedErr   string
	}{
		{"ls", SingleCheck{Command: Command{Name: "ls", Command: "ls"}}, ""},
		{"ls -a", SingleCheck{Command: Command{Name: "ls", Command: "ls -a"}}, ""},
		{"run: ls", SingleCheck{Command: Command{Name: "ls", Command: "ls"}}, ""},
		{"command: ls", SingleCheck{Command: Command{Name: "ls", Command: "ls"}}, ""},
		{"{command: ls, run: ls}", nil, "'run' must be the only key at /"},
		{"run: {command: ls}", SingleCheck{Command: Command{Name: "ls", Command: "ls"}}, ""},
		{`run: {command: ls, name: "list files"}`, SingleCheck{Command: Command{Name: "list files", Command: "ls"}}, ""},
		{"run: [ls, ls]", ManyChecks{
			Checks: []Check{
				SingleCheck{Command: Command{Name: "ls", Command: "ls"}},
				SingleCheck{Command: Command{Name: "ls:2", Command: "ls"}},
			},
			Parallel: false,
		}, ""},
		{"run: []", ManyChecks{Checks: []Check{}, Parallel: false}, ""},
		{"run:", SingleCheck{Command: Command{Name: "<empty command>"}}, ""},
		{"parallel:", ManyChecks{Checks: []Check{}, Parallel: true}, ""},
		{`parallel: [a, b]`, ManyChecks{
			Checks: []Check{
				SingleCheck{Command: Command{Name: "a", Command: "a"}},
				SingleCheck{Command: Command{Name: "b", Command: "b"}},
			},
			Parallel: true,
		}, ""},
	}

	for i, spec := range specs {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			check, err := parse(t, spec.data)
			if spec.expectedErr != "" {
				assert.EqualError(t, err, spec.expectedErr)
			} else {
				assert.Equal(t, spec.expectedCheck, check)
			}
		})
	}

}

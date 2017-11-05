package lib

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
)

var debug = false

func SetDebug(d bool) {
	debug = d
}

func MustRunCmd(name string, args ...string) string {
	if output, err := RunCmd(name, args...); err != nil {
		cmd := strings.Join(append([]string{name}, args...), " ")
		Failf(fmt.Sprintf("Command failed: %s\nError: %s\nOutput: %s\n", cmd, err, output))
		return ""
	} else {
		return output
	}
}

func Failf(msg string, args ...interface{}) {
	color.Red(msg, args...)
	os.Exit(1)
}

func MustRunTestCmd(msg string, name string, args ...string) {
	if msg != "" {
		color.Yellow("%s... ", msg)
	}
	start := time.Now()
	cmdDetails := strings.Join(append([]string{name}, args...), " ")
	color.Cyan(cmdDetails)
	cmd := exec.Command(name, args...) // #nosec
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	if err != nil {
		color.Red("Command: %s,\nError: %s\nOutput:\n%s\nFAIL (%s)", cmdDetails, err, output, duration)
		os.Exit(1)
	} else {
		color.Green("PASS (%s)", duration)
	}
}

func MustRunInteractiveCmd(name string, args ...string) {
	cmd := exec.Command(name, args...) // #nosec
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if debug {
		color.Magenta("[DEBUG] running '%s'", strings.Join(append([]string{name}, args...), " "))
	}
	if err := cmd.Run(); err != nil {
		cmdName := strings.Join(append([]string{name}, args...), " ")
		Failf(fmt.Sprintf("Command failed: %s\nError: %s\n\n", cmdName, err))
	}
}

func RunCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...) // #nosec
	if debug {
		color.Magenta("[DEBUG] running '%s'", strings.Join(append([]string{name}, args...), " "))
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

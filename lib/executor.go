package lib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Executor interface {
	Execute(ws Workspace, cmd Command) error
	ExecuteWithOutput(ws Workspace, cmd Command) ([]byte, error)
}

type CommandExecutor struct {
	DryRun bool
}

var CmdColors = []color.Attribute{
	color.FgRed,
	color.FgGreen,
	color.FgYellow,
	color.FgMagenta,
	color.FgCyan,
}

type CmdStatus int

const (
	PASS CmdStatus = iota
	FAIL
	INFO
)

var StatusToColor = map[CmdStatus]color.Attribute{
	PASS: color.FgGreen,
	FAIL: color.FgRed,
	INFO: color.FgCyan,
}

// Run Command returning output and error status
func (executor CommandExecutor) Execute(ws Workspace, cmd Command) error {
	_, err := executor.ExecuteWithOutput(ws, cmd)
	return err
}

func (executor CommandExecutor) ExecuteWithOutput(ws Workspace, cmd Command) ([]byte, error) {
	color := checkoutColor()
	defer releaseColor(color)

	output := []byte{}
	file, err := ioutil.TempFile("", cmd.Name)
	if err != nil {
		return output, err
	}

	file.Write([]byte("set -e\n"))
	file.Write([]byte(cmd.Command))
	file.Close()
	defer os.Remove(file.Name())

	start := time.Now()
	shellCmd := exec.Command("/bin/bash", file.Name()) /* #nosec */

	msg := "Running"
	if cmd.Description != "" {
		msg += fmt.Sprintf(" '%s'", cmd.Description)
	}
	msg += ":"
	if strings.Contains(strings.TrimSpace(cmd.Command), "\n") {
		msg += "\n"
	} else {
		msg += " "
	}
	if strings.TrimSpace(cmd.Command) == "" {
		msg += "<empty command>"
	} else {
		msg += cmd.Command
	}
	PrintCmdLine(INFO, cmd.Name, color, "%s", msg)

	if executor.DryRun {
		if contents, err := ioutil.ReadFile(file.Name()); err != nil {
			return output, err
		} else {
			PrintCmdLine(INFO, cmd.Name, color, "Would have run:\n=========\n%s\n========", contents)
		}
	} else {
		output, err = shellCmd.CombinedOutput()
		duration := time.Since(start)
		if err == nil && cmd.ExpectSilence && strings.TrimSpace(string(output)) != "" {
			err = errors.New("expected no output but output was present")
		}
		if err != nil {
			PrintCmdLine(FAIL, cmd.Name, color, "Command:\n%s\nError: %s\nOutput:\n%s\nFAIL (%0.3fs)", cmd.Command, err, output, seconds(duration))
			os.Exit(1)
		} else {
			PrintCmdLine(PASS, cmd.Name, color, "PASS (%0.3fs)", seconds(duration))
		}
	}
	return output, nil
}

func PrintCmdLine(status CmdStatus, name string, colorNum int, format string, args ...interface{}) {
	//statusColor := color.New(StatusToColor[status])
	cmdColor := color.New(CmdColors[colorNum])
	output := fmt.Sprintf(format, args...)
	for _, line := range strings.Split(output, "\n") {
		fmt.Print(cmdColor.Sprint("| " + name + " | " + line + "\n"))
	}
}

var colorLock sync.Mutex
var colorCounts map[int]int
var colorQueue []int

func init() {
	colorCounts = map[int]int{}
	colorQueue = make([]int, len(CmdColors))
	for i := 0; i < len(CmdColors); i++ {
		colorQueue[i] = i
	}
}

func checkoutColor() int {
	colorLock.Lock()
	defer colorLock.Unlock()

	// Find the first unused color in the queue, otherwise just pick the next color in queue
	var choiceIndex int
	for i, c := range colorQueue {
		if colorCounts[c] == 0 {
			choiceIndex = i
			break
		}
	}

	color := colorQueue[choiceIndex]

	// Put the new color at the back of the queue
	if choiceIndex < len(colorQueue)-1 {
		colorQueue = append(colorQueue[0:choiceIndex], append(colorQueue[choiceIndex+1:], colorQueue[choiceIndex])...)
	}
	colorCounts[color] += 1
	return color
}

func releaseColor(color int) {
	colorLock.Lock()
	defer colorLock.Unlock()
	colorCounts[color] -= 1
}

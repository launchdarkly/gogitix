package lib

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"gopkg.in/launchdarkly/gogitix.v2/lib/utils"
)

func Reformat(ws Workspace, executor Executor, check ReformatCheck, skipReformat bool) error {
	if len(ws.UpdatedFiles) > 0 {
		checkCommand := check.Check.Command
		if checkCommand.Description != "" {
			checkCommand.Description = fmt.Sprintf("Checking formatting ... (%d file(s) changed)", len(ws.UpdatedFiles))
		}
		output, err := executor.ExecuteWithOutput(ws, checkCommand)
		if err != nil {
			return fmt.Errorf("reformat check command failed: %s", err)
		}
		needsFormatting := string(output)
		if needsFormatting != "" {
			files := strings.Fields(needsFormatting)
			filesToUpdate := []string{}
			filesWithUnstagedChanges := utils.StrMap(ws.LocallyChangedFiles)
			for _, file := range files {
				if filesWithUnstagedChanges[file] {
					color.Red("Did not automatically reformat '%s' because it has un-staged changes.", file)
				} else {
					filesToUpdate = append(filesToUpdate, file)
				}
			}

			if len(filesToUpdate) > 0 && !skipReformat {
				color.White("The following files need formatting:\n" + needsFormatting)
				color.White("Automatically reformatting files.  Press <Enter> to review changes. Hit Ctrl-C at any point to abort commit.")

				var s string
				fmt.Scanln(&s)

				reformatCommand := check.Format.Command
				if reformatCommand.Description != "" {
					checkCommand.Description = fmt.Sprintf("Reformatting")
				}

				// Reformat and copy files from working dir to git work tree and then stage them
				if err := executor.Execute(ws, reformatCommand); err != nil {
					return fmt.Errorf("reformat check command failed: %s", err)
				}
				MustRunCmd("rsync", append([]string{"-R"}, append(filesToUpdate, ws.GitDir)...)...)

				MustRunInteractiveCmd("git", append([]string{"-C", ws.GitDir, "diff", "--"}, filesToUpdate...)...)
				MustRunInteractiveCmd("git", append([]string{"-C", ws.GitDir, "add", "--"}, filesToUpdate...)...)

				MustRunCmd("rsync", append([]string{"-R"}, append(filesToUpdate, ws.GitDir)...)...)

				output, err := executor.ExecuteWithOutput(ws, checkCommand)
				if err != nil {
					return fmt.Errorf("reformat check command failed: %s", err)
				}
				needsFormatting = string(output)
			}

			if needsFormatting != "" {
				Failf("The following files still need reformatting:\n" + needsFormatting + "\n")
			}
		} else {
			color.Green("No files need reformatting!")
		}
	}
	return nil
}

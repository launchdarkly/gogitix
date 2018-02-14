package lib

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/launchdarkly/gogitix/lib/utils"
)

type Workspace struct {
	GitDir              string   // Original git directory
	WorkDir             string   // Base of the temporary directory created with git index
	RootDir             string   // Base directory of the top-level go package in the git index
	UpdatedDirs         []string // Directories that have changed and still exist (sorted)
	UpdatedTrees        []string // Top directories that have changed and still exist (sorted)
	UpdatedFiles        []string // Files that have changed and still exist
	UpdatedPackages     []string // Packages that have changed and still exist
	LocallyChangedFiles []string // Files where the git index differs from what's in the working tree
}

func Start(gitRoot string, pathSpec []string, useLndir bool) (Workspace, error) {
	workDir, err := ioutil.TempDir("", path.Base(os.Args[0]))
	if err != nil {
		return Workspace{}, err
	}

	workDir, _ = filepath.EvalSymlinks(workDir)

	if err := os.Setenv("GOPATH", strings.Join([]string{workDir, os.Getenv("GOPATH")}, ":")); err != nil {
		return Workspace{}, err
	}

	yellow := color.New(color.FgYellow)
	yellow.Printf("Identifying changed files.")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer func() {
		ticker.Stop()
		yellow.Printf("\n")
	}()

	go func() {
		for {
			_, ok := <-ticker.C
			if !ok {
				break
			}
			yellow.Printf(".")
		}
	}()

	updatedFilesChan := make(chan []string, 1)
	locallyChangedFilesChan := make(chan []string, 1)
	updatedDirsChan := make(chan []string, 1)

	go func() {
		updatedFilesChan <- getUpdatedFiles(gitRoot, pathSpec)
	}()

	go func() {
		locallyChangedFilesChan <- getLocallyChangedFiles(gitRoot, pathSpec)
	}()

	go func() {
		updatedDirsChan <- getUpdatedDirs(gitRoot, pathSpec)
	}()

	rootPackage := strings.TrimSpace(MustRunCmd("sh", "-c", fmt.Sprintf("cd %s && go list -e .", gitRoot)))
	rootDir := path.Join(workDir, "src", rootPackage)

	// Try to create a shadow copy instead of checking out all the files
	lndir := ""
	lndirArgs := []string{"-silent"}
	if useLndir {
		if _, err := RunCmd("which", "go-lndir"); err == nil {
			lndir = "go-lndir"
			lndirArgs = append(lndirArgs, "-gitignore")
		} else if _, err := RunCmd("which", "lndir"); err == nil {
			lndir = "lndir"
		} else {
			Failf("Unable to find go-lndir or lndir")
		}
	}

	if lndir != "" {
		absGitRoot, err := filepath.Abs(gitRoot)
		if err != nil {
			return Workspace{}, err
		}
		if err := os.MkdirAll(rootDir, os.ModePerm); err != nil {
			return Workspace{}, err
		}
		// Start with a copy of the current workspace
		MustRunCmd(lndir, append(lndirArgs, absGitRoot, rootDir)...)
		// Copy out any files that have local changes from the index
		cmd := fmt.Sprintf("git diff --name-only | git checkout-index --stdin -f --prefix %s/", rootDir)
		MustRunCmd("sh", "-c", cmd)
		// Finally, copy out the files we want to test
		MustRunCmd("git", "-C", gitRoot, "checkout-index", "-f", "--prefix", rootDir+"/")
	} else {
		MustRunCmd("git", "-C", gitRoot, "checkout-index", "-a", "--prefix", rootDir+"/")
	}

	if err := os.Chdir(rootDir); err != nil {
		return Workspace{}, err
	}

	updatedDirs := <-updatedDirsChan
	updatedPackages := getUpdatedPackages(rootPackage, updatedDirs)

	updatedFiles := <-updatedFilesChan
	locallyChangedFiles := <-locallyChangedFilesChan

	return Workspace{
		GitDir:              gitRoot,
		WorkDir:             workDir,
		RootDir:             rootDir,
		UpdatedFiles:        utils.SortStrings(updatedFiles),
		UpdatedDirs:         utils.SortStrings(updatedDirs),
		UpdatedPackages:     utils.SortStrings(updatedPackages),
		UpdatedTrees:        utils.SortStrings(utils.ShortestPrefixes(updatedDirs)),
		LocallyChangedFiles: utils.SortStrings(locallyChangedFiles),
	}, nil
}
func getLocallyChangedFiles(gitRoot string, pathSpec []string) []string {
	return strings.Fields(MustRunCmd("git", append([]string{"-C", gitRoot, "diff", "--name-only", "--diff-filter=ACMR", "--"}, pathSpec...)...))
}

func getUpdatedFiles(gitRoot string, pathSpec []string) []string {
	return strings.Fields(MustRunCmd("git", append([]string{"-C", gitRoot, "diff", "--cached", "--name-only", "--diff-filter=ACMR", "--"}, pathSpec...)...))
}

func (ws Workspace) Close() error {
	return os.RemoveAll(ws.WorkDir)
}

// Must be run in rootDir
func getUpdatedPackages(rootPackage string, updatedDirs []string) []string {
	packages := strings.Fields(MustRunCmd("go", "list", "./..."))
	updatedPackages := map[string]bool{}

	updatedDirMap := utils.StrMap(updatedDirs)

	for _, p := range packages {
		dirName := strings.TrimPrefix(p, rootPackage+"/")
		if dirName == rootPackage {
			dirName = "."
		}
		if updatedDirMap[dirName] {
			updatedPackages[p] = true
		}
	}

	return utils.StrKeys(updatedPackages)
}

func getUpdatedDirs(gitRoot string, pathSpec []string) []string {
	fileStatus := MustRunCmd("git", append([]string{"-C", gitRoot, "diff", "--cached", "--name-status", "--diff-filter=ACDMR", "--"}, pathSpec...)...)
	scanner := bufio.NewScanner(strings.NewReader(fileStatus))
	var allFiles []string
	for scanner.Scan() {
		allFiles = append(allFiles, strings.Fields(scanner.Text())[1:]...)
	}
	updatedDirs := map[string]bool{}
	for _, f := range allFiles {
		updatedDirs[filepath.Dir(f)] = true
	}
	// Keep only the directories that still exist
	existingDirs := []string{}
	for d := range updatedDirs {
		if _, err := os.Stat(d); err == nil {
			existingDirs = append(existingDirs, d)
		}
	}

	return existingDirs
}

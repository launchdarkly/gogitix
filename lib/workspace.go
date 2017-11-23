package lib

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"

	"io/ioutil"

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

func Start(gitRoot string, pathSpec []string) (Workspace, error) {
	workDir, err := ioutil.TempDir("", os.Args[0])
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

	rootPackage := strings.TrimSpace(MustRunCmd("go", "list", "-e", "."))
	rootDir := path.Join(workDir, "src", rootPackage)

	go func() {
		updatedDirsChan <- getUpdatedDirs(gitRoot, pathSpec)
	}()

	MustRunCmd("git", "-C", gitRoot, "checkout-index", "-a", "--prefix", rootDir+"/")

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

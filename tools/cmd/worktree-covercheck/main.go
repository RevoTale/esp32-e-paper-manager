package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dr-dobermann/covercheck"
)

type changedLine = covercheck.ChangedLine
type parsedLines map[string][]changedLine
type changedLines map[string]map[int]changedLine

type gitRunner interface {
	output(directory string, arguments ...string) ([]byte, error)
}

type execGit struct{}

func (execGit) output(directory string, arguments ...string) ([]byte, error) {
	command := exec.Command("git", arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(arguments, " "), err, output)
	}
	return output, nil
}

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}

func execute(arguments []string, output, errorOutput io.Writer) int {
	flags := flag.NewFlagSet("worktree-covercheck", flag.ContinueOnError)
	flags.SetOutput(errorOutput)
	root := flags.String("root", ".", "module root inside the Git worktree")
	base := flags.String("base", "origin/main", "Git base ref")
	profiles := flags.String("profiles", "coverage.out", "comma-separated coverage profiles")
	minimum := flags.Float64("min", 90, "minimum changed executable-line coverage")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	code, err := run(*root, *base, *profiles, *minimum, output)
	if err != nil {
		if _, writeErr := fmt.Fprintln(errorOutput, "worktree-covercheck:", err); writeErr != nil {
			return 2
		}
		return 2
	}
	return code
}

func run(root, base, profileList string, minimum float64, output io.Writer) (int, error) {
	changed, err := collectChanged(root, base, execGit{})
	if err != nil {
		return 0, err
	}
	profiles, err := readProfiles(strings.Split(profileList, ","))
	if err != nil {
		return 0, err
	}

	result, err := evaluateCoverage(root, profiles, changed)
	if err != nil {
		return 0, err
	}
	if err := printResult(output, result); err != nil {
		return 0, err
	}
	percent := result.Ratio() * 100
	if percent+0.000001 < minimum {
		return 1, nil
	}
	return 0, nil
}

func collectChanged(root, base string, runner gitRunner) (parsedLines, error) {
	combined := changedLines{}
	prefixOutput, err := runner.output(root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSpace(string(prefixOutput))
	ancestor, err := runner.output(root, "merge-base", base, "HEAD")
	if err != nil {
		return nil, err
	}
	// One merge-base→worktree diff retains final coordinates and final files.
	// Unioning base→HEAD with HEAD→worktree resurrects deleted intermediate
	// files and charges statements to stale line numbers after insertions.
	// https://git-scm.com/docs/git-diff (comparing working tree with a commit)
	output, err := runner.output(root, "diff", "--unified=0", "--no-ext-diff", strings.TrimSpace(string(ancestor)), "--", "*.go")
	if err != nil {
		return nil, err
	}
	parsed, err := covercheck.ParseDiffLines(strings.NewReader(string(output)))
	if err != nil {
		return nil, fmt.Errorf("parse git diff: %w", err)
	}
	mergeChanged(combined, moduleRelative(parsed, prefix))
	if err := addUntracked(root, runner, combined); err != nil {
		return nil, err
	}
	return flattenChanged(combined), nil
}

func addUntracked(root string, runner gitRunner, destination changedLines) error {
	output, err := runner.output(root, "ls-files", "--others", "--exclude-standard", "--", "*.go")
	if err != nil {
		return err
	}
	for _, name := range strings.Fields(string(output)) {
		if err := addWholeFile(filepath.Join(root, name), filepath.ToSlash(name), destination); err != nil {
			return err
		}
	}
	return nil
}

func moduleRelative(source parsedLines, prefix string) parsedLines {
	if prefix == "" {
		return source
	}
	result := parsedLines{}
	for path, lines := range source {
		if strings.HasPrefix(path, prefix) {
			result[strings.TrimPrefix(path, prefix)] = lines
		}
	}
	return result
}

func addWholeFile(path, gitPath string, destination changedLines) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open untracked file %s: %w", path, err)
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		mergeChanged(destination, parsedLines{gitPath: {{Number: lineNumber, Text: scanner.Text()}}})
	}
	return errors.Join(scanner.Err(), file.Close())
}

func mergeChanged(destination changedLines, source parsedLines) {
	for path, lines := range source {
		if destination[path] == nil {
			destination[path] = map[int]changedLine{}
		}
		for _, line := range lines {
			destination[path][line.Number] = line
		}
	}
}

func flattenChanged(source changedLines) parsedLines {
	result := parsedLines{}
	for path, byNumber := range source {
		numbers := make([]int, 0, len(byNumber))
		for number := range byNumber {
			numbers = append(numbers, number)
		}
		sort.Ints(numbers)
		for _, number := range numbers {
			result[path] = append(result[path], byNumber[number])
		}
	}
	return result
}

func readProfiles(names []string) (map[string][]covercheck.Block, error) {
	result := map[string][]covercheck.Block{}
	for _, name := range names {
		file, err := os.Open(strings.TrimSpace(name))
		if err != nil {
			return nil, fmt.Errorf("open coverage profile: %w", err)
		}
		parsed, parseErr := covercheck.ParseProfiles(file)
		closeErr := file.Close()
		if err := errors.Join(parseErr, closeErr); err != nil {
			return nil, fmt.Errorf("read coverage profile: %w", err)
		}
		for path, blocks := range parsed {
			result[path] = append(result[path], blocks...)
		}
	}
	return result, nil
}

func excludeCoverageFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

func evaluateCoverage(
	root string,
	profiles map[string][]covercheck.Block,
	changed parsedLines,
) (covercheck.Result, error) {
	result := covercheck.EvaluateLines(profiles, changed, excludeCoverageFile, nil)
	for path, lines := range changed {
		if excludeCoverageFile(path) || hasProfile(profiles, path) {
			continue
		}
		statementLines, err := parseStatementLines(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return covercheck.Result{}, fmt.Errorf("inspect uncovered file %s: %w", path, err)
		}
		fileResult := result.PerFile[path]
		for _, line := range lines {
			if statementLines[line.Number] {
				fileResult.Coverable++
				result.Coverable++
			}
		}
		if fileResult.Coverable > 0 {
			result.PerFile[path] = fileResult
		}
	}
	return result, nil
}

func hasProfile(profiles map[string][]covercheck.Block, path string) bool {
	suffix := "/" + path
	for profilePath := range profiles {
		if profilePath == path || strings.HasSuffix(profilePath, suffix) {
			return true
		}
	}
	return false
}

func parseStatementLines(path string) (map[int]bool, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, contents, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	sourceLines := strings.Split(string(contents), "\n")
	result := map[int]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		statement, ok := node.(ast.Stmt)
		if !ok {
			return true
		}
		if _, block := statement.(*ast.BlockStmt); block {
			return true
		}
		start := set.Position(statement.Pos()).Line
		end := set.Position(statement.End()).Line
		for line := start; line <= end && line <= len(sourceLines); line++ {
			if isCodeLine(sourceLines[line-1]) {
				result[line] = true
			}
		}
		return true
	})
	return result, nil
}

func isCodeLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed != "" && trimmed != "{" && trimmed != "}" &&
		!strings.HasPrefix(trimmed, "//")
}

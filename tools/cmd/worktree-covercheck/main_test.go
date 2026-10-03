package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dr-dobermann/covercheck"
)

var errTest = errors.New("test failure")

type fixedGit struct {
	outputValue []byte
	err         error
}

func (runner fixedGit) output(string, ...string) ([]byte, error) {
	return runner.outputValue, runner.err
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errTest
}

func TestRunPassesAndFailsAtConfiguredThreshold(t *testing.T) {
	repo := newChangedRepo(t)
	coveredProfile := filepath.Join(repo, "covered.out")
	writeTestFile(t, repo, "covered.out", "mode: set\nexample.invalid/quality/tracked.go:3.1,3.16 1 1\n")
	var output bytes.Buffer
	code, err := run(repo, "HEAD", coveredProfile, 90, &output)
	if err != nil || code != 0 {
		t.Fatalf("covered run: code=%d err=%v output=%s", code, err, output.String())
	}

	uncoveredProfile := filepath.Join(repo, "uncovered.out")
	writeTestFile(t, repo, "uncovered.out", "mode: set\nexample.invalid/quality/tracked.go:3.1,3.16 1 0\n")
	code, err = run(repo, "HEAD", uncoveredProfile, 90, &output)
	if err != nil || code != 1 {
		t.Fatalf("uncovered run: code=%d err=%v output=%s", code, err, output.String())
	}
}

func TestExecuteRunsGateAndReportsCollectionError(t *testing.T) {
	repo := newChangedRepo(t)
	profile := filepath.Join(repo, "coverage.out")
	writeTestFile(t, repo, "coverage.out", "mode: set\nexample.invalid/quality/tracked.go:3.1,3.16 1 1\n")
	var output bytes.Buffer
	arguments := []string{"-root", repo, "-base", "HEAD", "-profiles", profile, "-min", "90"}
	if code := execute(arguments, &output, &output); code != 0 {
		t.Fatalf("execute code = %d, want 0; output=%s", code, output.String())
	}

	output.Reset()
	if code := execute([]string{"-root", filepath.Join(repo, "missing")}, &output, &output); code != 2 {
		t.Fatalf("error execute code = %d, want 2; output=%s", code, output.String())
	}
}

func TestRunReportsMissingCoverageProfile(t *testing.T) {
	repo := newChangedRepo(t)
	var output bytes.Buffer
	if _, err := run(repo, "HEAD", filepath.Join(repo, "missing.out"), 90, &output); err == nil {
		t.Fatal("run error = nil, want missing profile error")
	}
}

func TestRunReportsOutputFailure(t *testing.T) {
	repo := newChangedRepo(t)
	profile := filepath.Join(repo, "coverage.out")
	writeTestFile(t, repo, "coverage.out", "mode: set\nexample.invalid/quality/tracked.go:3.1,3.16 1 1\n")
	if _, err := run(repo, "HEAD", profile, 90, failingWriter{}); !errors.Is(err, errTest) {
		t.Fatalf("run error = %v, want output failure", err)
	}
}

func TestCollectionAndParsingErrorsAreReported(t *testing.T) {
	if _, err := collectChanged(t.TempDir(), "HEAD", fixedGit{err: errTest}); !errors.Is(err, errTest) {
		t.Fatalf("collect error = %v, want Git failure", err)
	}

	root := t.TempDir()
	if err := addUntracked(root, fixedGit{outputValue: []byte("missing.go\n")}, changedLines{}); err == nil {
		t.Fatal("add untracked error = nil, want missing file error")
	}

	writeTestFile(t, root, "invalid.go", "package invalid\nfunc {\n")
	changed := parsedLines{"invalid.go": {{Number: 2, Text: "func {"}}}
	if _, err := evaluateCoverage(root, map[string][]covercheck.Block{}, changed); err == nil {
		t.Fatal("evaluate error = nil, want parse error")
	}
}

func TestExecuteRejectsUnknownFlag(t *testing.T) {
	var output bytes.Buffer
	if code := execute([]string{"-unknown"}, &output, &output); code != 2 {
		t.Fatalf("execute code = %d, want 2; output=%s", code, output.String())
	}
}

func TestCollectChangedIncludesTrackedAndUntrackedLines(t *testing.T) {
	repo := t.TempDir()
	runGitTest(t, repo, "init")
	runGitTest(t, repo, "config", "user.email", "quality@example.invalid")
	runGitTest(t, repo, "config", "user.name", "Quality Test")
	writeTestFile(t, repo, "go.mod", "module example.invalid/quality\n\ngo 1.26.0\n")
	writeTestFile(t, repo, "tracked.go", "package quality\n\nfunc value() int { return 1 }\n")
	runGitTest(t, repo, "add", "go.mod", "tracked.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "baseline")

	writeTestFile(t, repo, "tracked.go", "package quality\n\nfunc value() int { return 2 }\n")
	writeTestFile(t, repo, "new.go", "package quality\n\nconst enabled = true\n")

	changed, err := collectChanged(repo, "HEAD", execGit{})
	if err != nil {
		t.Fatalf("collect changed lines: %v", err)
	}
	if !hasLine(changed["tracked.go"], 3) {
		t.Fatalf("tracked.go lines = %#v, want line 3", changed["tracked.go"])
	}
	for line := 1; line <= 3; line++ {
		if !hasLine(changed["new.go"], line) {
			t.Fatalf("new.go lines = %#v, want line %d", changed["new.go"], line)
		}
	}
}

func TestCollectChangedReturnsPathsRelativeToModuleRoot(t *testing.T) {
	repo := t.TempDir()
	module := filepath.Join(repo, "nested", "module")
	if err := os.MkdirAll(module, 0o700); err != nil {
		t.Fatalf("create module: %v", err)
	}
	runGitTest(t, repo, "init")
	runGitTest(t, repo, "config", "user.email", "quality@example.invalid")
	runGitTest(t, repo, "config", "user.name", "Quality Test")
	writeTestFile(t, module, "tracked.go", "package quality\n\nconst value = 1\n")
	runGitTest(t, repo, "add", "nested/module/tracked.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "baseline")
	writeTestFile(t, module, "tracked.go", "package quality\n\nconst value = 2\n")

	changed, err := collectChanged(module, "HEAD", execGit{})
	if err != nil {
		t.Fatalf("collect changed lines: %v", err)
	}
	if _, exists := changed["tracked.go"]; !exists {
		t.Fatalf("changed paths = %#v, want module-relative tracked.go", changed)
	}
	if _, exists := changed["nested/module/tracked.go"]; exists {
		t.Fatalf("changed paths = %#v, contains repository-relative path", changed)
	}
}

func TestMergeChangedDeduplicatesLineNumbers(t *testing.T) {
	destination := changedLines{"file.go": {2: {Number: 2, Text: "old"}}}
	mergeChanged(destination, parsedLines{
		"file.go": {
			{Number: 2, Text: "new"},
			{Number: 4, Text: "added"},
		},
	})

	got := flattenChanged(destination)["file.go"]
	if len(got) != 2 || got[0].Number != 2 || got[0].Text != "new" || got[1].Number != 4 {
		t.Fatalf("flattened lines = %#v", got)
	}
}

func TestEvaluateCoverageCountsStatementsMissingFromProfiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "device.go", "package device\n\nfunc run() {\n\tprintln(\"run\")\n}\n")
	changed := parsedLines{
		"device.go": {
			{Number: 1, Text: "package device"},
			{Number: 4, Text: "println(\"run\")"},
		},
	}

	result, err := evaluateCoverage(root, map[string][]covercheck.Block{}, changed)
	if err != nil {
		t.Fatalf("evaluate coverage: %v", err)
	}
	if result.Coverable != 1 || result.Covered != 0 {
		t.Fatalf("result = %#v, want one uncovered statement", result)
	}
}

func TestEvaluateCoverageDoesNotMeasureTestFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "device_test.go", "package device\n\nfunc testOnly() { println(\"test\") }\n")
	changed := parsedLines{"device_test.go": {{Number: 3, Text: "func testOnly()"}}}

	result, err := evaluateCoverage(root, map[string][]covercheck.Block{}, changed)
	if err != nil {
		t.Fatalf("evaluate coverage: %v", err)
	}
	if result.Coverable != 0 {
		t.Fatalf("result = %#v, want test file excluded", result)
	}
}

func runGitTest(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}

func writeTestFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func newChangedRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitTest(t, repo, "init")
	runGitTest(t, repo, "config", "user.email", "quality@example.invalid")
	runGitTest(t, repo, "config", "user.name", "Quality Test")
	writeTestFile(t, repo, "tracked.go", "package quality\n\nconst value = 1\n")
	runGitTest(t, repo, "add", "tracked.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "baseline")
	writeTestFile(t, repo, "tracked.go", "package quality\n\nconst value = 2\n")
	return repo
}

func hasLine(lines []changedLine, number int) bool {
	for _, line := range lines {
		if line.Number == number {
			return true
		}
	}
	return false
}

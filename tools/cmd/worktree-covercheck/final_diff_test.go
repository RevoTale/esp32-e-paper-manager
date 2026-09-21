package main

import (
	"os"
	"testing"
)

func TestCommittedAdditionDeletedInWorktreeIsNotCoverable(t *testing.T) {
	repo := newChangedRepo(t)
	runGitTest(t, repo, "tag", "coverage-base")
	writeTestFile(t, repo, "removed.go", "package quality\nfunc removed() int { return 1 }\n")
	runGitTest(t, repo, "add", "removed.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "intermediate implementation")
	if err := os.Remove(repo + "/removed.go"); err != nil {
		t.Fatal(err)
	}
	changed, err := collectChanged(repo, "coverage-base", execGit{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := changed["removed.go"]; ok {
		t.Fatal("removed intermediate file remains in changed coverage")
	}
}

func TestChangedLinesUseFinalWorktreeCoordinates(t *testing.T) {
	repo := newChangedRepo(t)
	runGitTest(t, repo, "tag", "coverage-base")
	writeTestFile(t, repo, "added.go", "package quality\n\nfunc value() int {\nreturn 1\n}\n")
	runGitTest(t, repo, "add", "added.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "intermediate lines")
	writeTestFile(t, repo, "added.go", "package quality\n// new offset\n\nfunc value() int {\nreturn 1\n}\n")
	changed, err := collectChanged(repo, "coverage-base", execGit{})
	if err != nil {
		t.Fatal(err)
	}
	lines := changed["added.go"]
	if len(lines) != 6 || lines[3].Text != "func value() int {" || lines[4].Text != "return 1" {
		t.Fatalf("stale intermediate coordinates: %#v", lines)
	}
}

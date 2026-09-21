package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/dr-dobermann/covercheck"
)

func TestAllFinalChangeSourcesRemainUncoveredWithoutProfiles(t *testing.T) {
	repo := newChangedRepo(t)
	runGitTest(t, repo, "tag", "coverage-base")
	writeTestFile(t, repo, "branch.go", "package quality\n\nfunc branch() int { return 1 }\n")
	runGitTest(t, repo, "add", "branch.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "branch implementation")
	writeTestFile(t, repo, "staged.go", "package quality\n\nfunc staged() int { return 2 }\n")
	runGitTest(t, repo, "add", "staged.go")
	writeTestFile(t, repo, "tracked.go", "package quality\n\nfunc unstaged() int { return 3 }\n")
	writeTestFile(t, repo, "untracked.go", "package quality\n\nfunc untracked() int { return 4 }\n")
	changed, err := collectChanged(repo, "coverage-base", execGit{})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"branch.go", "staged.go", "tracked.go", "untracked.go"} {
		if !hasLine(changed[path], 3) {
			t.Errorf("final executable line omitted from %s", path)
		}
	}
	result, err := evaluateCoverage(repo, map[string][]covercheck.Block{}, changed)
	if err != nil || result.Coverable != 4 || result.Covered != 0 {
		t.Fatalf("unprofiled final changes: %#v, %v", result, err)
	}
	writeTestFile(t, repo, "empty.out", "mode: set\n")
	var output bytes.Buffer
	code, err := run(repo, "coverage-base", filepath.Join(repo, "empty.out"), 90, &output)
	if err != nil || code != 1 {
		t.Fatalf("uncovered changes passed gate: code=%d err=%v output=%s", code, err, output.String())
	}
}

func TestDivergedBaseDoesNotChargeBaseOnlyChanges(t *testing.T) {
	repo := newChangedRepo(t)
	writeTestFile(t, repo, "unchanged.go", "package quality\n\nfunc unchanged() int { return 1 }\n")
	runGitTest(t, repo, "add", "unchanged.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "common implementation")
	runGitTest(t, repo, "branch", "-m", "feature")
	runGitTest(t, repo, "switch", "-c", "advanced-base")
	writeTestFile(t, repo, "unchanged.go", "package quality\n\nfunc unchanged() int { return 2 }\n")
	runGitTest(t, repo, "add", "unchanged.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "base-only implementation")
	runGitTest(t, repo, "switch", "feature")
	writeTestFile(t, repo, "feature.go", "package quality\n\nfunc feature() int { return 3 }\n")
	runGitTest(t, repo, "add", "feature.go")
	runGitTest(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "feature implementation")
	changed, err := collectChanged(repo, "advanced-base", execGit{})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := changed["unchanged.go"]; exists {
		t.Fatal("base-only changes were charged against the feature worktree")
	}
	if !hasLine(changed["feature.go"], 3) {
		t.Fatal("feature branch statement was omitted")
	}
}

type failedCollectionStage struct{ stage string }

func (f failedCollectionStage) output(_ string, arguments ...string) ([]byte, error) {
	if arguments[0] == f.stage {
		return nil, errTest
	}
	if arguments[0] == "merge-base" {
		return []byte("0123456789012345678901234567890123456789\n"), nil
	}
	return nil, nil
}

func TestEveryGitCollectionFailureIsFatal(t *testing.T) {
	for _, stage := range []string{"rev-parse", "merge-base", "diff", "ls-files"} {
		t.Run(stage, func(t *testing.T) {
			changed, err := collectChanged(t.TempDir(), "HEAD", failedCollectionStage{stage})
			if !errors.Is(err, errTest) || changed != nil {
				t.Fatalf("collection failed open: changed=%v err=%v", changed, err)
			}
		})
	}
}

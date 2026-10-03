package devicelink

import (
	"go/build"
	"testing"
)

// The manager adapter must not initialize host renderer dependencies on Pico.
// The task gate additionally checks TinyGo's complete actual dependency graph.
func TestManagerAdapterIsHostOnly(t *testing.T) {
	for _, tinygo := range []bool{false, true} {
		context := build.Default
		if tinygo {
			context.BuildTags = []string{"tinygo"}
		}
		matches, err := context.MatchFile(".", "manager.go")
		if err != nil || matches == tinygo {
			t.Fatal(tinygo, matches, err)
		}
	}
}

package main

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/dr-dobermann/covercheck"
)

func printResult(output io.Writer, result covercheck.Result) error {
	paths := make([]string, 0, len(result.PerFile))
	for path := range result.PerFile {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		file := result.PerFile[path]
		if _, err := fmt.Fprintf(output, "changed coverage %s: %d/%d\n", path, file.Covered, file.Coverable); err != nil {
			return err
		}
	}
	percent := result.Ratio() * 100
	_, err := fmt.Fprintln(output, "changed coverage total:", strconv.FormatFloat(percent, 'f', 1, 64)+"%")
	return err
}

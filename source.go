package logbench

import (
	"path/filepath"
	"runtime"
)

func getCallerLocation(skip int, cwd string) *sourceLocation {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return nil
	}

	if cwd != "" {
		if rel, err := filepath.Rel(cwd, file); err == nil {
			file = rel
		}
	}

	return &sourceLocation{
		FileName:   file,
		LineNumber: line,
	}
}

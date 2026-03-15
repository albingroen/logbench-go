package logbench

import (
	"errors"
	"math"
	"math/big"
	"os"
	"testing"
)

func TestIntegration(t *testing.T) {
	if os.Getenv("LOGBENCH_INTEGRATION") != "1" {
		t.Skip("skipping integration test; set LOGBENCH_INTEGRATION=1 to run")
	}

	projectID := os.Getenv("LOGBENCH_PROJECT_ID")
	if projectID == "" {
		t.Fatal("LOGBENCH_PROJECT_ID environment variable must be set")
	}
	logger := New(projectID)
	defer logger.Close()

	// Basic log levels
	logger.Info("Go SDK integration test - info")
	logger.Warn("Go SDK integration test - warn")
	logger.Err("Go SDK integration test - error")

	// Multiple args
	logger.Info("multiple", "args", 42, true, 3.14)

	// With metadata
	logger.InfoWith(LogOptions{Bookmark: true, Annotation: "Go integration test"}, "bookmarked log")

	// Special types
	logger.Info(
		math.NaN(),
		math.Inf(1),
		complex(1, 2),
		errors.New("test error"),
		[]byte("hello bytes"),
		big.NewInt(999999999999),
		map[string]any{"key": "value"},
	)

	// Struct
	type Point struct {
		X int
		Y int
	}
	logger.Info(Point{X: 10, Y: 20})

	// All levels with options
	logger.WarnWith(LogOptions{Annotation: "warn annotation"}, "annotated warning")
	logger.ErrWith(LogOptions{Bookmark: true}, "bookmarked error")
}

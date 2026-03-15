package logbench

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type capturedRequest struct {
	Body    logPayload
	RawBody string
	Path    string
}

func setupTestServer(t *testing.T) (*httptest.Server, *[]capturedRequest, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	var captured []capturedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var payload logPayload
		json.Unmarshal(body, &payload)

		mu.Lock()
		captured = append(captured, capturedRequest{
			Body:    payload,
			RawBody: string(body),
			Path:    r.URL.Path,
		})
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))

	t.Cleanup(server.Close)
	return server, &captured, &mu
}

func TestLogLevels(t *testing.T) {
	server, captured, mu := setupTestServer(t)

	logger := New("test-project", Options{URL: server.URL, DisableCaptureSource: true})

	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Err("error msg")
	logger.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(*captured) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(*captured))
	}

	// Goroutine ordering is not guaranteed, so check that all levels are present
	found := map[LogLevel]bool{}
	for _, req := range *captured {
		found[req.Body.Level] = true
	}
	for _, want := range []LogLevel{LevelInfo, LevelWarning, LevelError} {
		if !found[want] {
			t.Errorf("missing level %s in captured requests", want)
		}
	}
}

func TestSingleContent(t *testing.T) {
	server, captured, mu := setupTestServer(t)

	logger := New("test-project", Options{URL: server.URL, DisableCaptureSource: true})
	logger.Info("hello")
	logger.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(*captured) != 1 {
		t.Fatalf("expected 1 request, got %d", len(*captured))
	}

	// Single content should be unwrapped (not an array)
	if !strings.Contains((*captured)[0].RawBody, `"content":"hello"`) {
		t.Errorf("single content should be unwrapped string, got: %s", (*captured)[0].RawBody)
	}
}

func TestMultipleContent(t *testing.T) {
	server, captured, mu := setupTestServer(t)

	logger := New("test-project", Options{URL: server.URL, DisableCaptureSource: true})
	logger.Info("hello", 42, true)
	logger.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(*captured) != 1 {
		t.Fatalf("expected 1 request, got %d", len(*captured))
	}

	// Multiple content should be an array
	if !strings.Contains((*captured)[0].RawBody, `"content":["hello",42,true]`) {
		t.Errorf("multiple content should be array, got: %s", (*captured)[0].RawBody)
	}
}

func TestInfoWithBookmarkAndAnnotation(t *testing.T) {
	server, captured, mu := setupTestServer(t)

	logger := New("test-project", Options{URL: server.URL, DisableCaptureSource: true})
	logger.InfoWith(LogOptions{Bookmark: true, Annotation: "test note"}, "bookmarked")
	logger.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(*captured) != 1 {
		t.Fatalf("expected 1 request, got %d", len(*captured))
	}

	req := (*captured)[0]
	if req.Body.IsBookmarked == nil || !*req.Body.IsBookmarked {
		t.Error("expected isBookmarked to be true")
	}
	if req.Body.Annotation == nil || *req.Body.Annotation != "test note" {
		t.Error("expected annotation to be 'test note'")
	}
}

func TestSourceLocationCaptured(t *testing.T) {
	server, captured, mu := setupTestServer(t)

	logger := New("test-project", Options{URL: server.URL})
	logger.Info("with source")
	logger.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(*captured) != 1 {
		t.Fatalf("expected 1 request, got %d", len(*captured))
	}

	raw := (*captured)[0].RawBody
	if !strings.Contains(raw, `"fileName"`) || !strings.Contains(raw, `"lineNumber"`) {
		t.Errorf("expected source location in payload, got: %s", raw)
	}
	if !strings.Contains(raw, "logbench_test.go") {
		t.Errorf("expected filename to contain 'logbench_test.go', got: %s", raw)
	}
}

func TestSourceLocationDisabled(t *testing.T) {
	server, captured, mu := setupTestServer(t)

	logger := New("test-project", Options{URL: server.URL, DisableCaptureSource: true})
	logger.Info("no source")
	logger.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(*captured) != 1 {
		t.Fatalf("expected 1 request, got %d", len(*captured))
	}

	if strings.Contains((*captured)[0].RawBody, `"source"`) {
		t.Error("source should be omitted when disabled")
	}
}

func TestCorrectEndpointPath(t *testing.T) {
	server, captured, mu := setupTestServer(t)

	logger := New("my-project-id", Options{URL: server.URL, DisableCaptureSource: true})
	logger.Info("test")
	logger.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(*captured) != 1 {
		t.Fatalf("expected 1 request, got %d", len(*captured))
	}

	want := "/api/projects/my-project-id/logs/ingest"
	if (*captured)[0].Path != want {
		t.Errorf("expected path %s, got %s", want, (*captured)[0].Path)
	}
}

func TestNetworkErrorSwallowed(t *testing.T) {
	// Should not panic when server is unreachable
	logger := New("test-project", Options{URL: "http://127.0.0.1:1", DisableCaptureSource: true})
	logger.Info("this will fail")
	logger.Close() // should complete without panic
}

func TestCloseWaitsForInflight(t *testing.T) {
	server, captured, mu := setupTestServer(t)

	logger := New("test-project", Options{URL: server.URL, DisableCaptureSource: true})

	for i := 0; i < 10; i++ {
		logger.Info("msg", i)
	}
	logger.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(*captured) != 10 {
		t.Errorf("expected 10 requests after Close(), got %d", len(*captured))
	}
}

func TestDefaultURL(t *testing.T) {
	logger := New("test-project")
	want := "http://localhost:1447/api/projects/test-project/logs/ingest"
	if logger.endpoint != want {
		t.Errorf("expected endpoint %s, got %s", want, logger.endpoint)
	}
}

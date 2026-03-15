package logbench

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
)

const defaultURL = "http://localhost:1447"

// Logbench is a structured logging client that sends logs to a Logbench server.
type Logbench struct {
	endpoint             string
	disableCaptureSource bool
	cwd                  string
	client               *http.Client
	wg                   sync.WaitGroup
}

// New creates a new Logbench client. An optional Options value can configure the client.
func New(projectID string, opts ...Options) *Logbench {
	baseURL := defaultURL
	l := &Logbench{
		client: &http.Client{},
	}

	if len(opts) > 0 {
		o := opts[0]
		if o.URL != "" {
			baseURL = strings.TrimRight(o.URL, "/")
		}
		l.disableCaptureSource = o.DisableCaptureSource
		l.cwd = o.Cwd
	}

	l.endpoint = baseURL + "/api/projects/" + projectID + "/logs/ingest"
	return l
}

// Info sends an info-level log.
func (l *Logbench) Info(content ...any) {
	l.log(LevelInfo, nil, content)
}

// Warn sends a warning-level log.
func (l *Logbench) Warn(content ...any) {
	l.log(LevelWarning, nil, content)
}

// Err sends an error-level log.
func (l *Logbench) Err(content ...any) {
	l.log(LevelError, nil, content)
}

// InfoWith sends an info-level log with metadata options.
func (l *Logbench) InfoWith(opts LogOptions, content ...any) {
	l.log(LevelInfo, &opts, content)
}

// WarnWith sends a warning-level log with metadata options.
func (l *Logbench) WarnWith(opts LogOptions, content ...any) {
	l.log(LevelWarning, &opts, content)
}

// ErrWith sends an error-level log with metadata options.
func (l *Logbench) ErrWith(opts LogOptions, content ...any) {
	l.log(LevelError, &opts, content)
}

// Close waits for all in-flight log requests to complete and releases resources.
func (l *Logbench) Close() {
	l.wg.Wait()
	l.client.CloseIdleConnections()
}

func (l *Logbench) log(level LogLevel, opts *LogOptions, content []any) {
	// Capture source synchronously (correct call stack)
	var source *sourceLocation
	if !l.disableCaptureSource {
		source = getCallerLocation(3, l.cwd)
	}

	payload := logPayload{
		Content: prepareContent(content),
		Level:   level,
		Source:  source,
	}

	if opts != nil {
		if opts.Bookmark {
			b := true
			payload.IsBookmarked = &b
		}
		if opts.Annotation != "" {
			payload.Annotation = &opts.Annotation
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.send(body)
	}()
}

func (l *Logbench) send(body []byte) {
	resp, err := l.client.Post(l.endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

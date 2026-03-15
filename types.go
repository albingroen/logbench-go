package logbench

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LevelInfo    LogLevel = "INFO"
	LevelWarning LogLevel = "WARNING"
	LevelError   LogLevel = "ERROR"
)

// Options configures the Logbench client.
type Options struct {
	URL                  string // Server URL. Defaults to "http://localhost:1447".
	DisableCaptureSource bool   // If true, source location is not captured.
	Cwd                  string // Working directory for relative source paths.
}

// LogOptions provides per-log metadata.
type LogOptions struct {
	Bookmark   bool
	Annotation string
}

type sourceLocation struct {
	FileName   string `json:"fileName"`
	LineNumber int    `json:"lineNumber"`
}

type logPayload struct {
	Content      any             `json:"content"`
	Level        LogLevel        `json:"level"`
	Source       *sourceLocation `json:"source,omitempty"`
	IsBookmarked *bool           `json:"isBookmarked,omitempty"`
	Annotation   *string         `json:"annotation,omitempty"`
}

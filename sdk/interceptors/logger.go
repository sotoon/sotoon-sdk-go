package interceptors

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// LoggerOptions defines configuration options for the logger interceptor
type LoggerOptions struct {
	// Logger is the logger instance to use
	Logger *log.Logger

	// Logging control flags
	LogBasicInfo bool // Log method, URL, status code
	LogHeaders   bool // Log HTTP headers
	LogBody      bool // Log request/response bodies

	// MaxBodyLogSize is the maximum size of request/response body to log (in bytes). default is 1024 bytes
	MaxBodyLogSize int
	// SkipHeaders is a list of headers to exclude from logs (e.g., for security reasons)
	SkipHeaders []string
	// SkipPaths is a list of URL paths to exclude from logging
	SkipPaths []string
}

// Logger is an interceptor that logs HTTP requests and responses
type Logger struct {
	opts LoggerOptions
}

// NewLogger creates a new logger interceptor with the given options
func NewLogger(opts LoggerOptions) *Logger {
	// Set defaults if not provided
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	if opts.MaxBodyLogSize == 0 {
		opts.MaxBodyLogSize = 1024 // Default to 1KB
	}

	// Convert header names to lowercase for case-insensitive comparison
	for i, header := range opts.SkipHeaders {
		opts.SkipHeaders[i] = strings.ToLower(header)
	}

	return &Logger{opts: opts}
}

// BeforeRequest logs the outgoing request
func (l *Logger) BeforeRequest(data InterceptorData) (InterceptorData, error) {
	// Skip logging for specified paths
	for _, path := range l.opts.SkipPaths {
		if strings.HasPrefix(data.Request.URL.Path, path) {
			return data, nil
		}
	}

	var logBuilder strings.Builder

	if l.opts.LogBasicInfo {
		logBuilder.WriteString(fmt.Sprintf("[%s] --> %s %s\n", data.ID, data.Request.Method, data.Request.URL.String()))
	}

	if l.opts.LogHeaders {
		headerLogs := l.buildHeaderLogs("REQ", data.ID, data.Request.Header)
		if headerLogs != "" {
			logBuilder.WriteString(headerLogs)
		}
	}

	if l.opts.LogBody && data.Request.Body != nil {
		body, err := io.ReadAll(data.Request.Body)
		if err != nil {
			logBuilder.WriteString(fmt.Sprintf("[%s] Error reading request body: %v\n", data.ID, err))
		} else {
			data.Request.Body = io.NopCloser(bytes.NewReader(body))

			// Truncate body if it's too large
			truncated := false
			if len(body) > l.opts.MaxBodyLogSize {
				body = body[:l.opts.MaxBodyLogSize]
				truncated = true
			}
			logBuilder.WriteString(fmt.Sprintf("[%s] REQ BODY: %s%s\n", data.ID, string(body),
				map[bool]string{true: " [truncated...]", false: ""}[truncated]))

		}
	}

	logContent := logBuilder.String()
	if logContent != "" {
		l.opts.Logger.Print(logContent)
	}

	return data, nil
}

// AfterResponse logs the received response
func (l *Logger) AfterResponse(data InterceptorData) (InterceptorData, error) {

	for _, path := range l.opts.SkipPaths {
		if strings.HasPrefix(data.Request.URL.Path, path) {
			return data, nil
		}
	}

	var logBuilder strings.Builder

	if l.opts.LogBasicInfo {
		logBuilder.WriteString(fmt.Sprintf("[%s] <-- %d %s\n", data.ID,
			data.Response.StatusCode, http.StatusText(data.Response.StatusCode)))
	}
	if l.opts.LogHeaders {
		headerLogs := l.buildHeaderLogs("RESP", data.ID, data.Response.Header)
		if headerLogs != "" {
			logBuilder.WriteString(headerLogs)
		}
	}

	if l.opts.LogBody && data.Response.Body != nil {
		body, err := io.ReadAll(data.Response.Body)
		if err != nil {
			logBuilder.WriteString(fmt.Sprintf("[%s] Error reading response body: %v\n", data.ID, err))
		} else {
			data.Response.Body = io.NopCloser(bytes.NewReader(body))

			// Truncate body if it's too large
			truncated := false
			if len(body) > l.opts.MaxBodyLogSize {
				body = body[:l.opts.MaxBodyLogSize]
				truncated = true
			}
			logBuilder.WriteString(fmt.Sprintf("[%s] RESP BODY: %s%s\n", data.ID, string(body),
				map[bool]string{true: " [truncated...]", false: ""}[truncated]))
		}
	}

	logContent := logBuilder.String()
	if logContent != "" {
		l.opts.Logger.Print(logContent)
	}

	return data, nil
}

// buildHeaderLogs builds a string containing all header logs
func (l *Logger) buildHeaderLogs(prefix string, id string, headers http.Header) string {
	var logBuilder strings.Builder
	for name, values := range headers {
		if l.shouldSkipHeader(name) {
			continue
		}

		for _, value := range values {
			logBuilder.WriteString(fmt.Sprintf("[%s] %s HEADER: %s: %s\n", id, prefix, name, value))
		}
	}
	return logBuilder.String()
}

// shouldSkipHeader determines if a header should be skipped in logs
func (l *Logger) shouldSkipHeader(name string) bool {
	lowerName := strings.ToLower(name)
	for _, skip := range l.opts.SkipHeaders {
		if skip == lowerName {
			return true
		}
	}
	return false
}

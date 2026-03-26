package ics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arran4/golang-ical"
)

// SourceType represents the type of ICS source
type SourceType string

const (
	SourceTypeLocal   SourceType = "local"
	SourceTypeRemote  SourceType = "remote"
	SourceTypeCalDAV  SourceType = "caldav"
)

// AuthMethod represents different authentication methods for remote sources
type AuthMethod string

const (
	AuthNone  AuthMethod = "none"
	AuthBasic AuthMethod = "basic"
	AuthBearer AuthMethod = "bearer"
	AuthHeader AuthMethod = "header"
)

// Source represents a configuration for an ICS calendar source
type Source struct {
	Name            string            `yaml:"name"`
	Type            SourceType        `yaml:"type"`
	Path            string            `yaml:"path"`        // File path for local, URL for remote/caldav
	Auth            AuthMethod        `yaml:"auth"`
	Username        string            `yaml:"username"`    // For basic auth
	Password        string            `yaml:"password"`    // For basic auth
	Token           string            `yaml:"token"`       // For bearer auth
	Headers         map[string]string `yaml:"headers"`     // For custom headers
	Timeout         time.Duration     `yaml:"timeout"`     // HTTP timeout for remote sources
}

// ICSReader is an interface for reading ICS content from different sources
type ICSReader interface {
	ReadICS(ctx context.Context) (*ics.Calendar, error)
	GetSourceInfo() string
}

// LocalICSReader reads ICS files from local filesystem
type LocalICSReader struct {
	path string
}

// RemoteICSReader fetches ICS files from remote URLs with authentication
type RemoteICSReader struct {
	url        string
	auth       AuthMethod
	username   string
	password   string
	token      string
	headers    map[string]string
	httpClient *http.Client
}

// NewICSReader creates an appropriate ICSReader based on the source configuration
func NewICSReader(source Source) (ICSReader, error) {
	switch source.Type {
	case SourceTypeLocal:
		return NewLocalICSReader(source.Path)
	case SourceTypeRemote:
		timeout := source.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second // Default timeout
		}
		return NewRemoteICSReader(RemoteConfig{
			URL:      source.Path,
			Auth:     source.Auth,
			Username: source.Username,
			Password: source.Password,
			Token:    source.Token,
			Headers:  source.Headers,
			Timeout:  timeout,
		})
	default:
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}
}

// NewLocalICSReader creates a reader for local ICS files
func NewLocalICSReader(path string) (*LocalICSReader, error) {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("ICS file not found: %s", path)
	}

	return &LocalICSReader{path: path}, nil
}

// ReadICS implements ICSReader for local files
func (r *LocalICSReader) ReadICS(ctx context.Context) (*ics.Calendar, error) {
	file, err := os.Open(r.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open ICS file %s: %w", r.path, err)
	}
	defer file.Close()

	normalized, err := normalizeICSContent(file)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize ICS file %s: %w", r.path, err)
	}

	calendar, err := ics.ParseCalendar(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ICS file %s: %w", r.path, err)
	}

	return calendar, nil
}

// GetSourceInfo returns information about the local source
func (r *LocalICSReader) GetSourceInfo() string {
	abs, _ := filepath.Abs(r.path)
	return fmt.Sprintf("Local file: %s", abs)
}

// RemoteConfig holds configuration for remote ICS sources
type RemoteConfig struct {
	URL      string
	Auth     AuthMethod
	Username string
	Password string
	Token    string
	Headers  map[string]string
	Timeout  time.Duration
}

// NewRemoteICSReader creates a reader for remote ICS URLs
func NewRemoteICSReader(config RemoteConfig) (*RemoteICSReader, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("URL is required for remote ICS source")
	}

	if !strings.HasPrefix(config.URL, "http://") && !strings.HasPrefix(config.URL, "https://") {
		return nil, fmt.Errorf("invalid URL format: %s", config.URL)
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &RemoteICSReader{
		url:        config.URL,
		auth:       config.Auth,
		username:   config.Username,
		password:   config.Password,
		token:      config.Token,
		headers:    config.Headers,
		httpClient: httpClient,
	}, nil
}

// ReadICS implements ICSReader for remote URLs
func (r *RemoteICSReader) ReadICS(ctx context.Context) (*ics.Calendar, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Apply authentication
	switch r.auth {
	case AuthBasic:
		if r.username == "" || r.password == "" {
			return nil, fmt.Errorf("username and password required for basic auth")
		}
		req.SetBasicAuth(r.username, r.password)
	case AuthBearer:
		if r.token == "" {
			return nil, fmt.Errorf("token required for bearer auth")
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.token))
	case AuthHeader:
		// Custom headers are applied below
	case AuthNone:
		// No authentication
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", r.auth)
	}

	// Apply custom headers
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}

	// Set default headers for ICS requests
	req.Header.Set("Accept", "text/calendar, application/calendar")
	req.Header.Set("User-Agent", "caldav2markdown/1.0")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ICS from %s: %w", r.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: failed to fetch ICS from %s", resp.StatusCode, r.url)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(strings.ToLower(contentType), "calendar") &&
		!strings.Contains(strings.ToLower(contentType), "text") {
		fmt.Printf("Warning: unexpected content type for ICS: %s\n", contentType)
	}

	normalized, err := normalizeICSContent(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize ICS from %s: %w", r.url, err)
	}

	calendar, err := ics.ParseCalendar(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ICS from %s: %w", r.url, err)
	}

	return calendar, nil
}

// GetSourceInfo returns information about the remote source
func (r *RemoteICSReader) GetSourceInfo() string {
	authInfo := "no auth"
	switch r.auth {
	case AuthBasic:
		authInfo = fmt.Sprintf("basic auth (%s)", r.username)
	case AuthBearer:
		authInfo = "bearer token"
	case AuthHeader:
		authInfo = "custom headers"
	}
	return fmt.Sprintf("Remote URL: %s (%s)", r.url, authInfo)
}

// MultiSourceReader handles multiple ICS sources
type MultiSourceReader struct {
	readers []ICSReader
	sources []Source
}

// NewMultiSourceReader creates a reader that can handle multiple ICS sources
func NewMultiSourceReader(sources []Source) (*MultiSourceReader, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("at least one source is required")
	}

	readers := make([]ICSReader, 0, len(sources))
	for _, source := range sources {
		reader, err := NewICSReader(source)
		if err != nil {
			return nil, fmt.Errorf("failed to create reader for source %s: %w", source.Name, err)
		}
		readers = append(readers, reader)
	}

	return &MultiSourceReader{
		readers: readers,
		sources: sources,
	}, nil
}

// SourceProgressCallback is used for progress reporting during multi-source reading
type SourceProgressCallback func(message string, current, total int)

// ReadAllICS reads from all sources and returns combined results
func (m *MultiSourceReader) ReadAllICS(ctx context.Context, progressCallback SourceProgressCallback) ([]*ics.Calendar, error) {
	if progressCallback != nil {
		progressCallback("Starting to read ICS sources", 0, len(m.readers))
	}

	calendars := make([]*ics.Calendar, 0, len(m.readers))
	for i, reader := range m.readers {
		sourceName := fmt.Sprintf("Source %d", i+1)
		if i < len(m.sources) {
			sourceName = m.sources[i].Name
		}

		if progressCallback != nil {
			progressCallback(fmt.Sprintf("Reading from %s", sourceName), i+1, len(m.readers))
		}

		calendar, err := reader.ReadICS(ctx)
		if err != nil {
			fmt.Printf("Warning: failed to read from %s: %v\n", sourceName, err)
			continue
		}

		calendars = append(calendars, calendar)
	}

	if len(calendars) == 0 {
		return nil, fmt.Errorf("failed to read any ICS sources")
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Successfully read %d ICS sources", len(calendars)), len(m.readers), len(m.readers))
	}

	return calendars, nil
}

// normalizeICSContent fixes common non-conformant ICS content before parsing.
// Some servers (e.g. SOGo) place calendar-level properties like X-WR-CALNAME
// after component blocks (VEVENT, VTIMEZONE), which violates RFC 5545 and
// causes the golang-ical parser to fail. This function moves any such
// misplaced properties to before the first component.
func normalizeICSContent(r io.Reader) (io.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// Normalize line endings to LF
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")

	var header []string     // calendar-level properties before first component
	var calProps []string   // misplaced calendar-level properties found after components
	var components []string // all component blocks
	var footer []string     // END:VCALENDAR

	inComponent := false
	componentDepth := 0
	seenComponent := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "END:VCALENDAR" {
			footer = append(footer, line)
			continue
		}
		if !inComponent && (strings.HasPrefix(trimmed, "BEGIN:VEVENT") ||
			strings.HasPrefix(trimmed, "BEGIN:VTIMEZONE") ||
			strings.HasPrefix(trimmed, "BEGIN:VTODO") ||
			strings.HasPrefix(trimmed, "BEGIN:VJOURNAL") ||
			strings.HasPrefix(trimmed, "BEGIN:VFREEBUSY")) {
			inComponent = true
			seenComponent = true
			componentDepth = 1
			components = append(components, line)
			continue
		}
		if inComponent {
			components = append(components, line)
			if strings.HasPrefix(trimmed, "BEGIN:") {
				componentDepth++
			} else if strings.HasPrefix(trimmed, "END:") {
				componentDepth--
				if componentDepth == 0 {
					inComponent = false
				}
			}
			continue
		}
		// A non-BEGIN/END property line appearing after we've seen a component
		// is misplaced and needs to be moved before components.
		if seenComponent && trimmed != "" && !strings.HasPrefix(trimmed, "BEGIN:") && !strings.HasPrefix(trimmed, "END:") {
			calProps = append(calProps, line)
			continue
		}
		header = append(header, line)
	}

	// If nothing was misplaced, return original content to avoid any subtle changes
	if len(calProps) == 0 {
		return bytes.NewReader(data), nil
	}

	// Reconstruct: header + misplaced properties + components + END:VCALENDAR
	all := append(header, calProps...)
	all = append(all, components...)
	all = append(all, footer...)
	return strings.NewReader(strings.Join(all, "\n")), nil
}

// GetSourceInfos returns information about all sources
func (m *MultiSourceReader) GetSourceInfos() []string {
	infos := make([]string, len(m.readers))
	for i, reader := range m.readers {
		infos[i] = reader.GetSourceInfo()
	}
	return infos
}
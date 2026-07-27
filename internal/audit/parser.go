package audit

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/afcollins/kbx/internal/common"
)

type rawEvent struct {
	Verb      string `json:"verb"`
	ObjectRef struct {
		Resource   string `json:"resource"`
		APIGroup   string `json:"apiGroup"`
		APIVersion string `json:"apiVersion"`
		Namespace  string `json:"namespace"`
	} `json:"objectRef"`
	User struct {
		Username string `json:"username"`
	} `json:"user"`
	SourceIPs      []string `json:"sourceIPs"`
	UserAgent      string   `json:"userAgent"`
	ResponseStatus struct {
		Code int `json:"code"`
	} `json:"responseStatus"`
	StageTimestamp string `json:"stageTimestamp"`
}

// ParseResult holds events from a single file and the path to read raw JSON from.
type ParseResult struct {
	Events   []AuditEvent
	ReadPath string // path for offset-based raw JSON reads (temp file for gzip)
}

// ParseFile parses a .log or .log.gz file into AuditEvents, tracking byte offsets.
// For .gz files, it streams through a TeeReader: decompressing, writing to a temp
// file, and parsing in a single pass. The temp file path is returned in ReadPath
// for offset-based raw JSON re-reads.
func ParseFile(path string, fileIndex int) (*ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer common.CloseQuiet(f)

	var reader io.Reader
	readPath := path
	var tmpCleanup func()

	if isGzip(path) {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer common.GzCloseSafe(gz)
		tmp, err := os.CreateTemp("", "kbx-audit-*.log")
		if err != nil {
			return nil, err
		}
		tmpCleanup = func() { common.CloseQuiet(tmp) }
		readPath = tmp.Name()

		// TeeReader: reads from gzip, writes to temp file simultaneously
		reader = io.TeeReader(gz, tmp)
		slog.Info("streaming gzip", "file", path, "temp", readPath)
	} else {
		reader = f
	}

	start := time.Now()
	events, err := parseReader(reader, fileIndex)
	if tmpCleanup != nil {
		tmpCleanup()
	}
	if err != nil {
		return nil, err
	}

	slog.Info("parsed file", "file", path, "events", len(events), "elapsed", time.Since(start).Round(time.Millisecond))
	return &ParseResult{Events: events, ReadPath: readPath}, nil
}

func isGzip(path string) bool {
	n := len(path)
	return n > 3 && path[n-3:] == ".gz"
}

func parseReader(r io.Reader, fileIndex int) ([]AuditEvent, error) {
	// json.Decoder handles all formats: single objects, JSON arrays, and
	// JSON-lines (successive objects). InputOffset tracks byte positions
	// for offset-based raw JSON re-reads from disk.
	dec := json.NewDecoder(r)

	var events []AuditEvent
	for {
		startOffset := dec.InputOffset()
		var obj json.RawMessage
		if err := dec.Decode(&obj); err != nil {
			break
		}
		endOffset := dec.InputOffset()

		// If this is a JSON array, unwrap and parse each element.
		// Offset tracking is not available for individual array elements.
		if len(obj) > 0 && obj[0] == '[' {
			var arr []json.RawMessage
			if err := json.Unmarshal(obj, &arr); err != nil {
				continue
			}
			for _, item := range arr {
				if e, ok := parseRawEvent(item, fileIndex, 0, len(item)); ok {
					events = append(events, e)
				}
			}
			continue
		}

		lineLen := int(endOffset - startOffset)
		if e, ok := parseRawEvent(obj, fileIndex, startOffset, lineLen); ok {
			events = append(events, e)
		}
	}

	return events, nil
}

func parseRawEvent(data []byte, fileIndex int, offset int64, lineLen int) (AuditEvent, bool) {
	var raw rawEvent
	if err := json.Unmarshal(data, &raw); err != nil {
		return AuditEvent{}, false
	}

	ts, _ := time.Parse(time.RFC3339Nano, raw.StageTimestamp)

	sourceIP := ""
	if len(raw.SourceIPs) > 0 {
		sourceIP = raw.SourceIPs[0]
	}

	return AuditEvent{
		Verb:       raw.Verb,
		Resource:   raw.ObjectRef.Resource,
		APIGroup:   raw.ObjectRef.APIGroup,
		APIVersion: raw.ObjectRef.APIVersion,
		Namespace:  raw.ObjectRef.Namespace,
		Username:   raw.User.Username,
		SourceIP:   sourceIP,
		UserAgent:  raw.UserAgent,
		StatusCode: raw.ResponseStatus.Code,
		Timestamp:  ts,
		FileIndex:  fileIndex,
		FileOffset: offset,
		LineLength: lineLen,
	}, true
}

// ReadRawJSON reads the raw JSON for an event from the given file path.
func ReadRawJSON(path string, offset int64, length int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer common.CloseQuiet(f)

	buf := make([]byte, length)
	_, err = f.ReadAt(buf, offset)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

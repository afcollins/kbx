package audit

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"time"
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
// For .gz files, it decompresses to a temp file and returns that path in ReadPath.
func ParseFile(path string, fileIndex int) (*ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var reader io.Reader
	readPath := path

	if isGzip(path) {
		tmpFile, err := decompressToTemp(f)
		if err != nil {
			return nil, err
		}
		readPath = tmpFile
		// Re-open the decompressed temp file for parsing
		tf, err := os.Open(tmpFile)
		if err != nil {
			return nil, err
		}
		defer tf.Close()
		reader = tf
	} else {
		reader = f
	}

	events, err := parseReader(reader, fileIndex)
	if err != nil {
		return nil, err
	}

	return &ParseResult{Events: events, ReadPath: readPath}, nil
}

func isGzip(path string) bool {
	n := len(path)
	return n > 3 && path[n-3:] == ".gz"
}

func decompressToTemp(r io.Reader) (string, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tmp, err := os.CreateTemp("", "kube-audit-*.log")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, gz); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	return tmp.Name(), nil
}

func parseReader(r io.Reader, fileIndex int) ([]AuditEvent, error) {
	// json.Decoder handles all formats: single objects, JSON arrays, and
	// JSON-lines (successive objects). It reads complete JSON values
	// regardless of whitespace or pretty-printing.
	dec := json.NewDecoder(r)

	var events []AuditEvent
	for {
		var obj json.RawMessage
		if err := dec.Decode(&obj); err != nil {
			break
		}

		// If this is a JSON array, unwrap and parse each element
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

		if e, ok := parseRawEvent(obj, fileIndex, 0, len(obj)); ok {
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

// ReadRawJSON reads the raw JSON line for an event from the given file path.
func ReadRawJSON(path string, offset int64, length int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, length)
	_, err = f.ReadAt(buf, offset)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

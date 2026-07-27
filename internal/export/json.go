package export

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/afcollins/kbx/internal/store"
)

// ExportJSON writes the raw JSON of filtered events to the given file path.
func ExportJSON(s *store.EventStore, path string) (n int, retErr error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer func() {
		if cerr := f.Close(); retErr == nil {
			retErr = cerr
		}
	}()

	w := bufio.NewWriter(f)
	indices := s.Filtered()
	_, _ = w.WriteString("[\n")

	written := 0
	for i, idx := range indices {
		raw, err := s.ReadRawJSON(idx)
		if err != nil {
			return written, fmt.Errorf("reading event %d: %w", idx, err)
		}

		var obj interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			_, _ = w.Write(raw)
		} else {
			pretty, _ := json.MarshalIndent(obj, "  ", "  ")
			_, _ = w.WriteString("  ")
			_, _ = w.Write(pretty)
		}

		if i < len(indices)-1 {
			_, _ = w.WriteString(",")
		}
		_, _ = w.WriteString("\n")
		written++
	}

	_, _ = w.WriteString("]\n")
	if err := w.Flush(); err != nil {
		return written, err
	}
	return written, nil
}

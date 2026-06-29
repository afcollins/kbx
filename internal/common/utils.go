package common

import (
	"compress/gzip"
	"log/slog"
)

func GzCloseSafe(r *gzip.Reader) {
	if err := r.Close(); err != nil {
		slog.Debug("gz.Close", "error", err)
	}

}

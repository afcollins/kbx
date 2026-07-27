package common

import (
	"compress/gzip"
	"io"
	"log/slog"
)

func CloseQuiet(c io.Closer) {
	_ = c.Close()
}

func GzCloseSafe(r *gzip.Reader) {
	if err := r.Close(); err != nil {
		slog.Debug("gz.Close", "error", err)
	}

}

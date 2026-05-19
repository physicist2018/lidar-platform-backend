package ports

import (
	"context"
	"io"
)

type FileStorage interface {
	Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, path string) (io.ReadCloser, int64, error)
}

package sanitize

import (
	"fmt"
	"io"
)

// ReadAtMost reads the whole body unless it is larger than limit bytes, in
// which case it fails instead of holding an unbounded answer in memory.
func ReadAtMost(body io.Reader, limit int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return content, nil
}

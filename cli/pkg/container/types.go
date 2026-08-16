package container

import (
	"time"
)

// Metadata stores all essential file container attributes.
type Metadata struct {
	Version      int    `json:"version"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mimeType"`
	ByteLength   int64  `json:"byteLength"`
	SHA256       string `json:"sha256"`
	Timestamp    int64  `json:"timestamp"`
	Compression  string `json:"compression"` // "stored" (level 0) or "deflate" (level 6/9)
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AspectRatio  string `json:"aspectRatio"` // "1:1" or "16:9"
}

// EncodeOptions configures the encoding process.
type EncodeOptions struct {
	CompressionMode string // "auto", "stored", "deflate"
	AspectRatio     string // "1:1", "16:9"
	Filename        string // override original filename
	MimeType        string // optional MIME type
}

// DefaultEncodeOptions returns sensible defaults for encoding.
func DefaultEncodeOptions() EncodeOptions {
	return EncodeOptions{
		CompressionMode: "auto",
		AspectRatio:     "1:1",
	}
}

// EncodeResult contains the output PNG and analytics.
type EncodeResult struct {
	PNGData       []byte
	Metadata      Metadata
	OriginalSize  int64
	OutputSize    int64
	OverheadBytes int64
	RatioPercent  float64
	Duration      time.Duration
}

// DecodeResult contains the restored file data and verification info.
type DecodeResult struct {
	Data          []byte
	Metadata      Metadata
	SHA256Matches bool
	Duration      time.Duration
}

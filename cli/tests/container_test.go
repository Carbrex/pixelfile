package tests

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"pixelfile/pkg/container"
)

func TestLosslessRoundtrip(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		aspectRatio string
		mode        string
	}{
		{
			name:        "Empty file (0 bytes)",
			data:        []byte{},
			aspectRatio: "1:1",
			mode:        "auto",
		},
		{
			name:        "Single byte (1 byte)",
			data:        []byte{0x42},
			aspectRatio: "1:1",
			mode:        "auto",
		},
		{
			name:        "Two bytes",
			data:        []byte{0xDE, 0xAD},
			aspectRatio: "1:1",
			mode:        "auto",
		},
		{
			name:        "Three bytes (RGB boundary test)",
			data:        []byte{0xDE, 0xAD, 0xBE},
			aspectRatio: "1:1",
			mode:        "auto",
		},
		{
			name:        "Four bytes (RGBA 1 pixel)",
			data:        []byte{0xDE, 0xAD, 0xBE, 0xEF},
			aspectRatio: "1:1",
			mode:        "auto",
		},
		{
			name:        "Prime length (7 bytes)",
			data:        []byte("Hello!!"),
			aspectRatio: "16:9",
			mode:        "auto",
		},
		{
			name:        "Sample Plain Text / Code (Low Entropy)",
			data:        bytes.Repeat([]byte("func main() { fmt.Println(\"PixelFile!\") }\n"), 500),
			aspectRatio: "1:1",
			mode:        "deflate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := container.EncodeOptions{
				Filename:        fmt.Sprintf("test_%s.bin", tc.name),
				AspectRatio:     tc.aspectRatio,
				CompressionMode: tc.mode,
			}

			// Encode
			encResult, err := container.Encode(tc.data, opts)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			if len(encResult.PNGData) == 0 {
				t.Fatalf("Encoded PNG data is empty")
			}

			// Decode
			decResult, err := container.Decode(encResult.PNGData)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			// Validate
			if !decResult.SHA256Matches {
				t.Fatalf("SHA-256 validation failed")
			}

			if !bytes.Equal(decResult.Data, tc.data) {
				t.Fatalf("Restored bytes do not match original! Got %d bytes, want %d bytes", len(decResult.Data), len(tc.data))
			}

			t.Logf("[%s] Original: %d B -> PNG: %d B (Delta: %+.2f%%) [Mode: %s, Dimensions: %dx%d]",
				tc.name, encResult.OriginalSize, encResult.OutputSize, encResult.RatioPercent,
				encResult.Metadata.Compression, encResult.Metadata.Width, encResult.Metadata.Height)
		})
	}
}

func TestHighEntropyScaling(t *testing.T) {
	sizes := []struct {
		name      string
		sizeBytes int
		maxDelta  float64
	}{
		{"256 KB Random Data", 256 * 1024, 0.35}, // ~0.24%
		{"1 MB Random Data", 1024 * 1024, 0.15},  // ~0.09%
		{"4 MB Random Data", 4 * 1024 * 1024, 0.08}, // ~0.04%
		{"8 MB Random Data", 8 * 1024 * 1024, 0.08}, // ~0.05%
	}

	for _, s := range sizes {
		t.Run(s.name, func(t *testing.T) {
			randomData := make([]byte, s.sizeBytes)
			if _, err := rand.Read(randomData); err != nil {
				t.Fatalf("Failed to generate random bytes: %v", err)
			}

			opts := container.EncodeOptions{
				Filename:        "random_payload.bin",
				AspectRatio:     "1:1",
				CompressionMode: "auto",
			}

			encResult, err := container.Encode(randomData, opts)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			if encResult.RatioPercent > s.maxDelta {
				t.Errorf("Overhead too high: %f%% (wanted <= %f%%)", encResult.RatioPercent, s.maxDelta)
			}

			decResult, err := container.Decode(encResult.PNGData)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if !decResult.SHA256Matches {
				t.Fatalf("SHA-256 validation failed")
			}

			if !bytes.Equal(decResult.Data, randomData) {
				t.Fatalf("Decoded random data does not match original!")
			}

			t.Logf("[%s] Original: %d B -> PNG: %d B (Overhead: %d B / %.4f%%, Mode: %s) [Time: %v]",
				s.name, encResult.OriginalSize, encResult.OutputSize, encResult.OverheadBytes, encResult.RatioPercent, encResult.Metadata.Compression, encResult.Duration)
		})
	}
}

package tests

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"pixelfile/pkg/container"
)

func TestStreamingLargeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pixelfile_stream_test")
	if err != nil {
		t.Fatalf("Failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "large_source.bin")
	pngPath := filepath.Join(tmpDir, "large_encoded.png")
	restoredPath := filepath.Join(tmpDir, "large_restored.bin")

	// Create 20 MB of random data
	const size = 20 * 1024 * 1024
	origBytes := make([]byte, size)
	if _, err := rand.Read(origBytes); err != nil {
		t.Fatalf("Failed generating random bytes: %v", err)
	}
	if err := os.WriteFile(inputPath, origBytes, 0644); err != nil {
		t.Fatalf("Failed writing input file: %v", err)
	}

	opts := container.EncodeOptions{
		AspectRatio:     "1:1",
		CompressionMode: "stored",
	}

	// 1. Encode with Stream
	encRes, err := container.EncodeFile(inputPath, pngPath, opts)
	if err != nil {
		t.Fatalf("EncodeFile failed: %v", err)
	}

	t.Logf("[Streaming 20MB] Encoded in %v (Output Size: %d B, Delta: %+.4f%%)",
		encRes.Duration, encRes.OutputSize, encRes.RatioPercent)

	// 2. Decode with Stream
	decRes, err := container.DecodeFile(pngPath, restoredPath)
	if err != nil {
		t.Fatalf("DecodeFile failed: %v", err)
	}

	t.Logf("[Streaming 20MB] Decoded in %v", decRes.Duration)

	// 3. Verify byte equivalence
	restoredBytes, err := os.ReadFile(restoredPath)
	if err != nil {
		t.Fatalf("Failed reading restored file: %v", err)
	}

	if !bytes.Equal(origBytes, restoredBytes) {
		t.Fatalf("Restored 20MB file does not match original bytes!")
	}

	t.Logf("✓ 20MB Streaming encode/decode verified bit-perfect.")
}

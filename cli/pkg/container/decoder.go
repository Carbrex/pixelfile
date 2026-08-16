package container

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Decode extracts and validates original file bytes from a PixelFile PNG container.
func Decode(pngData []byte) (*DecodeResult, error) {
	startTime := time.Now()

	chunks, err := ParsePNGChunks(pngData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PNG chunks: %w", err)
	}

	var (
		ihdrData   []byte
		metaText   string
		idatBuffer bytes.Buffer
	)

	for _, chunk := range chunks {
		switch chunk.Type {
		case "IHDR":
			ihdrData = chunk.Data
		case "tEXt":
			// Find keyword 'PixelFile'
			nullIdx := bytes.IndexByte(chunk.Data, 0x00)
			if nullIdx > 0 {
				keyword := string(chunk.Data[:nullIdx])
				if keyword == "PixelFile" {
					metaText = string(chunk.Data[nullIdx+1:])
				}
			}
		case "IDAT":
			idatBuffer.Write(chunk.Data)
		}
	}

	if len(ihdrData) < 13 {
		return nil, fmt.Errorf("missing or invalid IHDR chunk in PNG")
	}

	width := int(binary.BigEndian.Uint32(ihdrData[0:4]))
	height := int(binary.BigEndian.Uint32(ihdrData[4:8]))

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid PNG dimensions: %dx%d", width, height)
	}

	if idatBuffer.Len() == 0 {
		return nil, fmt.Errorf("no IDAT image data found in PNG")
	}

	// Decompress IDAT zlib payload
	zReader, err := zlib.NewReader(&idatBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zlib decompression: %w", err)
	}
	defer zReader.Close()

	decompressedScanlines, err := io.ReadAll(zReader)
	if err != nil {
		return nil, fmt.Errorf("zlib decompression failed: %w", err)
	}

	// Reconstruct RGBA buffer from scanlines
	rowBytes := width * 4
	expectedScanlineLen := height * (1 + rowBytes)
	if len(decompressedScanlines) < expectedScanlineLen {
		return nil, fmt.Errorf("decompressed data shorter than expected scanlines (got %d, want %d)", len(decompressedScanlines), expectedScanlineLen)
	}

	rgbaBuffer := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		offset := y * (1 + rowBytes)
		// Filter byte is at decompressedScanlines[offset]
		copy(rgbaBuffer[y*rowBytes:(y+1)*rowBytes], decompressedScanlines[offset+1:offset+1+rowBytes])
	}

	// Parse Metadata if present
	var metadata Metadata
	if metaText != "" {
		if err := json.Unmarshal([]byte(metaText), &metadata); err != nil {
			return nil, fmt.Errorf("failed to parse PixelFile metadata JSON: %w", err)
		}
	} else {
		// Fallback if metadata chunk was stripped
		metadata = Metadata{
			Version:     1,
			Filename:    "restored_payload.bin",
			MimeType:    "application/octet-stream",
			ByteLength:  int64(len(rgbaBuffer)),
			Width:       width,
			Height:      height,
			Compression: "unknown",
		}
	}

	// Extract exact original byte slice
	restoredData, err := UnpackRGBAToBytes(rgbaBuffer, metadata.ByteLength)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack original byte payload: %w", err)
	}

	// Integrity validation
	sha256Matches := true
	if metadata.SHA256 != "" {
		sha256Matches = VerifySHA256(restoredData, metadata.SHA256)
		if !sha256Matches {
			return nil, fmt.Errorf("SHA-256 checksum mismatch! File is corrupted or altered. Expected: %s, Actual: %s", metadata.SHA256, ComputeSHA256(restoredData))
		}
	}

	return &DecodeResult{
		Data:          restoredData,
		Metadata:      metadata,
		SHA256Matches: sha256Matches,
		Duration:      time.Since(startTime),
	}, nil
}

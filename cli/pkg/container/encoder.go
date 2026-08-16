package container

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"mime"
	"path/filepath"
	"time"
)

// Encode transforms arbitrary binary file data into a valid, lossless PNG image.
func Encode(data []byte, options EncodeOptions) (*EncodeResult, error) {
	startTime := time.Now()

	totalBytes := int64(len(data))
	sha256Hash := ComputeSHA256(data)

	// Calculate optimal dimensions
	aspectRatio := options.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	width, height := CalculateDimensions(totalBytes, aspectRatio)

	// Pack data bytes into RGBA buffer
	rgbaBuffer := PackBytesToRGBA(data, width, height)

	// Build raw PNG scanlines (Filter None: 0x00 per row)
	rowBytes := width * 4
	scanlineData := make([]byte, height*(1+rowBytes))
	for y := 0; y < height; y++ {
		offset := y * (1 + rowBytes)
		scanlineData[offset] = 0 // Filter Type 0 (None)
		copy(scanlineData[offset+1:offset+1+rowBytes], rgbaBuffer[y*rowBytes:(y+1)*rowBytes])
	}

	// Compress scanlines adaptively
	var idatData []byte
	var compressionModeUsed string

	switch options.CompressionMode {
	case "stored":
		compressed, err := compressZlib(scanlineData, zlib.NoCompression)
		if err != nil {
			return nil, fmt.Errorf("zlib stored compression failed: %w", err)
		}
		idatData = compressed
		compressionModeUsed = "stored"

	case "deflate":
		compressed, err := compressZlib(scanlineData, zlib.BestCompression)
		if err != nil {
			return nil, fmt.Errorf("zlib deflate compression failed: %w", err)
		}
		idatData = compressed
		compressionModeUsed = "deflate"

	case "auto", "":
		fallthrough
	default:
		// Benchmark both modes to pick the smallest size
		storedData, err := compressZlib(scanlineData, zlib.NoCompression)
		if err != nil {
			return nil, fmt.Errorf("zlib stored compression failed: %w", err)
		}

		deflateData, err := compressZlib(scanlineData, zlib.BestCompression)
		if err != nil {
			return nil, fmt.Errorf("zlib deflate compression failed: %w", err)
		}

		// Prefer deflate only if it saves space compared to uncompressed stored blocks
		if len(deflateData) < len(storedData) {
			idatData = deflateData
			compressionModeUsed = "deflate"
		} else {
			idatData = storedData
			compressionModeUsed = "stored"
		}
	}

	// Determine MIME type
	filename := options.Filename
	if filename == "" {
		filename = "payload.bin"
	}
	mimeType := options.MimeType
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(filename))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	// Assemble Metadata
	metadata := Metadata{
		Version:     1,
		Filename:    filename,
		MimeType:    mimeType,
		ByteLength:  totalBytes,
		SHA256:      sha256Hash,
		Timestamp:   time.Now().UnixMilli(),
		Compression: compressionModeUsed,
		Width:       width,
		Height:      height,
		AspectRatio: aspectRatio,
	}

	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata JSON: %w", err)
	}

	// Build full PNG container
	var pngBuf bytes.Buffer
	pngBuf.Write(PNGSignature)
	pngBuf.Write(BuildIHDRChunk(uint32(width), uint32(height)))
	pngBuf.Write(BuildTextChunk("PixelFile", string(metaJSON)))
	pngBuf.Write(BuildChunk("IDAT", idatData))
	pngBuf.Write(BuildIENDChunk())

	finalPNG := pngBuf.Bytes()
	outputSize := int64(len(finalPNG))
	overheadBytes := outputSize - totalBytes
	ratioPercent := 0.0
	if totalBytes > 0 {
		ratioPercent = (float64(overheadBytes) / float64(totalBytes)) * 100.0
	}

	return &EncodeResult{
		PNGData:       finalPNG,
		Metadata:      metadata,
		OriginalSize:  totalBytes,
		OutputSize:    outputSize,
		OverheadBytes: overheadBytes,
		RatioPercent:  ratioPercent,
		Duration:      time.Since(startTime),
	}, nil
}

// compressZlib compresses input data at the specified compression level.
func compressZlib(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

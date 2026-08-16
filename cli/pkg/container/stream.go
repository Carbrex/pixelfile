package container

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultIDATChunkSize is the size of each IDAT chunk during streaming (64 KB).
	DefaultIDATChunkSize = 64 * 1024
)

// idatChunkWriter accumulates compressed data and flushes standard IDAT chunks to the underlying writer.
type idatChunkWriter struct {
	w         io.Writer
	chunkSize int
	buf       []byte
	written   int64
}

func newIDATChunkWriter(w io.Writer, chunkSize int) *idatChunkWriter {
	if chunkSize <= 0 {
		chunkSize = DefaultIDATChunkSize
	}
	return &idatChunkWriter{
		w:         w,
		chunkSize: chunkSize,
		buf:       make([]byte, 0, chunkSize),
	}
}

func (cw *idatChunkWriter) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 {
		avail := cw.chunkSize - len(cw.buf)
		if len(p) < avail {
			cw.buf = append(cw.buf, p...)
			break
		}
		cw.buf = append(cw.buf, p[:avail]...)
		p = p[avail:]
		if err := cw.flushChunk(); err != nil {
			return 0, err
		}
	}
	return n, nil
}

func (cw *idatChunkWriter) flushChunk() error {
	if len(cw.buf) == 0 {
		return nil
	}
	chunkBytes := BuildChunk("IDAT", cw.buf)
	if _, err := cw.w.Write(chunkBytes); err != nil {
		return err
	}
	cw.written += int64(len(cw.buf))
	cw.buf = cw.buf[:0]
	return nil
}

func (cw *idatChunkWriter) Close() error {
	return cw.flushChunk()
}

// EncodeStream converts a stream of bytes into a streaming lossless PNG with constant low memory usage.
func EncodeStream(r io.Reader, totalBytes int64, w io.Writer, options EncodeOptions) (*EncodeResult, error) {
	startTime := time.Now()

	aspectRatio := options.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	width, height := CalculateDimensions(totalBytes, aspectRatio)

	// Determine compression level
	compressionLevel := zlib.NoCompression
	compressionModeUsed := "stored"
	if options.CompressionMode == "deflate" {
		compressionLevel = zlib.BestCompression
		compressionModeUsed = "deflate"
	} else if options.CompressionMode == "auto" || options.CompressionMode == "" {
		// For small files, we can default to auto check, but for general streaming stored blocks is ideal
		// For streaming without double-reading, stored blocks guarantees <= 0.01% overhead.
		if totalBytes > 8*1024*1024 {
			compressionLevel = zlib.NoCompression
			compressionModeUsed = "stored"
		} else {
			// For small payload, we can use deflate if requested
			compressionLevel = zlib.DefaultCompression
			compressionModeUsed = "deflate"
		}
	}

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

	// 1. Write PNG Signature
	if _, err := w.Write(PNGSignature); err != nil {
		return nil, fmt.Errorf("failed to write PNG signature: %w", err)
	}

	// 2. Write IHDR Chunk
	ihdr := BuildIHDRChunk(uint32(width), uint32(height))
	if _, err := w.Write(ihdr); err != nil {
		return nil, fmt.Errorf("failed to write IHDR chunk: %w", err)
	}

	// 3. Prepare Metadata
	hasher := sha256.New()
	hashingReader := io.TeeReader(r, hasher)

	metadata := Metadata{
		Version:     1,
		Filename:    filename,
		MimeType:    mimeType,
		ByteLength:  totalBytes,
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

	// Write tEXt metadata chunk
	textChunk := BuildTextChunk("PixelFile", string(metaJSON))
	if _, err := w.Write(textChunk); err != nil {
		return nil, fmt.Errorf("failed to write tEXt chunk: %w", err)
	}

	// 4. Stream IDAT chunks row-by-row
	idatWriter := newIDATChunkWriter(w, DefaultIDATChunkSize)
	zWriter, err := zlib.NewWriterLevel(idatWriter, compressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zlib writer: %w", err)
	}

	rowBytes := width * 4
	rowBuf := make([]byte, rowBytes)
	filterPrefix := []byte{0x00} // Filter None

	var bytesReadTotal int64
	for y := 0; y < height; y++ {
		// Write scanline filter byte
		if _, err := zWriter.Write(filterPrefix); err != nil {
			return nil, fmt.Errorf("failed writing scanline filter: %w", err)
		}

		// Read exactly rowBytes
		n, err := io.ReadFull(hashingReader, rowBuf)
		bytesReadTotal += int64(n)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("error reading stream at row %d: %w", y, err)
		}

		// Zero-fill remaining row bytes if at the end of stream
		if n < rowBytes {
			for i := n; i < rowBytes; i++ {
				rowBuf[i] = 0
			}
		}

		if _, err := zWriter.Write(rowBuf); err != nil {
			return nil, fmt.Errorf("failed writing row data to zlib: %w", err)
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			// Fill remaining rows with zeros
			for remainingY := y + 1; remainingY < height; remainingY++ {
				if _, err := zWriter.Write(filterPrefix); err != nil {
					return nil, err
				}
				for i := range rowBuf {
					rowBuf[i] = 0
				}
				if _, err := zWriter.Write(rowBuf); err != nil {
					return nil, err
				}
			}
			break
		}
	}

	// Close zlib and flush IDAT
	if err := zWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed closing zlib stream: %w", err)
	}
	if err := idatWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed closing IDAT stream: %w", err)
	}

	// 5. Write IEND Chunk
	iend := BuildIENDChunk()
	if _, err := w.Write(iend); err != nil {
		return nil, fmt.Errorf("failed writing IEND chunk: %w", err)
	}

	computedHash := fmt.Sprintf("%x", hasher.Sum(nil))
	metadata.SHA256 = computedHash

	outputSize := int64(8 + len(ihdr) + len(textChunk) + int(idatWriter.written) + len(iend))
	overheadBytes := outputSize - totalBytes
	ratioPercent := 0.0
	if totalBytes > 0 {
		ratioPercent = (float64(overheadBytes) / float64(totalBytes)) * 100.0
	}

	return &EncodeResult{
		Metadata:      metadata,
		OriginalSize:  totalBytes,
		OutputSize:    outputSize,
		OverheadBytes: overheadBytes,
		RatioPercent:  ratioPercent,
		Duration:      time.Since(startTime),
	}, nil
}

// DecodeStream decodes a streaming PNG container into original file bytes with minimal RAM usage.
func DecodeStream(r io.Reader, w io.Writer) (*DecodeResult, error) {
	startTime := time.Now()

	// 1. Verify PNG Signature
	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		return nil, fmt.Errorf("failed reading PNG signature: %w", err)
	}
	if !bytes.Equal(sig, PNGSignature) {
		return nil, fmt.Errorf("invalid PNG signature")
	}

	// Stream chunk reader
	chunkReader := newStreamingChunkReader(r)

	// Read headers
	var (
		ihdrData   []byte
		metaText   string
		idatStream *idatPayloadStream
	)

	// Read until first IDAT
	for {
		chunk, err := chunkReader.nextChunkHeader()
		if err != nil {
			return nil, fmt.Errorf("error reading chunk header: %w", err)
		}
		if chunk.Type == "IHDR" {
			ihdrData = make([]byte, chunk.Length)
			if _, err := io.ReadFull(chunkReader.r, ihdrData); err != nil {
				return nil, err
			}
			chunkReader.skipCRC(chunk.Length, chunk.Type, ihdrData)
		} else if chunk.Type == "tEXt" {
			textData := make([]byte, chunk.Length)
			if _, err := io.ReadFull(chunkReader.r, textData); err != nil {
				return nil, err
			}
			chunkReader.skipCRC(chunk.Length, chunk.Type, textData)
			nullIdx := bytes.IndexByte(textData, 0x00)
			if nullIdx > 0 && string(textData[:nullIdx]) == "PixelFile" {
				metaText = string(textData[nullIdx+1:])
			}
		} else if chunk.Type == "IDAT" {
			// Positioned at first IDAT payload
			idatStream = newIDATPayloadStream(chunkReader, chunk.Length)
			break
		} else {
			// Skip unknown chunk
			if err := chunkReader.discardChunkData(chunk.Length); err != nil {
				return nil, err
			}
		}
	}

	if len(ihdrData) < 8 {
		return nil, fmt.Errorf("missing IHDR chunk in PNG")
	}

	width := int(binary.BigEndian.Uint32(ihdrData[0:4]))
	height := int(binary.BigEndian.Uint32(ihdrData[4:8]))

	var metadata Metadata
	if metaText != "" {
		_ = json.Unmarshal([]byte(metaText), &metadata)
	} else {
		metadata = Metadata{
			Version:    1,
			Filename:   "restored_payload.bin",
			ByteLength: int64(width * height * 4),
			Width:      width,
			Height:     height,
		}
	}

	// IDAT stream decompressed with zlib
	zReader, err := zlib.NewReader(idatStream)
	if err != nil {
		return nil, fmt.Errorf("failed creating zlib reader: %w", err)
	}
	defer zReader.Close()

	hasher := sha256.New()
	hashingWriter := io.MultiWriter(w, hasher)

	rowBytes := width * 4
	rowBuf := make([]byte, rowBytes)
	filterByte := make([]byte, 1)

	var remainingBytesToWrite = metadata.ByteLength

	for y := 0; y < height; y++ {
		// Read filter byte
		if _, err := io.ReadFull(zReader, filterByte); err != nil {
			return nil, fmt.Errorf("failed reading scanline filter at row %d: %w", y, err)
		}

		// Read row pixels
		if _, err := io.ReadFull(zReader, rowBuf); err != nil {
			return nil, fmt.Errorf("failed reading row %d data: %w", y, err)
		}

		if remainingBytesToWrite > 0 {
			toWrite := int64(len(rowBuf))
			if toWrite > remainingBytesToWrite {
				toWrite = remainingBytesToWrite
			}
			if _, err := hashingWriter.Write(rowBuf[:toWrite]); err != nil {
				return nil, fmt.Errorf("failed writing restored output: %w", err)
			}
			remainingBytesToWrite -= toWrite
		}
	}

	computedHash := fmt.Sprintf("%x", hasher.Sum(nil))
	sha256Matches := true
	if metadata.SHA256 != "" {
		sha256Matches = (computedHash == metadata.SHA256)
		if !sha256Matches {
			return nil, fmt.Errorf("SHA-256 integrity verification failed: expected %s, got %s", metadata.SHA256, computedHash)
		}
	}

	return &DecodeResult{
		Metadata:      metadata,
		SHA256Matches: sha256Matches,
		Duration:      time.Since(startTime),
	}, nil
}

// EncodeFile encodes a file from disk to disk with streaming I/O.
func EncodeFile(inputPath, outputPath string, options EncodeOptions) (*EncodeResult, error) {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed opening input file: %w", err)
	}
	defer inFile.Close()

	stat, err := inFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed reading file stats: %w", err)
	}

	if options.Filename == "" {
		options.Filename = filepath.Base(inputPath)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed creating output file: %w", err)
	}
	defer outFile.Close()

	return EncodeStream(inFile, stat.Size(), outFile, options)
}

// DecodeFile decodes a PNG container from disk to disk with streaming I/O.
func DecodeFile(inputPath, outputPath string) (*DecodeResult, error) {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed opening PNG file: %w", err)
	}
	defer inFile.Close()

	// If outputPath is not provided, decode to temporary or target
	var outFile *os.File
	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return nil, err
		}
		outFile, err = os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed creating output file: %w", err)
		}
		defer outFile.Close()
	} else {
		var buf bytes.Buffer
		res, err := DecodeStream(inFile, &buf)
		if err != nil {
			return nil, err
		}
		res.Data = buf.Bytes()
		return res, nil
	}

	return DecodeStream(inFile, outFile)
}

// Helper chunk reader for streaming
type rawChunkHeader struct {
	Length uint32
	Type   string
}

type streamingChunkReader struct {
	r io.Reader
}

func newStreamingChunkReader(r io.Reader) *streamingChunkReader {
	return &streamingChunkReader{r: r}
}

func (scr *streamingChunkReader) nextChunkHeader() (*rawChunkHeader, error) {
	hdr := make([]byte, 8)
	if _, err := io.ReadFull(scr.r, hdr); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(hdr[0:4])
	chunkType := string(hdr[4:8])
	return &rawChunkHeader{Length: length, Type: chunkType}, nil
}

func (scr *streamingChunkReader) skipCRC(dataLen uint32, chunkType string, data []byte) {
	crcBuf := make([]byte, 4)
	_, _ = io.ReadFull(scr.r, crcBuf)
}

func (scr *streamingChunkReader) discardChunkData(length uint32) error {
	discardBuf := make([]byte, 4096)
	remaining := int64(length + 4) // data + 4 bytes CRC
	for remaining > 0 {
		toRead := int64(len(discardBuf))
		if toRead > remaining {
			toRead = remaining
		}
		n, err := scr.r.Read(discardBuf[:toRead])
		if err != nil {
			return err
		}
		remaining -= int64(n)
	}
	return nil
}

// idatPayloadStream seamlessly concatenates sequential IDAT chunks into an io.Reader
type idatPayloadStream struct {
	scr             *streamingChunkReader
	currChunkRemain int64
	eof             bool
}

func newIDATPayloadStream(scr *streamingChunkReader, firstChunkLen uint32) *idatPayloadStream {
	return &idatPayloadStream{
		scr:             scr,
		currChunkRemain: int64(firstChunkLen),
	}
}

func (ips *idatPayloadStream) Read(p []byte) (int, error) {
	if ips.eof {
		return 0, io.EOF
	}

	for ips.currChunkRemain == 0 {
		// Read next chunk header
		chunk, err := ips.scr.nextChunkHeader()
		if err != nil {
			ips.eof = true
			return 0, err
		}
		if chunk.Type != "IDAT" {
			ips.eof = true
			return 0, io.EOF
		}
		ips.currChunkRemain = int64(chunk.Length)
	}

	toRead := int64(len(p))
	if toRead > ips.currChunkRemain {
		toRead = ips.currChunkRemain
	}

	n, err := ips.scr.r.Read(p[:toRead])
	ips.currChunkRemain -= int64(n)

	if ips.currChunkRemain == 0 {
		// Discard 4-byte CRC at end of this IDAT chunk
		crcBuf := make([]byte, 4)
		if _, crcErr := io.ReadFull(ips.scr.r, crcBuf); crcErr != nil {
			ips.eof = true
			return n, crcErr
		}
	}

	return n, err
}

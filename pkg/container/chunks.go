package container

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

// PNGSignature is the 8-byte magic header for all standard PNG files.
var PNGSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// Chunk represents a single PNG chunk.
type Chunk struct {
	Type string
	Data []byte
}

// BuildChunk creates a formatted PNG chunk byte slice with length, type, data, and CRC32.
func BuildChunk(chunkType string, data []byte) []byte {
	if len(chunkType) != 4 {
		panic("chunk type must be exactly 4 characters")
	}

	length := uint32(len(data))
	buf := make([]byte, 4+4+len(data)+4)

	// 4 bytes: Length
	binary.BigEndian.PutUint32(buf[0:4], length)

	// 4 bytes: Type
	copy(buf[4:8], []byte(chunkType))

	// Data
	copy(buf[8:8+len(data)], data)

	// 4 bytes: CRC32 calculated over Type + Data
	crc := crc32.ChecksumIEEE(buf[4 : 8+len(data)])
	binary.BigEndian.PutUint32(buf[8+len(data):], crc)

	return buf
}

// BuildIHDRChunk creates a standard 13-byte RGBA IHDR chunk.
func BuildIHDRChunk(width, height uint32) []byte {
	data := make([]byte, 13)
	binary.BigEndian.PutUint32(data[0:4], width)
	binary.BigEndian.PutUint32(data[4:8], height)
	data[8] = 8 // Bit depth: 8 bits per channel
	data[9] = 6 // Color type: 6 (RGBA)
	data[10] = 0 // Compression method: 0 (DEFLATE)
	data[11] = 0 // Filter method: 0 (standard)
	data[12] = 0 // Interlace method: 0 (no interlace)

	return BuildChunk("IHDR", data)
}

// BuildTextChunk creates a standard tEXt chunk with keyword and text value.
func BuildTextChunk(keyword, text string) []byte {
	var buf bytes.Buffer
	buf.WriteString(keyword)
	buf.WriteByte(0x00) // Null separator
	buf.WriteString(text)
	return BuildChunk("tEXt", buf.Bytes())
}

// BuildIENDChunk creates the standard terminal IEND chunk.
func BuildIENDChunk() []byte {
	return BuildChunk("IEND", nil)
}

// ParsePNGChunks parses and verifies all chunks in a valid PNG stream.
func ParsePNGChunks(pngData []byte) ([]Chunk, error) {
	if len(pngData) < 8 {
		return nil, fmt.Errorf("file too short to be a valid PNG (%d bytes)", len(pngData))
	}

	if !bytes.Equal(pngData[:8], PNGSignature) {
		return nil, fmt.Errorf("invalid PNG signature")
	}

	var chunks []Chunk
	offset := 8

	for offset < len(pngData) {
		if offset+8 > len(pngData) {
			return nil, fmt.Errorf("truncated chunk header at offset %d", offset)
		}

		length := binary.BigEndian.Uint32(pngData[offset : offset+4])
		chunkType := string(pngData[offset+4 : offset+8])
		dataStart := offset + 8
		dataEnd := dataStart + int(length)
		crcStart := dataEnd
		crcEnd := crcStart + 4

		if crcEnd > len(pngData) {
			return nil, fmt.Errorf("truncated chunk data or CRC for chunk '%s' at offset %d", chunkType, offset)
		}

		data := pngData[dataStart:dataEnd]
		expectedCRC := binary.BigEndian.Uint32(pngData[crcStart:crcEnd])
		actualCRC := crc32.ChecksumIEEE(pngData[offset+4 : dataEnd])

		if expectedCRC != actualCRC {
			return nil, fmt.Errorf("CRC mismatch for chunk '%s': expected 0x%08X, got 0x%08X", chunkType, expectedCRC, actualCRC)
		}

		chunks = append(chunks, Chunk{
			Type: chunkType,
			Data: data,
		})

		offset = crcEnd
		if chunkType == "IEND" {
			break
		}
	}

	return chunks, nil
}

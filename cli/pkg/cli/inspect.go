package cli

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pixelfile/pkg/container"
)

func RunInspect(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: missing input PNG file for 'inspect'")
		fmt.Fprintln(os.Stderr, "Usage: pixelfile inspect <image.png>")
		os.Exit(1)
	}

	inputFile := args[0]
	pngData, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading image '%s': %v\n", inputFile, err)
		os.Exit(1)
	}

	chunks, err := container.ParsePNGChunks(pngData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: '%s' is not a valid PNG container: %v\n", inputFile, err)
		os.Exit(1)
	}

	var (
		ihdrData []byte
		metaText string
		idatSize int64
	)

	chunkCounts := make(map[string]int)
	for _, chunk := range chunks {
		chunkCounts[chunk.Type]++
		switch chunk.Type {
		case "IHDR":
			ihdrData = chunk.Data
		case "tEXt":
			nullIdx := bytes.IndexByte(chunk.Data, 0x00)
			if nullIdx > 0 && string(chunk.Data[:nullIdx]) == "PixelFile" {
				metaText = string(chunk.Data[nullIdx+1:])
			}
		case "IDAT":
			idatSize += int64(len(chunk.Data))
		}
	}

	fmt.Println()
	fmt.Printf("🔍 PixelFile Container Inspection: %s\n", filepath.Base(inputFile))
	fmt.Println("═════════════════════════════════════════════════════════")

	if len(ihdrData) >= 8 {
		w := binary.BigEndian.Uint32(ihdrData[0:4])
		h := binary.BigEndian.Uint32(ihdrData[4:8])
		fmt.Printf("  Image Dimensions:  %dx%d pixels (%d total pixels)\n", w, h, uint64(w)*uint64(h))
		fmt.Printf("  Color Space:       32-bit RGBA (8-bit per channel)\n")
	}

	fmt.Printf("  Container Size:    %s (%d bytes)\n", FormatBytes(int64(len(pngData))), len(pngData))
	fmt.Printf("  IDAT Payload:      %s (%d bytes)\n", FormatBytes(idatSize), idatSize)

	if metaText != "" {
		var meta container.Metadata
		if err := json.Unmarshal([]byte(metaText), &meta); err == nil {
			fmt.Println("─────────────────────────────────────────────────────────")
			fmt.Printf("  Embedded Filename: %s\n", meta.Filename)
			fmt.Printf("  Original Size:     %s (%d bytes)\n", FormatBytes(meta.ByteLength), meta.ByteLength)
			fmt.Printf("  MIME Type:         %s\n", meta.MimeType)
			fmt.Printf("  Compression Mode:  %s\n", meta.Compression)
			fmt.Printf("  Aspect Ratio:      %s\n", meta.AspectRatio)
			fmt.Printf("  SHA-256 Checksum:  %s\n", meta.SHA256)
			if meta.Timestamp > 0 {
				t := time.UnixMilli(meta.Timestamp)
				fmt.Printf("  Encoded At:        %s\n", t.Format(time.RFC1123))
			}
			overhead := int64(len(pngData)) - meta.ByteLength
			deltaPct := 0.0
			if meta.ByteLength > 0 {
				deltaPct = (float64(overhead) / float64(meta.ByteLength)) * 100.0
			}
			fmt.Printf("  Size Overhead:     %+.2f%% (%+d bytes)\n", deltaPct, overhead)
		}
	} else {
		fmt.Println("  ⚠️  Notice: No 'PixelFile' metadata chunk found (may be a generic PNG).")
	}

	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Print("  Chunk Structure:   ")
	for t, count := range chunkCounts {
		fmt.Printf("[%s: %d] ", t, count)
	}
	fmt.Println()
	fmt.Println("═════════════════════════════════════════════════════════")
}

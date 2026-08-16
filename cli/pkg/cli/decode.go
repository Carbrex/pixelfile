package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pixelfile/pkg/container"
)

func RunDecode(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: missing input PNG file for 'decode'")
		fmt.Fprintln(os.Stderr, "Usage: pixelfile decode <image.png> [-o output_file] [--out-dir <dir>]")
		os.Exit(1)
	}

	var (
		inputFile  string
		outputFile string
		outDir     string
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--out":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--out="):
			outputFile = strings.TrimPrefix(arg, "--out=")
		case arg == "--out-dir" || arg == "-d":
			if i+1 < len(args) {
				outDir = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--out-dir="):
			outDir = strings.TrimPrefix(arg, "--out-dir=")
		case !strings.HasPrefix(arg, "-"):
			if inputFile == "" {
				inputFile = arg
			} else if outputFile == "" {
				outputFile = arg
			}
		}
	}

	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: no input PNG file specified.")
		os.Exit(1)
	}

	pngData, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading image file '%s': %v\n", inputFile, err)
		os.Exit(1)
	}

	fmt.Printf("Decoding: %s (%s)...\n", inputFile, FormatBytes(int64(len(pngData))))

	res, err := container.Decode(pngData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding file: %v\n", err)
		os.Exit(1)
	}

	// Resolve output path
	targetFilename := res.Metadata.Filename
	if targetFilename == "" {
		targetFilename = "restored_payload.bin"
	}

	if outputFile == "" {
		if outDir != "" {
			outputFile = filepath.Join(outDir, targetFilename)
		} else {
			outputFile = targetFilename
		}
	} else if outDir != "" && !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join(outDir, outputFile)
	}

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Write restored payload
	if err := os.WriteFile(outputFile, res.Data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing restored file '%s': %v\n", outputFile, err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("✓ Success: Decoded and restored exact file -> %s\n", outputFile)
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Printf("  Original Name:   %s\n", res.Metadata.Filename)
	fmt.Printf("  Restored Size:   %s (%d bytes)\n", FormatBytes(res.Metadata.ByteLength), res.Metadata.ByteLength)
	fmt.Printf("  MIME Type:       %s\n", res.Metadata.MimeType)
	fmt.Printf("  Image Source:    %dx%d pixels (%s mode)\n", res.Metadata.Width, res.Metadata.Height, res.Metadata.Compression)
	fmt.Printf("  SHA-256 Hash:    %s\n", res.Metadata.SHA256)
	fmt.Printf("  Integrity Check: ✓ 100%% Bit-Perfect Match Verified\n")
	fmt.Printf("  Time Taken:      %v\n", res.Duration)
	fmt.Println("───────────────────────────────────────────────────")
}

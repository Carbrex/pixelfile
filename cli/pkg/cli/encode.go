package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pixelfile/pkg/container"
)

func RunEncode(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: missing input file for 'encode'")
		fmt.Fprintln(os.Stderr, "Usage: pixelfile encode <input_file> [-o output.png] [--mode auto|stored|deflate] [--ratio 1:1|16:9]")
		os.Exit(1)
	}

	var (
		inputFile   string
		outputFile  string
		mode        = "auto"
		aspectRatio = "1:1"
	)

	// Parse arguments
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
		case arg == "-m" || arg == "--mode":
			if i+1 < len(args) {
				mode = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--mode="):
			mode = strings.TrimPrefix(arg, "--mode=")
		case arg == "-r" || arg == "--ratio":
			if i+1 < len(args) {
				aspectRatio = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--ratio="):
			aspectRatio = strings.TrimPrefix(arg, "--ratio=")
		case !strings.HasPrefix(arg, "-"):
			if inputFile == "" {
				inputFile = arg
			} else if outputFile == "" {
				outputFile = arg
			}
		}
	}

	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: no input file specified.")
		os.Exit(1)
	}

	fileInfo, err := os.Stat(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error inspecting file '%s': %v\n", inputFile, err)
		os.Exit(1)
	}

	baseName := filepath.Base(inputFile)
	if outputFile == "" {
		outputFile = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".png"
		if outputFile == inputFile {
			outputFile = inputFile + ".png"
		}
	}

	opts := container.EncodeOptions{
		Filename:        baseName,
		AspectRatio:     aspectRatio,
		CompressionMode: mode,
	}

	fileSize := fileInfo.Size()
	fmt.Printf("Encoding: %s (%s)...\n", inputFile, FormatBytes(fileSize))

	var res *container.EncodeResult

	// Use disk-to-disk streaming for large files (> 16 MB)
	if fileSize > 16*1024*1024 {
		res, err = container.EncodeFile(inputFile, outputFile, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error streaming encode: %v\n", err)
			os.Exit(1)
		}
	} else {
		data, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", inputFile, err)
			os.Exit(1)
		}
		res, err = container.Encode(data, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding file: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(outputFile, res.PNGData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output PNG '%s': %v\n", outputFile, err)
			os.Exit(1)
		}
	}

	fmt.Println()
	fmt.Printf("✓ Success: Encoded into lossless image -> %s\n", outputFile)
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Printf("  Original File:   %s (%s / %d bytes)\n", baseName, FormatBytes(res.OriginalSize), res.OriginalSize)
	fmt.Printf("  Output Image:    %s (%s / %d bytes)\n", filepath.Base(outputFile), FormatBytes(res.OutputSize), res.OutputSize)
	fmt.Printf("  Image Size:      %dx%d pixels (%s)\n", res.Metadata.Width, res.Metadata.Height, res.Metadata.AspectRatio)
	fmt.Printf("  Compression:     %s\n", res.Metadata.Compression)
	fmt.Printf("  Size Change:     %+.2f%% (%+d bytes)\n", res.RatioPercent, res.OverheadBytes)
	if res.Metadata.SHA256 != "" {
		fmt.Printf("  SHA-256:         %s\n", res.Metadata.SHA256)
	}
	fmt.Printf("  Time Taken:      %v\n", res.Duration)
	fmt.Println("───────────────────────────────────────────────────")
}

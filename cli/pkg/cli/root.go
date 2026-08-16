package cli

import (
	"fmt"
	"os"
)

const Version = "1.0.0"

func PrintUsage() {
	fmt.Printf(`PixelFile (Go) v%s - Universal Lossless File-to-Image Container

Usage:
  pixelfile <command> [arguments] [flags]

Commands:
  encode <file> [output.png]   Convert any file into a lossless PNG image
  decode <image.png> [output]  Restore original file from PNG with SHA-256 verification
  inspect <image.png>          Inspect metadata, dimensions, and checksum of a PNG
  verify <file> <image.png>    Verify bit-perfect equivalence between file and image
  benchmark <files...>         Benchmark compression ratio, speed, and overhead
  version                      Show version information

Flags for 'encode':
  -m, --mode <auto|stored|deflate>  Compression mode (default: auto)
  -r, --ratio <1:1|16:9>            Aspect ratio (default: 1:1)
  -o, --out <path>                  Output file path

Examples:
  pixelfile encode document.pdf
  pixelfile encode archive.zip -o archive.png --ratio 16:9
  pixelfile decode archive.png
  pixelfile inspect document.png
  pixelfile verify document.pdf document.png
  pixelfile benchmark doc.pdf data.json binary.exe

`, Version)
}

func Execute(args []string) {
	if len(args) < 1 {
		PrintUsage()
		os.Exit(0)
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "encode":
		RunEncode(cmdArgs)
	case "decode":
		RunDecode(cmdArgs)
	case "inspect":
		RunInspect(cmdArgs)
	case "verify":
		RunVerify(cmdArgs)
	case "benchmark":
		RunBenchmark(cmdArgs)
	case "version", "-v", "--version":
		fmt.Printf("PixelFile v%s (linux/amd64 Go)\n", Version)
	case "help", "-h", "--help":
		PrintUsage()
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n\n", command)
		PrintUsage()
		os.Exit(1)
	}
}

// FormatBytes formats byte count into human readable string.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

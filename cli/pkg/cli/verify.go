package cli

import (
	"bytes"
	"fmt"
	"os"

	"pixelfile/pkg/container"
)

func RunVerify(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: missing arguments for 'verify'")
		fmt.Fprintln(os.Stderr, "Usage: pixelfile verify <original_file> <image.png>")
		os.Exit(1)
	}

	origPath := args[0]
	pngPath := args[1]

	origData, err := os.ReadFile(origPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading original file '%s': %v\n", origPath, err)
		os.Exit(1)
	}

	pngData, err := os.ReadFile(pngPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading PNG image '%s': %v\n", pngPath, err)
		os.Exit(1)
	}

	origHash := container.ComputeSHA256(origData)
	fmt.Printf("Verifying '%s' against '%s'...\n", origPath, pngPath)

	decResult, err := container.Decode(pngData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ FAILED: Decoding PNG failed: %v\n", err)
		os.Exit(1)
	}

	decHash := container.ComputeSHA256(decResult.Data)

	fmt.Println()
	if bytes.Equal(origData, decResult.Data) && origHash == decHash {
		fmt.Println("✓ VERIFICATION SUCCESSFUL: 100% Bit-Perfect Match!")
		fmt.Println("───────────────────────────────────────────────────")
		fmt.Printf("  Original File Size: %s (%d bytes)\n", FormatBytes(int64(len(origData))), len(origData))
		fmt.Printf("  Decoded Data Size:  %s (%d bytes)\n", FormatBytes(int64(len(decResult.Data))), len(decResult.Data))
		fmt.Printf("  SHA-256 Hash:       %s\n", origHash)
		fmt.Printf("  Status:             Lossless & Identical\n")
		fmt.Println("───────────────────────────────────────────────────")
	} else {
		fmt.Println("✗ VERIFICATION FAILED: Data Mismatch!")
		fmt.Println("───────────────────────────────────────────────────")
		fmt.Printf("  Original Hash: %s (%d bytes)\n", origHash, len(origData))
		fmt.Printf("  Decoded Hash:  %s (%d bytes)\n", decHash, len(decResult.Data))
		fmt.Println("───────────────────────────────────────────────────")
		os.Exit(1)
	}
}

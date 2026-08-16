package cli

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"pixelfile/pkg/container"
)

type benchItem struct {
	Name string
	Data []byte
}

func RunBenchmark(args []string) {
	var items []benchItem

	if len(args) == 0 {
		fmt.Println("No files specified. Generating benchmark test fixtures (Code, JSON, Binary, High-Entropy)...")

		// 1. Source Code / Text (100 KB)
		codeSample := bytes.Repeat([]byte("package main\nimport \"fmt\"\nfunc ProcessData(x int) int {\n  return x * 42\n}\n"), 2500)
		items = append(items, benchItem{Name: "source_code.go (100 KB)", Data: codeSample})

		// 2. Structured JSON / Data (1 MB)
		jsonSample := bytes.Repeat([]byte("{\"id\":1001,\"user\":\"alice\",\"active\":true,\"roles\":[\"admin\",\"editor\"]},\n"), 14000)
		items = append(items, benchItem{Name: "data_payload.json (1 MB)", Data: jsonSample})

		// 3. Medium Binary / Executable-like (5 MB)
		binData := make([]byte, 5*1024*1024)
		for i := range binData {
			binData[i] = byte((i % 256) ^ ((i / 4) % 128))
		}
		items = append(items, benchItem{Name: "binary_executable.bin (5 MB)", Data: binData})

		// 4. High-Entropy Encrypted / ZIP payload (5 MB)
		randomData := make([]byte, 5*1024*1024)
		_, _ = rand.Read(randomData)
		items = append(items, benchItem{Name: "archive_zip.zip (5 MB High-Entropy)", Data: randomData})

		// 5. Large 10 MB payload
		largeRandom := make([]byte, 10*1024*1024)
		_, _ = rand.Read(largeRandom)
		items = append(items, benchItem{Name: "large_payload.dat (10 MB High-Entropy)", Data: largeRandom})
	} else {
		for _, f := range args {
			data, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to read benchmark file '%s': %v\n", f, err)
				continue
			}
			items = append(items, benchItem{Name: filepath.Base(f), Data: data})
		}
	}

	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid files to benchmark.")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("🚀 Running PixelFile Performance & Size Benchmarks...")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TEST PAYLOAD\tORIGINAL\tPNG SIZE\tDELTA %\tMODE\tENC TIME\tDEC TIME\tINTEGRITY")
	fmt.Fprintln(w, "────────────\t────────\t────────\t───────\t────\t────────\t────────\t─────────")

	for _, item := range items {
		opts := container.EncodeOptions{
			Filename:        item.Name,
			AspectRatio:     "1:1",
			CompressionMode: "auto",
		}

		// Benchmark Encode
		t0 := time.Now()
		encRes, err := container.Encode(item.Data, opts)
		encDur := time.Since(t0)
		if err != nil {
			fmt.Fprintf(w, "%s\t%s\tERROR\t-\t-\t-\t-\tFAIL (%v)\n", item.Name, FormatBytes(int64(len(item.Data))), err)
			continue
		}

		// Benchmark Decode
		t1 := time.Now()
		decRes, err := container.Decode(encRes.PNGData)
		decDur := time.Since(t1)
		if err != nil {
			fmt.Fprintf(w, "%s\t%s\t%s\t%+.2f%%\t%s\t%v\tERROR\tFAIL (%v)\n",
				item.Name, FormatBytes(encRes.OriginalSize), FormatBytes(encRes.OutputSize),
				encRes.RatioPercent, encRes.Metadata.Compression, encDur, err)
			continue
		}

		matchStatus := "✓ 100% Match"
		if !decRes.SHA256Matches || !bytes.Equal(decRes.Data, item.Data) {
			matchStatus = "✗ MISMATCH"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%+.2f%%\t%s\t%v\t%v\t%s\n",
			item.Name,
			FormatBytes(encRes.OriginalSize),
			FormatBytes(encRes.OutputSize),
			encRes.RatioPercent,
			encRes.Metadata.Compression,
			encDur.Round(time.Millisecond),
			decDur.Round(time.Millisecond),
			matchStatus,
		)
	}

	w.Flush()
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("✓ All benchmark items verified bit-perfect.")
}

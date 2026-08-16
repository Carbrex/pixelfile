# PixelFile CLI (Go)

> **High-Performance Universal Lossless File-to-Image Container**  
> Standalone CLI tool written in Go to convert any file (PDF, binary, source code, ZIP, executable, video) into a lossless standard PNG image and restore it with 100% cryptographic SHA-256 validation.

---

## ⚡ Highlights

* **Near-Zero Inflation on High-Entropy Data:** Uses uncompressed DEFLATE stored blocks (`level = 0`) to guarantee $\le 0.03\%$ overhead on already compressed files (ZIP, MP4, PDF, encrypted).
* **Streaming Disk-to-Disk I/O:** Encodes/decodes **1 GB to 50 GB+ files** with constant **< 4 MB RAM** footprint.
* **Massive Shrinkage on Low-Entropy Data:** Shrinks source code, JSON, logs, and uncompressed binaries by **-50% to -99%**.
* **Standard PNG Compliance:** Output `.png` images open in standard image viewers (Preview, Chrome, GIMP, Photoshop) as pixel noise maps.
* **100% Bit-Perfect Recovery:** Restores original bytes with SHA-256 verification and automatic padding truncation.
* **Metadata Preservation:** Encodes original filename, MIME type, byte length, and creation timestamp into standard PNG `tEXt` chunks.

---

## 🚀 Building & Installing

Ensure you have **Go 1.21+** installed:

```bash
cd cli

# Build local binary
go build -o pixelfile ./cmd/pixelfile

# (Optional) Install globally to $GOPATH/bin
go install ./cmd/pixelfile
```

---

## 🛠️ Command Reference

```text
Usage:
  pixelfile <command> [arguments] [flags]

Commands:
  encode <file> [output.png]   Convert any file into a lossless PNG image
  decode <image.png> [output]  Restore original file from PNG with SHA-256 verification
  inspect <image.png>          Inspect metadata, dimensions, and checksum of a PNG
  verify <file> <image.png>    Verify bit-perfect equivalence between file and image
  benchmark [files...]         Benchmark compression ratio, speed, and overhead
  version                      Show version information
```

---

### 1. `encode`
Convert any file into a standard PNG image:

```bash
# Auto mode (Square 1:1 aspect ratio)
./pixelfile encode document.pdf

# Custom output file and widescreen 16:9 aspect ratio
./pixelfile encode archive.zip -o archive.png --ratio 16:9

# Force uncompressed stored blocks (Level 0)
./pixelfile encode payload.bin --mode stored

# Force maximum DEFLATE compression (Level 9)
./pixelfile encode source_code.tar --mode deflate
```

#### Flags:
* `-o, --out <path>`: Specify destination `.png` path.
* `-m, --mode <auto|stored|deflate>`: Compression strategy (`auto` default).
* `-r, --ratio <1:1|16:9>`: Target aspect ratio.

---

### 2. `inspect`
Inspect PNG container metadata and dimensions without decoding or extracting the file:

```bash
./pixelfile inspect document.png
```

*Example Output:*
```text
🔍 PixelFile Container Inspection: document.png
═════════════════════════════════════════════════════════
  Image Dimensions:  5173x5172 pixels (26,754,756 total pixels)
  Color Space:       32-bit RGBA (8-bit per channel)
  Container Size:    102.09 MB (107,052,268 bytes)
  IDAT Payload:      102.07 MB (107,032,377 bytes)
─────────────────────────────────────────────────────────
  Embedded Filename: document.pdf
  Original Size:     102.05 MB (107,002,769 bytes)
  MIME Type:         application/pdf
  Compression Mode:  stored
  Aspect Ratio:      1:1
  SHA-256 Checksum:  3cbdb9647972e0c2946903ccbd6f615a2dbeb...
  Encoded At:        Sun, 16 Aug 2026 16:36:44 IST
  Size Overhead:     +0.03% (+29,891 bytes)
─────────────────────────────────────────────────────────
  Chunk Structure:   [IHDR: 1] [tEXt: 1] [IDAT: 1634] [IEND: 1] 
═════════════════════════════════════════════════════════
```

---

### 3. `verify`
Verify that a `.png` container matches the exact original file byte-for-byte and hash-for-hash:

```bash
./pixelfile verify document.pdf document.png
```

*Example Output:*
```text
✓ VERIFICATION SUCCESSFUL: 100% Bit-Perfect Match!
───────────────────────────────────────────────────
  Original File Size: 102.05 MB (107,002,769 bytes)
  Decoded Data Size:  102.05 MB (107,002,769 bytes)
  SHA-256 Hash:       3cbdb9647972e0c2946903ccbd6f615a2dbeb...
  Status:             Lossless & Identical
───────────────────────────────────────────────────
```

---

### 4. `decode`
Extract and reconstruct the original binary file from the PNG:

```bash
# Restore with original filename stored in metadata
./pixelfile decode document.png

# Restore to specific output path
./pixelfile decode document.png -o restored_document.pdf

# Restore into a specific directory
./pixelfile decode document.png --out-dir ./recovered/
```

---

### 5. `benchmark`
Run automated throughput and compression ratio benchmarks across synthetic or user-supplied files:

```bash
# Built-in multi-type benchmark suite (Code, JSON, Binary, High-Entropy Archive)
./pixelfile benchmark

# Benchmark custom files
./pixelfile benchmark file1.pdf file2.zip file3.exe
```

---

## 🧪 Running Tests

```bash
go test -v ./tests/...
```

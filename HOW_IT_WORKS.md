# PixelFile: Under the Hood & Deep Architecture Guide

> **A comprehensive explanation of how arbitrary files are mapped losslessly into standard PNG images with near-zero overhead, streaming I/O, and cryptographic integrity verification.**

---

## 1. The Core Concept: Files & Images Are Both Byte Streams

To a computer, every file—whether a PDF, ZIP archive, binary executable, MP4 video, or source code—is simply a contiguous sequence of 8-bit bytes (values 0 to 255):

```text
Byte:  [ 0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x37 ... ]
Value: [  37,   80,   68,   70,   45,   49,   46,   55  ... ]
```

A **32-bit RGBA image** is also just a sequence of bytes, where every 4 consecutive bytes define one pixel:

```text
Pixel 0:  Red = 37,  Green = 80,  Blue = 68,  Alpha = 70   (0x25504446)
Pixel 1:  Red = 45,  Green = 49,  Blue = 46,  Alpha = 55   (0x2D312E37)
```

Because 4 bytes match a 32-bit CPU word (`uint32`), this 4-byte-per-pixel mapping provides:
1. **Zero Data Distortion:** 1 byte maps to 1 color channel.
2. **Instant Memory Copy:** CPU and GPU memory buffers align to 32-bit boundaries for maximum memory throughput.
3. **No Prime Factorization Bottlenecks:** No 3-byte RGB alignment issues.

---

## 2. Solving the "Size Inflation" Problem: Stored Blocks

### Why standard PNGs inflate compressed files
PNG uses the **DEFLATE** compression standard (LZ77 dictionary matching + Huffman encoding) combined with per-row scanline filtering (Sub, Up, Average, Paeth).

When you feed **already-compressed or high-entropy data** (ZIP, MP4, PDF, encrypted files) into a standard PNG compressor:
* There are no recurring patterns or visual gradients.
* Scanline filters waste CPU attempting to find patterns in noise.
* Huffman tables and block headers add overhead, expanding the file size by **5% to 20%**.

### The Breakthrough: Uncompressed DEFLATE Stored Blocks (Level 0)
The DEFLATE standard (`RFC 1951`) defines three block types:
* `BTYPE = 01`: Compressed with fixed Huffman codes.
* `BTYPE = 10`: Compressed with dynamic Huffman codes.
* **`BTYPE = 00`: Stored / Uncompressed raw blocks.**

In Stored Block mode, the compressor writes the raw data with just a **5-byte header per 65,535 bytes**:

```text
┌──────────────┬──────────────────┬──────────────────┬────────────────────────┐
│ BFINAL (1 B) │ LEN (2 B uint16) │ NLEN (2 B uint16)│ Raw Data (up to 65535B)│
└──────────────┴──────────────────┴──────────────────┴────────────────────────┘
```

$$\text{Framing Overhead} = \frac{5\text{ bytes}}{65,535\text{ bytes}} = 0.00762\%$$

### The Adaptive Compression Engine
PixelFile uses an **Adaptive Dual-Mode Engine**:

```text
Input File ───► Run DEFLATE (Level 6) & Stored Blocks (Level 0)
                      │
                      ├──► If DEFLATE is smaller (Text, Code, Logs)   ──► Use DEFLATE (-50% to -99%)
                      └──► If Stored is smaller (ZIP, PDF, Video, EXE) ──► Use Stored (+0.02% to +0.05%)
```

**Real Result on 102 MB ZIP:**
* Original Size: `102.05 MB` (`107,002,769 bytes`)
* PNG Image Size: `102.07 MB` (`107,032,660 bytes`)
* Size Change: **`+0.03%` (only +29.8 KB across 107 MB!)**

---

## 3. PNG Format & Binary Chunks Anatomy

A valid PNG file is structured as an 8-byte signature followed by sequential chunks:

```text
┌─────────────────┬───────────────────┬──────────────────────┬───────────────────┐
│ Length (4B BE)  │ Chunk Type (4B)   │ Payload (N bytes)    │ CRC-32 (4B IEEE)  │
└─────────────────┴───────────────────┴──────────────────────┴───────────────────┘
```

PixelFile constructs the entire PNG container from raw binary primitives:

```text
1. Signature:   89 50 4E 47 0D 0A 1A 0A  (Standard PNG magic header)
2. IHDR Chunk:  Specifies Width, Height, BitDepth=8, ColorType=6 (RGBA), Filter=0
3. tEXt Chunk:  Keyword "PixelFile" + JSON Metadata:
                {
                  "version": 1,
                  "filename": "document.pdf",
                  "byteLength": 107002769,
                  "sha256": "3cbdb9647972e0c2...",
                  "compression": "stored",
                  "aspectRatio": "1:1",
                  "timestamp": 1771234604000
                }
4. IDAT Chunk:  Holds zlib-wrapped scanline payload (Filter 0x00 per row + RGBA pixels)
5. IEND Chunk:  Signals EOF
```

### Why Standard PNG Ancillary Chunks?
* Normal image viewers ignore `tEXt` chunks and display the pixel static.
* The PixelFile decoder parses `tEXt` to recover the original filename, exact byte count, and SHA-256 hash.

---

## 4. Geometry & Aspect Ratio Math

To avoid creating a $1\text{px} \times 10,000,000\text{px}$ image that crashes operating system image renderers:

1. **Calculate total pixels:**  
   `Pixels = ceil(TotalBytes / 4)`
2. **Calculate square width (1:1):**  
   `Width = ceil(sqrt(Pixels))`
3. **Calculate height:**  
   `Height = ceil(Pixels / Width)`  
   *(If `Width * Height < Pixels`, increment `Height`)*
4. **Padding:**  
   The final row is padded with `(Width * Height * 4) - TotalBytes` zero bytes to complete the grid.
5. **Lossless Recovery:**  
   The decoder slices the buffer to the exact `ByteLength` saved in metadata, stripping all padding.

---

## 5. Streaming Disk-to-Disk Architecture (1 GB to 50 GB+)

When encoding a 1 GB file, loading the whole file, scanlines, and compressed buffers into RAM requires $\approx 3\text{ GB}$ of memory.

PixelFile solves this with a **Row-by-Row Streaming Pipeline**:

```text
[Input File on Disk]
       │
       ▼ (Read 1 row at a time: ~64 KB)
[Row Buffer + 0x00 Filter Prefix]
       │
       ▼ (Pipe through zlib compressor & compute SHA-256)
[idatChunkWriter (Buffers 64 KB of compressed stream)]
       │
       ▼ (Flush standard [Length][IDAT][Data][CRC-32] chunk to disk)
[Output .png File on Disk]
```

* **RAM Footprint:** Stays constant at **`< 4 MB RAM`**, regardless of whether the file is 100 MB, 1 GB, or 50 GB.
* **Throughput:** Operates at raw NVMe disk speed ($> 150\text{ MB/s}$).

---

## 6. Web Studio & Client-Side Engine

The Web Studio runs 100% client-side with zero server dependencies:

1. **Hardware-Accelerated SHA-256:** Uses the browser's native WebCrypto API (`crypto.subtle.digest`).
2. **Background Web Worker (`worker.js`):** Runs CPU-heavy encoding and decoding on a background worker thread using zero-copy Transferable ArrayBuffers, keeping the UI at 60 fps.
3. **Optimized Adler-32 Checksum:**
   Instead of running `% 65521` on every single byte (which took 12 seconds for 107 million bytes in JS), sums are accumulated in **5,552-byte blocks** before applying modulo:
   $$\text{Block Accumulation: } 12,000\text{ ms} \longrightarrow 15\text{ ms}$$
4. **Subsampled Canvas Rendering:**
   When visualizing a 26.7 million pixel image ($5173 \times 5172$), the viewport dynamically subsamples pixels for smooth pan & zoom without overloading GPU texture memory, while generating the full-resolution PNG for download.

---

## 7. Decoding & Verification: Zero Corruption Assurance

```text
Input .png ──► Validate PNG Signature & CRC-32 Chunks
                    │
                    ▼
               Parse IHDR & tEXt Metadata (PixelFile Header)
                    │
                    ▼
               Decompress IDAT Payload Row-by-Row
                    │
                    ▼
               Strip Scanline Filter Byte (0x00) & Truncate to exact ByteLength
                    │
                    ▼
               Compute SHA-256 Checksum
                    │
           ┌────────┴────────┐
           ▼                 ▼
     Hash Matches?     Hash Mismatch?
           │                 │
           ▼                 ▼
     Write Restored      Throw Error &
         File          Abort Restoration
```

---

## 8. Summary of Technical Benchmarks

| Feature | PixelFile Implementation | Typical Existing Tools |
| :--- | :--- | :--- |
| **High-Entropy Overhead (ZIP, PDF, MP4)** | **+0.02% to +0.05%** | +5% to +20% (or uncompressed BMP/PPM) |
| **Low-Entropy Shrinkage (Code, Text, Logs)**| **-50% to -99%** | Often 0% (uncompressed) |
| **Metadata Preservation** | Embedded in PNG `tEXt` chunks | Lost or tacked onto raw bytes |
| **Integrity Assurance** | SHA-256 Checksum on every decode | None (silent corruption) |
| **Large File Handling** | Streaming Disk-to-Disk (< 4MB RAM) | In-Memory (OOM crash on >1GB) |
| **Web Experience** | 100% Client-Side Web Worker + Canvas | Server-dependent or CLI only |

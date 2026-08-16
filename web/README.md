# PixelFile Studio (Web App)

> **100% Client-Side Universal File-to-Image Container & Interactive Visualizer**  
> Run directly in any modern browser without servers, installations, or backend APIs.

---

## 🔒 100% Client-Side Privacy Architecture

* **Zero Server Uploads:** Files are processed strictly inside browser memory using standard Web APIs (`FileReader`, `WebCrypto`, and HTML5 Canvas).
* **Hardware-Accelerated Cryptography:** Uses `crypto.subtle.digest('SHA-256')` for instantaneous integrity calculation.
* **Non-Blocking Web Worker:** Heavy file processing and compression run in a background Web Worker ([`worker.js`](worker.js)) so the UI never stutters or locks up.
* **Offline Compatible:** Works completely offline as static files, or hosted on GitHub Pages / static CDN.

---

## ✨ Features & Interface Guide

### 1. Encode & Visualize Tab
* **Drag-and-Drop Dropzone:** Drop any file (PDF, ZIP, source code, executable, binary, video).
* **Settings:**
  * **Aspect Ratio:** Toggle between Square (`1:1`) and Widescreen (`16:9`).
  * **Compression Strategy:** Adaptive (Auto-selects smallest), Stored Blocks (Zero inflation), or DEFLATE (Maximum compression).
* **Interactive Canvas Viewport:**
  * Zoom in (up to 3000%) and pan around the image to explore raw byte distributions.
  * Subsampled dynamic rendering prevents browser GPU overload on multi-gigabyte/100M-pixel textures.
* **Live Pixel Inspector:**
  * Hover over any pixel to view its exact `(X, Y)` coordinate, `RGBA(r, g, b, a)` byte channel values, `HEX` color swatch, and file byte offset.
* **One-Click Download:** Download the fully standard lossless `.png` image.

### 2. Decode & Restore Tab
* **PNG Dropzone:** Drop any PixelFile PNG image.
* **Metadata Extraction:** Automatically reads embedded original filename, MIME type, dimensions, and creation timestamp.
* **Integrity Validation:** Computes SHA-256 hash in real-time and validates bit-perfect match with a status badge.
* **Download Restored File:** Restores the exact binary file with its original filename and extension.

---

## 🚀 Running Locally

Serve the `web/` folder with any static HTTP server:

```bash
cd web

# Python
python3 -m http.server 3000

# Node.js / npx
npx serve .

# Bun
bun x serve .
```

Then visit `http://localhost:3000` in your web browser.

---

## 🌐 Deploying to GitHub Pages

Because PixelFile Web Studio is 100% static client-side JavaScript, you can host it for free on GitHub Pages:
1. In your GitHub repository settings, navigate to **Pages**.
2. Set source to **Deploy from a branch**.
3. Select `main` branch and folder `/web`.
4. Your Studio is now live worldwide with zero server maintenance!

# PixelFile Studio (Web App)

> **100% Client-Side Universal File-to-Image Container & Interactive Visualizer**  
> Run directly in any modern browser without servers, installations, or backend APIs.

---

## 🔒 100% Client-Side Privacy
* Zero remote server uploads.
* Files are processed strictly inside client memory using standard `FileReader`, `WebCrypto` (`crypto.subtle.digest`), and HTML5 `CanvasRenderingContext2D`.
* Works completely offline or when hosted on static services like GitHub Pages.

---

## ⚡ Features
* **Interactive Canvas Visualizer:** Explore raw file byte structures with pan & zoom controls.
* **Live Pixel Inspector:** Hover over any pixel to inspect its `(X, Y)` coordinate, `RGBA` byte values, `HEX` color, and byte offset.
* **Lossless PNG Encoding:** Adaptive Stored/DEFLATE compression modes.
* **Lossless Decoding & SHA-256 Verification:** Restores exact original files and verifies cryptographic checksums.

---

## 🚀 Running Locally

You can serve this folder with any static HTTP server:

```bash
# Python
python3 -m http.server 3000

# Node.js
npx serve .
```
Then open `http://localhost:3000` in your browser.

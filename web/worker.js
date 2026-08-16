/**
 * PixelFile Background Web Worker
 * Offloads heavy CPU encoding & decoding from main UI thread
 */

import { encodeFileToPNG, decodePNGToFile } from './engine.js';

self.onmessage = async (e) => {
  const { id, type, payload } = e.data;

  try {
    if (type === 'encode') {
      const { fileBytes, filename, options } = payload;
      const res = await encodeFileToPNG(fileBytes, filename, options);

      // Create preview sample for canvas (max 1024x1024 sample to avoid main thread GPU choke)
      const previewDim = Math.min(1024, Math.max(res.width, res.height));
      const sampleCanvasData = generatePreviewSample(fileBytes, res.width, res.height, previewDim);

      self.postMessage({
        id,
        success: true,
        result: {
          metadata: res.metadata,
          width: res.width,
          height: res.height,
          originalSize: res.originalSize,
          outputSize: res.outputSize,
          overheadBytes: res.overheadBytes,
          ratioPercent: res.ratioPercent,
          durationMs: res.durationMs,
          pngBytes: res.pngBytes,
          previewSample: sampleCanvasData,
          previewDim: previewDim,
        }
      }, [res.pngBytes.buffer, sampleCanvasData.buffer]);

    } else if (type === 'decode') {
      const { pngBytes } = payload;
      const res = await decodePNGToFile(pngBytes);

      self.postMessage({
        id,
        success: true,
        result: {
          data: res.data,
          metadata: res.metadata,
          width: res.width,
          height: res.height,
          sha256Matches: res.sha256Matches,
          actualHash: res.actualHash,
          durationMs: res.durationMs,
        }
      }, [res.data.buffer]);
    }
  } catch (err) {
    self.postMessage({
      id,
      success: false,
      error: err.message || String(err),
    });
  }
};

// Generates a responsive preview sample for the canvas viewport
function generatePreviewSample(fileBytes, width, height, sampleDim) {
  const sampleBuf = new Uint8Array(sampleDim * sampleDim * 4);
  const totalPixels = width * height;
  const step = Math.max(1, Math.floor(totalPixels / (sampleDim * sampleDim)));

  for (let i = 0; i < sampleDim * sampleDim; i++) {
    const pixelIdx = i * step;
    const byteOffset = pixelIdx * 4;
    const targetOffset = i * 4;

    if (byteOffset < fileBytes.length) {
      sampleBuf[targetOffset] = fileBytes[byteOffset] || 0;
      sampleBuf[targetOffset + 1] = fileBytes[byteOffset + 1] || 0;
      sampleBuf[targetOffset + 2] = fileBytes[byteOffset + 2] || 0;
      sampleBuf[targetOffset + 3] = fileBytes[byteOffset + 3] || 255;
    } else {
      sampleBuf[targetOffset + 3] = 0;
    }
  }
  return sampleBuf;
}

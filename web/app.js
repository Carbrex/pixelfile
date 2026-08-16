import { encodeFileToPNG, decodePNGToFile } from './engine.js';

// DOM Elements
const themeToggleBtn = document.getElementById('themeToggleBtn');
const tabEncode = document.getElementById('tabEncode');
const tabDecode = document.getElementById('tabDecode');
const panelEncode = document.getElementById('panelEncode');
const panelDecode = document.getElementById('panelDecode');

// Encode Elements
const encodeDropzone = document.getElementById('encodeDropzone');
const encodeFileInput = document.getElementById('encodeFileInput');
const aspectRatioSelect = document.getElementById('aspectRatioSelect');
const compressionSelect = document.getElementById('compressionSelect');
const encodeStatsCard = document.getElementById('encodeStatsCard');
const statOriginalSize = document.getElementById('statOriginalSize');
const statOutputSize = document.getElementById('statOutputSize');
const statDelta = document.getElementById('statDelta');
const statDimensions = document.getElementById('statDimensions');
const statSHA256 = document.getElementById('statSHA256');
const downloadPNGBtn = document.getElementById('downloadPNGBtn');

// Canvas Elements
const canvasViewport = document.getElementById('canvasViewport');
const pixelCanvas = document.getElementById('pixelCanvas');
const canvasEmptyPlaceholder = document.getElementById('canvasEmptyPlaceholder');
const canvasDimLabel = document.getElementById('canvasDimLabel');
const zoomInBtn = document.getElementById('zoomInBtn');
const zoomOutBtn = document.getElementById('zoomOutBtn');
const resetViewBtn = document.getElementById('resetViewBtn');
const zoomLevelLabel = document.getElementById('zoomLevelLabel');

// Inspector Elements
const inspectColorSwatch = document.getElementById('inspectColorSwatch');
const inspectCoord = document.getElementById('inspectCoord');
const inspectRGBA = document.getElementById('inspectRGBA');
const inspectHEX = document.getElementById('inspectHEX');
const inspectOffset = document.getElementById('inspectOffset');

// Decode Elements
const decodeDropzone = document.getElementById('decodeDropzone');
const decodeFileInput = document.getElementById('decodeFileInput');
const decodeResultCard = document.getElementById('decodeResultCard');
const decodeFilename = document.getElementById('decodeFilename');
const decodeSize = document.getElementById('decodeSize');
const decodeMime = document.getElementById('decodeMime');
const decodeDim = document.getElementById('decodeDim');
const decodeSHA256 = document.getElementById('decodeSHA256');
const downloadRestoredBtn = document.getElementById('downloadRestoredBtn');
const downloadRestoredLabel = document.getElementById('downloadRestoredLabel');

// State
let currentEncodedResult = null;
let currentDecodedResult = null;
let currentSourceFile = null;
let currentZoom = 1;

// Format Bytes
function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Format Hex
function toHex(n) {
  return n.toString(16).padStart(2, '0').toUpperCase();
}

// 1. Theme Management
function initTheme() {
  const savedTheme = localStorage.getItem('pixelfile_theme');
  if (savedTheme === 'light') {
    document.body.classList.remove('dark-theme');
  } else {
    document.body.classList.add('dark-theme');
  }
}

themeToggleBtn.addEventListener('click', () => {
  document.body.classList.toggle('dark-theme');
  const isDark = document.body.classList.contains('dark-theme');
  localStorage.setItem('pixelfile_theme', isDark ? 'dark' : 'light');
});

// 2. Tab Navigation
tabEncode.addEventListener('click', () => {
  tabEncode.classList.add('active');
  tabDecode.classList.remove('active');
  panelEncode.classList.add('active');
  panelDecode.classList.remove('active');
});

tabDecode.addEventListener('click', () => {
  tabDecode.classList.add('active');
  tabEncode.classList.remove('active');
  panelDecode.classList.add('active');
  panelEncode.classList.remove('active');
});

// 3. Dropzone Drag & Drop Setup
function setupDropzone(dropzoneEl, fileInputEl, onFileSelect) {
  dropzoneEl.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropzoneEl.classList.add('dragover');
  });

  ['dragleave', 'dragend'].forEach(type => {
    dropzoneEl.addEventListener(type, () => {
      dropzoneEl.classList.remove('dragover');
    });
  });

  dropzoneEl.addEventListener('drop', (e) => {
    e.preventDefault();
    dropzoneEl.classList.remove('dragover');
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      onFileSelect(e.dataTransfer.files[0]);
    }
  });

  fileInputEl.addEventListener('change', (e) => {
    if (e.target.files && e.target.files.length > 0) {
      onFileSelect(e.target.files[0]);
    }
  });
}

// 4. Encode Workflow
setupDropzone(encodeDropzone, encodeFileInput, async (file) => {
  currentSourceFile = file;
  await processEncodeFile();
});

aspectRatioSelect.addEventListener('change', async () => {
  if (currentSourceFile) await processEncodeFile();
});

compressionSelect.addEventListener('change', async () => {
  if (currentSourceFile) await processEncodeFile();
});

async function processEncodeFile() {
  if (!currentSourceFile) return;

  try {
    const arrayBuffer = await currentSourceFile.arrayBuffer();
    const fileBytes = new Uint8Array(arrayBuffer);
    const aspectRatio = aspectRatioSelect.value;
    const mode = compressionSelect.value;

    const result = await encodeFileToPNG(fileBytes, currentSourceFile.name, {
      aspectRatio,
      mode,
      mimeType: currentSourceFile.type || 'application/octet-stream',
    });

    currentEncodedResult = result;

    // Update Stats Card
    statOriginalSize.textContent = formatBytes(result.originalSize);
    statOutputSize.textContent = formatBytes(result.outputSize);
    const sign = result.ratioPercent > 0 ? '+' : '';
    statDelta.textContent = `${sign}${result.ratioPercent.toFixed(2)}%`;
    statDelta.style.color = result.ratioPercent <= 0 ? 'var(--success-text)' : 'var(--accent-primary)';
    statDimensions.textContent = `${result.width} × ${result.height} px`;
    statSHA256.textContent = result.metadata.sha256;
    encodeStatsCard.classList.remove('hidden');

    // Render Canvas
    canvasEmptyPlaceholder.classList.add('hidden');
    canvasDimLabel.textContent = `${result.width} × ${result.height} px`;
    pixelCanvas.width = result.width;
    pixelCanvas.height = result.height;

    const ctx = pixelCanvas.getContext('2d');
    const imgData = new ImageData(new Uint8ClampedArray(result.rgbaBuffer.buffer), result.width, result.height);
    ctx.putImageData(imgData, 0, 0);

    resetCanvasZoom();
  } catch (err) {
    alert('Error encoding file: ' + err.message);
    console.error(err);
  }
}

// Download Encoded PNG
downloadPNGBtn.addEventListener('click', () => {
  if (!currentEncodedResult) return;
  const blob = new Blob([currentEncodedResult.pngBytes], { type: 'image/png' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  const base = currentEncodedResult.metadata.filename.replace(/\.[^/.]+$/, '');
  a.download = `${base}.png`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
});

// 5. Canvas Zoom & Pan
function setZoom(newZoom) {
  currentZoom = Math.max(0.2, Math.min(30, newZoom));
  pixelCanvas.style.transform = `scale(${currentZoom})`;
  zoomLevelLabel.textContent = `${Math.round(currentZoom * 100)}%`;
}

function resetCanvasZoom() {
  currentZoom = 1;
  setZoom(1);
}

zoomInBtn.addEventListener('click', () => setZoom(currentZoom * 1.5));
zoomOutBtn.addEventListener('click', () => setZoom(currentZoom / 1.5));
resetViewBtn.addEventListener('click', () => resetCanvasZoom());

// 6. Interactive Pixel Inspector
pixelCanvas.addEventListener('mousemove', (e) => {
  if (!currentEncodedResult) return;
  const rect = pixelCanvas.getBoundingClientRect();
  const scaleX = pixelCanvas.width / rect.width;
  const scaleY = pixelCanvas.height / rect.height;

  const x = Math.floor((e.clientX - rect.left) * scaleX);
  const y = Math.floor((e.clientY - rect.top) * scaleY);

  if (x >= 0 && x < pixelCanvas.width && y >= 0 && y < pixelCanvas.height) {
    const offset = (y * pixelCanvas.width + x) * 4;
    const r = currentEncodedResult.rgbaBuffer[offset];
    const g = currentEncodedResult.rgbaBuffer[offset + 1];
    const b = currentEncodedResult.rgbaBuffer[offset + 2];
    const a = currentEncodedResult.rgbaBuffer[offset + 3];

    inspectColorSwatch.style.backgroundColor = `rgba(${r}, ${g}, ${b}, ${a / 255})`;
    inspectCoord.textContent = `X:${x}, Y:${y}`;
    inspectRGBA.textContent = `${r}, ${g}, ${b}, ${a}`;
    inspectHEX.textContent = `#${toHex(r)}${toHex(g)}${toHex(b)}${toHex(a)}`;
    inspectOffset.textContent = `byte ${offset}`;
  }
});

// 7. Decode Workflow
setupDropzone(decodeDropzone, decodeFileInput, async (file) => {
  try {
    const arrayBuffer = await file.arrayBuffer();
    const pngBytes = new Uint8Array(arrayBuffer);

    const result = await decodePNGToFile(pngBytes);
    currentDecodedResult = result;

    decodeFilename.textContent = result.metadata.filename;
    decodeSize.textContent = `${formatBytes(result.metadata.byteLength)} (${result.metadata.byteLength} bytes)`;
    decodeMime.textContent = result.metadata.mimeType || 'application/octet-stream';
    decodeDim.textContent = `${result.width} × ${result.height} px`;
    decodeSHA256.textContent = result.actualHash;

    downloadRestoredLabel.textContent = `Download ${result.metadata.filename}`;
    decodeResultCard.classList.remove('hidden');
  } catch (err) {
    alert('Error decoding PNG image: ' + err.message);
    console.error(err);
  }
});

// Download Restored File
downloadRestoredBtn.addEventListener('click', () => {
  if (!currentDecodedResult) return;
  const blob = new Blob([currentDecodedResult.data], {
    type: currentDecodedResult.metadata.mimeType || 'application/octet-stream',
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = currentDecodedResult.metadata.filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
});

// Init
initTheme();

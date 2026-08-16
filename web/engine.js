/**
 * PixelFile Web Engine - Fast, Low-Memory Lossless Container
 */

const CRC_TABLE = new Uint32Array(256);
for (let i = 0; i < 256; i++) {
  let c = i;
  for (let k = 0; k < 8; k++) {
    c = (c & 1) ? (0xEDB88320 ^ (c >>> 1)) : (c >>> 1);
  }
  CRC_TABLE[i] = c >>> 0;
}

export function computeCRC32(data, prevCRC = 0) {
  let crc = (prevCRC ^ 0xFFFFFFFF) >>> 0;
  for (let i = 0; i < data.length; i++) {
    crc = (CRC_TABLE[(crc ^ data[i]) & 0xFF] ^ (crc >>> 8)) >>> 0;
  }
  return (crc ^ 0xFFFFFFFF) >>> 0;
}

export function computeAdler32(data) {
  let a = 1;
  let b = 0;
  const MOD = 65521;
  const len = data.length;
  let i = 0;
  while (i < len) {
    const blockEnd = Math.min(i + 5552, len);
    while (i < blockEnd) {
      a += data[i++];
      b += a;
    }
    a %= MOD;
    b %= MOD;
  }
  return ((b << 16) | a) >>> 0;
}

export async function computeSHA256(uint8Array) {
  const hashBuffer = await crypto.subtle.digest('SHA-256', uint8Array);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

export function calculateDimensions(totalBytes, aspectRatio = '1:1') {
  if (totalBytes <= 0) return { width: 1, height: 1, pixels: 1 };
  const pixels = Math.ceil(totalBytes / 4);
  let width, height;
  if (aspectRatio === '16:9') {
    width = Math.ceil(Math.sqrt(pixels * (16 / 9)));
    if (width < 1) width = 1;
    height = Math.ceil(pixels / width);
  } else {
    width = Math.ceil(Math.sqrt(pixels));
    if (width < 1) width = 1;
    height = Math.ceil(pixels / width);
  }
  while (width * height < pixels) {
    height++;
  }
  return { width, height, pixels };
}

export function buildChunk(typeStr, dataUint8 = new Uint8Array(0)) {
  const len = dataUint8.length;
  const totalLen = 4 + 4 + len + 4;
  const chunk = new Uint8Array(totalLen);
  const view = new DataView(chunk.buffer);

  view.setUint32(0, len, false);
  for (let i = 0; i < 4; i++) {
    chunk[4 + i] = typeStr.charCodeAt(i);
  }
  if (len > 0) {
    chunk.set(dataUint8, 8);
  }
  const typeAndData = chunk.subarray(4, 8 + len);
  const crc = computeCRC32(typeAndData);
  view.setUint32(8 + len, crc, false);
  return chunk;
}

export function buildIHDR(width, height) {
  const data = new Uint8Array(13);
  const view = new DataView(data.buffer);
  view.setUint32(0, width, false);
  view.setUint32(4, height, false);
  data[8] = 8;
  data[9] = 6;  // RGBA
  data[10] = 0;
  data[11] = 0;
  data[12] = 0;
  return buildChunk('IHDR', data);
}

export function buildText(keyword, text) {
  const encoder = new TextEncoder();
  const kwBytes = encoder.encode(keyword);
  const textBytes = encoder.encode(text);
  const data = new Uint8Array(kwBytes.length + 1 + textBytes.length);
  data.set(kwBytes, 0);
  data[kwBytes.length] = 0x00;
  data.set(textBytes, kwBytes.length + 1);
  return buildChunk('tEXt', data);
}

export function buildIEND() {
  return buildChunk('IEND', new Uint8Array(0));
}

// Low-memory streaming zlib stored builder (Level 0)
export function formatZlibStored(scanlineData) {
  const BLOCK_MAX = 65535;
  const numBlocks = Math.ceil(scanlineData.length / BLOCK_MAX) || 1;
  const totalLen = 2 + (numBlocks * 5) + scanlineData.length + 4;
  const out = new Uint8Array(totalLen);
  let pos = 0;

  out[pos++] = 0x78;
  out[pos++] = 0x01;

  let offset = 0;
  while (offset < scanlineData.length || (scanlineData.length === 0 && offset === 0)) {
    const chunkLen = Math.min(BLOCK_MAX, scanlineData.length - offset);
    const isFinal = (offset + chunkLen >= scanlineData.length) ? 1 : 0;
    
    out[pos++] = isFinal;
    out[pos++] = chunkLen & 0xFF;
    out[pos++] = (chunkLen >>> 8) & 0xFF;
    
    const nlen = (~chunkLen) & 0xFFFF;
    out[pos++] = nlen & 0xFF;
    out[pos++] = (nlen >>> 8) & 0xFF;

    if (chunkLen > 0) {
      out.set(scanlineData.subarray(offset, offset + chunkLen), pos);
      pos += chunkLen;
      offset += chunkLen;
    } else {
      break;
    }
  }

  const adler = computeAdler32(scanlineData);
  const view = new DataView(out.buffer);
  view.setUint32(pos, adler, false);
  pos += 4;

  return out.subarray(0, pos);
}

// Compress data using browser CompressionStream (DEFLATE)
export async function compressStreamDeflate(data) {
  if (typeof CompressionStream === 'undefined') {
    return formatZlibStored(data);
  }
  try {
    const cs = new CompressionStream('deflate');
    const writer = cs.writable.getWriter();
    writer.write(data);
    writer.close();
    const chunks = [];
    const reader = cs.readable.getReader();
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      chunks.push(value);
    }
    let totalLen = 0;
    for (const c of chunks) totalLen += c.length;
    const result = new Uint8Array(totalLen);
    let offset = 0;
    for (const c of chunks) {
      result.set(c, offset);
      offset += c.length;
    }
    return result;
  } catch (err) {
    console.warn('CompressionStream fallback to stored:', err);
    return formatZlibStored(data);
  }
}

// Decompress data using browser DecompressionStream (DEFLATE)
export async function decompressStreamDeflate(compressedData) {
  if (typeof DecompressionStream === 'undefined') {
    throw new Error('Browser does not support DecompressionStream.');
  }
  const ds = new DecompressionStream('deflate');
  const writer = ds.writable.getWriter();
  writer.write(compressedData);
  writer.close();
  const chunks = [];
  const reader = ds.readable.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
  }
  let totalLen = 0;
  for (const c of chunks) totalLen += c.length;
  const result = new Uint8Array(totalLen);
  let offset = 0;
  for (const c of chunks) {
    result.set(c, offset);
    offset += c.length;
  }
  return result;
}

// High-speed encode with fast path for large files
export async function encodeFileToPNG(fileBytes, filename = 'payload.bin', options = {}) {
  const t0 = performance.now();
  const totalBytes = fileBytes.length;
  const aspectRatio = options.aspectRatio || '1:1';
  const mode = options.mode || 'auto';

  // 1. Dimensions
  const { width, height } = calculateDimensions(totalBytes, aspectRatio);

  // 2. Direct Scanline Assembly (Filter 0x00 per row)
  const rowBytes = width * 4;
  const scanlines = new Uint8Array(height * (1 + rowBytes));
  
  let sourceOffset = 0;
  for (let y = 0; y < height; y++) {
    const scanOffset = y * (1 + rowBytes);
    scanlines[scanOffset] = 0; // Filter None
    if (sourceOffset < totalBytes) {
      const take = Math.min(rowBytes, totalBytes - sourceOffset);
      scanlines.set(fileBytes.subarray(sourceOffset, sourceOffset + take), scanOffset + 1);
      sourceOffset += take;
    }
  }

  // 3. Compute SHA-256
  const sha256 = await computeSHA256(fileBytes);

  // 4. Adaptive Compression
  let idatData;
  let compressionUsed = 'stored';

  // For large files (> 8 MB), stored mode is instant and avoids heavy CPU/RAM
  if (mode === 'stored' || (mode === 'auto' && totalBytes > 8 * 1024 * 1024)) {
    idatData = formatZlibStored(scanlines);
    compressionUsed = 'stored';
  } else if (mode === 'deflate') {
    idatData = await compressStreamDeflate(scanlines);
    compressionUsed = 'deflate';
  } else {
    // Auto on small files: benchmark stored vs deflate
    const stored = formatZlibStored(scanlines);
    const deflated = await compressStreamDeflate(scanlines);
    if (deflated.length < stored.length) {
      idatData = deflated;
      compressionUsed = 'deflate';
    } else {
      idatData = stored;
      compressionUsed = 'stored';
    }
  }

  // 5. Metadata JSON
  const metadata = {
    version: 1,
    filename: filename,
    mimeType: options.mimeType || 'application/octet-stream',
    byteLength: totalBytes,
    sha256: sha256,
    timestamp: Date.now(),
    compression: compressionUsed,
    width: width,
    height: height,
    aspectRatio: aspectRatio,
  };

  // 6. PNG Container Chunks
  const PNG_SIG = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]);
  const ihdrChunk = buildIHDR(width, height);
  const textChunk = buildText('PixelFile', JSON.stringify(metadata));
  const idatChunk = buildChunk('IDAT', idatData);
  const iendChunk = buildIEND();

  const totalPngLen = PNG_SIG.length + ihdrChunk.length + textChunk.length + idatChunk.length + iendChunk.length;
  const pngBytes = new Uint8Array(totalPngLen);
  let p = 0;
  pngBytes.set(PNG_SIG, p); p += PNG_SIG.length;
  pngBytes.set(ihdrChunk, p); p += ihdrChunk.length;
  pngBytes.set(textChunk, p); p += textChunk.length;
  pngBytes.set(idatChunk, p); p += idatChunk.length;
  pngBytes.set(iendChunk, p); p += iendChunk.length;

  const durationMs = performance.now() - t0;
  const outputSize = pngBytes.length;
  const overheadBytes = outputSize - totalBytes;
  const ratioPercent = totalBytes > 0 ? (overheadBytes / totalBytes) * 100 : 0;

  return {
    pngBytes,
    metadata,
    width,
    height,
    originalSize: totalBytes,
    outputSize,
    overheadBytes,
    ratioPercent,
    durationMs,
  };
}

// Decode PNG Container
export async function decodePNGToFile(pngBytes) {
  const t0 = performance.now();
  if (pngBytes.length < 8) {
    throw new Error('File is too small to be a valid PNG.');
  }

  const PNG_SIG = [137, 80, 78, 71, 13, 10, 26, 10];
  for (let i = 0; i < 8; i++) {
    if (pngBytes[i] !== PNG_SIG[i]) {
      throw new Error('Invalid PNG header signature.');
    }
  }

  let offset = 8;
  const view = new DataView(pngBytes.buffer, pngBytes.byteOffset, pngBytes.byteLength);
  let ihdrData = null;
  let metaText = '';
  const idatChunks = [];

  while (offset < pngBytes.length) {
    if (offset + 8 > pngBytes.length) break;
    const len = view.getUint32(offset, false);
    let chunkType = '';
    for (let i = 0; i < 4; i++) {
      chunkType += String.fromCharCode(pngBytes[offset + 4 + i]);
    }
    const dataStart = offset + 8;
    const dataEnd = dataStart + len;

    if (chunkType === 'IHDR') {
      ihdrData = pngBytes.subarray(dataStart, dataEnd);
    } else if (chunkType === 'tEXt') {
      const chunkData = pngBytes.subarray(dataStart, dataEnd);
      let nullIdx = -1;
      for (let i = 0; i < chunkData.length; i++) {
        if (chunkData[i] === 0x00) { nullIdx = i; break; }
      }
      if (nullIdx > 0) {
        const kw = new TextDecoder().decode(chunkData.subarray(0, nullIdx));
        if (kw === 'PixelFile') {
          metaText = new TextDecoder().decode(chunkData.subarray(nullIdx + 1));
        }
      }
    } else if (chunkType === 'IDAT') {
      idatChunks.push(pngBytes.subarray(dataStart, dataEnd));
    }

    offset = dataEnd + 4;
    if (chunkType === 'IEND') break;
  }

  if (!ihdrData || ihdrData.length < 8) {
    throw new Error('Missing or corrupt IHDR chunk.');
  }

  const ihdrView = new DataView(ihdrData.buffer, ihdrData.byteOffset, ihdrData.byteLength);
  const width = ihdrView.getUint32(0, false);
  const height = ihdrView.getUint32(4, false);

  if (idatChunks.length === 0) {
    throw new Error('No IDAT pixel data found in PNG.');
  }

  // Concatenate IDAT data
  let totalIdatLen = 0;
  for (const c of idatChunks) totalIdatLen += c.length;
  const combinedIDAT = new Uint8Array(totalIdatLen);
  let idatOffset = 0;
  for (const c of idatChunks) {
    combinedIDAT.set(c, idatOffset);
    idatOffset += c.length;
  }

  // Decompress scanlines
  const scanlines = await decompressStreamDeflate(combinedIDAT);
  const rowBytes = width * 4;

  let metadata;
  if (metaText) {
    try {
      metadata = JSON.parse(metaText);
    } catch (e) {
      console.warn('Failed to parse metadata JSON:', e);
    }
  }

  const targetLength = (metadata && metadata.byteLength) ? metadata.byteLength : (width * height * 4);
  const restoredBytes = new Uint8Array(targetLength);

  let writePos = 0;
  for (let y = 0; y < height; y++) {
    const scanOffset = y * (1 + rowBytes);
    const availableInRow = rowBytes;
    const needed = targetLength - writePos;
    if (needed <= 0) break;
    const toCopy = Math.min(availableInRow, needed);
    restoredBytes.set(scanlines.subarray(scanOffset + 1, scanOffset + 1 + toCopy), writePos);
    writePos += toCopy;
  }

  if (!metadata) {
    metadata = {
      filename: 'restored_payload.bin',
      byteLength: targetLength,
      mimeType: 'application/octet-stream',
      sha256: '',
      width,
      height,
    };
  }

  // Verify SHA-256
  const actualHash = await computeSHA256(restoredBytes);
  const sha256Matches = metadata.sha256 ? (actualHash.toLowerCase() === metadata.sha256.toLowerCase()) : true;
  const durationMs = performance.now() - t0;

  return {
    data: restoredBytes,
    metadata,
    width,
    height,
    sha256Matches,
    actualHash,
    durationMs,
  };
}

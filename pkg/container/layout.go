package container

import (
	"fmt"
	"math"
)

// CalculateDimensions computes width and height for a given byte size and aspect ratio.
func CalculateDimensions(totalBytes int64, aspectRatio string) (int, int) {
	if totalBytes <= 0 {
		return 1, 1
	}

	// 4 bytes per RGBA pixel
	pixels := int64(math.Ceil(float64(totalBytes) / 4.0))
	if pixels <= 0 {
		pixels = 1
	}

	var width, height int
	switch aspectRatio {
	case "16:9":
		// W = sqrt(P * 16 / 9)
		wFloat := math.Ceil(math.Sqrt(float64(pixels) * 16.0 / 9.0))
		width = int(wFloat)
		if width < 1 {
			width = 1
		}
		height = int(math.Ceil(float64(pixels) / float64(width)))
	case "1:1", "":
		fallthrough
	default:
		// W = sqrt(P)
		wFloat := math.Ceil(math.Sqrt(float64(pixels)))
		width = int(wFloat)
		if width < 1 {
			width = 1
		}
		height = int(math.Ceil(float64(pixels) / float64(width)))
	}

	// Ensure capacity covers all pixels
	for int64(width)*int64(height) < pixels {
		height++
	}

	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	return width, height
}

// PackBytesToRGBA places raw data bytes into an RGBA buffer of width x height x 4.
func PackBytesToRGBA(data []byte, width, height int) []byte {
	totalCapacity := width * height * 4
	rgba := make([]byte, totalCapacity)
	copy(rgba, data)
	return rgba
}

// UnpackRGBAToBytes truncates RGBA buffer back to the exact original byte length.
func UnpackRGBAToBytes(rgba []byte, originalLength int64) ([]byte, error) {
	if originalLength < 0 {
		return nil, fmt.Errorf("invalid negative original length: %d", originalLength)
	}
	if int64(len(rgba)) < originalLength {
		return nil, fmt.Errorf("buffer underrun: have %d bytes, need %d bytes", len(rgba), originalLength)
	}
	// Return a copy of the exact sliced bytes
	restored := make([]byte, originalLength)
	copy(restored, rgba[:originalLength])
	return restored, nil
}

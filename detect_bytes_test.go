package watermark

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Ensure byte-slice detection aligns with image-based detection.
func TestDetectWatermarkBytesMatchesImageDetection(t *testing.T) {
	inputPath := filepath.Join("cmd", "gwatermark", "image.png")

	data, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input image: %v", err)
	}

	presentBytes, scoreBytes, infoBytes, err := DetectWatermarkBytes(data)
	if err != nil {
		t.Fatalf("DetectWatermarkBytes error: %v", err)
	}
	if !presentBytes {
		t.Fatalf("expected watermark detection for bytes path")
	}
	if infoBytes.Size == 0 {
		t.Fatalf("expected watermark info to be populated")
	}

	img, _, err := DecodeImageBytes(data)
	if err != nil {
		t.Fatalf("decode image: %v", err)
	}

	presentImg, scoreImg, infoImg, err := DetectWatermark(img)
	if err != nil {
		t.Fatalf("DetectWatermark image path error: %v", err)
	}

	if presentBytes != presentImg {
		t.Fatalf("byte detection mismatch: bytes present=%v, image present=%v", presentBytes, presentImg)
	}
	if diff := math.Abs(scoreBytes - scoreImg); diff > 1e-9 {
		t.Fatalf("score mismatch: bytes %.6f image %.6f", scoreBytes, scoreImg)
	}
	if infoBytes != infoImg {
		t.Fatalf("info mismatch: bytes %+v image %+v", infoBytes, infoImg)
	}
}

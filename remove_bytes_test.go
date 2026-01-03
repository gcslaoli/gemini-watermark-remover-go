package watermark

import (
	"bytes"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"testing"
)

// Ensure byte-slice removal path matches the known cleaned image output.
func TestRemoveWatermarkBytesMatchesExpectedImage(t *testing.T) {
	inputPath := filepath.Join("cmd", "gwatermark", "image.png")
	expectedPath := filepath.Join("cmd", "gwatermark", "image_unwatermarked.png")

	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input image: %v", err)
	}

	outputBytes, present, score, info, err := RemoveWatermarkBytes(inputBytes)
	if err != nil {
		t.Fatalf("RemoveWatermarkBytes error: %v", err)
	}
	if !present {
		t.Fatalf("expected watermark detection, got present=false (score %.2f, info %+v)", score, info)
	}
	if len(outputBytes) == 0 {
		t.Fatalf("RemoveWatermarkBytes returned empty output")
	}

	outDir := filepath.Join("temp")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	outPath := filepath.Join(outDir, "image_cleaned.png")
	if err := os.WriteFile(outPath, outputBytes, 0o644); err != nil {
		t.Fatalf("write output image: %v", err)
	}
	t.Logf("wrote cleaned output to %s (score %.2f, watermark %dx%d at %v)", outPath, score, info.Size, info.Size, info.Position)

	gotImg, format, err := image.Decode(bytes.NewReader(outputBytes))
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if format != "png" {
		t.Fatalf("expected png output, got %q", format)
	}

	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected image: %v", err)
	}
	expectedImg, _, err := image.Decode(bytes.NewReader(expectedBytes))
	if err != nil {
		t.Fatalf("decode expected: %v", err)
	}

	if !imagesEqual(expectedImg, gotImg) {
		t.Fatalf("output image pixels differ from expected cleaned image")
	}
}

func imagesEqual(a, b image.Image) bool {
	if !a.Bounds().Eq(b.Bounds()) {
		return false
	}

	ab := imageToNRGBA(a)
	bb := imageToNRGBA(b)

	return bytes.Equal(ab.Pix, bb.Pix)
}

func imageToNRGBA(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	draw.Draw(out, bounds, img, bounds.Min, draw.Src)
	return out
}

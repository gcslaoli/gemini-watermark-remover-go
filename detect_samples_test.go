package watermark

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectWatermarkSampleImages(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		wantHit bool
	}{
		{name: "watermark_ref", path: filepath.Join("cmd", "gwatermark", "image.png"), wantHit: true},
		{name: "nowater", path: filepath.Join("cmd", "gwatermark", "nowater.jpg"), wantHit: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			img, err := readSample(tc.path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) && !tc.wantHit {
					t.Skipf("sample %s missing, skipping", tc.path)
					return
				}
				t.Fatalf("read %s: %v", tc.path, err)
			}

			bounds := img.Bounds()
			cfg := DetectWatermarkConfig(bounds.Dx(), bounds.Dy())
			rect, err := calculateWatermarkRect(bounds, cfg)
			if err != nil {
				t.Fatalf("rect: %v", err)
			}

			alpha, err := detectAlphaMap(cfg.LogoSize)
			if err != nil {
				t.Fatalf("alpha: %v", err)
			}

			band := cfg.LogoSize / 3
			if band < 8 {
				band = 8
			}
			outer := rect.Inset(-band).Intersect(bounds)

			_, innerCount := meanLuma(img, rect, image.Rectangle{})
			bgMean, outerCount := meanLuma(img, outer, rect)
			if innerCount == 0 || outerCount == 0 {
				t.Fatalf("insufficient pixels")
			}

			score, corr, err := scoreWatermark(img, rect, alpha, bgMean)
			if err != nil {
				t.Fatalf("score: %v", err)
			}

			present := score > detectionLumaThreshold && corr > detectionCorrelationThreshold
			t.Logf("%s: score=%.2f corr=%.3f", tc.name, score, corr)

			if present != tc.wantHit {
				t.Fatalf("present=%v want=%v (score=%.2f corr=%.3f)", present, tc.wantHit, score, corr)
			}
		})
	}
}

func readSample(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := Decode(f)
	return img, err
}

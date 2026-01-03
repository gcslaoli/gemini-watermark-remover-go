package watermark

import (
	"fmt"
	"image"
)

const (
	// Brightness difference threshold to consider a watermark present.
	// The watermark is white on darker pixels, so the mean luma in the
	// watermark rectangle should be noticeably higher than its surroundings.
	detectionLumaThreshold = 6.0
)

// DetectWatermark estimates whether the Gemini visible watermark is present.
// It compares the average luma inside the expected watermark rectangle with a
// surrounding band. A positive score above the threshold indicates likely
// watermark presence.
func DetectWatermark(img image.Image) (present bool, score float64, info Info, err error) {
	if img == nil {
		return false, 0, Info{}, fmt.Errorf("nil image provided")
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return false, 0, Info{}, fmt.Errorf("invalid image dimensions %dx%d", width, height)
	}

	cfg := DetectWatermarkConfig(width, height)
	rect, err := calculateWatermarkRect(bounds, cfg)
	if err != nil {
		return false, 0, Info{}, err
	}

	// Use a surrounding band to approximate the background without the watermark.
	band := cfg.LogoSize / 3
	if band < 8 {
		band = 8
	}

	outer := rect.Inset(-band).Intersect(bounds)

	wmMean, wmCount := meanLuma(img, rect, image.Rectangle{})
	bgMean, bgCount := meanLuma(img, outer, rect)

	if wmCount == 0 || bgCount == 0 {
		return false, 0, Info{}, fmt.Errorf("insufficient pixels to evaluate watermark")
	}

	score = wmMean - bgMean
	present = score > detectionLumaThreshold

	info = Info{Size: cfg.LogoSize, Position: rect}
	return present, score, info, nil
}

// meanLuma computes the average luma for pixels in region. If exclude is not
// empty, pixels inside exclude are skipped.
func meanLuma(img image.Image, region image.Rectangle, exclude image.Rectangle) (float64, int) {
	var sum float64
	var count int

	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			if exclude != (image.Rectangle{}) && (image.Point{X: x, Y: y}).In(exclude) {
				continue
			}

			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to luma in [0, 255].
			luma := 0.2126*float64(r)/257.0 + 0.7152*float64(g)/257.0 + 0.0722*float64(b)/257.0
			sum += luma
			count++
		}
	}

	if count == 0 {
		return 0, 0
	}

	return sum / float64(count), count
}

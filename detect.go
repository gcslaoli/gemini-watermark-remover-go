package watermark

import (
	"fmt"
	"image"
	"math"
	"sync"
)

const (
	// Brightness difference threshold to consider a watermark present.
	// The watermark is white on darker pixels, so the mean luma in the
	// watermark rectangle should be noticeably higher than its surroundings.
	detectionLumaThreshold = 6.0
	// Correlation gate to ensure the brightness increase matches the expected
	// watermark shape instead of arbitrary bright content near the corner.
	detectionCorrelationThreshold = 0.20
)

var detectAlphaCache = struct {
	once map[int]*sync.Once
	maps map[int][]float32
	errs map[int]error
}{
	once: map[int]*sync.Once{
		48: new(sync.Once),
		96: new(sync.Once),
	},
	maps: make(map[int][]float32),
	errs: make(map[int]error),
}

// DetectWatermark estimates whether the Gemini visible watermark is present.
// It compares the luma inside the expected watermark rectangle against a
// surrounding band and gates on correlation with the watermark alpha mask so
// bright corners without the watermark are not misclassified.
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

	alphaMap, err := detectAlphaMap(cfg.LogoSize)
	if err != nil {
		return false, 0, Info{}, err
	}

	// Use a surrounding band to approximate the background without the watermark.
	band := cfg.LogoSize / 3
	if band < 8 {
		band = 8
	}

	outer := rect.Inset(-band).Intersect(bounds)

	_, bgCount := meanLuma(img, rect, image.Rectangle{})
	bgMean, outerCount := meanLuma(img, outer, rect)

	if bgCount == 0 || outerCount == 0 {
		return false, 0, Info{}, fmt.Errorf("insufficient pixels to evaluate watermark")
	}

	score, corr, err := scoreWatermark(img, rect, alphaMap, bgMean)
	if err != nil {
		return false, 0, Info{}, err
	}

	present = score > detectionLumaThreshold && corr > detectionCorrelationThreshold

	info = Info{Size: cfg.LogoSize, Position: rect}
	return present, score, info, nil
}

func detectAlphaMap(size int) ([]float32, error) {
	once, ok := detectAlphaCache.once[size]
	if !ok {
		return nil, fmt.Errorf("unsupported watermark size %d", size)
	}

	once.Do(func() {
		detectAlphaCache.maps[size], detectAlphaCache.errs[size] = decodeAlphaAsset(size)
	})

	if err := detectAlphaCache.errs[size]; err != nil {
		return nil, err
	}

	alpha, ok := detectAlphaCache.maps[size]
	if !ok {
		return nil, fmt.Errorf("alpha map not available for size %d", size)
	}

	return alpha, nil
}

// scoreWatermark compares the expected watermark alpha mask with the image
// brightness to produce a luma delta and a shape correlation score.
func scoreWatermark(img image.Image, rect image.Rectangle, alphaMap []float32, bgMean float64) (delta float64, corr float64, err error) {
	stride := rect.Dx()
	if stride <= 0 || rect.Dy() <= 0 {
		return 0, 0, fmt.Errorf("invalid watermark rectangle %v", rect)
	}

	required := rect.Dx() * rect.Dy()
	if len(alphaMap) != required {
		return 0, 0, fmt.Errorf("alpha map size mismatch: have %d, want %d", len(alphaMap), required)
	}

	residuals := make([]float64, required)

	const clearAlphaCutoff = 0.02

	var (
		sumResidual float64
		sumAlpha    float64
		sumAlphaSq  float64
		clearSum    float64
		clearCount  int
	)

	idx := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			luma := 0.2126*float64(r)/257.0 + 0.7152*float64(g)/257.0 + 0.0722*float64(b)/257.0

			residual := luma - bgMean
			residuals[idx] = residual

			alpha := float64(alphaMap[idx])
			sumResidual += residual
			sumAlpha += alpha
			sumAlphaSq += alpha * alpha

			if alpha < clearAlphaCutoff {
				clearSum += residual
				clearCount++
			}

			idx++
		}
	}

	if clearCount == 0 {
		return 0, 0, fmt.Errorf("alpha map missing clear pixels for scoring")
	}
	if sumAlpha == 0 {
		return 0, 0, fmt.Errorf("alpha map missing opaque pixels for scoring")
	}

	clearMean := clearSum / float64(clearCount)

	weightedResidual := 0.0
	for i, res := range residuals {
		weightedResidual += res * float64(alphaMap[i])
	}

	delta = weightedResidual/sumAlpha - clearMean

	residualMean := sumResidual / float64(required)
	alphaMean := sumAlpha / float64(required)

	var numerator, resVar float64
	idx = 0
	for range residuals {
		a := float64(alphaMap[idx]) - alphaMean
		r := residuals[idx] - residualMean
		numerator += a * r
		resVar += r * r
		idx++
	}

	alphaVar := sumAlphaSq - float64(required)*alphaMean*alphaMean
	denominator := math.Sqrt(alphaVar * resVar)
	if denominator == 0 {
		return delta, 0, nil
	}

	corr = numerator / denominator
	return delta, corr, nil
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

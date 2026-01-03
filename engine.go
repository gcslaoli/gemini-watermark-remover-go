package watermark

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"sync"
)

const (
	alphaThreshold = 0.002
	maxAlpha       = 0.99
	logoValue      = 255.0
)

type watermarkConfig struct {
	LogoSize     int
	MarginRight  int
	MarginBottom int
}

// Info captures the watermark size and placement for a given image.
type Info struct {
	Size     int
	Position image.Rectangle
}

// Engine holds cached alpha maps and performs reverse alpha blending.
type Engine struct {
	alphaMaps map[int][]float32
	alphaErrs map[int]error
	once      map[int]*sync.Once
}

// NewEngine constructs an Engine with lazily loaded alpha maps.
func NewEngine() *Engine {
	return &Engine{
		alphaMaps: make(map[int][]float32),
		alphaErrs: make(map[int]error),
		once: map[int]*sync.Once{
			48: new(sync.Once),
			96: new(sync.Once),
		},
	}
}

var defaultEngine struct {
	once sync.Once
	eng  *Engine
}

// RemoveWatermark applies the default engine to the provided image.
func RemoveWatermark(img image.Image) (*image.RGBA, error) {
	defaultEngine.once.Do(func() {
		defaultEngine.eng = NewEngine()
	})

	return defaultEngine.eng.RemoveWatermark(img)
}

// RemoveWatermark applies reverse alpha blending to remove the Gemini
// watermark. The result is returned as a new *image.RGBA.
func (e *Engine) RemoveWatermark(img image.Image) (*image.RGBA, error) {
	if img == nil {
		return nil, fmt.Errorf("nil image provided")
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid image dimensions %dx%d", width, height)
	}

	cfg := DetectWatermarkConfig(width, height)
	rect, err := calculateWatermarkRect(bounds, cfg)
	if err != nil {
		return nil, err
	}

	alphaMap, err := e.getAlphaMap(cfg.LogoSize)
	if err != nil {
		return nil, err
	}

	expected := rect.Dx() * rect.Dy()
	if len(alphaMap) != expected {
		return nil, fmt.Errorf("alpha map size mismatch: have %d, want %d", len(alphaMap), expected)
	}

	rgba := cloneToRGBA(img)
	applyReverseAlpha(rgba, alphaMap, rect)

	return rgba, nil
}

// WatermarkInfo reports the detected watermark size and rectangle for display.
func WatermarkInfo(width, height int) Info {
	cfg := DetectWatermarkConfig(width, height)
	rect, _ := calculateWatermarkRect(image.Rect(0, 0, width, height), cfg)
	return Info{Size: cfg.LogoSize, Position: rect}
}

// DetectWatermarkConfig selects the Gemini watermark parameters based on the
// original JS rules: if both width and height are greater than 1024, use 96x96
// with 64px margins; otherwise use 48x48 with 32px margins.
func DetectWatermarkConfig(width, height int) watermarkConfig {
	if width > 1024 && height > 1024 {
		return watermarkConfig{LogoSize: 96, MarginRight: 64, MarginBottom: 64}
	}
	return watermarkConfig{LogoSize: 48, MarginRight: 32, MarginBottom: 32}
}

// calculateWatermarkRect computes the watermark rectangle in image coordinates.
func calculateWatermarkRect(bounds image.Rectangle, cfg watermarkConfig) (image.Rectangle, error) {
	x := bounds.Max.X - cfg.MarginRight - cfg.LogoSize
	y := bounds.Max.Y - cfg.MarginBottom - cfg.LogoSize

	rect := image.Rect(x, y, x+cfg.LogoSize, y+cfg.LogoSize)
	if !rect.In(bounds) {
		return image.Rectangle{}, fmt.Errorf("watermark rectangle %v out of bounds %v", rect, bounds)
	}
	return rect, nil
}

// cloneToRGBA copies the image into a mutable RGBA buffer.
func cloneToRGBA(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}

// getAlphaMap lazily loads and caches the alpha map for the requested size.
func (e *Engine) getAlphaMap(size int) ([]float32, error) {
	once, ok := e.once[size]
	if !ok {
		return nil, fmt.Errorf("unsupported watermark size %d", size)
	}

	once.Do(func() {
		e.alphaMaps[size], e.alphaErrs[size] = decodeAlphaAsset(size)
	})

	if err := e.alphaErrs[size]; err != nil {
		return nil, err
	}

	if alpha, ok := e.alphaMaps[size]; ok {
		return alpha, nil
	}

	return nil, fmt.Errorf("alpha map not available for size %d", size)
}

// applyReverseAlpha performs the reverse alpha blending within the watermark
// rectangle. It mutates the provided RGBA buffer in place.
func applyReverseAlpha(img *image.RGBA, alphaMap []float32, rect image.Rectangle) {
	stride := rect.Dx()

	for row := 0; row < rect.Dy(); row++ {
		for col := 0; col < rect.Dx(); col++ {
			alpha := float64(alphaMap[row*stride+col])
			if alpha < alphaThreshold {
				continue
			}

			if alpha > maxAlpha {
				alpha = maxAlpha
			}

			oneMinusAlpha := 1.0 - alpha
			offset := img.PixOffset(rect.Min.X+col, rect.Min.Y+row)

			for c := 0; c < 3; c++ {
				watermarked := float64(img.Pix[offset+c])
				original := (watermarked - alpha*logoValue) / oneMinusAlpha

				original = math.Max(0, math.Min(255, original))
				img.Pix[offset+c] = uint8(math.Round(original))
			}
		}
	}
}

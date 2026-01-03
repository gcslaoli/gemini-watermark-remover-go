package watermark

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
)

//go:embed assets/bg_48.png assets/bg_64.png assets/bg_96.png
var embeddedAssets embed.FS

// decodeAlphaAsset loads the pre-captured watermark background and converts it
// into an alpha map normalized to [0, 1].
func decodeAlphaAsset(size int) ([]float32, error) {
	filename := fmt.Sprintf("assets/bg_%d.png", size)

	data, err := embeddedAssets.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filename, err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", filename, err)
	}

	return calculateAlphaMap(img), nil
}

// calculateAlphaMap extracts the maximum RGB channel per pixel and scales it to
// [0, 1], matching the JavaScript implementation.
func calculateAlphaMap(img image.Image) []float32 {
	bounds := img.Bounds()
	alpha := make([]float32, bounds.Dx()*bounds.Dy())

	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			max := r
			if g > max {
				max = g
			}
			if b > max {
				max = b
			}

			alpha[idx] = float32(max) / 65535.0
			idx++
		}
	}

	return alpha
}

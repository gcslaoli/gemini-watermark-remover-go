package watermark

import "fmt"

// DetectWatermarkBytes checks raw image bytes for the Gemini watermark without
// performing any cleanup. It decodes the bytes into an image and delegates to
// DetectWatermark for the score and placement details.
func DetectWatermarkBytes(data []byte) (present bool, score float64, info Info, err error) {
	if len(data) == 0 {
		return false, 0, Info{}, fmt.Errorf("empty image data")
	}

	img, _, err := DecodeImageBytes(data)
	if err != nil {
		return false, 0, Info{}, err
	}

	return DetectWatermark(img)
}

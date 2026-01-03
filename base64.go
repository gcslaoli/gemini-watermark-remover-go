package watermark

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"
)

// DecodeBase64Image decodes a base64-encoded image (optionally a data URL) into
// an image.Image. It returns the decoded image and the detected format string
// ("png", "jpeg", "webp", etc.).
func DecodeBase64Image(input string) (image.Image, string, error) {
	raw := stripDataPrefix(input)

	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, "", fmt.Errorf("decode base64: %w", err)
	}

	img, format, err := Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}

	return img, format, nil
}

// EncodePNGToBase64 encodes an image as PNG and returns a base64 string.
func EncodePNGToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// RemoveWatermarkBase64 removes the watermark from a base64-encoded image. It
// returns the cleaned image as base64 PNG, whether a watermark was detected,
// the detection score, watermark info, and an error if any.
func RemoveWatermarkBase64(input string) (output string, present bool, score float64, info Info, err error) {
	img, _, err := DecodeBase64Image(input)
	if err != nil {
		return "", false, 0, Info{}, err
	}

	present, score, info, err = DetectWatermark(img)
	if err != nil {
		return "", false, 0, Info{}, err
	}

	if !present {
		return "", false, score, info, nil
	}

	engine := NewEngine()
	cleaned, err := engine.RemoveWatermark(img)
	if err != nil {
		return "", false, 0, Info{}, err
	}

	output, err = EncodePNGToBase64(cleaned)
	if err != nil {
		return "", false, 0, Info{}, err
	}

	return output, true, score, info, nil
}

func stripDataPrefix(input string) string {
	lower := strings.ToLower(input)
	if strings.HasPrefix(lower, "data:") {
		if idx := strings.Index(input, ","); idx != -1 {
			return input[idx+1:]
		}
	}
	return input
}

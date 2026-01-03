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

// DecodeImageBytes decodes raw image bytes into an image.Image. It returns the
// decoded image and detected format string.
func DecodeImageBytes(data []byte) (image.Image, string, error) {
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty image data")
	}

	return Decode(bytes.NewReader(data))
}

// EncodePNGToBase64 encodes an image as PNG and returns a base64 string.
func EncodePNGToBase64(img image.Image) (string, error) {
	data, err := EncodePNGToBytes(img)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// EncodePNGToBytes encodes an image as PNG and returns the raw bytes.
func EncodePNGToBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// RemoveWatermarkBase64 removes the watermark from a base64-encoded image. It
// returns the cleaned image as base64 PNG, whether a watermark was detected,
// the detection score, watermark info, and an error if any.
func RemoveWatermarkBase64(input string) (output string, present bool, score float64, info Info, err error) {
	raw := stripDataPrefix(input)

	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", false, 0, Info{}, fmt.Errorf("decode base64: %w", err)
	}

	bytesOut, present, score, info, err := RemoveWatermarkBytes(data)
	if err != nil {
		return "", false, 0, Info{}, err
	}

	if !present {
		return "", false, score, info, nil
	}

	return base64.StdEncoding.EncodeToString(bytesOut), true, score, info, nil
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

// RemoveWatermarkBytes removes the watermark from raw image bytes. It returns
// the cleaned PNG bytes when a watermark is detected, along with the detection
// score and watermark info.
func RemoveWatermarkBytes(input []byte) (output []byte, present bool, score float64, info Info, err error) {
	if len(input) == 0 {
		return nil, false, 0, Info{}, fmt.Errorf("empty image data")
	}

	img, _, err := DecodeImageBytes(input)
	if err != nil {
		return nil, false, 0, Info{}, err
	}

	present, score, info, err = DetectWatermark(img)
	if err != nil {
		return nil, false, 0, Info{}, err
	}

	if !present {
		return nil, false, score, info, nil
	}

	engine := NewEngine()
	cleaned, err := engine.RemoveWatermark(img)
	if err != nil {
		return nil, false, 0, Info{}, err
	}

	output, err = EncodePNGToBytes(cleaned)
	if err != nil {
		return nil, false, 0, Info{}, err
	}

	return output, true, score, info, nil
}

package watermark

import (
	"image"
	"image/png"
	"io"

	// Register common decoders, including WebP via x/image/webp.
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// Decode reads an image from the reader, returning the decoded image and the
// detected format string ("png", "jpeg", "webp", etc.).
func Decode(r io.Reader) (image.Image, string, error) {
	return image.Decode(r)
}

// EncodePNG writes the provided image to the writer as PNG.
func EncodePNG(w io.Writer, img image.Image) error {
	return png.Encode(w, img)
}

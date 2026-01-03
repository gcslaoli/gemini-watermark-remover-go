# Gemini Watermark Remover (Go)

A Go port of the reverse alpha blending algorithm used to remove visible Gemini AI watermarks without AI inpainting. The library ships with embedded watermark captures for both 48x48 and 96x96 variants and works entirely in memory.

## Install

```bash
go get github.com/gcslaoli/gemini-watermark-remover-go
```

## Usage

```go
package main

import (
    "os"

    watermark "github.com/gcslaoli/gemini-watermark-remover-go"
)

func main() {
    f, _ := os.Open("watermarked.png")
    img, _, _ := watermark.Decode(f)

    engine := watermark.NewEngine()
    cleaned, _ := engine.RemoveWatermark(img)

    out, _ := os.Create("unwatermarked.png")
    defer out.Close()
    _ = watermark.EncodePNG(out, cleaned)
}
```

Quick detection (visible watermark only):

```go
present, score, info, err := watermark.DetectWatermark(img)
if err != nil {
    // handle error
}
// present is a bool; score is luma contrast; info contains size and rect.
```

Base64 in/out helper:

```go
outB64, present, score, info, err := watermark.RemoveWatermarkBase64(inB64)
if err != nil {
    // handle error
}
if !present {
    // no visible watermark detected
}
// outB64 is PNG base64 when present is true
```

Byte slice helper (raw image bytes → PNG bytes):

```go
outBytes, present, score, info, err := watermark.RemoveWatermarkBytes(inBytes)
if err != nil {
    // handle error
}
if !present {
    // no visible watermark detected
}
// outBytes is PNG bytes when present is true
```

`DetectWatermarkConfig` follows the original rule set:

- If width > 1024 **and** height > 1024 → 96x96 logo with 64px margins
- Otherwise → 48x48 logo with 32px margins

## CLI example

A small helper binary is available:

```bash
go run ./cmd/gwatermark -in image.png -out image_unwatermarked.png
```

## License

MIT

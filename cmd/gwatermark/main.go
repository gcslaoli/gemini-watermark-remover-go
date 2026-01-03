package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	watermark "github.com/gcslaoli/gemini-watermark-remover-go"
)

// go run main.go -in image.png -out image_unwatermarked.png
// go run main.go -in nowater.jpg -out nowater_unwatermarked.png
// go run main.go -in image2.png -out image2_unwatermarked.png
// go run main.go -in image3.jpg -out image3_unwatermarked.png
// go run main.go -in image4.jpg -out image4_unwatermarked.jpg
// go run main.go -in image_cleaned.png -out image_cleaned_unwatermarked.png
// go run main.go -in nowater.jpg --out nowater_unwatermarked.png

func main() {
	input := flag.String("in", "", "Path to the watermarked image (png/jpg/webp)")
	inputBase64 := flag.String("inbase64", "", "Base64 image input (optionally data URL)")
	output := flag.String("out", "", "Output path (defaults to <name>_unwatermarked.png)")
	outputBase64 := flag.Bool("outbase64", false, "Write cleaned PNG as base64 to stdout instead of file")
	flag.Parse()

	if *input == "" && *inputBase64 == "" {
		flag.Usage()
		os.Exit(1)
	}

	var (
		img    image.Image
		format string
		source string
		err    error
	)

	if *inputBase64 != "" {
		img, format, err = watermark.DecodeBase64Image(*inputBase64)
		source = "base64"
	} else {
		inFile, openErr := os.Open(*input)
		if openErr != nil {
			fmt.Fprintf(os.Stderr, "open input: %v\n", openErr)
			os.Exit(1)
		}
		defer inFile.Close()

		img, format, err = watermark.Decode(inFile)
		source = *input
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "decode input: %v\n", err)
		os.Exit(1)
	}

	present, score, info, err := watermark.DetectWatermark(img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "detect watermark: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Detected visible Gemini watermark (score %.2f) at %dx%d position %v.\n", score, info.Size, info.Size, info.Position)

	if !present {
		fmt.Printf("No visible Gemini watermark detected (score %.2f). Skipping removal.\n", score)
		os.Exit(0)
	}

	engine := watermark.NewEngine()
	cleaned, err := engine.RemoveWatermark(img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "remove watermark: %v\n", err)
		os.Exit(1)
	}

	if *outputBase64 {
		encoded, encErr := watermark.EncodePNGToBase64(cleaned)
		if encErr != nil {
			fmt.Fprintf(os.Stderr, "encode base64 output: %v\n", encErr)
			os.Exit(1)
		}
		fmt.Println(encoded)
		fmt.Printf("Processed %s (%s) -> base64 [watermark %dx%d at %v]\n", source, format, info.Size, info.Size, info.Position)
		return
	}

	outPath := *output
	if outPath == "" {
		base := "output"
		if *input != "" {
			base = strings.TrimSuffix(filepath.Base(*input), filepath.Ext(*input))
		}
		outPath = filepath.Join(filepath.Dir(*input), base+"_unwatermarked.png")
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if err := watermark.EncodePNG(outFile, cleaned); err != nil {
		fmt.Fprintf(os.Stderr, "encode output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processed %s (%s) -> %s [watermark %dx%d at %v]\n", source, format, outPath, info.Size, info.Size, info.Position)
}

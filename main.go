package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"

	_ "github.com/chai2010/webp"
	_ "github.com/strukturag/libheif/go/heif"
)

func colorModelName(cm color.Model) string {
	switch cm {
	case color.RGBAModel:
		return "RGBA"
	case color.RGBA64Model:
		return "RGBA64"
	case color.NRGBAModel:
		return "NRGBA"
	case color.NRGBA64Model:
		return "NRGBA64"
	case color.AlphaModel:
		return "Alpha"
	case color.Alpha16Model:
		return "Alpha16"
	case color.GrayModel:
		return "Gray"
	case color.Gray16Model:
		return "Gray16"
	case color.YCbCrModel:
		return "YCbCr"
	case color.CMYKModel:
		return "CMYK"
	default:
		return "Unknown"
	}
}

func estimateDecodedSize(filename string) (int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Get original file size
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}
	originalSize := fileInfo.Size()

	// DecodeConfig reads only the image header, not the full image
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, err
	}

	width := config.Width
	height := config.Height

	var bytesPerPixel int

	// Determine bytes per pixel based on format and color model
	// This reflects how Go's image decoders actually work
	switch format {
	case "jpeg":
		// JPEG always decodes to YCbCr in Go (3 bytes/pixel: Y + Cb + Cr)
		bytesPerPixel = 3
	case "png":
		// PNG decoding depends on color model
		switch config.ColorModel {
		case color.GrayModel:
			bytesPerPixel = 1
		case color.Gray16Model:
			bytesPerPixel = 2
		case color.RGBA64Model, color.NRGBA64Model:
			bytesPerPixel = 8
		default:
			// Most PNG (RGB, RGBA, NRGBA) decode to 4 bytes/pixel
			// Even RGB without alpha uses RGBA format in Go
			bytesPerPixel = 4
		}
	case "heif", "avif":
		// HEIF/HEIC and AVIF support multiple color models
		// They can be YCbCr (most common), RGB, or monochrome
		// Check the actual color model from the header
		switch config.ColorModel {
		case color.GrayModel:
			// Monochrome/grayscale HEIF
			bytesPerPixel = 1
		case color.Gray16Model:
			// 10-bit or 16-bit grayscale
			bytesPerPixel = 2
		case color.YCbCrModel:
			// Most common: YCbCr (3 bytes/pixel)
			bytesPerPixel = 3
		case color.RGBA64Model, color.NRGBA64Model:
			// HDR content with 10-bit+ per channel
			bytesPerPixel = 8
		default:
			// RGB or RGBA typically decode to RGBA (4 bytes/pixel)
			// or YCbCr (3 bytes/pixel) - default to YCbCr as most common
			bytesPerPixel = 3
		}
	case "webp":
		// WebP decodes to RGBA (4 bytes/pixel)
		bytesPerPixel = 4
	default:
		// Fallback: use color model to estimate
		switch config.ColorModel {
		case color.GrayModel, color.AlphaModel:
			bytesPerPixel = 1
		case color.Gray16Model, color.Alpha16Model:
			bytesPerPixel = 2
		case color.RGBA64Model, color.NRGBA64Model:
			bytesPerPixel = 8
		default:
			bytesPerPixel = 4
		}
	}

	decodedSize := int64(width) * int64(height) * int64(bytesPerPixel)

	fmt.Printf("Format: %s\n", format)
	fmt.Printf("Dimensions: %dx%d\n", width, height)
	fmt.Printf("Color Model: %s\n", colorModelName(config.ColorModel))
	fmt.Printf("Original file size: %d bytes (%.2f MB)\n",
		originalSize, float64(originalSize)/(1024*1024))
	fmt.Printf("Estimated decoded size: %d bytes (%.2f MB)\n",
		decodedSize, float64(decodedSize)/(1024*1024))
	fmt.Printf("Compression ratio: %.1fx\n",
		float64(decodedSize)/float64(originalSize))

	return decodedSize, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: testdecode <image-file>")
		fmt.Println("Supported formats: PNG, JPEG, HEIF/HEIC, AVIF, WebP")
		os.Exit(1)
	}

	filename := os.Args[1]

	_, err := estimateDecodedSize(filename)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}

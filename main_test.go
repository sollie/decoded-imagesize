package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/chai2010/webp"
	"github.com/strukturag/libheif/go/heif"
)

var testDimensions = []struct {
	width  int
	height int
	name   string
}{
	{100, 100, "100x100"},
	{500, 500, "500x500"},
	{1000, 1000, "1000x1000"},
	{2000, 1500, "2000x1500"},
	{4000, 3000, "4000x3000"},
}

func generateGrayImage(width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetGray(x, y, color.Gray{Y: uint8((x + y) % 256)})
		}
	}
	return img
}

func generateRGBAImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}
	return img
}

func generateGray16Image(width, height int) *image.Gray16 {
	img := image.NewGray16(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetGray16(x, y, color.Gray16{Y: uint16((x + y) % 65536)})
		}
	}
	return img
}

func getActualDecodedSize(filename string) (int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var bytesPerPixel int
	switch img.(type) {
	case *image.RGBA, *image.NRGBA:
		bytesPerPixel = 4
	case *image.RGBA64, *image.NRGBA64:
		bytesPerPixel = 8
	case *image.YCbCr:
		bytesPerPixel = 3
	case *image.Gray:
		bytesPerPixel = 1
	case *image.Gray16:
		bytesPerPixel = 2
	case *image.Paletted:
		bytesPerPixel = 1
	case *image.CMYK:
		bytesPerPixel = 4
	default:
		bytesPerPixel = 4
	}

	return int64(width) * int64(height) * int64(bytesPerPixel), nil
}

func generatePalettedImage(width, height int) *image.Paletted {
	palette := make(color.Palette, 256)
	for i := 0; i < 256; i++ {
		palette[i] = color.RGBA{
			R: uint8(i),
			G: uint8(255 - i),
			B: uint8((i * 2) % 256),
			A: 255,
		}
	}

	img := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetColorIndex(x, y, uint8((x+y)%256))
		}
	}
	return img
}

func generateRGBA64Image(width, height int) *image.RGBA64 {
	img := image.NewRGBA64(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA64(x, y, color.RGBA64{
				R: uint16((x * 65535) / width),
				G: uint16((y * 65535) / height),
				B: uint16((x + y) % 65536),
				A: 65535,
			})
		}
	}
	return img
}

func TestPNGRGBAEstimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generateRGBAImage(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_rgba_"+dim.name+".png")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = png.Encode(file, img)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode PNG: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 4

			t.Logf("PNG RGBA %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != actual {
				t.Errorf("Size mismatch for %s: estimated=%d, actual=%d, diff=%d",
					dim.name, estimated, actual, estimated-actual)
			}

			if actual != expectedSize {
				t.Errorf("Unexpected actual size for %s: expected=%d, got=%d",
					dim.name, expectedSize, actual)
			}
		})
	}
}

func TestPNGGrayscaleEstimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generateGrayImage(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_gray_"+dim.name+".png")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = png.Encode(file, img)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode PNG: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 1

			t.Logf("PNG Grayscale %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != actual {
				t.Errorf("Size mismatch for %s: estimated=%d, actual=%d, diff=%d",
					dim.name, estimated, actual, estimated-actual)
			}

			if actual != expectedSize {
				t.Errorf("Unexpected actual size for %s: expected=%d, got=%d",
					dim.name, expectedSize, actual)
			}
		})
	}
}

func TestPNGGray16Estimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generateGray16Image(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_gray16_"+dim.name+".png")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = png.Encode(file, img)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode PNG: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 2

			t.Logf("PNG Gray16 %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != actual {
				t.Errorf("Size mismatch for %s: estimated=%d, actual=%d, diff=%d",
					dim.name, estimated, actual, estimated-actual)
			}

			if actual != expectedSize {
				t.Errorf("Unexpected actual size for %s: expected=%d, got=%d",
					dim.name, expectedSize, actual)
			}
		})
	}
}

func TestJPEGEstimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generateRGBAImage(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_"+dim.name+".jpg")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode JPEG: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 3

			t.Logf("JPEG %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != expectedSize {
				t.Errorf("Estimated size mismatch for %s: estimated=%d, expected=%d",
					dim.name, estimated, expectedSize)
			}

			if actual != expectedSize {
				t.Errorf("Actual size mismatch for %s: actual=%d, expected=%d",
					dim.name, actual, expectedSize)
			}

			if estimated != actual {
				t.Errorf("Estimation vs actual mismatch for %s: estimated=%d, actual=%d",
					dim.name, estimated, actual)
			}
		})
	}
}

func TestAccuracyAcrossAllFormats(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		width       int
		height      int
		generator   func(int, int) image.Image
		encoder     func(*os.File, image.Image) error
		extension   string
		expectBytes int
	}{
		{"PNG_RGBA_500x500", 500, 500, func(w, h int) image.Image { return generateRGBAImage(w, h) },
			func(f *os.File, img image.Image) error { return png.Encode(f, img) }, ".png", 4},
		{"PNG_Gray_500x500", 500, 500, func(w, h int) image.Image { return generateGrayImage(w, h) },
			func(f *os.File, img image.Image) error { return png.Encode(f, img) }, ".png", 1},
		{"PNG_Gray16_500x500", 500, 500, func(w, h int) image.Image { return generateGray16Image(w, h) },
			func(f *os.File, img image.Image) error { return png.Encode(f, img) }, ".png", 2},
		{"JPEG_1000x1000", 1000, 1000, func(w, h int) image.Image { return generateRGBAImage(w, h) },
			func(f *os.File, img image.Image) error { return jpeg.Encode(f, img, &jpeg.Options{Quality: 90}) }, ".jpg", 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			img := tc.generator(tc.width, tc.height)
			filename := filepath.Join(tmpDir, tc.name+tc.extension)

			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = tc.encoder(file, img)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode image: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expected := int64(tc.width) * int64(tc.height) * int64(tc.expectBytes)

			if estimated != expected || actual != expected {
				t.Errorf("%s: estimated=%d, actual=%d, expected=%d",
					tc.name, estimated, actual, expected)
			}
		})
	}
}

func TestWebPEstimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generateRGBAImage(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_"+dim.name+".webp")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = webp.Encode(file, img, &webp.Options{Lossless: false, Quality: 90})
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode WebP: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 4

			t.Logf("WebP %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != expectedSize {
				t.Errorf("Estimated size mismatch for %s: estimated=%d, expected=%d",
					dim.name, estimated, expectedSize)
			}

			if actual != expectedSize {
				t.Errorf("Actual size mismatch for %s: actual=%d, expected=%d",
					dim.name, actual, expectedSize)
			}

			if estimated != actual {
				t.Errorf("Estimation vs actual mismatch for %s: estimated=%d, actual=%d",
					dim.name, estimated, actual)
			}
		})
	}
}

func TestHEIFEstimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generateRGBAImage(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_"+dim.name+".heic")

			ctx, err := heif.EncodeFromImage(img, heif.CompressionHEVC, 90, heif.LosslessModeDisabled, heif.LoggingLevelNone)
			if err != nil {
				t.Fatalf("Failed to encode HEIF: %v", err)
			}

			err = ctx.WriteToFile(filename)
			if err != nil {
				t.Fatalf("Failed to write HEIF file: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 3

			t.Logf("HEIF %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != expectedSize {
				t.Errorf("Estimated size mismatch for %s: estimated=%d, expected=%d",
					dim.name, estimated, expectedSize)
			}

			if actual != expectedSize {
				t.Errorf("Actual size mismatch for %s: actual=%d, expected=%d",
					dim.name, actual, expectedSize)
			}

			if estimated != actual {
				t.Errorf("Estimation vs actual mismatch for %s: estimated=%d, actual=%d",
					dim.name, estimated, actual)
			}
		})
	}
}

func TestAVIFEstimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generateRGBAImage(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_"+dim.name+".avif")

			ctx, err := heif.EncodeFromImage(img, heif.CompressionAV1, 90, heif.LosslessModeDisabled, heif.LoggingLevelNone)
			if err != nil {
				t.Skipf("AVIF encoding not available (libheif may not be built with AV1 support): %v", err)
			}

			err = ctx.WriteToFile(filename)
			if err != nil {
				t.Fatalf("Failed to write AVIF file: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 3

			t.Logf("AVIF %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != expectedSize {
				t.Errorf("Estimated size mismatch for %s: estimated=%d, expected=%d",
					dim.name, estimated, expectedSize)
			}

			if actual != expectedSize {
				t.Errorf("Actual size mismatch for %s: actual=%d, expected=%d",
					dim.name, actual, expectedSize)
			}

			if estimated != actual {
				t.Errorf("Estimation vs actual mismatch for %s: estimated=%d, actual=%d",
					dim.name, estimated, actual)
			}
		})
	}
}

func TestWebPLosslessEstimation(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("RGBA_Lossless", func(t *testing.T) {
		img := generateRGBAImage(1000, 1000)
		filename := filepath.Join(tmpDir, "test_lossless.webp")
		file, err := os.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err = webp.Encode(file, img, &webp.Options{Lossless: true})
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatalf("Failed to encode WebP: %v", err)
		}

		estimated, err := estimateDecodedSize(filename)
		if err != nil {
			t.Fatalf("estimateDecodedSize failed: %v", err)
		}

		actual, err := getActualDecodedSize(filename)
		if err != nil {
			t.Fatalf("getActualDecodedSize failed: %v", err)
		}

		expectedSize := int64(1000 * 1000 * 4)

		fmt.Printf("Test result: estimated=%d bytes, actual=%d bytes, expected=%d bytes\n",
			estimated, actual, expectedSize)

		t.Logf("WebP Lossless: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
			estimated, actual, expectedSize)

		if estimated != expectedSize || actual != expectedSize {
			t.Errorf("Size mismatch: estimated=%d, actual=%d, expected=%d",
				estimated, actual, expectedSize)
		}
	})
}

func TestMultipleColorModels(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		format      string
		generator   func() image.Image
		encode      func(string, image.Image) error
		expectBytes int
	}{
		{
			name:   "PNG_Grayscale",
			format: "PNG",
			generator: func() image.Image {
				return generateGrayImage(500, 500)
			},
			encode: func(fn string, img image.Image) error {
				f, err := os.Create(fn)
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()
				return png.Encode(f, img)
			},
			expectBytes: 1,
		},
		{
			name:   "PNG_RGBA",
			format: "PNG",
			generator: func() image.Image {
				return generateRGBAImage(500, 500)
			},
			encode: func(fn string, img image.Image) error {
				f, err := os.Create(fn)
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()
				return png.Encode(f, img)
			},
			expectBytes: 4,
		},
		{
			name:   "JPEG_YCbCr",
			format: "JPEG",
			generator: func() image.Image {
				return generateRGBAImage(500, 500)
			},
			encode: func(fn string, img image.Image) error {
				f, err := os.Create(fn)
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()
				return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
			},
			expectBytes: 3,
		},
		{
			name:   "WebP_RGBA",
			format: "WebP",
			generator: func() image.Image {
				return generateRGBAImage(500, 500)
			},
			encode: func(fn string, img image.Image) error {
				f, err := os.Create(fn)
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()
				return webp.Encode(f, img, &webp.Options{Lossless: false, Quality: 90})
			},
			expectBytes: 4,
		},
		{
			name:   "HEIF_YCbCr",
			format: "HEIF",
			generator: func() image.Image {
				return generateRGBAImage(500, 500)
			},
			encode: func(fn string, img image.Image) error {
				ctx, err := heif.EncodeFromImage(img, heif.CompressionHEVC, 90, heif.LosslessModeDisabled, heif.LoggingLevelNone)
				if err != nil {
					return err
				}
				return ctx.WriteToFile(fn)
			},
			expectBytes: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img := tc.generator()
			filename := filepath.Join(tmpDir, tc.name+".img")

			err := tc.encode(filename, img)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expected := int64(500 * 500 * tc.expectBytes)

			fmt.Printf("Test result: estimated=%d bytes, actual=%d bytes, expected=%d bytes\n",
				estimated, actual, expected)

			t.Logf("%s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				tc.name, estimated, actual, expected)

			if estimated != expected {
				t.Errorf("%s: estimated size mismatch: got=%d, want=%d",
					tc.name, estimated, expected)
			}

			if actual != expected {
				t.Errorf("%s: actual size mismatch: got=%d, want=%d",
					tc.name, actual, expected)
			}

			if estimated != actual {
				t.Errorf("%s: estimated vs actual mismatch: estimated=%d, actual=%d",
					tc.name, estimated, actual)
			}
		})
	}
}

func TestPNGPalettedEstimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generatePalettedImage(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_paletted_"+dim.name+".png")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = png.Encode(file, img)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode PNG: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 1

			t.Logf("PNG Paletted %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != actual {
				t.Errorf("Size mismatch for %s: estimated=%d, actual=%d, diff=%d",
					dim.name, estimated, actual, estimated-actual)
			}

			if actual != expectedSize {
				t.Errorf("Unexpected actual size for %s: expected=%d, got=%d",
					dim.name, expectedSize, actual)
			}
		})
	}
}

func TestPNGRGBA64Estimation(t *testing.T) {
	tmpDir := t.TempDir()

	for _, dim := range testDimensions {
		t.Run(dim.name, func(t *testing.T) {
			img := generateRGBA64Image(dim.width, dim.height)

			filename := filepath.Join(tmpDir, "test_rgba64_"+dim.name+".png")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = png.Encode(file, img)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode PNG: %v", err)
			}

			estimated, err := estimateDecodedSize(filename)
			if err != nil {
				t.Fatalf("estimateDecodedSize failed: %v", err)
			}

			actual, err := getActualDecodedSize(filename)
			if err != nil {
				t.Fatalf("getActualDecodedSize failed: %v", err)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 8

			t.Logf("PNG RGBA64 %s: estimated=%d bytes, actual=%d bytes, expected=%d bytes",
				dim.name, estimated, actual, expectedSize)

			if estimated != actual {
				t.Errorf("Size mismatch for %s: estimated=%d, actual=%d, diff=%d",
					dim.name, estimated, actual, estimated-actual)
			}

			if actual != expectedSize {
				t.Errorf("Unexpected actual size for %s: expected=%d, got=%d",
					dim.name, expectedSize, actual)
			}
		})
	}
}

func TestBitDepthDetection(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		img      image.Image
		expected int
	}{
		{"Gray8", generateGrayImage(100, 100), 8},
		{"Gray16", generateGray16Image(100, 100), 16},
		{"RGBA", generateRGBAImage(100, 100), 8},
		{"RGBA64", generateRGBA64Image(100, 100), 16},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filename := filepath.Join(tmpDir, tc.name+".png")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = png.Encode(file, tc.img)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode PNG: %v", err)
			}

			f, err := os.Open(filename)
			if err != nil {
				t.Fatalf("Failed to open file: %v", err)
			}
			defer func() { _ = f.Close() }()
			bitDepth := detectPNGBitDepth(f)

			if bitDepth != tc.expected {
				t.Errorf("%s: bit depth mismatch: got=%d, want=%d", tc.name, bitDepth, tc.expected)
			}
		})
	}
}

func TestYCbCrSubsamplingDetection(t *testing.T) {
	tmpDir := t.TempDir()

	img := generateRGBAImage(500, 500)
	filename := filepath.Join(tmpDir, "test_ycbcr.jpg")

	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	err = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	if closeErr := file.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatalf("Failed to encode JPEG: %v", err)
	}

	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer func() { _ = f.Close() }()

	subsampling := detectJPEGSubsampling(f)
	t.Logf("Detected YCbCr subsampling: %s", subsampling)

	if subsampling == "Unknown" {
		t.Errorf("Failed to detect YCbCr subsampling")
	}
}

func TestImageInfoPNG(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name             string
		img              image.Image
		expectedModel    ColorModel
		expectedBitDepth int
		expectedAlpha    bool
		expectedChroma   ChromaSubsampling
		expectedHDR      HDRType
		expectedComp     CompressionType
	}{
		{
			name:             "PNG_RGBA",
			img:              generateRGBAImage(100, 100),
			expectedModel:    ColorModelRGB,
			expectedBitDepth: 8,
			expectedAlpha:    true,
			expectedChroma:   ChromaSubsamplingNA,
			expectedHDR:      HDRNone,
			expectedComp:     CompressionLossless,
		},
		{
			name:             "PNG_Gray",
			img:              generateGrayImage(100, 100),
			expectedModel:    ColorModelGrayscale,
			expectedBitDepth: 8,
			expectedAlpha:    false,
			expectedChroma:   ChromaSubsamplingNA,
			expectedHDR:      HDRNone,
			expectedComp:     CompressionLossless,
		},
		{
			name:             "PNG_Gray16",
			img:              generateGray16Image(100, 100),
			expectedModel:    ColorModelGrayscale,
			expectedBitDepth: 16,
			expectedAlpha:    false,
			expectedChroma:   ChromaSubsamplingNA,
			expectedHDR:      HDRNone,
			expectedComp:     CompressionLossless,
		},
		{
			name:             "PNG_RGBA64",
			img:              generateRGBA64Image(100, 100),
			expectedModel:    ColorModelRGB,
			expectedBitDepth: 16,
			expectedAlpha:    true,
			expectedChroma:   ChromaSubsamplingNA,
			expectedHDR:      HDRNone,
			expectedComp:     CompressionLossless,
		},
		{
			name:             "PNG_Paletted",
			img:              generatePalettedImage(100, 100),
			expectedModel:    ColorModelIndexed,
			expectedBitDepth: 8,
			expectedAlpha:    false,
			expectedChroma:   ChromaSubsamplingNA,
			expectedHDR:      HDRNone,
			expectedComp:     CompressionLossless,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filename := filepath.Join(tmpDir, tc.name+".png")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = png.Encode(file, tc.img)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode PNG: %v", err)
			}

			info, err := analyzeImage(filename)
			if err != nil {
				t.Fatalf("analyzeImage failed: %v", err)
			}

			if info.Format != "png" {
				t.Errorf("Format mismatch: got=%s, want=png", info.Format)
			}

			if info.ColorModel != tc.expectedModel {
				t.Errorf("ColorModel mismatch: got=%s, want=%s", info.ColorModel, tc.expectedModel)
			}

			if info.BitDepth != tc.expectedBitDepth {
				t.Errorf("BitDepth mismatch: got=%d, want=%d", info.BitDepth, tc.expectedBitDepth)
			}

			if info.HasAlpha != tc.expectedAlpha {
				t.Errorf("HasAlpha mismatch: got=%v, want=%v", info.HasAlpha, tc.expectedAlpha)
			}

			if info.ChromaSubsampling != tc.expectedChroma {
				t.Errorf("ChromaSubsampling mismatch: got=%s, want=%s", info.ChromaSubsampling, tc.expectedChroma)
			}

			if info.HDRType != tc.expectedHDR {
				t.Errorf("HDRType mismatch: got=%s, want=%s", info.HDRType, tc.expectedHDR)
			}

			if info.CompressionType != tc.expectedComp {
				t.Errorf("CompressionType mismatch: got=%s, want=%s", info.CompressionType, tc.expectedComp)
			}

			if info.ColorSpace != ColorSpaceSRGB {
				t.Errorf("ColorSpace mismatch: got=%s, want=sRGB", info.ColorSpace)
			}
		})
	}
}

func TestImageInfoJPEG(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name             string
		img              image.Image
		expectedModel    ColorModel
		expectedBitDepth int
		expectedAlpha    bool
		expectedHDR      HDRType
		expectedComp     CompressionType
	}{
		{
			name:             "JPEG_Color",
			img:              generateRGBAImage(100, 100),
			expectedModel:    ColorModelYCbCr,
			expectedBitDepth: 8,
			expectedAlpha:    false,
			expectedHDR:      HDRNone,
			expectedComp:     CompressionLossy,
		},
		{
			name:             "JPEG_Grayscale",
			img:              generateGrayImage(100, 100),
			expectedModel:    ColorModelGrayscale,
			expectedBitDepth: 8,
			expectedAlpha:    false,
			expectedHDR:      HDRNone,
			expectedComp:     CompressionLossy,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filename := filepath.Join(tmpDir, tc.name+".jpg")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = jpeg.Encode(file, tc.img, &jpeg.Options{Quality: 90})
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode JPEG: %v", err)
			}

			info, err := analyzeImage(filename)
			if err != nil {
				t.Fatalf("analyzeImage failed: %v", err)
			}

			t.Logf("JPEG Analysis: ColorModel=%s, ChromaSubsampling=%s, BitDepth=%d",
				info.ColorModel, info.ChromaSubsampling, info.BitDepth)

			if info.Format != "jpeg" {
				t.Errorf("Format mismatch: got=%s, want=jpeg", info.Format)
			}

			if info.ColorModel != tc.expectedModel {
				t.Errorf("ColorModel mismatch: got=%s, want=%s", info.ColorModel, tc.expectedModel)
			}

			if info.BitDepth != tc.expectedBitDepth {
				t.Errorf("BitDepth mismatch: got=%d, want=%d", info.BitDepth, tc.expectedBitDepth)
			}

			if info.HasAlpha != tc.expectedAlpha {
				t.Errorf("HasAlpha mismatch: got=%v, want=%v", info.HasAlpha, tc.expectedAlpha)
			}

			if info.HDRType != tc.expectedHDR {
				t.Errorf("HDRType mismatch: got=%s, want=%s", info.HDRType, tc.expectedHDR)
			}

			if info.CompressionType != tc.expectedComp {
				t.Errorf("CompressionType mismatch: got=%s, want=%s", info.CompressionType, tc.expectedComp)
			}

			if tc.expectedModel == ColorModelYCbCr {
				if info.ChromaSubsampling == ChromaSubsamplingUnknown {
					t.Errorf("ChromaSubsampling should be detected for YCbCr JPEG")
				}
			}
		})
	}
}

func TestImageInfoWebP(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		img            image.Image
		lossless       bool
		expectedModel  ColorModel
		expectedComp   CompressionType
		expectedChroma ChromaSubsampling
	}{
		{
			name:           "WebP_Lossless",
			img:            generateRGBAImage(100, 100),
			lossless:       true,
			expectedModel:  ColorModelRGB,
			expectedComp:   CompressionLossless,
			expectedChroma: ChromaSubsamplingNA,
		},
		{
			name:           "WebP_Lossy",
			img:            generateRGBAImage(100, 100),
			lossless:       false,
			expectedModel:  ColorModelRGB,
			expectedComp:   CompressionLossy,
			expectedChroma: ChromaSubsampling420,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filename := filepath.Join(tmpDir, tc.name+".webp")
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			err = webp.Encode(file, tc.img, &webp.Options{Lossless: tc.lossless, Quality: 90})
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("Failed to encode WebP: %v", err)
			}

			info, err := analyzeImage(filename)
			if err != nil {
				t.Fatalf("analyzeImage failed: %v", err)
			}

			if info.Format != "webp" {
				t.Errorf("Format mismatch: got=%s, want=webp", info.Format)
			}

			if info.ColorModel != tc.expectedModel {
				t.Errorf("ColorModel mismatch: got=%s, want=%s", info.ColorModel, tc.expectedModel)
			}

			if info.CompressionType != tc.expectedComp {
				t.Errorf("CompressionType mismatch: got=%s, want=%s", info.CompressionType, tc.expectedComp)
			}

			if info.ChromaSubsampling != tc.expectedChroma {
				t.Errorf("ChromaSubsampling mismatch: got=%s, want=%s", info.ChromaSubsampling, tc.expectedChroma)
			}

			if info.BitDepth != 8 {
				t.Errorf("BitDepth mismatch: got=%d, want=8", info.BitDepth)
			}

			if info.HDRType != HDRNone {
				t.Errorf("HDRType should be None for WebP, got=%s", info.HDRType)
			}
		})
	}
}

func TestStringMethods(t *testing.T) {
	t.Run("ColorModel", func(t *testing.T) {
		tests := []struct {
			model    ColorModel
			expected string
		}{
			{ColorModelRGB, "RGB"},
			{ColorModelYCbCr, "YCbCr"},
			{ColorModelGrayscale, "Grayscale"},
			{ColorModelIndexed, "Indexed"},
			{ColorModelUnknown, "Unknown"},
			{ColorModel(999), "Unknown"},
		}

		for _, tc := range tests {
			if got := tc.model.String(); got != tc.expected {
				t.Errorf("ColorModel(%d).String() = %s, want %s", tc.model, got, tc.expected)
			}
		}
	})

	t.Run("ColorSpace", func(t *testing.T) {
		tests := []struct {
			space    ColorSpace
			expected string
		}{
			{ColorSpaceSRGB, "sRGB"},
			{ColorSpaceAdobeRGB, "Adobe RGB"},
			{ColorSpaceBT709, "BT.709"},
			{ColorSpaceBT2020, "BT.2020"},
			{ColorSpaceDisplayP3, "Display P3"},
			{ColorSpaceUnknown, "Unknown"},
			{ColorSpace(999), "Unknown"},
		}

		for _, tc := range tests {
			if got := tc.space.String(); got != tc.expected {
				t.Errorf("ColorSpace(%d).String() = %s, want %s", tc.space, got, tc.expected)
			}
		}
	})

	t.Run("HDRType", func(t *testing.T) {
		tests := []struct {
			hdr      HDRType
			expected string
		}{
			{HDRNone, "None"},
			{HDRPQ, "PQ (SMPTE ST 2084)"},
			{HDRHLG, "HLG (ARIB STD-B67)"},
			{HDRLimited, "Limited"},
			{HDRType(999), "Unknown"},
		}

		for _, tc := range tests {
			if got := tc.hdr.String(); got != tc.expected {
				t.Errorf("HDRType(%d).String() = %s, want %s", tc.hdr, got, tc.expected)
			}
		}
	})

	t.Run("ChromaSubsampling", func(t *testing.T) {
		tests := []struct {
			chroma   ChromaSubsampling
			expected string
		}{
			{ChromaSubsampling444, "4:4:4"},
			{ChromaSubsampling422, "4:2:2"},
			{ChromaSubsampling420, "4:2:0"},
			{ChromaSubsamplingNA, "N/A"},
			{ChromaSubsamplingUnknown, "Unknown"},
			{ChromaSubsampling(999), "Unknown"},
		}

		for _, tc := range tests {
			if got := tc.chroma.String(); got != tc.expected {
				t.Errorf("ChromaSubsampling(%d).String() = %s, want %s", tc.chroma, got, tc.expected)
			}
		}
	})

	t.Run("CompressionType", func(t *testing.T) {
		tests := []struct {
			comp     CompressionType
			expected string
		}{
			{CompressionLossless, "Lossless"},
			{CompressionLossy, "Lossy"},
			{CompressionHybrid, "Lossy/Lossless"},
			{CompressionUnknown, "Unknown"},
			{CompressionType(999), "Unknown"},
		}

		for _, tc := range tests {
			if got := tc.comp.String(); got != tc.expected {
				t.Errorf("CompressionType(%d).String() = %s, want %s", tc.comp, got, tc.expected)
			}
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := analyzeImage("/nonexistent/file.png")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("InvalidImageFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "invalid.png")

		err := os.WriteFile(filename, []byte("not a valid image"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = analyzeImage(filename)
		if err == nil {
			t.Error("Expected error for invalid image file, got nil")
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "empty.png")

		err := os.WriteFile(filename, []byte{}, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = analyzeImage(filename)
		if err == nil {
			t.Error("Expected error for empty file, got nil")
		}
	})

	t.Run("EstimateDecodedSize_NonExistent", func(t *testing.T) {
		_, err := estimateDecodedSize("/nonexistent/file.png")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("EstimateDecodedSize_Invalid", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "invalid.jpg")

		err := os.WriteFile(filename, []byte("not a valid image"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = estimateDecodedSize(filename)
		if err == nil {
			t.Error("Expected error for invalid image file, got nil")
		}
	})
}

func TestParseColorSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected ColorSpace
	}{
		{"sRGB", ColorSpaceSRGB},
		{"sRGB (ICC)", ColorSpaceSRGB},
		{"Adobe RGB", ColorSpaceAdobeRGB},
		{"BT.709", ColorSpaceBT709},
		{"BT.2020", ColorSpaceBT2020},
		{"Display P3", ColorSpaceDisplayP3},
		{"Unknown Profile", ColorSpaceSRGB},
		{"", ColorSpaceSRGB},
	}

	for _, tc := range tests {
		if got := parseColorSpace(tc.input); got != tc.expected {
			t.Errorf("parseColorSpace(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestDetectColorSpaceFromICC(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "TooShort",
			data:     make([]byte, 100),
			expected: "sRGB",
		},
		{
			name:     "DisplayP3",
			data:     append(make([]byte, 128), []byte("Display P3 profile data")...),
			expected: "Display P3",
		},
		{
			name:     "DisplayP3_ShortName",
			data:     append(make([]byte, 128), []byte("P3 profile")...),
			expected: "Display P3",
		},
		{
			name:     "BT2020",
			data:     append(make([]byte, 128), []byte("BT.2020 profile data")...),
			expected: "BT.2020",
		},
		{
			name:     "BT2020_AltName",
			data:     append(make([]byte, 128), []byte("Rec. 2020 profile")...),
			expected: "BT.2020",
		},
		{
			name:     "BT709",
			data:     append(make([]byte, 128), []byte("BT.709 profile data")...),
			expected: "BT.709",
		},
		{
			name:     "BT709_AltName",
			data:     append(make([]byte, 128), []byte("Rec. 709 profile")...),
			expected: "BT.709",
		},
		{
			name:     "AdobeRGB",
			data:     append(make([]byte, 128), []byte("Adobe RGB profile data")...),
			expected: "Adobe RGB",
		},
		{
			name:     "DefaultSRGB",
			data:     append(make([]byte, 128), []byte("Some other profile")...),
			expected: "sRGB (ICC)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectColorSpaceFromICC(tc.data); got != tc.expected {
				t.Errorf("detectColorSpaceFromICC() = %s, want %s", got, tc.expected)
			}
		})
	}
}

func createPNGWithICCProfile(filename string, img image.Image, iccProfileName string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return err
	}

	pngData := buf.Bytes()
	if len(pngData) < 8 {
		return fmt.Errorf("invalid PNG data")
	}

	iccProfile := make([]byte, 128)
	copy(iccProfile, []byte("ICC Profile Header Padding"))

	switch iccProfileName {
	case "Display P3":
		iccProfile = append(iccProfile, []byte("Display P3 color profile embedded data for testing purposes")...)
	case "Adobe RGB":
		iccProfile = append(iccProfile, []byte("Adobe RGB color profile embedded data for testing purposes")...)
	case "BT.709":
		iccProfile = append(iccProfile, []byte("BT.709 color profile embedded data for testing purposes")...)
	case "BT.2020":
		iccProfile = append(iccProfile, []byte("BT.2020 color profile embedded data for testing purposes")...)
	default:
		iccProfile = append(iccProfile, []byte("sRGB color profile embedded data")...)
	}

	var compressed bytes.Buffer
	zlibWriter := zlib.NewWriter(&compressed)
	_, _ = zlibWriter.Write(iccProfile)
	_ = zlibWriter.Close()

	profileName := []byte(iccProfileName)
	profileName = append(profileName, 0)
	profileName = append(profileName, 0)
	iccpChunk := append(profileName, compressed.Bytes()...)

	var newPNG bytes.Buffer
	newPNG.Write(pngData[:8])

	pos := 8
	for pos < len(pngData) {
		if pos+8 > len(pngData) {
			break
		}

		length := binary.BigEndian.Uint32(pngData[pos : pos+4])
		chunkType := string(pngData[pos+4 : pos+8])

		if chunkType == "IHDR" {
			totalChunkSize := int(length) + 12
			if pos+totalChunkSize > len(pngData) {
				break
			}
			newPNG.Write(pngData[pos : pos+totalChunkSize])

			iccpLength := uint32(len(iccpChunk))
			var iccpHeader [8]byte
			binary.BigEndian.PutUint32(iccpHeader[0:4], iccpLength)
			copy(iccpHeader[4:8], "iCCP")
			newPNG.Write(iccpHeader[:])
			newPNG.Write(iccpChunk)

			crc := crc32PNG(append([]byte("iCCP"), iccpChunk...))
			var crcBytes [4]byte
			binary.BigEndian.PutUint32(crcBytes[:], crc)
			newPNG.Write(crcBytes[:])

			pos += totalChunkSize
		} else {
			totalChunkSize := int(length) + 12
			if pos+totalChunkSize > len(pngData) {
				newPNG.Write(pngData[pos:])
				break
			}
			newPNG.Write(pngData[pos : pos+totalChunkSize])
			pos += totalChunkSize
		}
	}

	_, err = file.Write(newPNG.Bytes())
	return err
}

func crc32PNG(data []byte) uint32 {
	crc := uint32(0xFFFFFFFF)
	for _, b := range data {
		crc ^= uint32(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xEDB88320
			} else {
				crc = crc >> 1
			}
		}
	}
	return crc ^ 0xFFFFFFFF
}

func createJPEGWithICCProfile(filename string, img image.Image, iccProfileName string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	if err != nil {
		return err
	}

	jpegData := buf.Bytes()

	iccProfile := make([]byte, 128)
	copy(iccProfile, []byte("ICC Profile Header Padding"))

	switch iccProfileName {
	case "Display P3":
		iccProfile = append(iccProfile, []byte("Display P3 color profile embedded in JPEG data for testing")...)
	case "Adobe RGB":
		iccProfile = append(iccProfile, []byte("Adobe RGB color profile embedded in JPEG data for testing")...)
	case "BT.709":
		iccProfile = append(iccProfile, []byte("BT.709 color profile embedded in JPEG data for testing")...)
	case "BT.2020":
		iccProfile = append(iccProfile, []byte("BT.2020 color profile embedded in JPEG data for testing")...)
	default:
		iccProfile = append(iccProfile, []byte("sRGB color profile embedded in JPEG data")...)
	}

	iccMarker := []byte{0xFF, 0xE2}
	iccHeader := []byte("ICC_PROFILE\x00")
	iccSeqNum := []byte{1, 1}
	iccData := append(iccHeader, iccSeqNum...)
	iccData = append(iccData, iccProfile...)

	markerLength := uint16(len(iccData) + 2)
	var lengthBytes [2]byte
	binary.BigEndian.PutUint16(lengthBytes[:], markerLength)

	var newJPEG bytes.Buffer
	newJPEG.Write(jpegData[:2])
	newJPEG.Write(iccMarker)
	newJPEG.Write(lengthBytes[:])
	newJPEG.Write(iccData)
	newJPEG.Write(jpegData[2:])

	_, err = file.Write(newJPEG.Bytes())
	return err
}

func TestICCProfileDetection(t *testing.T) {
	tmpDir := t.TempDir()

	colorSpaces := []string{"Display P3", "Adobe RGB", "BT.709", "BT.2020"}

	for _, cs := range colorSpaces {
		t.Run("PNG_"+cs, func(t *testing.T) {
			img := generateRGBAImage(100, 100)
			filename := filepath.Join(tmpDir, "icc_"+cs+".png")

			err := createPNGWithICCProfile(filename, img, cs)
			if err != nil {
				t.Fatalf("Failed to create PNG with ICC profile: %v", err)
			}

			info, err := analyzeImage(filename)
			if err != nil {
				t.Fatalf("analyzeImage failed: %v", err)
			}

			if !info.HasICCProfile {
				t.Error("Expected ICC profile to be detected")
			}

			if info.ICCProfileSize == 0 {
				t.Error("Expected ICC profile size > 0")
			}

		})

		t.Run("JPEG_"+cs, func(t *testing.T) {
			img := generateRGBAImage(100, 100)
			filename := filepath.Join(tmpDir, "icc_"+cs+".jpg")

			err := createJPEGWithICCProfile(filename, img, cs)
			if err != nil {
				t.Fatalf("Failed to create JPEG with ICC profile: %v", err)
			}

			info, err := analyzeImage(filename)
			if err != nil {
				t.Fatalf("analyzeImage failed: %v", err)
			}

			if !info.HasICCProfile {
				t.Error("Expected ICC profile to be detected")
			}

			if info.ICCProfileSize == 0 {
				t.Error("Expected ICC profile size > 0")
			}

			expectedColorSpace := parseColorSpace(cs)
			if info.ColorSpace != expectedColorSpace {
				t.Errorf("ColorSpace mismatch: got=%s, want=%s", info.ColorSpace, expectedColorSpace)
			}
		})
	}
}

func TestJPEGSubsamplingDetection(t *testing.T) {
	tests := []struct {
		name              string
		yH, yV, cbH, cbV  uint8
		expectedSubsample string
	}{
		{"4:4:4", 1, 1, 1, 1, "4:4:4"},
		{"4:2:2", 2, 1, 1, 1, "4:2:2"},
		{"4:2:0", 2, 2, 1, 1, "4:2:0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jpegData := createMinimalJPEGData(100, 100, tc.yH, tc.yV, tc.cbH, tc.cbV, 8)
			reader := bytes.NewReader(jpegData)

			result := detectJPEGSubsampling(reader)
			if result != tc.expectedSubsample {
				t.Errorf("Subsampling mismatch: got=%s, want=%s", result, tc.expectedSubsample)
			}
		})
	}
}

func Test12BitJPEGDetection(t *testing.T) {
	tests := []struct {
		name       string
		precision  uint8
		expected12 bool
	}{
		{"8-bit", 8, false},
		{"12-bit", 12, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jpegData := createMinimalJPEGData(100, 100, 2, 2, 1, 1, tc.precision)
			reader := bytes.NewReader(jpegData)

			result := is12BitJPEG(reader)
			if result != tc.expected12 {
				t.Errorf("12-bit detection mismatch: got=%v, want=%v", result, tc.expected12)
			}
		})
	}
}

func createMinimalJPEGData(width, height int, yH, yV, cbH, cbV, precision uint8) []byte {
	var buf bytes.Buffer

	buf.Write([]byte{0xFF, 0xD8})

	marker := uint8(0xC0)
	if precision == 12 {
		marker = 0xC1
	}
	buf.Write([]byte{0xFF, marker})

	sofLength := uint16(8 + 3*3)
	_ = binary.Write(&buf, binary.BigEndian, sofLength)
	buf.WriteByte(precision)
	_ = binary.Write(&buf, binary.BigEndian, uint16(height))
	_ = binary.Write(&buf, binary.BigEndian, uint16(width))
	buf.WriteByte(3)

	buf.WriteByte(1)
	buf.WriteByte((yH << 4) | yV)
	buf.WriteByte(0)

	buf.WriteByte(2)
	buf.WriteByte((cbH << 4) | cbV)
	buf.WriteByte(1)

	buf.WriteByte(3)
	buf.WriteByte((cbH << 4) | cbV)
	buf.WriteByte(1)

	buf.Write([]byte{0xFF, 0xD9})

	return buf.Bytes()
}

func TestHEIFMetadataBoxParsing(t *testing.T) {
	tests := []struct {
		name               string
		colorPrimaries     uint16
		transferChar       uint16
		bitDepth           uint8
		hasAlpha           bool
		expectedColorSpace ColorSpace
		expectedHDR        HDRType
	}{
		{"BT709_SDR", 1, 1, 8, false, ColorSpaceBT709, HDRNone},
		{"BT2020_PQ", 9, 16, 10, false, ColorSpaceBT2020, HDRPQ},
		{"BT2020_HLG", 9, 18, 10, false, ColorSpaceBT2020, HDRHLG},
		{"DisplayP3", 12, 1, 8, false, ColorSpaceDisplayP3, HDRNone},
		{"WithAlpha", 1, 1, 8, true, ColorSpaceBT709, HDRNone},
		{"10bit", 1, 1, 10, false, ColorSpaceBT709, HDRNone},
		{"12bit", 1, 1, 12, false, ColorSpaceBT709, HDRNone},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			heifData := createMinimalHEIFMetadata(tc.colorPrimaries, tc.transferChar, tc.bitDepth, tc.hasAlpha)
			reader := bytes.NewReader(heifData)

			metadata := parseHEIFMetadata(reader)

			if metadata.ColorSpace != tc.expectedColorSpace {
				t.Errorf("ColorSpace mismatch: got=%s, want=%s", metadata.ColorSpace, tc.expectedColorSpace)
			}

			if metadata.HDRType != tc.expectedHDR {
				t.Errorf("HDRType mismatch: got=%s, want=%s", metadata.HDRType, tc.expectedHDR)
			}

			if metadata.BitDepth != int(tc.bitDepth) {
				t.Errorf("BitDepth mismatch: got=%d, want=%d", metadata.BitDepth, tc.bitDepth)
			}

			if metadata.HasAlpha != tc.hasAlpha {
				t.Errorf("HasAlpha mismatch: got=%v, want=%v", metadata.HasAlpha, tc.hasAlpha)
			}
		})
	}
}

func createMinimalHEIFMetadata(colorPrimaries, transferChar uint16, bitDepth uint8, hasAlpha bool) []byte {
	var buf bytes.Buffer

	writeBox := func(boxType string, data []byte) {
		length := uint32(len(data) + 8)
		_ = binary.Write(&buf, binary.BigEndian, length)
		buf.WriteString(boxType)
		buf.Write(data)
	}

	var ftypData bytes.Buffer
	ftypData.WriteString("heic")
	_ = binary.Write(&ftypData, binary.BigEndian, uint32(0))
	ftypData.WriteString("heic")
	writeBox("ftyp", ftypData.Bytes())

	var metaData bytes.Buffer
	_ = binary.Write(&metaData, binary.BigEndian, uint32(0))

	var iprpData bytes.Buffer
	var ipcoData bytes.Buffer

	var pixiData bytes.Buffer
	pixiData.WriteByte(0)
	pixiData.WriteByte(3)
	pixiData.WriteByte(bitDepth)
	pixiData.WriteByte(bitDepth)
	pixiData.WriteByte(bitDepth)
	pixiLength := uint32(len(pixiData.Bytes()) + 8)
	_ = binary.Write(&ipcoData, binary.BigEndian, pixiLength)
	ipcoData.WriteString("pixi")
	ipcoData.Write(pixiData.Bytes())

	var colrData bytes.Buffer
	colrData.WriteString("nclx")
	_ = binary.Write(&colrData, binary.BigEndian, colorPrimaries)
	_ = binary.Write(&colrData, binary.BigEndian, transferChar)
	_ = binary.Write(&colrData, binary.BigEndian, uint16(1))
	colrData.WriteByte(1)
	colrLength := uint32(len(colrData.Bytes()) + 8)
	_ = binary.Write(&ipcoData, binary.BigEndian, colrLength)
	ipcoData.WriteString("colr")
	ipcoData.Write(colrData.Bytes())

	if hasAlpha {
		var auxCData bytes.Buffer
		auxCData.WriteString("urn:mpeg:mpegB:cicp:systems:auxiliary:alpha")
		auxCData.WriteByte(0)
		auxCLength := uint32(len(auxCData.Bytes()) + 8)
		_ = binary.Write(&ipcoData, binary.BigEndian, auxCLength)
		ipcoData.WriteString("auxC")
		ipcoData.Write(auxCData.Bytes())
	}

	ipcoLength := uint32(ipcoData.Len() + 8)
	_ = binary.Write(&iprpData, binary.BigEndian, ipcoLength)
	iprpData.WriteString("ipco")
	iprpData.Write(ipcoData.Bytes())

	iprpLength := uint32(iprpData.Len() + 8)
	_ = binary.Write(&metaData, binary.BigEndian, iprpLength)
	metaData.WriteString("iprp")
	metaData.Write(iprpData.Bytes())

	writeBox("meta", metaData.Bytes())

	return buf.Bytes()
}

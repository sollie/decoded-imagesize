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

func TestWebPFormatDetection(t *testing.T) {
	t.Run("VP8X_Extended", func(t *testing.T) {
		webpData := createWebPData("VP8X")
		reader := bytes.NewReader(webpData)

		isLossless, chroma := detectWebPFormat(reader)
		if isLossless {
			t.Error("VP8X should not be detected as lossless")
		}
		if chroma != ChromaSubsamplingUnknown {
			t.Errorf("VP8X chroma: got=%s, want=%s", chroma, ChromaSubsamplingUnknown)
		}
	})

	t.Run("VP8L_Lossless", func(t *testing.T) {
		webpData := createWebPData("VP8L")
		reader := bytes.NewReader(webpData)

		isLossless, chroma := detectWebPFormat(reader)
		if !isLossless {
			t.Error("VP8L should be detected as lossless")
		}
		if chroma != ChromaSubsamplingNA {
			t.Errorf("VP8L chroma: got=%s, want=%s", chroma, ChromaSubsamplingNA)
		}
	})

	t.Run("VP8_Lossy", func(t *testing.T) {
		webpData := createWebPData("VP8 ")
		reader := bytes.NewReader(webpData)

		isLossless, chroma := detectWebPFormat(reader)
		if isLossless {
			t.Error("VP8 should not be detected as lossless")
		}
		if chroma != ChromaSubsampling420 {
			t.Errorf("VP8 chroma: got=%s, want=%s", chroma, ChromaSubsampling420)
		}
	})

	t.Run("TruncatedHeader", func(t *testing.T) {
		webpData := []byte("RIFF")
		reader := bytes.NewReader(webpData)

		isLossless, chroma := detectWebPFormat(reader)
		if isLossless {
			t.Error("Truncated file should not be lossless")
		}
		if chroma != ChromaSubsamplingUnknown {
			t.Errorf("Truncated chroma: got=%s, want=%s", chroma, ChromaSubsamplingUnknown)
		}
	})

	t.Run("InvalidRIFF", func(t *testing.T) {
		webpData := []byte("JUNK____WEBP____")
		reader := bytes.NewReader(webpData)

		isLossless, chroma := detectWebPFormat(reader)
		if isLossless {
			t.Error("Invalid RIFF should not be lossless")
		}
		if chroma != ChromaSubsamplingUnknown {
			t.Errorf("Invalid RIFF chroma: got=%s, want=%s", chroma, ChromaSubsamplingUnknown)
		}
	})

	t.Run("InvalidWEBP", func(t *testing.T) {
		webpData := []byte("RIFF____JUNK____")
		reader := bytes.NewReader(webpData)

		isLossless, chroma := detectWebPFormat(reader)
		if isLossless {
			t.Error("Invalid WEBP should not be lossless")
		}
		if chroma != ChromaSubsamplingUnknown {
			t.Errorf("Invalid WEBP chroma: got=%s, want=%s", chroma, ChromaSubsamplingUnknown)
		}
	})

	t.Run("TruncatedChunkHeader", func(t *testing.T) {
		webpData := []byte("RIFF\x00\x00\x00\x00WEBP")
		reader := bytes.NewReader(webpData)

		isLossless, chroma := detectWebPFormat(reader)
		if isLossless {
			t.Error("Truncated chunk should not be lossless")
		}
		if chroma != ChromaSubsamplingUnknown {
			t.Errorf("Truncated chunk chroma: got=%s, want=%s", chroma, ChromaSubsamplingUnknown)
		}
	})
}

func createWebPData(fourCC string) []byte {
	var buf bytes.Buffer

	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(100))
	buf.WriteString("WEBP")
	buf.WriteString(fourCC)

	return buf.Bytes()
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

func TestHEIFMetadataEdgeCases(t *testing.T) {
	t.Run("SmallFile_LessThan12Bytes", func(t *testing.T) {
		data := []byte("short")
		reader := bytes.NewReader(data)

		metadata := parseHEIFMetadata(reader)

		if metadata.BitDepth != 8 {
			t.Errorf("Expected default BitDepth=8, got=%d", metadata.BitDepth)
		}
		if metadata.ColorSpace != ColorSpaceBT709 {
			t.Errorf("Expected default ColorSpace=BT.709, got=%s", metadata.ColorSpace)
		}
	})

	t.Run("MissingFtypBox", func(t *testing.T) {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.BigEndian, uint32(12))
		buf.WriteString("junk")
		buf.Write(make([]byte, 4))

		reader := bytes.NewReader(buf.Bytes())
		metadata := parseHEIFMetadata(reader)

		if metadata.BitDepth != 8 {
			t.Errorf("Expected default BitDepth=8, got=%d", metadata.BitDepth)
		}
	})

	t.Run("BoxSizeZero", func(t *testing.T) {
		var buf bytes.Buffer

		_ = binary.Write(&buf, binary.BigEndian, uint32(16))
		buf.WriteString("ftyp")
		buf.WriteString("heic")
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))

		_ = binary.Write(&buf, binary.BigEndian, uint32(0))
		buf.WriteString("meta")

		reader := bytes.NewReader(buf.Bytes())
		metadata := parseHEIFMetadata(reader)

		if metadata.ColorSpace != ColorSpaceBT709 {
			t.Errorf("Expected default ColorSpace=BT.709, got=%s", metadata.ColorSpace)
		}
	})

	t.Run("BoxSizeLessThan8", func(t *testing.T) {
		var buf bytes.Buffer

		_ = binary.Write(&buf, binary.BigEndian, uint32(16))
		buf.WriteString("ftyp")
		buf.WriteString("heic")
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))

		_ = binary.Write(&buf, binary.BigEndian, uint32(4))
		buf.WriteString("meta")

		reader := bytes.NewReader(buf.Bytes())
		metadata := parseHEIFMetadata(reader)

		if metadata.ColorSpace != ColorSpaceBT709 {
			t.Errorf("Expected default ColorSpace=BT.709, got=%s", metadata.ColorSpace)
		}
	})

	t.Run("BoxSizeExceedsData", func(t *testing.T) {
		var buf bytes.Buffer

		_ = binary.Write(&buf, binary.BigEndian, uint32(16))
		buf.WriteString("ftyp")
		buf.WriteString("heic")
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))

		_ = binary.Write(&buf, binary.BigEndian, uint32(10000))
		buf.WriteString("meta")
		buf.Write([]byte("truncated"))

		reader := bytes.NewReader(buf.Bytes())
		metadata := parseHEIFMetadata(reader)

		if metadata.ColorSpace != ColorSpaceBT709 {
			t.Errorf("Expected ColorSpace=BT.709, got=%s", metadata.ColorSpace)
		}
	})

	t.Run("DataTruncatedDuringParsing", func(t *testing.T) {
		var buf bytes.Buffer

		_ = binary.Write(&buf, binary.BigEndian, uint32(16))
		buf.WriteString("ftyp")
		buf.WriteString("heic")
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))

		_ = binary.Write(&buf, binary.BigEndian, uint32(20))
		buf.WriteString("meta")
		buf.Write([]byte("data"))

		reader := bytes.NewReader(buf.Bytes())
		metadata := parseHEIFMetadata(reader)

		if metadata.ColorSpace != ColorSpaceBT709 {
			t.Errorf("Expected ColorSpace=BT.709, got=%s", metadata.ColorSpace)
		}
	})
}

func TestMapStdColorModel(t *testing.T) {
	t.Run("AlphaModel", func(t *testing.T) {
		cm, hasAlpha := mapStdColorModel(color.AlphaModel)
		if cm != ColorModelGrayscale {
			t.Errorf("AlphaModel: expected Grayscale, got %s", cm)
		}
		if !hasAlpha {
			t.Error("AlphaModel: expected hasAlpha=true")
		}
	})

	t.Run("Alpha16Model", func(t *testing.T) {
		cm, hasAlpha := mapStdColorModel(color.Alpha16Model)
		if cm != ColorModelGrayscale {
			t.Errorf("Alpha16Model: expected Grayscale, got %s", cm)
		}
		if !hasAlpha {
			t.Error("Alpha16Model: expected hasAlpha=true")
		}
	})

	t.Run("RGBA64Model", func(t *testing.T) {
		cm, hasAlpha := mapStdColorModel(color.RGBA64Model)
		if cm != ColorModelRGB {
			t.Errorf("RGBA64Model: expected RGB, got %s", cm)
		}
		if !hasAlpha {
			t.Error("RGBA64Model: expected hasAlpha=true")
		}
	})

	t.Run("NRGBAModel", func(t *testing.T) {
		cm, hasAlpha := mapStdColorModel(color.NRGBAModel)
		if cm != ColorModelRGB {
			t.Errorf("NRGBAModel: expected RGB, got %s", cm)
		}
		if !hasAlpha {
			t.Error("NRGBAModel: expected hasAlpha=true")
		}
	})

	t.Run("NRGBA64Model", func(t *testing.T) {
		cm, hasAlpha := mapStdColorModel(color.NRGBA64Model)
		if cm != ColorModelRGB {
			t.Errorf("NRGBA64Model: expected RGB, got %s", cm)
		}
		if !hasAlpha {
			t.Error("NRGBA64Model: expected hasAlpha=true")
		}
	})

	t.Run("Gray16Model", func(t *testing.T) {
		cm, hasAlpha := mapStdColorModel(color.Gray16Model)
		if cm != ColorModelGrayscale {
			t.Errorf("Gray16Model: expected Grayscale, got %s", cm)
		}
		if hasAlpha {
			t.Error("Gray16Model: expected hasAlpha=false")
		}
	})

	t.Run("PaletteModel", func(t *testing.T) {
		palette := color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}}
		cm, hasAlpha := mapStdColorModel(palette)
		if cm != ColorModelIndexed {
			t.Errorf("Palette: expected Indexed, got %s", cm)
		}
		if hasAlpha {
			t.Error("Palette: expected hasAlpha=false")
		}
	})

	t.Run("UnknownModel", func(t *testing.T) {
		type customModel struct{}
		var custom customModel
		customColorModel := struct{ customModel }{custom}
		// Create a minimal color.Model implementation
		modelFunc := color.ModelFunc(func(c color.Color) color.Color { return c })
		cm, hasAlpha := mapStdColorModel(modelFunc)
		if cm != ColorModelUnknown {
			t.Errorf("Custom model: expected Unknown, got %s", cm)
		}
		if hasAlpha {
			t.Error("Custom model: expected hasAlpha=false")
		}
		_ = customColorModel
	})
}

func TestCalculateBytesPerPixel(t *testing.T) {
	tests := []struct {
		name        string
		colorModel  ColorModel
		bitDepth    int
		hasAlpha    bool
		expectedBPP int
	}{
		{"Grayscale_8bit_NoAlpha", ColorModelGrayscale, 8, false, 1},
		{"Grayscale_8bit_WithAlpha", ColorModelGrayscale, 8, true, 2},
		{"Grayscale_16bit_NoAlpha", ColorModelGrayscale, 16, false, 2},
		{"Grayscale_16bit_WithAlpha", ColorModelGrayscale, 16, true, 4},
		{"Grayscale_10bit_NoAlpha", ColorModelGrayscale, 10, false, 2},
		{"Grayscale_10bit_WithAlpha", ColorModelGrayscale, 10, true, 4},
		{"Grayscale_12bit_NoAlpha", ColorModelGrayscale, 12, false, 2},
		{"Grayscale_12bit_WithAlpha", ColorModelGrayscale, 12, true, 4},

		{"Indexed_8bit", ColorModelIndexed, 8, false, 1},
		{"Indexed_4bit", ColorModelIndexed, 4, false, 1},
		{"Indexed_1bit", ColorModelIndexed, 1, false, 1},

		{"RGB_8bit_NoAlpha", ColorModelRGB, 8, false, 3},
		{"RGB_8bit_WithAlpha", ColorModelRGB, 8, true, 4},
		{"RGB_16bit_NoAlpha", ColorModelRGB, 16, false, 6},
		{"RGB_16bit_WithAlpha", ColorModelRGB, 16, true, 8},
		{"RGB_10bit_NoAlpha", ColorModelRGB, 10, false, 6},
		{"RGB_10bit_WithAlpha", ColorModelRGB, 10, true, 8},
		{"RGB_12bit_NoAlpha", ColorModelRGB, 12, false, 6},
		{"RGB_12bit_WithAlpha", ColorModelRGB, 12, true, 8},

		{"YCbCr_8bit", ColorModelYCbCr, 8, false, 3},
		{"YCbCr_10bit", ColorModelYCbCr, 10, false, 6},
		{"YCbCr_12bit", ColorModelYCbCr, 12, false, 6},
		{"YCbCr_16bit", ColorModelYCbCr, 16, false, 6},

		{"Unknown_Default", ColorModelUnknown, 8, false, 4},
		{"Unknown_16bit", ColorModelUnknown, 16, false, 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := &ImageInfo{
				ColorModel: tc.colorModel,
				BitDepth:   tc.bitDepth,
				HasAlpha:   tc.hasAlpha,
			}
			bpp := calculateBytesPerPixel(info)
			if bpp != tc.expectedBPP {
				t.Errorf("Expected %d bytes per pixel, got %d", tc.expectedBPP, bpp)
			}
		})
	}
}

func TestAnalyzeJPEG_GrayscaleAndUnknown(t *testing.T) {
	t.Run("Grayscale_JPEG_1Component", func(t *testing.T) {
		jpegData := createGrayscaleJPEG(100, 100, 8)
		reader := bytes.NewReader(jpegData)

		subsampling := detectJPEGSubsampling(reader)
		if subsampling != "Grayscale" {
			t.Errorf("Subsampling: got=%s, want=Grayscale", subsampling)
		}
	})

	t.Run("CustomSubsampling_Unknown", func(t *testing.T) {
		jpegData := createCustomSubsamplingJPEG(100, 100, 3, 3, 1, 1, 8)
		reader := bytes.NewReader(jpegData)

		subsampling := detectJPEGSubsampling(reader)
		expected := "Custom (3x3:1x1)"
		if subsampling != expected {
			t.Errorf("Subsampling: got=%s, want=%s", subsampling, expected)
		}
	})

	t.Run("NoICCProfile_DefaultsToSRGB", func(t *testing.T) {
		jpegData := createMinimalJPEGData(100, 100, 2, 2, 1, 1, 8)
		reader := bytes.NewReader(jpegData)

		iccData, colorSpace := detectJPEGICCProfile(reader)
		if iccData != nil {
			t.Error("Expected nil ICC data")
		}
		if colorSpace != "sRGB" {
			t.Errorf("ColorSpace: got=%s, want=sRGB", colorSpace)
		}
	})
}

func TestJPEGSubsampling_AllMarkers(t *testing.T) {
	t.Run("SOF2_Progressive_420", func(t *testing.T) {
		jpegData := createJPEGWithSOFMarker(0xC2, 8, 3, 100, 100, 2, 2, 1, 1)
		reader := bytes.NewReader(jpegData)

		result := detectJPEGSubsampling(reader)
		if result != "4:2:0" {
			t.Errorf("SOF2 subsampling: got=%s, want=4:2:0", result)
		}
	})

	t.Run("Grayscale_1Component", func(t *testing.T) {
		jpegData := createGrayscaleJPEG(100, 100, 8)
		reader := bytes.NewReader(jpegData)

		result := detectJPEGSubsampling(reader)
		if result != "Grayscale" {
			t.Errorf("Grayscale subsampling: got=%s, want=Grayscale", result)
		}
	})

	t.Run("CustomSubsampling_3x3_1x1", func(t *testing.T) {
		jpegData := createCustomSubsamplingJPEG(100, 100, 3, 3, 1, 1, 8)
		reader := bytes.NewReader(jpegData)

		result := detectJPEGSubsampling(reader)
		if result != "Custom (3x3:1x1)" {
			t.Errorf("Custom subsampling: got=%s, want=Custom (3x3:1x1)", result)
		}
	})

	t.Run("EOI_WithoutSOF", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xD9})
		reader := bytes.NewReader(buf.Bytes())

		result := detectJPEGSubsampling(reader)
		if result != "Unknown" {
			t.Errorf("EOI without SOF: got=%s, want=Unknown", result)
		}
	})

	t.Run("TruncatedSOF", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xC0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(10))
		buf.Write([]byte{8, 0, 100, 0, 100})
		reader := bytes.NewReader(buf.Bytes())

		result := detectJPEGSubsampling(reader)
		if result != "Unknown" {
			t.Errorf("Truncated SOF: got=%s, want=Unknown", result)
		}
	})

	t.Run("InvalidNumComponents", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xC0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(20))
		buf.Write([]byte{8})
		_ = binary.Write(&buf, binary.BigEndian, uint16(100))
		_ = binary.Write(&buf, binary.BigEndian, uint16(100))
		buf.WriteByte(10)
		buf.Write(make([]byte, 5))
		reader := bytes.NewReader(buf.Bytes())

		result := detectJPEGSubsampling(reader)
		if result != "Unknown" {
			t.Errorf("Invalid components: got=%s, want=Unknown", result)
		}
	})
}

func TestJPEGICCProfile_EdgeCases(t *testing.T) {
	t.Run("NoICCProfile_ReachesEOI", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xD9})
		reader := bytes.NewReader(buf.Bytes())

		iccData, colorSpace := detectJPEGICCProfile(reader)
		if iccData != nil {
			t.Error("Expected nil ICC data")
		}
		if colorSpace != "sRGB" {
			t.Errorf("ColorSpace: got=%s, want=sRGB", colorSpace)
		}
	})

	t.Run("NonICCAPP2Marker", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xE2})
		_ = binary.Write(&buf, binary.BigEndian, uint16(20))
		buf.WriteString("NOT_ICC_PROFILE\x00")
		buf.Write(make([]byte, 4))
		buf.Write([]byte{0xFF, 0xD9})
		reader := bytes.NewReader(buf.Bytes())

		iccData, colorSpace := detectJPEGICCProfile(reader)
		if iccData != nil {
			t.Error("Expected nil ICC data for non-ICC APP2")
		}
		if colorSpace != "sRGB" {
			t.Errorf("ColorSpace: got=%s, want=sRGB", colorSpace)
		}
	})

	t.Run("ShortICCData", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xE2})
		_ = binary.Write(&buf, binary.BigEndian, uint16(18))
		buf.WriteString("ICC_PROFILE\x00")
		buf.Write([]byte{1, 1})
		buf.Write([]byte{0, 0})
		buf.Write([]byte{0xFF, 0xD9})
		reader := bytes.NewReader(buf.Bytes())

		_, colorSpace := detectJPEGICCProfile(reader)
		if colorSpace != "sRGB" {
			t.Errorf("ColorSpace: got=%s, want=sRGB", colorSpace)
		}
	})

	t.Run("InvalidJPEGHeader", func(t *testing.T) {
		buf := bytes.NewReader([]byte{0x00, 0x00})

		iccData, colorSpace := detectJPEGICCProfile(buf)
		if iccData != nil {
			t.Error("Expected nil ICC data for invalid header")
		}
		if colorSpace != "sRGB" {
			t.Errorf("ColorSpace: got=%s, want=sRGB", colorSpace)
		}
	})

	t.Run("TruncatedMarkerLength", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xE1})
		reader := bytes.NewReader(buf.Bytes())

		iccData, colorSpace := detectJPEGICCProfile(reader)
		if iccData != nil {
			t.Error("Expected nil ICC data for truncated marker")
		}
		if colorSpace != "sRGB" {
			t.Errorf("ColorSpace: got=%s, want=sRGB", colorSpace)
		}
	})
}

func Test12BitJPEG_AllSOFMarkers(t *testing.T) {
	t.Run("SOF2_Progressive_8bit", func(t *testing.T) {
		jpegData := createJPEGWithSOFMarker(0xC2, 8, 3, 100, 100, 2, 2, 1, 1)
		reader := bytes.NewReader(jpegData)

		result := is12BitJPEG(reader)
		if result {
			t.Error("Expected false for 8-bit progressive JPEG")
		}
	})

	t.Run("SOF2_Progressive_12bit", func(t *testing.T) {
		jpegData := createJPEGWithSOFMarker(0xC2, 12, 3, 100, 100, 2, 2, 1, 1)
		reader := bytes.NewReader(jpegData)

		result := is12BitJPEG(reader)
		if !result {
			t.Error("Expected true for 12-bit progressive JPEG")
		}
	})

	t.Run("EmptySOFData", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xC0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(2))
		reader := bytes.NewReader(buf.Bytes())

		result := is12BitJPEG(reader)
		if result {
			t.Error("Expected false for empty SOF data")
		}
	})

	t.Run("ReachesEOI_Without12Bit", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xD9})
		reader := bytes.NewReader(buf.Bytes())

		result := is12BitJPEG(reader)
		if result {
			t.Error("Expected false when reaching EOI without SOF")
		}
	})

	t.Run("InvalidJPEGHeader", func(t *testing.T) {
		buf := bytes.NewReader([]byte{0x00, 0x00})

		result := is12BitJPEG(buf)
		if result {
			t.Error("Expected false for invalid JPEG header")
		}
	})

	t.Run("TruncatedSOF", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0xFF, 0xD8})
		buf.Write([]byte{0xFF, 0xC0})
		reader := bytes.NewReader(buf.Bytes())

		result := is12BitJPEG(reader)
		if result {
			t.Error("Expected false for truncated SOF")
		}
	})
}

func createGrayscaleJPEG(width, height int, precision uint8) []byte {
	return createJPEGWithSOFMarker(0xC0, precision, 1, width, height, 1, 1, 0, 0)
}

func createCustomSubsamplingJPEG(width, height int, yH, yV, cbH, cbV, precision uint8) []byte {
	return createJPEGWithSOFMarker(0xC0, precision, 3, width, height, yH, yV, cbH, cbV)
}

func createJPEGWithSOFMarker(sofMarker, precision uint8, numComponents int, width, height int, yH, yV, cbH, cbV uint8) []byte {
	var buf bytes.Buffer

	buf.Write([]byte{0xFF, 0xD8})

	buf.Write([]byte{0xFF, sofMarker})

	sofLength := uint16(8 + numComponents*3)
	_ = binary.Write(&buf, binary.BigEndian, sofLength)
	buf.WriteByte(precision)
	_ = binary.Write(&buf, binary.BigEndian, uint16(height))
	_ = binary.Write(&buf, binary.BigEndian, uint16(width))
	buf.WriteByte(uint8(numComponents))

	switch numComponents {
	case 1:
		buf.WriteByte(1)
		buf.WriteByte((1 << 4) | 1)
		buf.WriteByte(0)
	case 3:
		buf.WriteByte(1)
		buf.WriteByte((yH << 4) | yV)
		buf.WriteByte(0)

		buf.WriteByte(2)
		buf.WriteByte((cbH << 4) | cbV)
		buf.WriteByte(1)

		buf.WriteByte(3)
		buf.WriteByte((cbH << 4) | cbV)
		buf.WriteByte(1)
	}

	buf.Write([]byte{0xFF, 0xD9})

	return buf.Bytes()
}

func TestDetectPNGBitDepth_EdgeCases(t *testing.T) {
	t.Run("TruncatedAfterSignature", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		reader := bytes.NewReader(buf.Bytes())

		bitDepth := detectPNGBitDepth(reader)
		if bitDepth != 8 {
			t.Errorf("Expected default 8, got %d", bitDepth)
		}
	})

	t.Run("InvalidIHDRChunkType", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		_ = binary.Write(&buf, binary.BigEndian, uint32(13))
		buf.Write([]byte("IXXX"))
		reader := bytes.NewReader(buf.Bytes())

		bitDepth := detectPNGBitDepth(reader)
		if bitDepth != 8 {
			t.Errorf("Expected default 8, got %d", bitDepth)
		}
	})

	t.Run("InvalidIHDRLength", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		_ = binary.Write(&buf, binary.BigEndian, uint32(10))
		buf.Write([]byte("IHDR"))
		reader := bytes.NewReader(buf.Bytes())

		bitDepth := detectPNGBitDepth(reader)
		if bitDepth != 8 {
			t.Errorf("Expected default 8, got %d", bitDepth)
		}
	})

	t.Run("TruncatedIHDRData", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		_ = binary.Write(&buf, binary.BigEndian, uint32(13))
		buf.Write([]byte("IHDR"))
		buf.Write([]byte{0, 0, 0, 100})
		reader := bytes.NewReader(buf.Bytes())

		bitDepth := detectPNGBitDepth(reader)
		if bitDepth != 8 {
			t.Errorf("Expected default 8, got %d", bitDepth)
		}
	})

	t.Run("ValidIHDR_16bit", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		_ = binary.Write(&buf, binary.BigEndian, uint32(13))
		buf.Write([]byte("IHDR"))
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		buf.WriteByte(16)
		buf.WriteByte(6)
		buf.WriteByte(0)
		buf.WriteByte(0)
		buf.WriteByte(0)

		reader := bytes.NewReader(buf.Bytes())
		bitDepth := detectPNGBitDepth(reader)
		if bitDepth != 16 {
			t.Errorf("Expected 16, got %d", bitDepth)
		}
	})

	t.Run("ValidIHDR_4bit", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		_ = binary.Write(&buf, binary.BigEndian, uint32(13))
		buf.Write([]byte("IHDR"))
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		buf.WriteByte(4)
		buf.WriteByte(3)
		buf.WriteByte(0)
		buf.WriteByte(0)
		buf.WriteByte(0)

		reader := bytes.NewReader(buf.Bytes())
		bitDepth := detectPNGBitDepth(reader)
		if bitDepth != 4 {
			t.Errorf("Expected 4, got %d", bitDepth)
		}
	})
}

func TestDetectPNGICCProfile_EdgeCases(t *testing.T) {
	t.Run("TruncatedAfterSignature", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		reader := bytes.NewReader(buf.Bytes())

		iccData, colorSpace := detectPNGICCProfile(reader)
		if iccData != nil {
			t.Error("Expected nil ICC data")
		}
		if colorSpace != "sRGB" {
			t.Errorf("Expected sRGB, got %s", colorSpace)
		}
	})

	t.Run("ReachesIEND_NoICC", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		_ = binary.Write(&buf, binary.BigEndian, uint32(13))
		buf.Write([]byte("IHDR"))
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		buf.WriteByte(8)
		buf.WriteByte(6)
		buf.WriteByte(0)
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))
		buf.Write([]byte("IEND"))

		reader := bytes.NewReader(buf.Bytes())
		iccData, colorSpace := detectPNGICCProfile(reader)
		if iccData != nil {
			t.Error("Expected nil ICC data")
		}
		if colorSpace != "sRGB" {
			t.Errorf("Expected sRGB, got %s", colorSpace)
		}
	})

	t.Run("SkipsNonICCPChunks", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		_ = binary.Write(&buf, binary.BigEndian, uint32(4))
		buf.Write([]byte("gAMA"))
		buf.Write([]byte{0, 0, 177, 143})
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))
		buf.Write([]byte("IEND"))

		reader := bytes.NewReader(buf.Bytes())
		iccData, colorSpace := detectPNGICCProfile(reader)
		if iccData != nil {
			t.Error("Expected nil ICC data")
		}
		if colorSpace != "sRGB" {
			t.Errorf("Expected sRGB, got %s", colorSpace)
		}
	})

	t.Run("ICCPChunk_TruncatedData", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		buf.Write([]byte("iCCP"))
		buf.Write([]byte("profile\x00"))

		reader := bytes.NewReader(buf.Bytes())
		iccData, colorSpace := detectPNGICCProfile(reader)
		if iccData != nil {
			t.Error("Expected nil ICC data on truncated iCCP")
		}
		if colorSpace != "sRGB" {
			t.Errorf("Expected sRGB, got %s", colorSpace)
		}
	})

	t.Run("ICCPChunk_ValidData", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})

		iccProfile := []byte("fake-icc-profile-data-here")
		_ = binary.Write(&buf, binary.BigEndian, uint32(len(iccProfile)))
		buf.Write([]byte("iCCP"))
		buf.Write(iccProfile)

		reader := bytes.NewReader(buf.Bytes())
		iccData, colorSpace := detectPNGICCProfile(reader)
		if iccData == nil {
			t.Error("Expected ICC data")
		}
		if len(iccData) != len(iccProfile) {
			t.Errorf("ICC data length mismatch: got %d, want %d", len(iccData), len(iccProfile))
		}
		if colorSpace != "sRGB" {
			t.Errorf("Expected sRGB (detectColorSpaceFromICC default), got %s", colorSpace)
		}
	})

	t.Run("MultipleChunks_FindsICC", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})

		_ = binary.Write(&buf, binary.BigEndian, uint32(13))
		buf.Write([]byte("IHDR"))
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		_ = binary.Write(&buf, binary.BigEndian, uint32(100))
		buf.WriteByte(8)
		buf.WriteByte(6)
		buf.WriteByte(0)
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))

		_ = binary.Write(&buf, binary.BigEndian, uint32(4))
		buf.Write([]byte("gAMA"))
		buf.Write([]byte{0, 0, 177, 143})
		_ = binary.Write(&buf, binary.BigEndian, uint32(0))

		iccProfile := []byte("test-icc")
		_ = binary.Write(&buf, binary.BigEndian, uint32(len(iccProfile)))
		buf.Write([]byte("iCCP"))
		buf.Write(iccProfile)

		reader := bytes.NewReader(buf.Bytes())
		iccData, _ := detectPNGICCProfile(reader)
		if iccData == nil {
			t.Error("Expected ICC data after skipping other chunks")
		}
		if len(iccData) != len(iccProfile) {
			t.Errorf("ICC data length mismatch: got %d, want %d", len(iccData), len(iccProfile))
		}
	})
}

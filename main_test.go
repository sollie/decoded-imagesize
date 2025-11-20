package main

import (
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

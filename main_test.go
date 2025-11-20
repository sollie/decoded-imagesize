package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
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
	defer file.Close()

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
	default:
		bytesPerPixel = 4
	}

	return int64(width) * int64(height) * int64(bytesPerPixel), nil
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
			file.Close()
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

			if estimated != actual {
				t.Errorf("Size mismatch for %s: estimated=%d, actual=%d, diff=%d",
					dim.name, estimated, actual, estimated-actual)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 4
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
			file.Close()
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

			if estimated != actual {
				t.Errorf("Size mismatch for %s: estimated=%d, actual=%d, diff=%d",
					dim.name, estimated, actual, estimated-actual)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 1
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
			file.Close()
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

			if estimated != actual {
				t.Errorf("Size mismatch for %s: estimated=%d, actual=%d, diff=%d",
					dim.name, estimated, actual, estimated-actual)
			}

			expectedSize := int64(dim.width) * int64(dim.height) * 2
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
			file.Close()
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
			file.Close()
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

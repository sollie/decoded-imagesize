package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
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
		if _, ok := cm.(color.Palette); ok {
			return "Paletted"
		}
		return "Unknown"
	}
}

func estimateDecodedSize(filename string) (int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}
	originalSize := fileInfo.Size()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, err
	}

	width := config.Width
	height := config.Height

	var bytesPerPixel int
	var bitDepth int

	_, _ = file.Seek(0, 0)
	iccProfile, colorSpace := detectICCProfile(file, format)
	if colorSpace == "" {
		colorSpace = "sRGB"
	}

	_, _ = file.Seek(0, 0)

	switch format {
	case "jpeg":
		_, _ = file.Seek(0, 0)
		subsampling := detectJPEGSubsampling(file)
		isCMYK := config.ColorModel == color.CMYKModel

		if isCMYK {
			bytesPerPixel = 4
			bitDepth = 8
		} else {
			switch subsampling {
			case "4:4:4":
				bytesPerPixel = 3
			case "4:2:2":
				bytesPerPixel = 2
			case "4:2:0":
				bytesPerPixel = 2
			default:
				bytesPerPixel = 3
			}
			bitDepth = 8
			_, _ = file.Seek(0, 0)
			if is12BitJPEG(file) {
				bitDepth = 12
			}
		}

	case "png":
		_, _ = file.Seek(0, 0)
		bitDepth = detectPNGBitDepth(file)

		switch config.ColorModel {
		case color.GrayModel:
			bytesPerPixel = 1
		case color.Gray16Model:
			bytesPerPixel = 2
		case color.RGBA64Model, color.NRGBA64Model:
			bytesPerPixel = 8
		default:
			if _, ok := config.ColorModel.(color.Palette); ok {
				bytesPerPixel = 1
			} else {
				bytesPerPixel = 4
			}
		}

	case "heif", "avif":
		switch config.ColorModel {
		case color.GrayModel:
			bytesPerPixel = 1
			bitDepth = 8
		case color.Gray16Model:
			bytesPerPixel = 2
			bitDepth = 10
		case color.YCbCrModel:
			bytesPerPixel = 3
			bitDepth = 8
		case color.RGBA64Model, color.NRGBA64Model:
			bytesPerPixel = 8
			bitDepth = 10
		default:
			bytesPerPixel = 3
			bitDepth = 8
		}
		colorSpace = detectHEIFColorSpace(format)

	case "webp":
		bytesPerPixel = 4
		bitDepth = 8

	default:
		switch config.ColorModel {
		case color.GrayModel, color.AlphaModel:
			bytesPerPixel = 1
			bitDepth = 8
		case color.Gray16Model, color.Alpha16Model:
			bytesPerPixel = 2
			bitDepth = 16
		case color.RGBA64Model, color.NRGBA64Model:
			bytesPerPixel = 8
			bitDepth = 16
		case color.CMYKModel:
			bytesPerPixel = 4
			bitDepth = 8
		default:
			if _, ok := config.ColorModel.(color.Palette); ok {
				bytesPerPixel = 1
				bitDepth = 8
			} else {
				bytesPerPixel = 4
				bitDepth = 8
			}
		}
	}

	decodedSize := int64(width) * int64(height) * int64(bytesPerPixel)

	fmt.Printf("Format: %s\n", format)
	fmt.Printf("Dimensions: %dx%d\n", width, height)
	fmt.Printf("Color Model: %s\n", colorModelName(config.ColorModel))
	if len(iccProfile) > 0 {
		fmt.Printf("ICC Profile: Present (%d bytes)\n", len(iccProfile))
	} else {
		fmt.Printf("ICC Profile: Not detected\n")
	}
	fmt.Printf("Color Space: %s\n", colorSpace)
	fmt.Printf("Bit Depth: %d\n", bitDepth)
	if format == "jpeg" && config.ColorModel != color.CMYKModel {
		_, _ = file.Seek(0, 0)
		subsampling := detectJPEGSubsampling(file)
		fmt.Printf("YCbCr Subsampling: %s\n", subsampling)
	}
	fmt.Printf("Original file size: %d bytes (%.2f MB)\n",
		originalSize, float64(originalSize)/(1024*1024))
	fmt.Printf("Estimated decoded size: %d bytes (%.2f MB)\n",
		decodedSize, float64(decodedSize)/(1024*1024))
	fmt.Printf("Compression ratio: %.1fx\n",
		float64(decodedSize)/float64(originalSize))

	return decodedSize, nil
}

func detectICCProfile(r io.ReadSeeker, format string) ([]byte, string) {
	switch format {
	case "png":
		return detectPNGICCProfile(r)
	case "jpeg":
		return detectJPEGICCProfile(r)
	default:
		return nil, "sRGB"
	}
}

func detectPNGICCProfile(r io.ReadSeeker) ([]byte, string) {
	_, _ = r.Seek(8, 0)

	buf := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, "sRGB"
		}

		length := binary.BigEndian.Uint32(buf[:4])
		chunkType := string(buf[4:8])

		if chunkType == "iCCP" {
			iccData := make([]byte, length)
			if _, err := io.ReadFull(r, iccData); err != nil {
				return nil, "sRGB"
			}
			return iccData, detectColorSpaceFromICC(iccData)
		}

		if chunkType == "IEND" {
			break
		}

		_, _ = r.Seek(int64(length+4), 1)
	}

	return nil, "sRGB"
}

func detectJPEGICCProfile(r io.ReadSeeker) ([]byte, string) {
	_, _ = r.Seek(0, 0)

	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, "sRGB"
	}

	if buf[0] != 0xFF || buf[1] != 0xD8 {
		return nil, "sRGB"
	}

	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, "sRGB"
		}

		if buf[0] != 0xFF {
			return nil, "sRGB"
		}

		marker := buf[1]

		if marker == 0xD9 {
			break
		}

		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, "sRGB"
		}

		length := int(binary.BigEndian.Uint16(buf)) - 2

		if marker == 0xE2 {
			data := make([]byte, length)
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, "sRGB"
			}

			if len(data) > 12 && string(data[:12]) == "ICC_PROFILE\x00" {
				return data[14:], detectColorSpaceFromICC(data[14:])
			}
		} else {
			_, _ = r.Seek(int64(length), 1)
		}
	}

	return nil, "sRGB"
}

func detectColorSpaceFromICC(iccData []byte) string {
	if len(iccData) < 128 {
		return "sRGB"
	}

	if bytes.Contains(iccData, []byte("Display P3")) || bytes.Contains(iccData, []byte("P3")) {
		return "Display P3"
	}
	if bytes.Contains(iccData, []byte("BT.2020")) || bytes.Contains(iccData, []byte("Rec. 2020")) {
		return "BT.2020"
	}
	if bytes.Contains(iccData, []byte("BT.709")) || bytes.Contains(iccData, []byte("Rec. 709")) {
		return "BT.709"
	}
	if bytes.Contains(iccData, []byte("Adobe RGB")) {
		return "Adobe RGB"
	}

	return "sRGB (ICC)"
}

func detectJPEGSubsampling(r io.ReadSeeker) string {
	_, _ = r.Seek(0, 0)

	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "Unknown"
	}

	if buf[0] != 0xFF || buf[1] != 0xD8 {
		return "Unknown"
	}

	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return "Unknown"
		}

		if buf[0] != 0xFF {
			return "Unknown"
		}

		marker := buf[1]

		if marker == 0xC0 || marker == 0xC1 || marker == 0xC2 {
			if _, err := io.ReadFull(r, buf); err != nil {
				return "Unknown"
			}

			length := int(binary.BigEndian.Uint16(buf))
			sofData := make([]byte, length-2)
			if _, err := io.ReadFull(r, sofData); err != nil {
				return "Unknown"
			}

			if len(sofData) < 6 {
				return "Unknown"
			}

			numComponents := sofData[3]
			if numComponents < 3 {
				return "Grayscale"
			}

			if len(sofData) < 6+int(numComponents)*3 {
				return "Unknown"
			}

			ySample := sofData[5]
			cbSample := sofData[8]

			yH := (ySample >> 4) & 0x0F
			yV := ySample & 0x0F
			cbH := (cbSample >> 4) & 0x0F
			cbV := cbSample & 0x0F

			if yH == 1 && yV == 1 && cbH == 1 && cbV == 1 {
				return "4:4:4"
			} else if yH == 2 && yV == 1 && cbH == 1 && cbV == 1 {
				return "4:2:2"
			} else if yH == 2 && yV == 2 && cbH == 1 && cbV == 1 {
				return "4:2:0"
			}

			return fmt.Sprintf("Custom (%dx%d:%dx%d)", yH, yV, cbH, cbV)
		}

		if marker == 0xD9 {
			break
		}

		if _, err := io.ReadFull(r, buf); err != nil {
			return "Unknown"
		}

		length := int(binary.BigEndian.Uint16(buf)) - 2
		_, _ = r.Seek(int64(length), 1)
	}

	return "Unknown"
}

func is12BitJPEG(r io.ReadSeeker) bool {
	_, _ = r.Seek(0, 0)

	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return false
	}

	if buf[0] != 0xFF || buf[1] != 0xD8 {
		return false
	}

	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return false
		}

		if buf[0] != 0xFF {
			return false
		}

		marker := buf[1]

		if marker == 0xC0 || marker == 0xC1 || marker == 0xC2 {
			if _, err := io.ReadFull(r, buf); err != nil {
				return false
			}

			length := int(binary.BigEndian.Uint16(buf))
			sofData := make([]byte, length-2)
			if _, err := io.ReadFull(r, sofData); err != nil {
				return false
			}

			if len(sofData) > 0 {
				precision := sofData[0]
				return precision == 12
			}
		}

		if marker == 0xD9 {
			break
		}

		if _, err := io.ReadFull(r, buf); err != nil {
			return false
		}

		length := int(binary.BigEndian.Uint16(buf)) - 2
		_, _ = r.Seek(int64(length), 1)
	}

	return false
}

func detectPNGBitDepth(r io.ReadSeeker) int {
	_, _ = r.Seek(8, 0)

	buf := make([]byte, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 8
	}

	length := binary.BigEndian.Uint32(buf[:4])
	chunkType := string(buf[4:8])

	if chunkType != "IHDR" || length != 13 {
		return 8
	}

	ihdr := make([]byte, 13)
	if _, err := io.ReadFull(r, ihdr); err != nil {
		return 8
	}

	bitDepth := int(ihdr[8])
	return bitDepth
}

func detectHEIFColorSpace(format string) string {
	if format == "heif" {
		return "BT.709"
	}
	return "BT.709"
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

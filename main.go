package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
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

const (
	ExitSuccess         = 0
	ExitUsageError      = 1
	ExitFileNotFound    = 2
	ExitInvalidFormat   = 3
	ExitProcessingError = 4
)

type ColorModel int

const (
	ColorModelUnknown ColorModel = iota
	ColorModelRGB
	ColorModelYCbCr
	ColorModelGrayscale
	ColorModelIndexed
)

func (cm ColorModel) String() string {
	switch cm {
	case ColorModelRGB:
		return "RGB"
	case ColorModelYCbCr:
		return "YCbCr"
	case ColorModelGrayscale:
		return "Grayscale"
	case ColorModelIndexed:
		return "Indexed"
	default:
		return "Unknown"
	}
}

func (cm ColorModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(cm.String())
}

type ColorSpace int

const (
	ColorSpaceUnknown ColorSpace = iota
	ColorSpaceSRGB
	ColorSpaceAdobeRGB
	ColorSpaceBT709
	ColorSpaceBT2020
	ColorSpaceDisplayP3
)

func (cs ColorSpace) String() string {
	switch cs {
	case ColorSpaceSRGB:
		return "sRGB"
	case ColorSpaceAdobeRGB:
		return "Adobe RGB"
	case ColorSpaceBT709:
		return "BT.709"
	case ColorSpaceBT2020:
		return "BT.2020"
	case ColorSpaceDisplayP3:
		return "Display P3"
	default:
		return "Unknown"
	}
}

func (cs ColorSpace) MarshalJSON() ([]byte, error) {
	return json.Marshal(cs.String())
}

type HDRType int

const (
	HDRNone HDRType = iota
	HDRPQ
	HDRHLG
	HDRLimited
)

func (h HDRType) String() string {
	switch h {
	case HDRPQ:
		return "PQ (SMPTE ST 2084)"
	case HDRHLG:
		return "HLG (ARIB STD-B67)"
	case HDRLimited:
		return "Limited"
	case HDRNone:
		return "None"
	default:
		return "Unknown"
	}
}

func (h HDRType) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.String())
}

type ChromaSubsampling int

const (
	ChromaSubsamplingNA ChromaSubsampling = iota
	ChromaSubsampling444
	ChromaSubsampling422
	ChromaSubsampling420
	ChromaSubsamplingUnknown
)

func (cs ChromaSubsampling) String() string {
	switch cs {
	case ChromaSubsampling444:
		return "4:4:4"
	case ChromaSubsampling422:
		return "4:2:2"
	case ChromaSubsampling420:
		return "4:2:0"
	case ChromaSubsamplingNA:
		return "N/A"
	default:
		return "Unknown"
	}
}

func (cs ChromaSubsampling) MarshalJSON() ([]byte, error) {
	return json.Marshal(cs.String())
}

type CompressionType int

const (
	CompressionUnknown CompressionType = iota
	CompressionLossless
	CompressionLossy
	CompressionHybrid
)

func (ct CompressionType) String() string {
	switch ct {
	case CompressionLossless:
		return "Lossless"
	case CompressionLossy:
		return "Lossy"
	case CompressionHybrid:
		return "Lossy/Lossless"
	default:
		return "Unknown"
	}
}

func (ct CompressionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.String())
}

type ImageInfo struct {
	Format            string            `json:"format"`
	Width             int               `json:"width"`
	Height            int               `json:"height"`
	ColorModel        ColorModel        `json:"color_model"`
	ColorSpace        ColorSpace        `json:"color_space"`
	BitDepth          int               `json:"bit_depth"`
	HasAlpha          bool              `json:"has_alpha"`
	HasICCProfile     bool              `json:"has_icc_profile"`
	ICCProfileSize    int               `json:"icc_profile_size,omitempty"`
	HDRType           HDRType           `json:"hdr_type"`
	ChromaSubsampling ChromaSubsampling `json:"chroma_subsampling"`
	CompressionType   CompressionType   `json:"compression_type"`
	OriginalSize      int64             `json:"original_size_bytes"`
	DecodedSize       int64             `json:"decoded_size_bytes"`
	CompressionRatio  float64           `json:"compression_ratio"`
}

func analyzeImage(filename string) (*ImageInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, err
	}

	info := &ImageInfo{
		Format: format,
		Width:  config.Width,
		Height: config.Height,
	}

	_, _ = file.Seek(0, 0)

	switch format {
	case "png":
		analyzePNG(file, config, info)
	case "jpeg":
		analyzeJPEG(file, config, info)
	case "webp":
		analyzeWebP(file, config, info)
	case "heif":
		analyzeHEIF(file, config, info)
	case "avif":
		analyzeAVIF(file, config, info)
	default:
		info.ColorModel = ColorModelUnknown
		info.ColorSpace = ColorSpaceUnknown
		info.BitDepth = 8
	}

	return info, nil
}

func mapStdColorModel(cm color.Model) (ColorModel, bool) {
	switch cm {
	case color.RGBAModel, color.RGBA64Model, color.NRGBAModel, color.NRGBA64Model:
		hasAlpha := true
		return ColorModelRGB, hasAlpha
	case color.GrayModel, color.Gray16Model:
		return ColorModelGrayscale, false
	case color.AlphaModel, color.Alpha16Model:
		return ColorModelGrayscale, true
	case color.YCbCrModel:
		return ColorModelYCbCr, false
	default:
		if _, ok := cm.(color.Palette); ok {
			return ColorModelIndexed, false
		}
		return ColorModelUnknown, false
	}
}

func analyzePNG(r io.ReadSeeker, config image.Config, info *ImageInfo) {
	info.ColorModel, info.HasAlpha = mapStdColorModel(config.ColorModel)
	info.CompressionType = CompressionLossless
	info.ChromaSubsampling = ChromaSubsamplingNA
	info.HDRType = HDRNone

	_, _ = r.Seek(0, 0)
	info.BitDepth = detectPNGBitDepth(r)

	if info.BitDepth == 16 {
		info.HDRType = HDRLimited
	}

	_, _ = r.Seek(0, 0)
	iccProfile, colorSpace := detectPNGICCProfile(r)
	if len(iccProfile) > 0 {
		info.HasICCProfile = true
		info.ICCProfileSize = len(iccProfile)
		info.ColorSpace = parseColorSpace(colorSpace)
	} else {
		info.ColorSpace = ColorSpaceSRGB
	}
}

func analyzeJPEG(r io.ReadSeeker, config image.Config, info *ImageInfo) {
	info.CompressionType = CompressionLossy
	info.HasAlpha = false
	info.HDRType = HDRNone

	_, _ = r.Seek(0, 0)
	if is12BitJPEG(r) {
		info.BitDepth = 12
	} else {
		info.BitDepth = 8
	}

	_, _ = r.Seek(0, 0)
	subsampling := detectJPEGSubsampling(r)
	switch subsampling {
	case "4:4:4":
		info.ColorModel = ColorModelYCbCr
		info.ChromaSubsampling = ChromaSubsampling444
	case "4:2:2":
		info.ColorModel = ColorModelYCbCr
		info.ChromaSubsampling = ChromaSubsampling422
	case "4:2:0":
		info.ColorModel = ColorModelYCbCr
		info.ChromaSubsampling = ChromaSubsampling420
	case "Grayscale":
		info.ColorModel = ColorModelGrayscale
		info.ChromaSubsampling = ChromaSubsamplingNA
	default:
		info.ColorModel = ColorModelYCbCr
		info.ChromaSubsampling = ChromaSubsamplingUnknown
	}

	_, _ = r.Seek(0, 0)
	iccProfile, colorSpace := detectJPEGICCProfile(r)
	if len(iccProfile) > 0 {
		info.HasICCProfile = true
		info.ICCProfileSize = len(iccProfile)
		info.ColorSpace = parseColorSpace(colorSpace)
	} else {
		info.ColorSpace = ColorSpaceSRGB
	}
}

func analyzeWebP(r io.ReadSeeker, config image.Config, info *ImageInfo) {
	info.BitDepth = 8
	info.HDRType = HDRNone

	info.ColorModel, info.HasAlpha = mapStdColorModel(config.ColorModel)

	_, _ = r.Seek(0, 0)
	isLossless, chromaSub := detectWebPFormat(r)
	if isLossless {
		info.CompressionType = CompressionLossless
		info.ChromaSubsampling = ChromaSubsamplingNA
	} else {
		info.CompressionType = CompressionLossy
		info.ChromaSubsampling = chromaSub
	}

	info.ColorSpace = ColorSpaceSRGB
}

type heifMetadata struct {
	ColorModel        ColorModel
	HasAlpha          bool
	BitDepth          int
	ColorSpace        ColorSpace
	ChromaSubsampling ChromaSubsampling
	HDRType           HDRType
}

func parseHEIFMetadata(r io.ReadSeeker) heifMetadata {
	meta := heifMetadata{
		ColorModel:        ColorModelYCbCr,
		HasAlpha:          false,
		BitDepth:          8,
		ColorSpace:        ColorSpaceBT709,
		ChromaSubsampling: ChromaSubsampling420,
		HDRType:           HDRNone,
	}

	_, _ = r.Seek(0, 0)
	data := make([]byte, 16384)
	n, _ := r.Read(data)
	if n < 12 {
		return meta
	}
	data = data[:n]

	if string(data[4:8]) != "ftyp" {
		return meta
	}

	offset := 0
	for offset+8 < len(data) {
		if offset+4 > len(data) {
			break
		}

		boxSize := binary.BigEndian.Uint32(data[offset : offset+4])
		if boxSize == 0 || boxSize < 8 {
			break
		}

		if offset+8 > len(data) {
			break
		}

		boxType := string(data[offset+4 : offset+8])

		if int(boxSize) > len(data)-offset {
			boxSize = uint32(len(data) - offset)
		}

		boxData := data[offset+8 : offset+int(boxSize)]

		switch boxType {
		case "meta":
			parseMetaBox(boxData, &meta)

		case "pixi":
			if len(boxData) >= 3 {
				meta.BitDepth = int(boxData[2])
			}

		case "colr":
			if len(boxData) >= 4 {
				colorType := string(boxData[0:4])
				if colorType == "nclx" && len(boxData) >= 8 {
					colorPrimaries := binary.BigEndian.Uint16(boxData[4:6])
					transferChar := binary.BigEndian.Uint16(boxData[6:8])

					switch colorPrimaries {
					case 1:
						meta.ColorSpace = ColorSpaceBT709
					case 9:
						meta.ColorSpace = ColorSpaceBT2020
					case 12:
						meta.ColorSpace = ColorSpaceDisplayP3
					}

					switch transferChar {
					case 16:
						meta.HDRType = HDRPQ
					case 18:
						meta.HDRType = HDRHLG
					}
				}
			}

		case "auxC":
			if bytes.Contains(boxData, []byte("urn:mpeg:mpegB:cicp:systems:auxiliary:alpha")) {
				meta.HasAlpha = true
			}
		}

		offset += int(boxSize)
	}

	return meta
}

func parseMetaBox(data []byte, meta *heifMetadata) {
	offset := 4

	for offset+8 < len(data) {
		boxSize := binary.BigEndian.Uint32(data[offset : offset+4])
		boxType := string(data[offset+4 : offset+8])

		if boxSize < 8 || offset+int(boxSize) > len(data) {
			break
		}

		switch boxType {
		case "iprp":
			parseIprpBox(data[offset+8:offset+int(boxSize)], meta)
		}

		offset += int(boxSize)
	}
}

func parseIprpBox(data []byte, meta *heifMetadata) {
	offset := 0

	for offset+8 < len(data) {
		boxSize := binary.BigEndian.Uint32(data[offset : offset+4])
		boxType := string(data[offset+4 : offset+8])

		if boxSize < 8 || offset+int(boxSize) > len(data) {
			break
		}

		boxData := data[offset+8 : offset+int(boxSize)]

		switch boxType {
		case "ipco":
			parseIpcoBox(boxData, meta)
		}

		offset += int(boxSize)
	}
}

func parseIpcoBox(data []byte, meta *heifMetadata) {
	offset := 0

	for offset+8 < len(data) {
		boxSize := binary.BigEndian.Uint32(data[offset : offset+4])
		boxType := string(data[offset+4 : offset+8])

		if boxSize < 8 || offset+int(boxSize) > len(data) {
			break
		}

		boxData := data[offset+8 : offset+int(boxSize)]

		switch boxType {
		case "pixi":
			if len(boxData) >= 3 {
				numChannels := int(boxData[1])
				if numChannels > 0 && len(boxData) >= 2+numChannels {
					meta.BitDepth = int(boxData[2])
				}
			}

		case "colr":
			if len(boxData) >= 4 {
				colorType := string(boxData[0:4])
				if colorType == "nclx" && len(boxData) >= 8 {
					colorPrimaries := binary.BigEndian.Uint16(boxData[4:6])
					transferChar := binary.BigEndian.Uint16(boxData[6:8])

					switch colorPrimaries {
					case 1:
						meta.ColorSpace = ColorSpaceBT709
					case 9:
						meta.ColorSpace = ColorSpaceBT2020
					case 12:
						meta.ColorSpace = ColorSpaceDisplayP3
					}

					switch transferChar {
					case 16:
						meta.HDRType = HDRPQ
					case 18:
						meta.HDRType = HDRHLG
					}
				}
			}

		case "auxC":
			if bytes.Contains(boxData, []byte("urn:mpeg:mpegB:cicp:systems:auxiliary:alpha")) {
				meta.HasAlpha = true
			}
		}

		offset += int(boxSize)
	}
}

func analyzeHEIF(r io.ReadSeeker, config image.Config, info *ImageInfo) {
	info.CompressionType = CompressionHybrid

	metadata := parseHEIFMetadata(r)

	info.ColorModel = metadata.ColorModel
	info.HasAlpha = metadata.HasAlpha
	info.BitDepth = metadata.BitDepth
	info.ColorSpace = metadata.ColorSpace
	info.ChromaSubsampling = metadata.ChromaSubsampling
	info.HDRType = metadata.HDRType
}

func analyzeAVIF(r io.ReadSeeker, config image.Config, info *ImageInfo) {
	info.CompressionType = CompressionHybrid

	metadata := parseHEIFMetadata(r)

	info.ColorModel = metadata.ColorModel
	info.HasAlpha = metadata.HasAlpha
	info.BitDepth = metadata.BitDepth
	info.ColorSpace = metadata.ColorSpace
	info.ChromaSubsampling = metadata.ChromaSubsampling
	info.HDRType = metadata.HDRType
}

func parseColorSpace(cs string) ColorSpace {
	switch cs {
	case "sRGB", "sRGB (ICC)":
		return ColorSpaceSRGB
	case "Adobe RGB":
		return ColorSpaceAdobeRGB
	case "BT.709":
		return ColorSpaceBT709
	case "BT.2020":
		return ColorSpaceBT2020
	case "Display P3":
		return ColorSpaceDisplayP3
	default:
		return ColorSpaceSRGB
	}
}

func detectWebPFormat(r io.ReadSeeker) (bool, ChromaSubsampling) {
	_, _ = r.Seek(0, 0)

	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return false, ChromaSubsamplingUnknown
	}

	if string(header[0:4]) != "RIFF" {
		return false, ChromaSubsamplingUnknown
	}

	if string(header[8:12]) != "WEBP" {
		return false, ChromaSubsamplingUnknown
	}

	chunkHeader := make([]byte, 4)
	if _, err := io.ReadFull(r, chunkHeader); err != nil {
		return false, ChromaSubsamplingUnknown
	}

	fourCC := string(chunkHeader)
	switch fourCC {
	case "VP8L":
		return true, ChromaSubsamplingNA
	case "VP8 ":
		return false, ChromaSubsampling420
	default:
		return false, ChromaSubsamplingUnknown
	}
}

func estimateDecodedSize(filename string, jsonOutput bool) (*ImageInfo, error) {
	info, err := analyzeImage(filename)
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	originalSize := fileInfo.Size()

	bytesPerPixel := calculateBytesPerPixel(info)
	decodedSize := int64(info.Width) * int64(info.Height) * int64(bytesPerPixel)

	info.OriginalSize = originalSize
	info.DecodedSize = decodedSize
	info.CompressionRatio = float64(decodedSize) / float64(originalSize)

	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(info); err != nil {
			return nil, err
		}
	} else {
		fmt.Printf("Format: %s\n", info.Format)
		fmt.Printf("Dimensions: %dx%d\n", info.Width, info.Height)
		fmt.Printf("Color Model: %s\n", info.ColorModel)
		if info.HasICCProfile {
			fmt.Printf("ICC Profile: Present (%d bytes)\n", info.ICCProfileSize)
		} else {
			fmt.Printf("ICC Profile: Not detected\n")
		}
		fmt.Printf("Color Space: %s\n", info.ColorSpace)
		fmt.Printf("Bit Depth: %d\n", info.BitDepth)
		fmt.Printf("Alpha Channel: %v\n", info.HasAlpha)
		fmt.Printf("Chroma Subsampling: %s\n", info.ChromaSubsampling)
		fmt.Printf("HDR Support: %s\n", info.HDRType)
		fmt.Printf("Compression Type: %s\n", info.CompressionType)
		fmt.Printf("Original file size: %d bytes (%.2f MB)\n",
			originalSize, float64(originalSize)/(1024*1024))
		fmt.Printf("Estimated decoded size: %d bytes (%.2f MB)\n",
			decodedSize, float64(decodedSize)/(1024*1024))
		fmt.Printf("Compression ratio: %.1fx\n",
			float64(decodedSize)/float64(originalSize))
	}

	return info, nil
}

func calculateBytesPerPixel(info *ImageInfo) int {
	bytesPerChannel := (info.BitDepth + 7) / 8

	switch info.ColorModel {
	case ColorModelGrayscale:
		if info.HasAlpha {
			return 2 * bytesPerChannel
		}
		return bytesPerChannel
	case ColorModelIndexed:
		return 1
	case ColorModelRGB:
		if info.HasAlpha {
			return 4 * bytesPerChannel
		}
		return 3 * bytesPerChannel
	case ColorModelYCbCr:
		return 3 * bytesPerChannel
	default:
		return 4
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

			numComponents := sofData[5]
			if numComponents < 3 {
				return "Grayscale"
			}

			if len(sofData) < 6+int(numComponents)*3 {
				return "Unknown"
			}

			ySample := sofData[7]
			cbSample := sofData[10]

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

func main() {
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: decoded-imagesize [-json] <image-file>")
		fmt.Println("Supported formats: PNG, JPEG, HEIF/HEIC, AVIF, WebP")
		fmt.Println("\nFlags:")
		fmt.Println("  -json    Output in JSON format")
		fmt.Println("\nExit Codes:")
		fmt.Println("  0 - Success")
		fmt.Println("  1 - Usage error")
		fmt.Println("  2 - File not found")
		fmt.Println("  3 - Invalid or unsupported format")
		fmt.Println("  4 - Processing error")
		os.Exit(ExitUsageError)
	}

	filename := flag.Arg(0)

	_, err := estimateDecodedSize(filename, *jsonOutput)
	if err != nil {
		exitCode := categorizeError(err)
		if *jsonOutput {
			errJSON, _ := json.Marshal(map[string]interface{}{
				"error":     err.Error(),
				"exit_code": exitCode,
			})
			fmt.Println(string(errJSON))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(exitCode)
	}
}

func categorizeError(err error) int {
	if err == nil {
		return ExitSuccess
	}

	errMsg := err.Error()

	if os.IsNotExist(err) || contains(errMsg, "no such file", "cannot find") {
		return ExitFileNotFound
	}

	if contains(errMsg, "unknown format", "invalid", "decode", "unsupported") {
		return ExitInvalidFormat
	}

	return ExitProcessingError
}

func contains(s string, substrs ...string) bool {
	lower := ""
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			lower += string(s[i] + 32)
		} else {
			lower += string(s[i])
		}
	}
	for _, substr := range substrs {
		lowerSubstr := ""
		for i := 0; i < len(substr); i++ {
			if substr[i] >= 'A' && substr[i] <= 'Z' {
				lowerSubstr += string(substr[i] + 32)
			} else {
				lowerSubstr += string(substr[i])
			}
		}
		if containsSubstring(lower, lowerSubstr) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

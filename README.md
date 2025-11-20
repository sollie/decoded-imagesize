# Image Decoded Size Estimator

A Go tool that estimates the decoded (uncompressed) memory size of image files without fully decoding them. Supports comprehensive color model and color space detection across modern image formats.

## Features

### Supported Image Formats

- **PNG** - Portable Network Graphics
- **JPEG** - Joint Photographic Experts Group
- **HEIF/HEIC** - High Efficiency Image Format
- **AVIF** - AV1 Image File Format
- **WebP** - Google's WebP format

### Color Model Support

| Format | RGB | YCbCr | Grayscale | Indexed | CMYK |
|--------|-----|-------|-----------|---------|------|
| **HEIF**  | ✓ | ✓ | ✓ | - | - |
| **PNG**   | ✓ | - | ✓ | ✓ | - |
| **JPEG**  | ✓ | ✓ | ✓ | - | ✓ |
| **AVIF**  | ✓ | ✓ | ✓ | - | - |
| **WebP**  | ✓ | - | ✓ | - | - |

### Advanced Detection

#### ICC Profile Support
- Extracts and identifies ICC color profiles from PNG and JPEG files
- Detects color spaces: sRGB, Display P3, BT.709, BT.2020, Adobe RGB

#### Bit Depth Detection
- **PNG**: 1-16 bits per channel
- **JPEG**: 8-bit and 12-bit (extended)
- **HEIF/AVIF**: 8-bit, 10-bit, 12-bit
- **WebP**: 8-bit

#### HDR Support
- HEIF/AVIF: 10-bit and 12-bit HDR content
- PNG: 16-bit per channel (RGBA64)
- Detects and reports HDR color models

#### YCbCr Subsampling (JPEG)
- 4:4:4 (no subsampling)
- 4:2:2 (horizontal subsampling)
- 4:2:0 (horizontal and vertical subsampling)
- Custom subsampling ratios

#### Color Space Detection
- **HEIF/AVIF**: BT.709, BT.2020, Display P3
- **PNG/JPEG**: via ICC profile metadata
- **WebP**: sRGB

## Installation

```bash
go get github.com/chai2010/webp
go get github.com/strukturag/libheif/go/heif
go build
```

## Usage

```bash
./testdecode <image-file>
```

### Example Output

```
Format: png
Dimensions: 2000x1500
Color Model: RGBA64
ICC Profile: Present (3144 bytes)
Color Space: Display P3
Bit Depth: 16
Original file size: 57254 bytes (0.05 MB)
Estimated decoded size: 24000000 bytes (22.89 MB)
Compression ratio: 419.2x
```

### JPEG with YCbCr Subsampling

```
Format: jpeg
Dimensions: 4000x3000
Color Model: YCbCr
ICC Profile: Not detected
Color Space: sRGB
Bit Depth: 8
YCbCr Subsampling: 4:2:0
Original file size: 809378 bytes (0.77 MB)
Estimated decoded size: 36000000 bytes (34.33 MB)
Compression ratio: 44.5x
```

### HEIF HDR Image

```
Format: heif
Dimensions: 1920x1080
Color Model: RGBA64
ICC Profile: Not detected
Color Space: BT.2020
Bit Depth: 10
Original file size: 245678 bytes (0.23 MB)
Estimated decoded size: 16588800 bytes (15.82 MB)
Compression ratio: 67.5x
```

## How It Works

The tool uses `image.DecodeConfig()` to read only the image header without decoding the entire image. It then:

1. Detects the image format and color model
2. Reads ICC profile data (if present) to identify color space
3. Determines bit depth from format-specific headers
4. For JPEG: analyzes YCbCr subsampling from SOF markers
5. Calculates bytes per pixel based on color model and format
6. Estimates total decoded size: `width × height × bytes_per_pixel`

This approach is memory-efficient and fast, as it avoids loading the full image data.

## Accuracy

The size estimation reflects how Go's image decoders work:
- **PNG RGBA**: 4 bytes/pixel (even for RGB without alpha)
- **PNG Grayscale**: 1 byte/pixel (8-bit) or 2 bytes/pixel (16-bit)
- **PNG Paletted**: 1 byte/pixel
- **PNG RGBA64**: 8 bytes/pixel
- **JPEG YCbCr**: 2-3 bytes/pixel (depends on subsampling)
- **JPEG CMYK**: 4 bytes/pixel
- **HEIF/AVIF YCbCr**: 3 bytes/pixel
- **HEIF/AVIF HDR**: 8 bytes/pixel (10-bit RGBA64)
- **WebP RGBA**: 4 bytes/pixel

## Testing

Comprehensive test suite with 15 test suites covering:
- All supported formats
- Multiple color models
- Various bit depths
- Different image dimensions (100x100 to 4000x3000)
- Edge cases (paletted, RGBA64, YCbCr subsampling)

Run tests:
```bash
go test -v
```

## Code Quality

- Zero golangci-lint issues
- Full error handling
- No memory leaks (proper file closure)
- Efficient header-only parsing

## Dependencies

- [github.com/chai2010/webp](https://github.com/chai2010/webp) - WebP decoder
- [github.com/strukturag/libheif](https://github.com/strukturag/libheif) - HEIF/HEIC/AVIF decoder

## Use Cases

- **Memory planning**: Estimate RAM requirements before loading images
- **Image processing pipelines**: Validate available memory
- **Batch processing**: Calculate total memory needed for image sets
- **Image analysis**: Understand compression efficiency
- **Format comparison**: Compare decoded sizes across formats

## License

See LICENSE file for details.

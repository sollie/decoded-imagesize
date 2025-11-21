# Image Decoded Size Estimator

[![Go Version](https://img.shields.io/badge/Go-1.25.4-blue.svg)](https://golang.org)
[![Test Coverage](https://img.shields.io/badge/coverage-80.0%25-green.svg)](https://github.com/sollie/decoded-imagesize)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go tool that estimates the decoded (uncompressed) memory size of image files without fully decoding them. Supports comprehensive color model and color space detection across modern image formats.

## Features

### Comprehensive Image Format Support

A detailed comparison of supported image format characteristics:

| Feature | PNG | JPEG | HEIF | AVIF | WebP |
|---------|-----|------|------|------|------|
| **Color Model** | RGB, Grayscale, Indexed | YCbCr, Grayscale | YCbCr | YCbCr | RGB, YCbCr |
| **Color Space** | sRGB, Adobe RGB, Display P3, BT.709, BT.2020 (ICC) | sRGB, Adobe RGB, Display P3, BT.709, BT.2020 (ICC) | sRGB, BT.709, BT.2020, Display P3 | sRGB, BT.709, BT.2020, Display P3 | sRGB |
| **Bit Depth** | 1, 2, 4, 8, 16 | 8, 12 | 8, 10, 12 | 8, 10, 12 | 8 |
| **Alpha Channel** | ✓ | ✗ | ✓ | ✓ | ✓ |
| **Chroma Subsampling** | N/A | 4:4:4, 4:2:2, 4:2:0 | 4:4:4, 4:2:2, 4:2:0 | 4:4:4, 4:2:2, 4:2:0 | 4:2:0 |
| **HDR Support** | Limited (16-bit) | ✗ | ✓ (PQ, HLG) | ✓ (PQ, HLG) | ✗ |
| **Compression** | Lossless | Lossy | Lossy/Lossless | Lossy/Lossless | Lossy/Lossless |
| **Max Resolution** | Unlimited | 65535×65535 | Unlimited | Unlimited | 16383×16383 |
| **Typical Use Cases** | Web graphics, screenshots | Photography, web images | Mobile photos, HDR | Next-gen web, HDR | Web images, transparency |

### Detection Capabilities

#### Color Model Detection
- **RGB**: PNG, WebP, JPEG (rare)
- **YCbCr**: JPEG, HEIF, AVIF, WebP (lossy)
- **Grayscale**: PNG, JPEG
- **Indexed (Palette)**: PNG
- **CMYK**: JPEG (detected but not decoded)

#### Color Space Support
- **sRGB**: All formats (default)
- **Display P3**: HEIF/AVIF (native), PNG/JPEG (via ICC)
- **BT.709**: HEIF/AVIF (native), PNG/JPEG (via ICC)
- **BT.2020**: HEIF/AVIF (native), PNG/JPEG (via ICC)
- **Adobe RGB**: PNG/JPEG (via ICC)

#### Bit Depth Detection
- **PNG**: Accurately detects 1, 2, 4, 8, 16 bits per channel (16-bit marked as Limited HDR)
- **JPEG**: Detects 8-bit (baseline) and 12-bit (extended)
- **HEIF/AVIF**: Parses `pixi` box for 8, 10, 12-bit detection
- **WebP**: Always 8-bit

#### HDR Detection
- **PNG**: Reports 16-bit images as "Limited" HDR (extended dynamic range without HDR metadata)
- **HEIF/AVIF**: Detects PQ (SMPTE ST 2084) and HLG (ARIB STD-B67) transfer functions
- **Detection method**: 
  - PNG: Checks bit depth from IHDR chunk
  - HEIF/AVIF: Parses `colr` box transfer characteristics

#### Chroma Subsampling Detection
- **JPEG**: Analyzes SOF (Start of Frame) markers for Y, Cb, Cr sampling factors
  - 4:4:4 (1:1:1) - No subsampling
  - 4:2:2 (2:1:1) - Horizontal subsampling
  - 4:2:0 (2:2:1) - Horizontal and vertical subsampling
- **HEIF/AVIF**: Detected from container metadata (typically 4:2:0)
- **WebP Lossy**: 4:2:0 subsampling
- **PNG/WebP Lossless**: N/A (no subsampling)

#### Compression Type Detection
- **Lossless**: PNG, WebP (VP8L)
- **Lossy**: JPEG, WebP (VP8)
- **Hybrid (Lossy/Lossless)**: HEIF, AVIF
- **Detection method**: WebP uses FourCC code analysis ('VP8 ' vs 'VP8L')

## Installation

```bash
go install github.com/sollie/decoded-imagesize@latest
```

Or clone and build from source:

```bash
git clone https://github.com/sollie/decoded-imagesize.git
cd decoded-imagesize
go mod tidy
go build
```

## Usage

### Basic Usage

```bash
# Human-readable output
./decoded-imagesize <image-file>

# JSON output for scripting
./decoded-imagesize -json <image-file>
```

### Examples

**Normal output:**
```bash
./decoded-imagesize image.png
```

**JSON output:**
```bash
./decoded-imagesize -json image.png
```

Output:
```json
{
  "format": "png",
  "width": 2000,
  "height": 1500,
  "color_model": "RGB",
  "color_space": "sRGB",
  "bit_depth": 16,
  "has_alpha": true,
  "has_icc_profile": false,
  "hdr_type": "Limited",
  "chroma_subsampling": "N/A",
  "compression_type": "Lossless",
  "original_size_bytes": 57254,
  "decoded_size_bytes": 24000000,
  "compression_ratio": 419.2
}
```

## CLI Features

### Output Formats

**Human-Readable** (default):
- Clear, formatted output for manual inspection
- Easy to read metadata display

**JSON Output** (`-json` flag):
- Machine-readable structured data
- All metadata fields included
- Ideal for scripting and automation
- Example: `./decoded-imagesize -json image.png`

### Exit Codes

The tool returns standardized exit codes for scripting:
- `0` - Success
- `1` - Usage error (invalid arguments)  
- `2` - File not found
- `3` - Invalid or unsupported image format
- `4` - Processing error

Exit codes are included in JSON error output when using `-json` flag.

**Example error handling in scripts:**
```bash
./decoded-imagesize -json image.png > output.json
case $? in
  0) echo "Success" ;;
  2) echo "File not found" ;;
  3) echo "Invalid format" ;;
  *) echo "Error occurred" ;;
esac
```

### Example Output

#### PNG with 16-bit RGB and ICC Profile
```
Format: png
Dimensions: 2000x1500
Color Model: RGB
ICC Profile: Present (3144 bytes)
Color Space: Display P3
Bit Depth: 16
Alpha Channel: true
Chroma Subsampling: N/A
HDR Support: Limited
Compression Type: Lossless
Original file size: 57254 bytes (0.05 MB)
Estimated decoded size: 24000000 bytes (22.89 MB)
Compression ratio: 419.2x
```

#### JPEG with YCbCr Subsampling
```
Format: jpeg
Dimensions: 4000x3000
Color Model: YCbCr
ICC Profile: Not detected
Color Space: sRGB
Bit Depth: 8
Alpha Channel: false
Chroma Subsampling: 4:2:0
HDR Support: None
Compression Type: Lossy
Original file size: 809378 bytes (0.77 MB)
Estimated decoded size: 36000000 bytes (34.33 MB)
Compression ratio: 44.5x
```

#### HEIF with HDR Support
```
Format: heif
Dimensions: 3840x2160
Color Model: YCbCr
ICC Profile: Not detected
Color Space: BT.2020
Bit Depth: 10
Alpha Channel: false
Chroma Subsampling: 4:2:0
HDR Support: PQ (SMPTE ST 2084)
Compression Type: Lossy/Lossless
Original file size: 1245678 bytes (1.19 MB)
Estimated decoded size: 24883200 bytes (23.73 MB)
Compression ratio: 20.0x
```

#### AVIF with Alpha Channel
```
Format: avif
Dimensions: 1920x1080
Color Model: YCbCr
ICC Profile: Not detected
Color Space: BT.709
Bit Depth: 8
Alpha Channel: true
Chroma Subsampling: 4:2:0
HDR Support: None
Compression Type: Lossy/Lossless
Original file size: 45678 bytes (0.04 MB)
Estimated decoded size: 6220800 bytes (5.93 MB)
Compression ratio: 136.2x
```

#### WebP Lossless
```
Format: webp
Dimensions: 1000x1000
Color Model: RGB
ICC Profile: Not detected
Color Space: sRGB
Bit Depth: 8
Alpha Channel: true
Chroma Subsampling: N/A
HDR Support: None
Compression Type: Lossless
Original file size: 4376 bytes (0.00 MB)
Estimated decoded size: 4000000 bytes (3.81 MB)
Compression ratio: 914.1x
```

## How It Works

The tool uses `image.DecodeConfig()` to read only the image header without decoding the entire image. It then performs format-specific analysis:

### Detection Pipeline

1. **Format Detection**: Identifies image format from file signature
2. **Basic Metadata**: Extracts dimensions and Go's native color model
3. **Format-Specific Analysis**:
   - **PNG**: Parses IHDR chunk for bit depth, color type, and iCCP chunk for ICC profiles
   - **JPEG**: Analyzes SOF markers for bit depth and chroma subsampling, APP2 markers for ICC profiles
   - **HEIF/AVIF**: Parses ISO Base Media File Format boxes:
     - `meta` → `iprp` → `ipco` → `pixi` for bit depth
     - `meta` → `iprp` → `ipco` → `colr` for color space and HDR transfer functions
     - `meta` → `iprp` → `ipco` → `auxC` for alpha channel detection
   - **WebP**: Analyzes FourCC codes ('VP8 ' for lossy, 'VP8L' for lossless)
4. **Size Calculation**: `width × height × bytes_per_pixel`

### Bytes Per Pixel Calculation

The tool accurately reflects how Go's `image` package decodes formats:

| Format | Color Model | Bit Depth | Bytes/Pixel | Notes |
|--------|-------------|-----------|-------------|-------|
| PNG | RGBA | 8 | 4 | Even without alpha |
| PNG | RGB | 16 | 8 | Decoded as RGBA64 |
| PNG | Grayscale | 8 | 1 | Gray |
| PNG | Grayscale | 16 | 2 | Gray16 |
| PNG | Indexed | 8 | 1 | Paletted |
| JPEG | YCbCr | 8 | 3 | 4:4:4 subsampling |
| JPEG | YCbCr | 8 | 2 | 4:2:0 subsampling (decoded form) |
| JPEG | Grayscale | 8 | 1 | Gray |
| HEIF/AVIF | YCbCr | 8 | 3 | Standard dynamic range |
| HEIF/AVIF | YCbCr | 10/12 | 8 | HDR (RGBA64) |
| WebP | RGBA | 8 | 4 | Both lossy and lossless |

### Memory Efficiency

This approach is extremely memory-efficient:
- Reads only file headers (typically < 64KB)
- No full image decoding required
- Constant memory usage regardless of image size
- Fast execution (< 1ms per image)

## Testing

Comprehensive test suite with **100% accuracy** verified across:

### Test Coverage
- **16 test suites** covering all supported formats
- **Coverage**: 80.0% overall (100% on critical parsing functions like parseIprpBox)
- **Test images**: Generated programmatically for consistency
- **Dimensions tested**: 100×100, 500×500, 1000×1000, 2000×1500, 4000×3000
- **Color models**: RGB, RGBA, Grayscale, Gray16, RGBA64, YCbCr, Indexed
- **Bit depths**: 1, 2, 4, 8, 12, 16
- **Special cases**: Chroma subsampling, ICC profiles, HDR detection, malformed data handling

### Test Results
```bash
$ go test -v
PASS
ok      github.com/sollie/decoded-imagesize       9.645s

$ go test -cover
coverage: 80.0% of statements
```

Run tests:
```bash
go test -v              # Verbose output
go test -cover          # With coverage
go test -run TestJPEG   # Specific format
go test -short          # Quick test run
```

## Quality Metrics

- ✅ **Test Coverage**: 80.0% (100% on critical parsing functions)
- ✅ **Linter**: 0 issues (golangci-lint)
- ✅ **Security**: 0 vulnerabilities (trivy)
- ✅ **All Tests**: Passing
- ✅ **Dependencies**: Up to date
- ✅ **Exit Codes**: Standardized for scripting
- ✅ **JSON Output**: Full machine-readable support

### Quality Checks
```bash
$ golangci-lint run
0 issues.

$ trivy fs --scanners vuln .
0 vulnerabilities found
```

## Dependencies

- [github.com/chai2010/webp](https://github.com/chai2010/webp) - WebP decoder
- [github.com/strukturag/libheif](https://github.com/strukturag/libheif) - HEIF/HEIC/AVIF decoder

## Use Cases

### Production Applications
- **Memory planning**: Estimate RAM requirements before loading images in image processing servers
- **Resource allocation**: Determine optimal worker pool sizes for batch processing
- **API rate limiting**: Calculate processing costs based on decoded size
- **CDN optimization**: Assess memory impact of different image formats

### Development & Analysis
- **Format comparison**: Compare compression efficiency across PNG, JPEG, HEIF, AVIF, WebP
- **HDR validation**: Verify HDR metadata (PQ/HLG) without full decode
- **Color space verification**: Check ICC profiles and color space metadata
- **Image pipeline debugging**: Validate chroma subsampling and bit depth

### Performance Optimization
- **Batch processing**: Calculate total memory needed before starting
- **Memory-aware scheduling**: Prioritize small images when memory is limited
- **Cache sizing**: Determine optimal image cache sizes
- **Format selection**: Choose best format based on memory constraints

### Examples

**Check if server can handle image**:
```go
size, err := estimateDecodedSize("upload.jpg")
if size > availableRAM {
    return errors.New("insufficient memory")
}
```

**Compare format efficiency**:
```bash
$ ./decoded-imagesize photo.png
Compression ratio: 514.7x

$ ./decoded-imagesize photo.avif
Compression ratio: 231.1x
```

**Validate HDR content**:
```bash
$ ./decoded-imagesize hdr_photo.heic | grep HDR
HDR Support: PQ (Perceptual Quantizer)
```

## License

See LICENSE file for details.

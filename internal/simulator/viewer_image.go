package simulator

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	_ "golang.org/x/image/webp"
)

// readImageInfo reads image metadata and generates preview
func readImageInfo(path string, maxPreviewHeight, maxPreviewWidth int) (*ImageInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Check if it's an SVG file by extension or content
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".svg" {
		return readSVGInfo(path, stat.Size(), maxPreviewHeight, maxPreviewWidth)
	}

	// For files without extension, check if content looks like SVG
	if ext == "" {
		// Read first 512 bytes to check for SVG content
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err == nil && n > 0 {
			content := strings.ToLower(string(buffer[:n]))
			if strings.Contains(content, "<?xml") &&
				(strings.Contains(content, "<svg") || strings.Contains(content, "xmlns=\"http://www.w3.org/2000/svg\"")) {
				// Reset file position
				_ = file.Close()
				return readSVGInfo(path, stat.Size(), maxPreviewHeight, maxPreviewWidth)
			}
		}
		// Reset file position for image decoding
		_, _ = file.Seek(0, 0)
	}

	// Decode image to get dimensions
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("not a valid image: %w", err)
	}

	info := &ImageInfo{
		Format: format,
		Width:  config.Width,
		Height: config.Height,
		Size:   stat.Size(),
	}

	// Generate preview if requested
	if maxPreviewHeight > 15 { // Only generate preview if we have reasonable space
		// Reset file position
		_, _ = file.Seek(0, 0)

		// Decode full image for preview
		img, _, err := image.Decode(file)
		if err == nil {
			// Calculate available space
			// The maxPreviewHeight already accounts for UI overhead from model.go
			// Just reserve 4 lines for the image info header in the viewer
			availableHeight := maxPreviewHeight - 4
			if availableHeight > 0 {
				// Use the actual available width minus some padding
				// Account for content box padding (4 chars)
				availableWidth := maxPreviewWidth - 4
				if availableWidth < 20 {
					availableWidth = 20
				}
				info.Preview = generateImagePreview(img, availableWidth, availableHeight)
			}
		}
	}

	return info, nil
}

// generateImagePreview creates a terminal-renderable preview of an image
func generateImagePreview(img image.Image, maxWidth, maxHeight int) *ImagePreview {
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Calculate target dimensions maintaining aspect ratio
	// We use half-blocks, so each character can show 2 vertical pixels
	aspectRatio := float64(imgWidth) / float64(imgHeight)

	// Target dimensions in characters (height will use half-blocks for 2x resolution)
	var targetWidth, targetHeight int

	// Account for half-blocks: actual pixel height is 2x character height
	effectiveMaxHeight := maxHeight * 2

	if aspectRatio > float64(maxWidth)/float64(effectiveMaxHeight) {
		// Width-constrained
		targetWidth = maxWidth
		targetHeight = int(float64(maxWidth) / aspectRatio)
	} else {
		// Height-constrained
		targetHeight = effectiveMaxHeight
		targetWidth = int(float64(effectiveMaxHeight) * aspectRatio)
	}

	// Ensure even height for half-blocks
	if targetHeight%2 != 0 {
		targetHeight--
	}

	// Character dimensions
	charHeight := targetHeight / 2
	charWidth := targetWidth

	// Create preview
	preview := &ImagePreview{
		Width:  charWidth,
		Height: charHeight,
		Rows:   make([]string, charHeight),
	}

	// Scale factors
	xScale := float64(imgWidth) / float64(targetWidth)
	yScale := float64(imgHeight) / float64(targetHeight)

	// Process each row
	for row := 0; row < charHeight; row++ {
		var rowStr strings.Builder

		for col := 0; col < charWidth; col++ {
			// Sample two pixels for half-block (upper and lower)
			y1 := int(float64(row*2) * yScale)
			y2 := int(float64(row*2+1) * yScale)
			x := int(float64(col) * xScale)

			// Bounds checking
			if x >= imgWidth {
				x = imgWidth - 1
			}
			if y1 >= imgHeight {
				y1 = imgHeight - 1
			}
			if y2 >= imgHeight {
				y2 = imgHeight - 1
			}

			// Get colors
			c1 := img.At(x, y1)
			c2 := img.At(x, y2)

			// Convert to RGB
			r1, g1, b1, _ := c1.RGBA()
			r2, g2, b2, _ := c2.RGBA()

			// Convert to 8-bit color
			r1, g1, b1 = r1>>8, g1>>8, b1>>8
			r2, g2, b2 = r2>>8, g2>>8, b2>>8

			// Generate ANSI escape sequences
			// Upper half block with foreground color
			// Lower half block with background color
			fmt.Fprintf(&rowStr, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀\x1b[0m", r1, g1, b1, r2, g2, b2)
		}

		preview.Rows[row] = rowStr.String()
	}

	return preview
}

// readSVGInfo reads SVG metadata and generates preview
func readSVGInfo(path string, fileSize int64, maxPreviewHeight, maxPreviewWidth int) (*ImageInfo, error) {
	// Read SVG file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Try to extract dimensions from SVG content
	svgStr := string(data)
	width, height := extractSVGDimensions(svgStr)

	info := &ImageInfo{
		Format: "svg",
		Width:  width,
		Height: height,
		Size:   fileSize,
	}

	// Generate preview if requested
	if maxPreviewHeight > 15 { // Only generate preview if we have reasonable space
		// Calculate available space
		availableHeight := maxPreviewHeight - 4 // Same as regular images
		if availableHeight > 0 {
			// Use the actual available width minus padding
			availableWidth := maxPreviewWidth - 4
			if availableWidth < 20 {
				availableWidth = 20
			}

			// Calculate render size to fit in preview area
			renderWidth := width
			renderHeight := height

			// Scale down if needed to fit preview constraints
			if renderHeight > availableHeight*10 || renderWidth > availableWidth*10 {
				aspectRatio := float64(width) / float64(height)
				if aspectRatio > float64(availableWidth)/float64(availableHeight) {
					renderWidth = availableWidth * 10
					renderHeight = int(float64(renderWidth) / aspectRatio)
				} else {
					renderHeight = availableHeight * 10
					renderWidth = int(float64(renderHeight) * aspectRatio)
				}
			}

			// Use oksvg implementation
			icon, parseErr := oksvg.ReadIconStream(bytes.NewReader(data))
			if parseErr != nil {
				info.Preview = &ImagePreview{
					Width:  1,
					Height: 1,
					Rows:   []string{fmt.Sprintf("[SVG parsing error: %v]", parseErr)},
				}
			} else {
				// Try oksvg rendering
				img, renderErr := rasterizeSVG(icon, renderWidth, renderHeight)
				if renderErr != nil {
					info.Preview = &ImagePreview{
						Width:  1,
						Height: 1,
						Rows:   []string{fmt.Sprintf("[SVG rendering error: %v]", renderErr)},
					}
				} else {
					info.Preview = generateImagePreview(img, availableWidth, availableHeight)
				}
			}
		}
	}

	return info, nil
}

// extractSVGDimensions tries to extract width and height from SVG content
func extractSVGDimensions(svgContent string) (int, int) {
	width, height := 256, 256 // defaults

	// Try to extract width
	if widthMatch := strings.Index(svgContent, `width="`); widthMatch != -1 {
		widthStart := widthMatch + 7
		widthEnd := strings.Index(svgContent[widthStart:], `"`)
		if widthEnd != -1 {
			if w, err := strconv.Atoi(svgContent[widthStart : widthStart+widthEnd]); err == nil {
				width = w
			}
		}
	}

	// Try to extract height
	if heightMatch := strings.Index(svgContent, `height="`); heightMatch != -1 {
		heightStart := heightMatch + 8
		heightEnd := strings.Index(svgContent[heightStart:], `"`)
		if heightEnd != -1 {
			if h, err := strconv.Atoi(svgContent[heightStart : heightStart+heightEnd]); err == nil {
				height = h
			}
		}
	}

	// Try viewBox if width/height not found
	if viewBoxMatch := strings.Index(svgContent, `viewBox="`); viewBoxMatch != -1 {
		viewBoxStart := viewBoxMatch + 9
		viewBoxEnd := strings.Index(svgContent[viewBoxStart:], `"`)
		if viewBoxEnd != -1 {
			viewBox := svgContent[viewBoxStart : viewBoxStart+viewBoxEnd]
			parts := strings.Fields(viewBox)
			if len(parts) >= 4 {
				if w, err := strconv.ParseFloat(parts[2], 64); err == nil {
					width = int(w)
				}
				if h, err := strconv.ParseFloat(parts[3], 64); err == nil {
					height = int(h)
				}
			}
		}
	}

	return width, height
}

// rasterizeSVG converts an SVG icon to a raster image using oksvg.
// A recovered panic from the rasterizer is returned as an error via
// the named return rather than printed to stdout — printing would
// corrupt the TUI since stdout is the rendering surface.
func rasterizeSVG(icon *oksvg.SvgIcon, width, height int) (img image.Image, err error) {
	// Limit render size to prevent memory issues
	maxSize := 1024
	if width > maxSize || height > maxSize {
		scale := float64(maxSize) / float64(max(width, height))
		width = int(float64(width) * scale)
		height = int(float64(height) * scale)
	}

	// Create a new RGBA image
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	// Set icon size to fit the canvas
	icon.SetTarget(0, 0, float64(width), float64(height))

	// Create a rasterizer
	scanner := rasterx.NewScannerGV(width, height, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(width, height, scanner)

	// Try to draw the SVG
	defer func() {
		if r := recover(); r != nil {
			img = nil
			err = fmt.Errorf("SVG rendering panic: %v", r)
		}
	}()

	// Draw the icon
	icon.Draw(raster, 1.0)

	return rgba, nil
}

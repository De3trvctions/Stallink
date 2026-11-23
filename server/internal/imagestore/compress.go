package imagestore

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	MaxUploadBytes  = 6 << 20
	maxCatalogBytes = 1 << 20
	maxPayCodeBytes = 4 << 20
	maxCatalogEdge  = 1600
	maxPixels       = 24_000_000
)

var (
	ErrInvalidImage  = errors.New("invalid image")
	ErrImageTooLarge = errors.New("image exceeds size limit")
)

// Compress normalizes catalog images and losslessly optimizes payment images.
func Compress(src io.Reader, preservePixels bool) ([]byte, string, error) {
	original, err := io.ReadAll(io.LimitReader(src, MaxUploadBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(original) > MaxUploadBytes {
		return nil, "", ErrImageTooLarge
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(original))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return nil, "", ErrInvalidImage
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return nil, "", ErrInvalidImage
	}
	if config.Width > maxPixels/config.Height {
		return nil, "", ErrImageTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(original))
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	if format == "jpeg" {
		img = orientJPEG(img, original)
	}
	if preservePixels {
		encoded, err := encodeImage(img, "png")
		if err != nil {
			return nil, "", err
		}
		if len(encoded) <= maxPayCodeBytes && len(encoded) < len(original) {
			return encoded, "image/png", nil
		}
		if len(original) <= maxPayCodeBytes {
			return original, "image/" + format, nil
		}
		return nil, "", ErrImageTooLarge
	}

	bounds := img.Bounds()
	longEdge := max(bounds.Dx(), bounds.Dy())
	edge := min(longEdge, maxCatalogEdge)
	opaque, ok := img.(interface{ Opaque() bool })
	outputFormat := "png"
	if ok && opaque.Opaque() && format != "png" {
		outputFormat = "jpeg"
	}
	for {
		scaled := img
		if edge < longEdge {
			width := max(1, bounds.Dx()*edge/longEdge)
			height := max(1, bounds.Dy()*edge/longEdge)
			dst := image.NewNRGBA(image.Rect(0, 0, width, height))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Src, nil)
			scaled = dst
		}
		encoded, err := encodeImage(scaled, outputFormat)
		if err != nil {
			return nil, "", err
		}
		if edge == longEdge && len(original) <= maxCatalogBytes && len(original) <= len(encoded) {
			return original, "image/" + format, nil
		}
		if len(encoded) <= maxCatalogBytes {
			return encoded, "image/" + outputFormat, nil
		}
		edge = max(1, edge*3/4)
	}
}

func encodeImage(img image.Image, format string) ([]byte, error) {
	var output bytes.Buffer
	var err error
	if format == "jpeg" {
		err = jpeg.Encode(&output, img, &jpeg.Options{Quality: 80})
	} else {
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&output, img)
	}
	return output.Bytes(), err
}

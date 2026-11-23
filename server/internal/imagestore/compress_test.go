package imagestore

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestCompressImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1800, 810))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	var input bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(&input, img); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		preservePixels bool
		width          int
		height         int
	}{
		{name: "catalog", width: 1600, height: 720},
		{name: "pay code", preservePixels: true, width: 1800, height: 810},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, mime, err := Compress(bytes.NewReader(input.Bytes()), tc.preservePixels)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) >= input.Len() || mime != "image/png" {
				t.Fatalf("compressed to %d bytes (%s), original %d", len(got), mime, input.Len())
			}
			decoded, err := png.Decode(bytes.NewReader(got))
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Bounds().Dx() != tc.width || decoded.Bounds().Dy() != tc.height {
				t.Fatalf("dimensions = %v", decoded.Bounds())
			}
			if tc.preservePixels && decoded.At(0, 0) != img.At(0, 0) {
				t.Fatal("pay code pixel changed")
			}
		})
	}
}

func TestCompressRejectsInvalidImage(t *testing.T) {
	if _, _, err := Compress(bytes.NewReader([]byte("\x89PNG\r\n\x1a\n")), false); err == nil {
		t.Fatal("accepted a truncated PNG")
	}
}

package imagestore

import (
	"bytes"
	"encoding/binary"
	"image"
)

func orientJPEG(src image.Image, jpeg []byte) image.Image {
	orientation := jpegOrientation(jpeg)
	if orientation < 2 || orientation > 8 {
		return src
	}
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	dimensions := image.Rect(0, 0, width, height)
	if orientation >= 5 {
		dimensions = image.Rect(0, 0, height, width)
	}
	dst := image.NewNRGBA(dimensions)
	for y := range height {
		for x := range width {
			dx, dy := x, y
			switch orientation {
			case 2:
				dx = width - 1 - x
			case 3:
				dx, dy = width-1-x, height-1-y
			case 4:
				dy = height - 1 - y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = height-1-y, x
			case 7:
				dx, dy = height-1-y, width-1-x
			case 8:
				dx, dy = y, width-1-x
			}
			dst.Set(dx, dy, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}

func jpegOrientation(jpeg []byte) uint16 {
	for segment := jpeg[2:]; len(segment) >= 4 && segment[0] == 0xff; {
		if segment[1] == 0xff {
			segment = segment[1:]
			continue
		}
		if segment[1] == 0xda || segment[1] == 0xd9 {
			break
		}
		length := int(binary.BigEndian.Uint16(segment[2:4]))
		if length < 2 || length > len(segment)-2 {
			break
		}
		payload := segment[4 : 2+length]
		if segment[1] == 0xe1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
			return tiffOrientation(payload[6:])
		}
		segment = segment[2+length:]
	}
	return 1
}

func tiffOrientation(tiff []byte) uint16 {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	offset := int(order.Uint32(tiff[4:8]))
	if order.Uint16(tiff[2:4]) != 42 || offset < 8 || offset > len(tiff)-2 {
		return 1
	}
	entries := int(order.Uint16(tiff[offset : offset+2]))
	for entry := tiff[offset+2:]; entries > 0 && len(entry) >= 12; entry = entry[12:] {
		entries--
		if order.Uint16(entry[:2]) == 0x112 && order.Uint16(entry[2:4]) == 3 && order.Uint32(entry[4:8]) == 1 {
			return order.Uint16(entry[8:10])
		}
	}
	return 1
}

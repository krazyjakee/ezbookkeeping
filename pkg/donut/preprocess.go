package donut

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ErrImageTooLarge is returned when the image has more pixels than MaxImagePixels
var ErrImageTooLarge = errors.New("image is too large")

// MaxImagePixels is the maximum number of pixels of an image which is decoded
const MaxImagePixels = 60_000_000

// DecodeImage decodes a jpeg, png, gif or webp image and applies the exif orientation of jpeg images (phone photos are often stored rotated)
func DecodeImage(data []byte) (image.Image, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))

	if err != nil {
		return nil, err
	}

	if config.Width*config.Height > MaxImagePixels {
		return nil, ErrImageTooLarge
	}

	img, _, err := image.Decode(bytes.NewReader(data))

	if err != nil {
		return nil, err
	}

	return applyOrientation(img, jpegOrientation(data)), nil
}

// Preprocess converts the image into the normalized CHW float tensor data of the Donut image processor:
// the shortest edge is resized to the shortest side of the model input (bilinear), the image is shrunk to fit the input (bicubic),
// centered on a black canvas of the input size, and the pixel values are scaled to [-1, 1]
func Preprocess(img image.Image, height int, width int) []float32 {
	bounds := img.Bounds()
	sourceWidth, sourceHeight := bounds.Dx(), bounds.Dy()
	shortest := min(height, width)
	resizedWidth, resizedHeight := shortest, shortest

	if sourceWidth <= sourceHeight {
		resizedHeight = int(float64(sourceHeight) * float64(shortest) / float64(sourceWidth))
	} else {
		resizedWidth = int(float64(sourceWidth) * float64(shortest) / float64(sourceHeight))
	}

	// images with transparency are flattened onto white, like a scanned page
	flattened := image.NewRGBA(image.Rect(0, 0, sourceWidth, sourceHeight))
	draw.Draw(flattened, flattened.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(flattened, flattened.Bounds(), img, bounds.Min, draw.Over)

	resized := image.NewRGBA(image.Rect(0, 0, resizedWidth, resizedHeight))
	xdraw.BiLinear.Scale(resized, resized.Bounds(), flattened, flattened.Bounds(), xdraw.Src, nil)

	thumbnailWidth, thumbnailHeight := min(resizedWidth, width), min(resizedHeight, height)
	thumbnail := resized

	if thumbnailWidth != resizedWidth || thumbnailHeight != resizedHeight {
		if resizedHeight > resizedWidth {
			thumbnailWidth = int(float64(resizedWidth) * float64(thumbnailHeight) / float64(resizedHeight))
		} else if resizedWidth > resizedHeight {
			thumbnailHeight = int(float64(resizedHeight) * float64(thumbnailWidth) / float64(resizedWidth))
		}

		thumbnail = image.NewRGBA(image.Rect(0, 0, thumbnailWidth, thumbnailHeight))
		xdraw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), resized, resized.Bounds(), xdraw.Src, nil)
	}

	left := (width - thumbnailWidth) / 2
	top := (height - thumbnailHeight) / 2
	planeSize := height * width
	result := make([]float32, 3*planeSize)

	// the padding is black, which is -1 after normalization
	for i := range result {
		result[i] = -1
	}

	for y := 0; y < thumbnailHeight; y++ {
		row := thumbnail.Pix[y*thumbnail.Stride:]
		offset := (top+y)*width + left

		for x := 0; x < thumbnailWidth; x++ {
			pixel := row[x*4:]
			result[offset+x] = float32(pixel[0])/127.5 - 1
			result[planeSize+offset+x] = float32(pixel[1])/127.5 - 1
			result[2*planeSize+offset+x] = float32(pixel[2])/127.5 - 1
		}
	}

	return result
}

// jpegOrientation returns the exif orientation (1-8) of a jpeg image, or 1 if it has none
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 1
	}

	offset := 2

	for offset+4 <= len(data) {
		if data[offset] != 0xff {
			return 1
		}

		marker := data[offset+1]
		length := int(binary.BigEndian.Uint16(data[offset+2:]))

		if marker == 0xda || length < 2 || offset+2+length > len(data) {
			return 1
		}

		segment := data[offset+4 : offset+2+length]

		if marker == 0xe1 && len(segment) > 14 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:])
		}

		offset += 2 + length
	}

	return 1
}

func tiffOrientation(tiff []byte) int {
	var order binary.ByteOrder

	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}

	ifdOffset := int(order.Uint32(tiff[4:]))

	if ifdOffset+2 > len(tiff) {
		return 1
	}

	count := int(order.Uint16(tiff[ifdOffset:]))

	for i := 0; i < count; i++ {
		entry := ifdOffset + 2 + i*12

		if entry+12 > len(tiff) {
			return 1
		}

		if order.Uint16(tiff[entry:]) == 0x0112 {
			orientation := int(order.Uint16(tiff[entry+8:]))

			if orientation >= 1 && orientation <= 8 {
				return orientation
			}

			return 1
		}
	}

	return 1
}

// applyOrientation returns the image transformed according to the exif orientation
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	swap := orientation >= 5
	resultWidth, resultHeight := w, h

	if swap {
		resultWidth, resultHeight = h, w
	}

	result := image.NewRGBA(image.Rect(0, 0, resultWidth, resultHeight))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int

			switch orientation {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}

			result.Set(dx, dy, img.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}

	return result
}

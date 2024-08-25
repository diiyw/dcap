package dcap

import "C"
import (
	"fmt"
	"github.com/diiyw/dcap/internal/clipboard"
	"image"
)

// NewImage create new image
func (d *DCap) NewImage(x, y, width, height int) {
	if d.im == nil {
		d.im = image.NewRGBA(image.Rect(0, 0, width, height))
	}
	if d.im.Bounds().Dx() != width || d.im.Bounds().Dy() != height {
		d.im = image.NewRGBA(image.Rect(0, 0, width, height))
	}
}

func (d *DCap) CaptureDisplay(displayIndex int) error {
	if len(d.Displays)-1 < displayIndex {
		return fmt.Errorf("index %d out of range", displayIndex)
	}
	rect := d.Displays[displayIndex]
	return d.Capture(rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy())
}

// ClipboardSet set text to clipboard
func (d *DCap) ClipboardSet(text string) error {
	return clipboard.Set(text)
}

// ClipboardGet get text from clipboard
func (d *DCap) ClipboardGet() (string, error) {
	return clipboard.Get()
}

// ImageNoCopy return image.RGBA without copy
func (d *DCap) ImageNoCopy() *image.RGBA {
	return d.im
}

// Image return image.RGBA with copy
func (d *DCap) Image() *image.RGBA {
	im := image.NewRGBA(d.im.Bounds())
	copy(im.Pix, d.im.Pix)
	return im
}

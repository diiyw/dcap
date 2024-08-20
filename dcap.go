package dcap

import "C"
import (
	"errors"
	"github.com/diiyw/dcap/internal/clipboard"
	"image"
)

// NewImage create new image
func (d *DCap) NewImage(x, y, width, height int) {
	if d.Img == nil {
		d.Img = image.NewRGBA(image.Rect(0, 0, width, height))
	}
	if d.Img.Bounds().Dx() != width || d.Img.Bounds().Dy() != height {
		d.Img = image.NewRGBA(image.Rect(0, 0, width, height))
	}
}

func (d *DCap) CaptureDisplay(displayIndex int) error {
	if len(d.Displays)-1 < displayIndex {
		return errors.New("display not found")
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

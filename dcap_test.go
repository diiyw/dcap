package dcap

import (
	"bytes"
	"fmt"
	"testing"
)

func TestDCap(t *testing.T) {
	d, err := NewDCap()
	if err != nil {
		t.Fatal(err)
	}
	for _, display := range d.Displays {
		fmt.Printf("Display Size: %dx%d\n", display.Dx(), display.Dy())
	}
	if err = d.CaptureDisplay(0); err != nil {
		t.Fatal(err)
	}
	im := d.Image()
	if err = d.CaptureDisplay(0); err != nil {
		t.Fatal(err)
	}
	im2 := d.Image()
	if im.Bounds().Dx() != im2.Bounds().Dx() || im.Bounds().Dy() != im2.Bounds().Dy() {
		t.Fatal("image size not equal")
	}
	if bytes.Equal(im.Pix, im2.Pix) {
		t.Fatal("image data equal")
	}
}

func TestClipboard(t *testing.T) {
	d, err := NewDCap()
	if err != nil {
		t.Fatal(err)
	}
	if err = d.ClipboardSet("Hello World"); err != nil {
		t.Fatal(err)
	}
	text, err := d.ClipboardGet()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(text)
}

func TestMouseMove(t *testing.T) {
	d, err := NewDCap()
	if err != nil {
		t.Fatal(err)
	}
	if err = d.MouseMove(100, 100); err != nil {
		t.Fatal(err)
	}
}

func TestToggleKey(t *testing.T) {
	d, err := NewDCap()
	if err != nil {
		t.Fatal(err)
	}
	if err = d.ToggleKey("a", true); err != nil {
		t.Fatal(err)
	}
	if err = d.ToggleKey("a", false); err != nil {
		t.Fatal(err)
	}
}

func TestToggleMouse(t *testing.T) {
	d, err := NewDCap()
	if err != nil {
		t.Fatal(err)
	}
	if err = d.ToggleMouse(MouseRight, true); err != nil {
		t.Fatal(err)
	}
	if err = d.ToggleMouse(MouseRight, false); err != nil {
		t.Fatal(err)
	}
}

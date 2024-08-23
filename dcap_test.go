package dcap

import (
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

func TestClient_ToggleKey(t *testing.T) {
	d, err := NewDCap()
	if err != nil {
		t.Fatal(err)
	}
	if err = d.ToggleKey("esc", true); err != nil {
		t.Fatal(err)
	}
	if err = d.ToggleKey("esc", false); err != nil {
		t.Fatal(err)
	}
}

func TestClient_ToggleMouse(t *testing.T) {
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

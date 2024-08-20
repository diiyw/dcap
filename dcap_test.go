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

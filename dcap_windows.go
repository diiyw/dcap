package dcap

/*
#include "mouse_windows.h"
#include "keyboard_windows.h"
*/
import "C"

import (
	"errors"
	"github.com/diiyw/dcap/internal/windef"
	"github.com/lxn/win"
	"image"
	"syscall"
	"unsafe"
)

type DCap struct {
	im           *image.RGBA
	Displays     []image.Rectangle
	hdc          win.HDC
	memoryDevice win.HDC
	bitmap       win.HBITMAP
}

func NewDCap() (*DCap, error) {
	var d = &DCap{}
	hWnd := windef.GetDesktopWindow()
	d.hdc = win.GetDC(hWnd)
	if d.hdc == 0 {
		return nil, errors.New("GetDC failed")
	}
	d.memoryDevice = win.CreateCompatibleDC(d.hdc)
	var count = 0
	windef.EnumDisplayMonitors(win.HDC(0), nil, syscall.NewCallback(windef.CountUpMonitorCallback), uintptr(unsafe.Pointer(&count)))
	d.Displays = make([]image.Rectangle, count)
	for i := 0; i < count; i++ {
		d.Displays[i] = windef.GetDisplayBounds(i)
	}
	return d, nil
}

func (d *DCap) Close() {
	win.ReleaseDC(win.HWND(0), d.hdc)
	win.DeleteDC(d.memoryDevice)
	win.DeleteObject(win.HGDIOBJ(d.bitmap))
}

func (d *DCap) Capture(x, y, width, height int) error {
	d.NewImage(x, y, width, height)
	if d.bitmap == 0 {
		d.bitmap = win.CreateCompatibleBitmap(d.hdc, int32(width), int32(height))
		if d.bitmap == 0 {
			return errors.New("CreateCompatibleBitmap failed")
		}
	}

	var header win.BITMAPINFOHEADER
	header.BiSize = uint32(unsafe.Sizeof(header))
	header.BiPlanes = 1
	header.BiBitCount = 32
	header.BiWidth = int32(width)
	header.BiHeight = int32(-height)
	header.BiCompression = win.BI_RGB
	header.BiSizeImage = 0

	// GetDIBits balks at using Go memory on some systems. The MSDN example uses
	// GlobalAlloc, so we'll do that too. See:
	// https://docs.microsoft.com/en-gb/windows/desktop/gdi/capturing-an-image
	bitmapDataSize := uintptr(((int64(width)*int64(header.BiBitCount) + 31) / 32) * 4 * int64(height))
	hMem := win.GlobalAlloc(win.GMEM_MOVEABLE, bitmapDataSize)
	defer win.GlobalFree(hMem)
	memPtr := win.GlobalLock(hMem)
	defer win.GlobalUnlock(hMem)

	old := win.SelectObject(d.memoryDevice, win.HGDIOBJ(d.bitmap))
	if old == 0 {
		return errors.New("SelectObject failed")
	}
	defer win.SelectObject(d.memoryDevice, old)

	if !win.BitBlt(d.memoryDevice, 0, 0, int32(width), int32(height), d.hdc, int32(x), int32(y), win.SRCCOPY) {
		return errors.New("BitBlt failed")
	}

	if win.GetDIBits(d.hdc, d.bitmap, 0, uint32(height), (*uint8)(memPtr), (*win.BITMAPINFO)(unsafe.Pointer(&header)), win.DIB_RGB_COLORS) == 0 {
		return errors.New("GetDIBits failed")
	}

	i := 0
	src := uintptr(memPtr)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v0 := *(*uint8)(unsafe.Pointer(src))
			v1 := *(*uint8)(unsafe.Pointer(src + 1))
			v2 := *(*uint8)(unsafe.Pointer(src + 2))

			// BGRA => RGBA, and set A to 255
			d.im.Pix[i], d.im.Pix[i+1], d.im.Pix[i+2], d.im.Pix[i+3] = v2, v1, v0, 255

			i += 4
			src += 4
		}
	}
	return nil
}

// MouseMove move mouse to x,y
func (d *DCap) MouseMove(x, y int) error {
	C.mouse_move(C.uint(x), C.uint(y))
	return nil
}

// ToggleMouse toggle mouse button event
func (d *DCap) ToggleMouse(button MouseButton, down bool) error {
	switch button {
	case MouseLeft:
		C.mouse_toggle(0, C.bool(down))
	case MouseMiddle:
		C.mouse_toggle(2, C.bool(down))
	case MouseRight:
		C.mouse_toggle(1, C.bool(down))
	}
	return nil
}

// ToggleKey toggle keyboard event
func (d *DCap) ToggleKey(key string, down bool) error {
	code := checkKeycodes(key)
	C.keyboard_toggle(C.uint(code), C.bool(down))
	return nil
}

// Scroll mouse scroll
func (d *DCap) Scroll(x, y int) {
	C.scroll(C.uint(x), C.uint(y))
}

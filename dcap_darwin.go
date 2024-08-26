package dcap

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework AppKit
#include <CoreGraphics/CoreGraphics.h>

CGEventRef createWheelEvent(int x, int y) {
	return CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitPixel, 2, y, x);
}

// void get_cursor_size(int *width, int *height);
// void cursor_copy(unsigned char* pixels, int width, int height);
void* CompatCGDisplayCreateImageForRect(CGDirectDisplayID display, CGRect rect) {
	return CGDisplayCreateImageForRect(display, rect);
}

void CompatCGImageRelease(void* image) {
	CGImageRelease(image);
}

void* CompatCGImageCreateCopyWithColorSpace(void* image, CGColorSpaceRef space) {
	return CGImageCreateCopyWithColorSpace((CGImageRef)image, space);
}

void CompatCGContextDrawImage(CGContextRef c, CGRect rect, void* image) {
	CGContextDrawImage(c, rect, (CGImageRef)image);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"image"
	"time"
	"unsafe"
)

type DCap struct {
	im                  *image.RGBA
	Displays            []image.Rectangle
	displayIds          []C.CGDirectDisplayID
	ctrlDown            bool
	altDown             bool
	shiftDown           bool
	cmdDown             bool
	currentDisplay      int
	bitmapContext       C.CGContextRef
	colorSpace          C.CGColorSpaceRef
	cgMainDisplayBounds C.CGRect
}

// NewDCap create new dcap
func NewDCap() (*DCap, error) {
	var d = &DCap{}
	num := numActiveDisplays()
	if num == 0 {
		return nil, fmt.Errorf("can not get active displays")
	}
	d.Displays = make([]image.Rectangle, num)
	for i := 0; i < num; i++ {
		d.Displays[i] = getDisplayBounds(i)
	}
	d.displayIds = activeDisplayList()
	d.cgMainDisplayBounds = C.CGRectMake(C.CGFloat(d.Displays[0].Min.X), C.CGFloat(d.Displays[0].Min.Y),
		C.CGFloat(d.Displays[0].Dx()), C.CGFloat(d.Displays[0].Dy()))
	return d, nil
}

func (d *DCap) Close() {
	C.CGColorSpaceRelease(d.colorSpace)
	return
}

func (d *DCap) Capture(x, y, width, height int) error {
	if width <= 0 || height <= 0 {
		return errors.New("width or height should be > 0")
	}
	d.NewImage(x, y, width, height)

	winBottomLeft := C.CGPointMake(C.CGFloat(x), C.CGFloat(y+height))
	cgBottomLeft := getCoreGraphicsCoordinateFromWindowsCoordinate(winBottomLeft, d.cgMainDisplayBounds)
	cgCaptureBounds := C.CGRectMake(cgBottomLeft.x, cgBottomLeft.y, C.CGFloat(width), C.CGFloat(height))

	if d.bitmapContext == 0 {
		d.bitmapContext = createBitmapContext(width, height, (*C.uint32_t)(unsafe.Pointer(&d.im.Pix[0])), d.im.Stride)
		if d.bitmapContext == 0 {
			return errors.New("cannot create bitmap context")
		}
	}
	if d.colorSpace == 0 {
		d.colorSpace = createColorspace()
		if d.colorSpace == 0 {
			return errors.New("cannot create colorspace")
		}
	}

	for _, id := range d.displayIds {
		cgBounds := getCoreGraphicsCoordinateOfDisplay(id)
		cgIntersect := C.CGRectIntersection(cgBounds, cgCaptureBounds)
		if C.CGRectIsNull(cgIntersect) {
			continue
		}
		if cgIntersect.size.width <= 0 || cgIntersect.size.height <= 0 {
			continue
		}

		// CGDisplayCreateImageForRect potentially fail in case width/height is odd number.
		if int(cgIntersect.size.width)%2 != 0 {
			cgIntersect.size.width = C.CGFloat(int(cgIntersect.size.width) + 1)
		}
		if int(cgIntersect.size.height)%2 != 0 {
			cgIntersect.size.height = C.CGFloat(int(cgIntersect.size.height) + 1)
		}

		diIntersectDisplayLocal := C.CGRectMake(cgIntersect.origin.x-cgBounds.origin.x,
			cgBounds.origin.y+cgBounds.size.height-(cgIntersect.origin.y+cgIntersect.size.height),
			cgIntersect.size.width, cgIntersect.size.height)
		captured := C.CompatCGDisplayCreateImageForRect(id, diIntersectDisplayLocal)
		if captured == nil {
			return errors.New("cannot capture display")
		}
		defer C.CompatCGImageRelease(captured)

		image := C.CompatCGImageCreateCopyWithColorSpace(captured, d.colorSpace)
		if image == nil {
			return errors.New("failed copying captured image")
		}
		defer C.CompatCGImageRelease(image)

		cgDrawRect := C.CGRectMake(cgIntersect.origin.x-cgCaptureBounds.origin.x, cgIntersect.origin.y-cgCaptureBounds.origin.y,
			cgIntersect.size.width, cgIntersect.size.height)
		C.CompatCGContextDrawImage(d.bitmapContext, cgDrawRect, image)
	}

	i := 0
	for iy := 0; iy < height; iy++ {
		j := i
		for ix := 0; ix < width; ix++ {
			// ARGB => RGBA, and set A to 255
			d.im.Pix[j], d.im.Pix[j+1], d.im.Pix[j+2], d.im.Pix[j+3] = d.im.Pix[j+1], d.im.Pix[j+2], d.im.Pix[j+3], 255
			j += 4
		}
		i += d.im.Stride
	}

	return nil
}

// MouseMove move mouse to x,y
func (d *DCap) MouseMove(x, y int) error {
	pt := C.CGPointMake(C.double(x), C.double(y))
	err := C.CGDisplayMoveCursorToPoint(C.uint(d.currentDisplay), pt)
	if err != 0 {
		return fmt.Errorf("can not move: %d", err)
	}
	return nil
}

func getMousePosition() C.CGPoint {
	event := C.CGEventCreate(C.CGEventSourceRef(0))
	defer C.CFRelease(C.CFTypeRef(event))
	return C.CGEventGetLocation(event)
}

// ToggleMouse toggle mouse button event
func (d *DCap) ToggleMouse(button MouseButton, down bool) error {
	var t C.CGEventType
	var btn C.CGMouseButton
	switch button {
	case MouseLeft:
		if down {
			t = C.kCGEventLeftMouseDown
		} else {
			t = C.kCGEventLeftMouseUp
		}
		btn = 0
	case MouseMiddle:
		if down {
			t = C.kCGEventOtherMouseDown
		} else {
			t = C.kCGEventOtherMouseUp
		}
		btn = 2
	case MouseRight:
		if down {
			t = C.kCGEventRightMouseDown
		} else {
			t = C.kCGEventRightMouseUp
		}
		btn = 1
	}
	event := C.CGEventCreateMouseEvent(C.CGEventSourceRef(0), t, getMousePosition(), btn)
	defer C.CFRelease(C.CFTypeRef(event))
	C.CGEventPost(C.kCGSessionEventTap, event)
	return nil
}

// ToggleKey toggle keyboard event
func (d *DCap) ToggleKey(key string, down bool) error {
	code := checkKeycodes(key)
	event := C.CGEventCreateKeyboardEvent(C.CGEventSourceRef(0), C.CGKeyCode(code), true)
	if event == 0 {
		return nil
	}
	defer C.CFRelease(C.CFTypeRef(event))

	if down {
		C.CGEventSetType(event, C.kCGEventKeyDown)
	} else {
		C.CGEventSetType(event, C.kCGEventKeyUp)
	}

	flag := 0
	if d.ctrlDown {
		flag |= C.kCGEventFlagMaskControl
	}
	if d.altDown {
		flag |= C.kCGEventFlagMaskAlternate
	}
	if d.cmdDown {
		flag |= C.kCGEventFlagMaskCommand
	}
	if d.shiftDown {
		flag |= C.kCGEventFlagMaskShift
	}
	if flag != 0 {
		C.CGEventSetFlags(event, C.CGEventFlags(flag))
	}

	C.CGEventPost(C.kCGSessionEventTap, event)

	switch key {
	case "cmd":
		d.cmdDown = down
	case "alt":
		d.altDown = down
	case "control":
		d.ctrlDown = down
	case "shift":
		d.shiftDown = down
	}

	time.Sleep(0)
	return nil
}

// Scroll mouse scroll
func (d *DCap) Scroll(x, y int) {
	event := C.createWheelEvent(C.int(x), C.int(y))
	defer C.CFRelease(C.CFTypeRef(event))
	C.CGEventPost(C.kCGHIDEventTap, event)
}

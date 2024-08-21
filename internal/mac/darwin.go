//go:build cgo && darwin

package mac

/*
   #if __ENVIRONMENT_MAC_OS_X_VERSION_MIN_REQUIRED__ > MAC_OS_VERSION_14_4
   #cgo CFLAGS: -x objective-c
   #cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ScreenCaptureKit
   #include <ScreenCaptureKit/ScreenCaptureKit.h>
   #else
   #cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
   #endif
   #include <CoreGraphics/CoreGraphics.h>

   static CGImageRef capture(CGDirectDisplayID id, CGRect diIntersectDisplayLocal, CGColorSpaceRef colorSpace) {
   #if __ENVIRONMENT_MAC_OS_X_VERSION_MIN_REQUIRED__ > MAC_OS_VERSION_14_4
       dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);
       __block CGImageRef result = nil;
       [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent* content, NSError* error) {
           @autoreleasepool {
               if (error) {
                   dispatch_semaphore_signal(semaphore);
                   return;
               }
               SCDisplay* target = nil;
               for (SCDisplay *display in content.displays) {
                   if (display.displayID == id) {
                       target = display;
                       break;
                   }
               }
               if (!target) {
                   dispatch_semaphore_signal(semaphore);
                   return;
               }
               SCContentFilter* filter = [[SCContentFilter alloc] initWithDisplay:target excludingWindows:@[]];
               SCStreamConfiguration* config = [[SCStreamConfiguration alloc] init];
               config.sourceRect = diIntersectDisplayLocal;
               config.width = diIntersectDisplayLocal.size.width;
               config.height = diIntersectDisplayLocal.size.height;
               [SCScreenshotManager captureImageWithFilter:filter
                                             configuration:config
                                         completionHandler:^(CGImageRef img, NSError* error) {
                   if (!error) {
                       result = CGImageCreateCopyWithColorSpace(img, colorSpace);
                   }
                   dispatch_semaphore_signal(semaphore);
               }];
           }
       }];
       dispatch_semaphore_wait(semaphore, DISPATCH_TIME_FOREVER);
       dispatch_release(semaphore);
       return result;
   #else
       CGImageRef img = CGDisplayCreateImageForRect(id, diIntersectDisplayLocal);
       if (!img) {
           return nil;
       }
       CGImageRef copy = CGImageCreateCopyWithColorSpace(img, colorSpace);
       CGImageRelease(img);
       if (!copy) {
           return nil;
       }
       return copy;
   #endif
   }
*/
import "C"

import (
	"image"
	"unsafe"
)

func NumActiveDisplays() int {
	var count C.uint32_t = 0
	if C.CGGetActiveDisplayList(0, nil, &count) == C.kCGErrorSuccess {
		return int(count)
	} else {
		return 0
	}
}

func GetDisplayBounds(displayIndex int) image.Rectangle {
	id := getDisplayId(displayIndex)
	main := C.CGMainDisplayID()

	var rect image.Rectangle

	bounds := GetCoreGraphicsCoordinateOfDisplay(id)
	rect.Min.X = int(bounds.origin.x)
	if main == id {
		rect.Min.Y = 0
	} else {
		mainBounds := GetCoreGraphicsCoordinateOfDisplay(main)
		mainHeight := mainBounds.size.height
		rect.Min.Y = int(mainHeight - (bounds.origin.y + bounds.size.height))
	}
	rect.Max.X = rect.Min.X + int(bounds.size.width)
	rect.Max.Y = rect.Min.Y + int(bounds.size.height)

	return rect
}

func getDisplayId(displayIndex int) C.CGDirectDisplayID {
	main := C.CGMainDisplayID()
	if displayIndex == 0 {
		return main
	} else {
		n := NumActiveDisplays()
		ids := make([]C.CGDirectDisplayID, n)
		if C.CGGetActiveDisplayList(C.uint32_t(n), (*C.CGDirectDisplayID)(unsafe.Pointer(&ids[0])), nil) != C.kCGErrorSuccess {
			return 0
		}
		index := 0
		for i := 0; i < n; i++ {
			if ids[i] == main {
				continue
			}
			index++
			if index == displayIndex {
				return ids[i]
			}
		}
	}

	return 0
}

func GetCoreGraphicsCoordinateOfDisplay(id C.CGDirectDisplayID) C.CGRect {
	main := C.CGDisplayBounds(C.CGMainDisplayID())
	r := C.CGDisplayBounds(id)
	return C.CGRectMake(r.origin.x, -r.origin.y-r.size.height+main.size.height,
		r.size.width, r.size.height)
}

func GetCoreGraphicsCoordinateFromWindowsCoordinate(p C.CGPoint, mainDisplayBounds C.CGRect) C.CGPoint {
	return C.CGPointMake(p.x, mainDisplayBounds.size.height-p.y)
}

func CreateBitmapContext(width int, height int, data *C.uint32_t, bytesPerRow int) C.CGContextRef {
	colorSpace := CreateColorspace()
	if colorSpace == 0 {
		return 0
	}
	defer C.CGColorSpaceRelease(colorSpace)

	return C.CGBitmapContextCreate(unsafe.Pointer(data),
		C.size_t(width),
		C.size_t(height),
		8, // bits per component
		C.size_t(bytesPerRow),
		colorSpace,
		C.kCGImageAlphaNoneSkipFirst)
}

func CreateColorspace() C.CGColorSpaceRef {
	return C.CGColorSpaceCreateWithName(C.kCGColorSpaceSRGB)
}

func ActiveDisplayList() []C.CGDirectDisplayID {
	count := C.uint32_t(NumActiveDisplays())
	ret := make([]C.CGDirectDisplayID, count)
	if count > 0 && C.CGGetActiveDisplayList(count, (*C.CGDirectDisplayID)(unsafe.Pointer(&ret[0])), nil) == C.kCGErrorSuccess {
		return ret
	} else {
		return make([]C.CGDirectDisplayID, 0)
	}
}

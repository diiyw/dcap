package dcap

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/gen2brain/shm"
	"github.com/jezek/xgb"
	mshm "github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xinerama"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
)

type DCap struct {
	im                *image.RGBA
	Displays          []image.Rectangle
	xgbConn           *xgb.Conn
	useShm            bool
	defaultScreen     *xproto.ScreenInfo
	wholeScreenBounds image.Rectangle
}

func NewDCap() (*DCap, error) {
	c, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	if err = xinerama.Init(c); err != nil {
		return nil, err
	}
	if err = xtest.Init(c); err != nil {
		return nil, err
	}
	reply, err := xinerama.QueryScreens(c).Reply()
	if err != nil {
		return nil, err
	}
	var d = &DCap{
		xgbConn:  c,
		Displays: make([]image.Rectangle, len(reply.ScreenInfo)),
	}

	primary := reply.ScreenInfo[0]
	x0 := int(primary.XOrg)
	y0 := int(primary.YOrg)
	for i, screenInfo := range reply.ScreenInfo {
		x := int(screenInfo.XOrg) - x0
		y := int(screenInfo.YOrg) - y0
		w := int(screenInfo.Width)
		h := int(screenInfo.Height)
		d.Displays[i] = image.Rect(x, y, x+w, y+h)
	}

	d.useShm = true
	err = mshm.Init(d.xgbConn)
	if err != nil {
		d.useShm = false
	}
	d.defaultScreen = xproto.Setup(c).DefaultScreen(c)
	d.wholeScreenBounds = image.Rect(0, 0, int(d.defaultScreen.WidthInPixels), int(d.defaultScreen.HeightInPixels))
	return d, nil
}

// Close close connection
func (d *DCap) Close() {
	d.xgbConn.Close()
}

func (d *DCap) Capture(x, y, width, height int) error {
	d.NewImage(x, y, width, height)
	reply, err := xinerama.QueryScreens(d.xgbConn).Reply()
	if err != nil {
		return err
	}

	primary := reply.ScreenInfo[0]
	x0 := int(primary.XOrg)
	y0 := int(primary.YOrg)

	targetBounds := image.Rect(x+x0, y+y0, x+x0+width, y+y0+height)
	intersect := d.wholeScreenBounds.Intersect(targetBounds)

	if !intersect.Empty() {
		var data []byte

		if d.useShm {
			shmSize := intersect.Dx() * intersect.Dy() * 4
			shmId, err := shm.Get(shm.IPC_PRIVATE, shmSize, shm.IPC_CREAT|0777)
			if err != nil {
				return err
			}

			seg, err := mshm.NewSegId(d.xgbConn)
			if err != nil {
				return err
			}

			data, err = shm.At(shmId, 0, 0)
			if err != nil {
				return err
			}

			mshm.Attach(d.xgbConn, seg, uint32(shmId), false)

			defer mshm.Detach(d.xgbConn, seg)
			defer func() {
				_ = shm.Rm(shmId)
			}()
			defer func() {
				_ = shm.Dt(data)
			}()

			_, err = mshm.GetImage(d.xgbConn, xproto.Drawable(d.defaultScreen.Root),
				int16(intersect.Min.X), int16(intersect.Min.Y),
				uint16(intersect.Dx()), uint16(intersect.Dy()), 0xffffffff,
				byte(xproto.ImageFormatZPixmap), seg, 0).Reply()
			if err != nil {
				return err
			}
		} else {
			xImg, err := xproto.GetImage(d.xgbConn, xproto.ImageFormatZPixmap, xproto.Drawable(d.defaultScreen.Root),
				int16(intersect.Min.X), int16(intersect.Min.Y),
				uint16(intersect.Dx()), uint16(intersect.Dy()), 0xffffffff).Reply()
			if err != nil {
				return err
			}

			data = xImg.Data
		}

		// BitBlt by hand
		offset := 0
		for iy := intersect.Min.Y; iy < intersect.Max.Y; iy++ {
			for ix := intersect.Min.X; ix < intersect.Max.X; ix++ {
				r := data[offset+2]
				g := data[offset+1]
				b := data[offset]
				d.im.SetRGBA(ix-(x+x0), iy-(y+y0), color.RGBA{r, g, b, 255})
				offset += 4
			}
		}
	}
	return nil
}

// MouseMove move mouse to x,y
func (d *DCap) MouseMove(x, y int) error {
	cookie := xproto.WarpPointerChecked(d.xgbConn, xproto.WindowNone, d.defaultScreen.Root, 0, 0, 0, 0, int16(x), int16(y))
	if err := cookie.Check(); err != nil {
		return err
	}
	d.xgbConn.Sync()
	return nil
}

// ToggleMouse toggle mouse button event, https://www.x.org/releases/X11R7.7/doc/xextproto/xtest.html
func (d *DCap) ToggleMouse(button MouseButton, down bool) error {
	var typ byte = xproto.ButtonPress
	if !down {
		typ = xproto.ButtonRelease
	}
	// Simulate a left mouse button press event
	cookie := xtest.FakeInputChecked(d.xgbConn, typ, byte(button)+1, 0, d.defaultScreen.Root, 0, 0, 0)
	if err := cookie.Check(); err != nil {
		return err
	}
	d.xgbConn.Sync()
	return nil
}

// ToggleKey toggle keyboard event
func (d *DCap) ToggleKey(key string, down bool) error {
	var code byte = byte(checkKeycodes(key))
	if code == 0 {
		return fmt.Errorf("key not found: %s", key)
	}
	var eventType byte = xproto.KeyPress // key down
	if !down {
		eventType = xproto.KeyRelease // key up
	}

	cookie := xtest.FakeInputChecked(d.xgbConn, eventType, code, 0, d.defaultScreen.Root, 0, 0, 0)
	if err := cookie.Check(); err != nil {
		return err
	}
	return nil
}
func (d *DCap) Scroll(x, y int) {
	var ydir byte = 4 /* Button 4 is up, 5 is down. */
	var xdir byte = 6

	if y < 0 {
		ydir = 5
	}
	if x < 0 {
		xdir = 7
	}

	for xi := 0; xi < int(math.Abs(float64(x))); xi++ {
		xtest.FakeInput(d.xgbConn, xdir, 1, 0, d.defaultScreen.Root, int16(x), int16(y), 0)
		xtest.FakeInput(d.xgbConn, xdir, 0, 0, d.defaultScreen.Root, int16(x), int16(y), 0)
	}
	for yi := 0; yi < int(math.Abs(float64(y))); yi++ {
		xtest.FakeInput(d.xgbConn, ydir, 1, 0, d.defaultScreen.Root, int16(x), int16(y), 0)
		xtest.FakeInput(d.xgbConn, ydir, 0, 0, d.defaultScreen.Root, int16(x), int16(y), 0)
	}
	d.xgbConn.Sync()
}

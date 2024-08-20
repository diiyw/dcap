package dcap

import (
	"errors"
	"image"
	"image/color"

	"github.com/gen2brain/shm"
	"github.com/jezek/xgb"
	mshm "github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xinerama"
	"github.com/jezek/xgb/xproto"
)

type DCap struct {
	Img               *image.RGBA
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
				d.Img.SetRGBA(ix-(x+x0), iy-(y+y0), color.RGBA{r, g, b, 255})
				offset += 4
			}
		}
	}
	return nil
}

func (d *DCap) CaptureDisplay(displayIndex int) error {
	if len(d.Displays)-1 < displayIndex {
		return errors.New("display not found")
	}
	rect := d.Displays[displayIndex]
	return d.Capture(rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy())
}

// MouseMove move mouse to x,y
func (cli *Client) MouseMove(x, y int) error {
	return cli.cli.WarpPointer(uint16(x), uint16(y))
}

// ToggleMouse toggle mouse button event, https://www.x.org/releases/X11R7.7/doc/xextproto/xtest.html
func (cli *Client) ToggleMouse(button MouseButton, down bool) error {
	t := 4 // button down
	if !down {
		t = 5 // button up
	}
	return cli.cli.TestFakeInput(byte(t), byte(button)+1)
}

// ToggleKey toggle keyboard event
func (cli *Client) ToggleKey(key string, down bool) error {
	code := checkKeycodes(key)
	if code == 0 {
		return fmt.Errorf("key not found: %s", key)
	}
	t := 2 // key down
	if !down {
		t = 3 // key up
	}
	n := cli.cli.KeysymToKeycode(code)
	return cli.cli.TestFakeInput(byte(t), n)
}

// Scroll https://github.com/go-vgo/robotgo/blob/master/mouse/mouse_c.h#L313
func (cli *Client) Scroll(x, y int) {
	run := func(dir byte, cnt int) {
		for i := 0; i < cnt; i++ {
			// https://gitlab.freedesktop.org/xorg/lib/libxtst/-/blob/master/src/XTest.c#L181
			// transform press to 4 and up to 5
			cli.cli.TestFakeInput(4, dir)
			cli.cli.TestFakeInput(5, dir)
		}
	}
	if x != 0 {
		dir := 6 // up
		if x < 0 {
			dir = 7 // down
		}
		run(byte(dir), int(math.Abs(float64(x))))
	}
	if y != 0 {
		dir := 4 // up
		if y < 0 {
			dir = 5 // down
		}
		run(byte(dir), int(math.Abs(float64(y))))
	}
}

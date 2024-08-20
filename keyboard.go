package dcap

import "github.com/diiyw/dcap/internal/keycode"

// https://github.com/go-vgo/robotgo/blob/master/key/goKey.h#L142
func checkKeycodes(key string) int {
	if len(key) == 1 {
		return keycode.ForChar(key)
	}
	return keycode.Maps[key]
}

package dcap

import "github.com/diiyw/dcap/clipboard"

// ClipboardSet set text to clipboard
func (d *DCap) ClipboardSet(text string) error {
	return clipboard.Set(text)
}

// ClipboardGet get text from clipboard
func (d *DCap) ClipboardGet() (string, error) {
	return clipboard.Get()
}

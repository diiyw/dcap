# dcap
Cross platform desktop capture library for golang

## Installation
```bash
go get github.com/diiyw/dcap
```

## Usage
```go
package main

import (
    "fmt"
    "github.com/diiyw/dcap"
)

func main() {
    d, err := NewDCap()
	if err != nil {
		fmt.Println(err)
        return
	}
    defer d.Close()
    if err = d.CaptureDisplay(0); err != nil {
		fmt.Println(err)
        return
	}
    fi, err := os.Create("test.png")
    if err != nil {
        fmt.Println(err)
        return
    }
    png.Encode(fi, d.Image())
    fi.Close()
}
```
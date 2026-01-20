package connhelper

import "fmt"

type ConnHelper interface {
	Write(payload []byte)
	Read() ([]byte, error)
}

const maxFrameLength = 1500

var ErrFrameTooLarge = fmt.Errorf("frame too large")

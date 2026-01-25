package framereaderwriter

import (
	"fmt"
)

type FrameReaderWriter interface {
	Write([]byte) error
	Read() ([]byte, error)
}

const maxFrameLength = 1500

var ErrFrameTooLarge = fmt.Errorf("frame too large")

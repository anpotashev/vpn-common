package framereaderwriter

import (
	"fmt"
	"time"
)

type FrameReaderWriter interface {
	Write([]byte) error
	Read() ([]byte, error)
	SetReadDeadline(time.Time) error
}

const maxFrameLength = 1500

var ErrFrameTooLarge = fmt.Errorf("frame too large")

package framereaderwriter

import (
	"fmt"
	"time"
)

type FrameReaderWriter interface {
	Write(payload []byte)
	Read() ([]byte, error)
	SetReadDeadline(deadline time.Time) error
}

const maxFrameLength = 1500

var ErrFrameTooLarge = fmt.Errorf("frame too large")

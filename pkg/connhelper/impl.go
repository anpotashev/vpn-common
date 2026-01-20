package connhelper

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"time"
)

type impl struct {
	conn    net.Conn
	outChan chan []byte
	ctx     context.Context
}

func New(ctx context.Context, conn net.Conn) ConnHelper {
	result := &impl{
		conn:    conn,
		outChan: make(chan []byte, 1024),
		ctx:     ctx,
	}
	go result.startListeningOutChan()
	return result
}

func (i *impl) Write(payload []byte) {
	select {
	case i.outChan <- payload:
	case <-i.ctx.Done():
	}
}

func (i *impl) startListeningOutChan() {
	defer i.conn.Close()
	for {
		select {
		case payload := <-i.outChan:
			err := i.writeFrame(payload)
			if err != nil {
				return
			}
		case <-i.ctx.Done():
			return
		}
	}
}

func (i *impl) writeFrame(payload []byte) error {
	if len(payload) > maxFrameLength {
		return ErrFrameTooLarge
	}
	var pktLength [4]byte
	binary.BigEndian.PutUint32(pktLength[:], uint32(len(payload)))
	if _, err := i.conn.Write(pktLength[:]); err != nil {
		return err
	}
	_, err := i.conn.Write(payload)
	return err
}

func (i *impl) SetReadDeadline(deadline time.Time) error {
	return i.conn.SetReadDeadline(deadline)
}

func (i *impl) Read() ([]byte, error) {
	select {
	case <-i.ctx.Done():
		return nil, i.ctx.Err()
	default:
	}
	var lenBuf [4]byte
	_, err := io.ReadFull(i.conn, lenBuf[:])
	if err != nil {
		return nil, err
	}
	frameLen := binary.BigEndian.Uint32(lenBuf[:])
	if frameLen > maxFrameLength {
		return nil, ErrFrameTooLarge
	}
	result := make([]byte, frameLen)
	_, err = io.ReadFull(i.conn, result)
	if err != nil {
		return nil, err
	}
	return result[:], nil
}

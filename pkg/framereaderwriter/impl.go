package framereaderwriter

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/anpotashev/vpn-common/pkg/logconfig"
)

type impl struct {
	conn    net.Conn
	outChan chan []byte
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *slog.Logger
}

var ping = []byte{0}
var pong = []byte{1}

const pingInterval = time.Second * 10

func New(ctx context.Context, logger *slog.Logger, conn net.Conn) FrameReaderWriter {
	ctx, cancel := context.WithCancel(ctx)
	result := &impl{
		conn:    conn,
		outChan: make(chan []byte, 1024),
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger.With("pkt", "framereaderwriter"),
	}
	go result.startListeningOutChan()
	go result.startPing()
	return result
}

func (i *impl) Write(payload []byte) error {
	select {
	case i.outChan <- payload:
		return nil
	case <-i.ctx.Done():
		return i.ctx.Err()
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

func (i *impl) Read() ([]byte, error) {
	select {
	case <-i.ctx.Done():
		return nil, i.ctx.Err()
	default:
	}
	for {
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
		if bytes.Equal(result, ping[:]) {
			i.logger.Log(nil, logconfig.TraceLogLevel, "Received the ping message.")
			err = i.writeFrame(pong[:])
			if err != nil {
				return nil, err
			}
			continue
		}
		if bytes.Equal(result, pong[:]) {
			i.logger.Log(nil, logconfig.TraceLogLevel, "Received the pong message.")
			err = i.conn.SetReadDeadline(time.Now().Add(pingInterval))
			continue
		}
		return result[:], nil
	}
}

func (i impl) startPing() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	err := i.conn.SetReadDeadline(time.Now().Add(pingInterval))
	if err != nil {
		i.cancel()
		return
	}
	for {
		select {
		case <-ticker.C:
			i.logger.Log(nil, logconfig.TraceLogLevel, "Sending ping")
			err = i.Write(ping)
			if err != nil {
				i.cancel()
				return
			}
		case <-i.ctx.Done():
			return
		}
	}
}

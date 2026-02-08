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
	inChan  chan []byte
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *slog.Logger
}

var ping = []byte{0}
var pong = []byte{1}

const pingInterval = time.Second * 10
const pingGap = time.Second

func New(ctx context.Context, logger *slog.Logger, conn net.Conn) FrameReaderWriter {
	ctx, cancel := context.WithCancel(ctx)
	result := &impl{
		conn:    conn,
		outChan: make(chan []byte, 1024),
		inChan:  make(chan []byte, 1024),
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger.With("pkt", "framereaderwriter"),
	}
	go result.startListeningOutChan()
	go result.startReading()
	return result
}

func (i *impl) Write(payload []byte) error {
	i.logger.Log(nil, logconfig.TraceLogLevel, "Writing payload to the outChan.", "Payload", payload)
	select {
	case i.outChan <- payload:
		i.logger.Log(nil, logconfig.TraceLogLevel, "The payload successfully written to the outChan.", "Payload", payload)
		return nil
	case <-i.ctx.Done():
		i.logger.Warn("Received ctx.done", "Error", i.ctx.Err())
		return i.ctx.Err()
	}
}

func (i *impl) startListeningOutChan() {
	defer func() { _ = i.conn.Close() }()
	for {
		i.logger.Log(nil, logconfig.TraceLogLevel, "Next iteration of reading outChan")
		select {
		case payload := <-i.outChan:
			i.logger.Log(nil, logconfig.TraceLogLevel, "Read message from the outChan", "Payload", payload)
			err := i.writeFrame(payload)
			if err != nil {
				i.logger.Error("Error writing the frame", "Error", err)
				return
			}
			i.logger.Log(nil, logconfig.TraceLogLevel, "The message from the outChan sent", "Payload", payload)
		case <-i.ctx.Done():
			i.logger.Warn("Received ctx.done", "Error", i.ctx.Err())
			return
		}
	}
}

func (i *impl) writeFrame(payload []byte) error {
	i.logger.Log(nil, logconfig.TraceLogLevel, "Start writing frame.", "Payload", payload)
	if len(payload) > maxFrameLength {
		i.logger.Error("Trying to write the large payload", "Payload length", len(payload))
		return ErrFrameTooLarge
	}
	var pktLength [4]byte
	binary.BigEndian.PutUint32(pktLength[:], uint32(len(payload)))
	i.logger.Log(nil, logconfig.TraceLogLevel, "Writing frame length.")
	if _, err := i.conn.Write(pktLength[:]); err != nil {
		i.logger.Error("Error writing the length header to the connection", "Error", err)
		return err
	}
	i.logger.Log(nil, logconfig.TraceLogLevel, "Writing payload.")
	_, err := i.conn.Write(payload)
	if err != nil {
		i.logger.Error("Error writing the payload to the connection", "Error", err)
	}
	return err
}

func (i *impl) startReading() {
	defer i.cancel()
	select {
	case <-i.ctx.Done():
		i.logger.Warn("Received ctx.done", "Error", i.ctx.Err())
		return
	default:
	}
	i.schedulePing()
	for {
		var lenBuf [4]byte
		_, err := io.ReadFull(i.conn, lenBuf[:])
		if err != nil {
			i.logger.Error("Error reading the length header from the connection", "Error", err)
			if err == io.EOF {
				i.logger.Error("EOF: peer closed connection")
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				i.logger.Error("Timeout")
			} else {
				i.logger.Error("Read error", "err", err)
			}
			return
		}
		i.logger.Log(nil, logconfig.TraceLogLevel, "Read frame length.", "Frame length", binary.BigEndian.Uint32(lenBuf[:]))
		frameLen := binary.BigEndian.Uint32(lenBuf[:])
		if frameLen > maxFrameLength {
			i.logger.Error("Received length is too large.")
			return
		}
		payload := make([]byte, frameLen)
		_, err = io.ReadFull(i.conn, payload)
		if err != nil {
			i.logger.Error("Error reading the the payload from the connection", "Error", err)
			return
		}
		i.logger.Log(nil, logconfig.TraceLogLevel, "Read frame payload.")
		if bytes.Equal(payload, ping[:]) {
			i.logger.Log(nil, logconfig.TraceLogLevel, "Received the ping message.")
			err = i.Write(pong[:])
			if err != nil {
				return
			}
			continue
		}
		if bytes.Equal(payload, pong[:]) {
			i.logger.Log(nil, logconfig.TraceLogLevel, "Received the pong message.")
			i.schedulePing()
		}
		i.inChan <- payload[:]
	}
}

func (i *impl) Read() ([]byte, error) {
	select {
	case payload := <-i.inChan:
		return payload, nil
	case <-i.ctx.Done():
		return nil, i.ctx.Err()
	}
}

func (i *impl) schedulePing() {
	i.logger.Log(nil, logconfig.TraceLogLevel, "Scheduling the next ping")
	go func() {
		time.Sleep(pingInterval)
		err := i.Write(ping)
		if err != nil {
			i.logger.Error("Error writing the ping message", "Error", err)
			i.cancel()
		}
		err = i.conn.SetReadDeadline(time.Now().Add(pingInterval).Add(pingGap))
		if err != nil {
			i.cancel()
		}
	}()
}

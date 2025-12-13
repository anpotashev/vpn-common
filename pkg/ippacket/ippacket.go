package ippacket

import (
	"fmt"
	"net"
)

type IPPacket interface {
	GetSource() net.IP
	GetDestination() net.IP
	GetBytes() []byte
}

type impl []byte

var ErrNotAnIPPacket = fmt.Errorf("not an IP-packet")

func New(data []byte) (IPPacket, error) {
	// todo расшифровка ??
	if len(data) < 20 {
		return nil, ErrNotAnIPPacket
	}
	// todo прочие проверки является ли массив IP пакетом.
	return impl(data), nil
}

func (i impl) GetSource() net.IP {
	return net.IP(i[12:16])
}

func (i impl) GetDestination() net.IP {
	return net.IP(i[16:20])
}

func (i impl) GetBytes() []byte {
	return i
}

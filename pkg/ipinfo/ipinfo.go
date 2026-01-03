package ipinfo

import (
	"encoding/binary"
	"net"
)

type IPInfo [14]byte

func New(ip net.IP, mask net.IPMask, gw net.IP, mtu uint16) IPInfo {
	var r IPInfo
	binary.LittleEndian.PutUint16(r[0:2], mtu)
	copy(r[2:6], ip.To4())
	copy(r[6:10], gw.To4())
	copy(r[10:14], mask)
	return r
}

func (data IPInfo) Parse() (ip net.IP, mask net.IPMask, gw net.IP, mtu uint16) {
	mtu = binary.LittleEndian.Uint16(data[:2])
	ip = make(net.IP, 4)
	gw = make(net.IP, 4)
	mask = make(net.IPMask, 4)

	copy(ip, data[2:6])
	copy(gw, data[6:10])
	copy(mask, data[10:14])
	return
}

package ippacket

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net"
	"strings"

	"github.com/anpotashev/vpn-common/pkg/logconfig"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type IPPacket interface {
	Source() net.IP
	Destination() net.IP
	Bytes() []byte
	HashByDestination() (uint16, error)
	HashBySource() (uint16, error)
	fmt.Stringer
}
type impl struct {
	gopacket.Packet
	logger *slog.Logger
}

var ErrNoIPv4Layer = fmt.Errorf("no IPv4 layer found")
var ErrNoTCPLayer = fmt.Errorf("no TCP layer found in packet")
var ErrNoUDPLayer = fmt.Errorf("no UDP layer found in packet")

func New(data []byte) (IPPacket, error) {
	packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.NoCopy)
	layer := packet.Layer(layers.LayerTypeIPv4)
	if layer == nil {
		return nil, ErrNoIPv4Layer
	}
	var logger = slog.Default().With("pkg", "ippacket")
	return impl{
		Packet: packet,
		logger: logger,
	}, nil
}

func (i impl) HashByDestination() (uint16, error) {
	layer := i.Layer(layers.LayerTypeIPv4)
	if layer == nil {
		return 0, ErrNoIPv4Layer
	}
	ip := layer.(*layers.IPv4)
	switch ip.Protocol {
	case layers.IPProtocolTCP:
		tcpLayer := i.Layer(layers.LayerTypeTCP).(*layers.TCP)
		if tcpLayer == nil {
			return 0, ErrNoTCPLayer
		}
		i.logger.Log(nil, logconfig.TraceLogLevel, "Calculating by destination. TCP.", "Source IP", ip.SrcIP, "Destination", ip.DstIP, "Source Port", tcpLayer.SrcPort, "Destination Port", tcpLayer.DstPort)
		return hash(ip.DstIP, uint16(tcpLayer.DstPort), byte(layers.IPProtocolTCP)), nil
	case layers.IPProtocolUDP:
		udpLayer := i.Layer(layers.LayerTypeUDP).(*layers.UDP)
		if udpLayer == nil {
			return 0, ErrNoUDPLayer
		}
		i.logger.Log(nil, logconfig.TraceLogLevel, "Calculating by destination. UDP.", "Source IP", ip.SrcIP, "Destination", ip.DstIP, "Source Port", udpLayer.SrcPort, "Destination Port", udpLayer.DstPort)
		return hash(ip.DstIP, uint16(udpLayer.SrcPort), byte(layers.IPProtocolUDP)), nil
	default:
		i.logger.Log(nil, logconfig.TraceLogLevel, "Calculating by source. Other.", "Source IP", ip.SrcIP, "Destination", ip.DstIP)
		return hash(ip.DstIP, 0, byte(ip.Protocol)), nil
	}
}

func (i impl) HashBySource() (uint16, error) {
	layer := i.Layer(layers.LayerTypeIPv4)
	if layer == nil {
		return 0, ErrNoIPv4Layer
	}
	ip := layer.(*layers.IPv4)
	switch ip.Protocol {
	case layers.IPProtocolTCP:
		tcpLayer := i.Layer(layers.LayerTypeTCP).(*layers.TCP)
		if tcpLayer == nil {
			return 0, ErrNoTCPLayer
		}
		i.logger.Log(nil, logconfig.TraceLogLevel, "Calculating by source. TCP.", "Source IP", ip.SrcIP, "Destination", ip.DstIP, "Source Port", tcpLayer.SrcPort, "Destination Port", tcpLayer.DstPort)
		return hash(ip.SrcIP, uint16(tcpLayer.DstPort), byte(layers.IPProtocolTCP)), nil
	case layers.IPProtocolUDP:
		udpLayer := i.Layer(layers.LayerTypeUDP).(*layers.UDP)
		if udpLayer == nil {
			return 0, ErrNoUDPLayer
		}
		i.logger.Log(nil, logconfig.TraceLogLevel, "Calculating by source. UDP.", "Source IP", ip.SrcIP, "Destination", ip.DstIP, "Source Port", udpLayer.SrcPort, "Destination Port", udpLayer.DstPort)
		return hash(ip.SrcIP, uint16(udpLayer.DstPort), byte(layers.IPProtocolUDP)), nil
	default:
		i.logger.Log(nil, logconfig.TraceLogLevel, "Calculating by source. Other.", "Source IP", ip.SrcIP, "Destination", ip.DstIP)
		return hash(ip.SrcIP, 0, byte(ip.Protocol)), nil
	}
}

func hash(ip net.IP, port uint16, protocol byte) uint16 {
	h := fnv.New32a()
	ipv4 := ip.To4()
	if ipv4 != nil {
		h.Write(ipv4)
	} else {
		h.Write(ip.To16())
	}
	var buf [3]byte
	binary.BigEndian.PutUint16(buf[:2], port)
	buf[2] = protocol
	h.Write(buf[:])
	return uint16(h.Sum32())
}

func (i impl) Source() net.IP {
	layer := i.Layer(layers.LayerTypeIPv4)
	return layer.(*layers.IPv4).SrcIP.To4()
}

func (i impl) Destination() net.IP {
	layer := i.Layer(layers.LayerTypeIPv4)
	return layer.(*layers.IPv4).DstIP.To4()
}

func (i impl) Bytes() []byte {
	return i.Data()
}

func (i impl) String() string {
	data := i.Data()
	strArr := make([]string, min(len(data), 20))
	for i := range strArr {
		strArr[i] = fmt.Sprintf("0x%02X", i)
	}
	suffix := ""
	if len(data) > 20 {
		suffix = "..."
	}
	return strings.Join(strArr, " ") + suffix
}

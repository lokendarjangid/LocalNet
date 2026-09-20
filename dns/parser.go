package dns

import (
	"encoding/binary"
	"strings"
)

type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

func ParseHeader(buf []byte) DNSHeader {
	return DNSHeader{
		ID:      binary.BigEndian.Uint16(buf[0:2]),
		Flags:   binary.BigEndian.Uint16(buf[2:4]),
		QDCount: binary.BigEndian.Uint16(buf[4:6]),
		ANCount: binary.BigEndian.Uint16(buf[6:8]),
		NSCount: binary.BigEndian.Uint16(buf[8:10]),
		ARCount: binary.BigEndian.Uint16(buf[10:12]),
	}
}

func ParseName(buf []byte, offset int) (string, int) {
	var labels []string
	for {
		length := int(buf[offset])
		if length == 0 {
			offset++ // move past the terminating zero byte
			break
		}
		offset++ // move past the length byte
		labels = append(labels, string(buf[offset:offset+length]))
		offset += length
	}
	return strings.Join(labels, "."), offset
}

package dns

import (
	"encoding/binary"
	"net"
)

func BuildResponseHeader(qID uint16) []byte {
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], qID)    // same ID as the query
	binary.BigEndian.PutUint16(header[2:4], 0x8180) // flags: QR=1 (response), RD=1, RA=1
	binary.BigEndian.PutUint16(header[4:6], 1)      // QDCOUNT = 1 (we echo the question)
	binary.BigEndian.PutUint16(header[6:8], 1)      // ANCOUNT = 1 (one answer)
	binary.BigEndian.PutUint16(header[8:10], 0)     // NSCOUNT
	binary.BigEndian.PutUint16(header[10:12], 0)    // ARCOUNT
	return header
}

func BuildResponseQuestions(buf []byte, startOffset int) (questions []byte, nameByte []byte) {
	offset := startOffset

	for {
		length := int(buf[offset])
		offset++
		if length == 0 {
			break
		}
		offset += length
	}
	end := offset + 4

	return buf[startOffset:end], buf[startOffset:offset]
}

func BuildResponseAnswer(name []byte, ip net.IP) []byte {
	var out []byte

	out = append(out, name...) // NAME

	typeField := make([]byte, 2)
	binary.BigEndian.PutUint16(typeField, 1) // TYPE = A
	out = append(out, typeField...)

	classField := make([]byte, 2)
	binary.BigEndian.PutUint16(classField, 1) // CLASS = IN
	out = append(out, classField...)

	ttlField := make([]byte, 4) // note: 4 bytes, not 2 — needs PutUint32
	binary.BigEndian.PutUint32(ttlField, 300)
	out = append(out, ttlField...)

	ipv4 := ip.To4()
	rdlengthField := make([]byte, 2)
	binary.BigEndian.PutUint16(rdlengthField, uint16(len(ipv4))) // RDLENGTH = 4
	out = append(out, rdlengthField...)

	out = append(out, ipv4...) // RDATA — the raw IP bytes, no encoding needed

	return out
}

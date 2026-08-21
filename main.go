package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func main() {
	addr := net.UDPAddr{
		Port: 5352,
		IP:   net.ParseIP("127.0.0.1"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	fmt.Println("Listening on port 5352")

	buf := make([]byte, 512)

	for {
		_, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("read error:", err)
			continue
		}
		header := parseHeader(buf)
		name, nextOffset := parseName(buf, 12) // question starts right after the 12-byte header
		qtype := binary.BigEndian.Uint16(buf[nextOffset : nextOffset+2])
		qclass := binary.BigEndian.Uint16(buf[nextOffset+2 : nextOffset+4])

		fmt.Printf("Query ID=%d from %s: name=%s type=%d class=%d\n",
			header.ID, clientAddr, name, qtype, qclass)
	}
}

package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/lokendarjangid/LocalNet/dns"
)

func getEnvAsInt(envkey string, fallback int) int {
	if val := os.Getenv(envkey); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnv(envkey string, fallback string) string {
	if val := os.Getenv(envkey); val != "" {
		return val
	}
	return fallback
}

// It is a function to load data from records file
func loadData(path string) (map[string]string, error) {
	_, err := os.Stat(path)

	// create a records file in
	if os.IsNotExist(err) {
		emptyRecords := map[string]string{}
		data, err := json.MarshalIndent(emptyRecords, "", "  ")
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(path, data, 0644)
		if err != nil {
			return nil, err
		}
		log.Println("no config found — created empty file at", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var records map[string]string
	err = json.Unmarshal(data, &records)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func forwardToUpstream(query []byte) ([]byte, error) {
	upstreamAddr, err := net.ResolveUDPAddr("udp", "8.8.8.8:53")
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, upstreamAddr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Println("failed to close upstream connection:", closeErr)
		}
	}()

	// set a timeout so a dead/slow upstream doesn't hang your server forever
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return nil, err
	}

	_, err = conn.Write(query)
	if err != nil {
		return nil, err
	}

	response := make([]byte, 512)
	n, err := conn.Read(response)
	if err != nil {
		return nil, err
	}

	return response[:n], nil
}

func handleQuery(buf []byte, clientAddr *net.UDPAddr, conn *net.UDPConn, records map[string]string) {
	header := dns.ParseHeader(buf)
	name, nextOffset := dns.ParseName(buf, 12) // question starts right after the 12-byte header
	qtype := binary.BigEndian.Uint16(buf[nextOffset : nextOffset+2])
	qclass := binary.BigEndian.Uint16(buf[nextOffset+2 : nextOffset+4])

	log.Printf("Query ID=%d from=%s name=%s type=%d class=%d\n",
		header.ID, clientAddr, name, qtype, qclass)

	ipstring, found := records[name]

	if found {
		response := dns.BuildResponseHeader(header.ID)
		questions, nameByte := dns.BuildResponseQuestions(buf, 12)
		response = append(response, questions...)
		answer := dns.BuildResponseAnswer(nameByte, net.ParseIP(ipstring))
		response = append(response, answer...)
		if _, err := conn.WriteToUDP(response, clientAddr); err != nil {
			log.Println("failed to write local response:", err)
		}
		return
	}

	upstreamResponse, err := forwardToUpstream(buf)
	if err != nil {
		log.Println("upstream forward failed:", err)
		return
	}
	if _, err := conn.WriteToUDP(upstreamResponse, clientAddr); err != nil {
		log.Println("failed to write upstream response:", err)
	}
}

func main() {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		log.Println("failed to load .env file:", err)
	}

	records, err := loadData(getEnv("RECORDS_PATH", "/etc/LocalNet/records.json"))
	if err != nil {
		panic(err)
	}

	// Dynamically pull PORT from env, defaulting to 53
	port := getEnvAsInt("PORT", 53)

	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("127.0.0.1"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Println("failed to close listener:", closeErr)
		}
	}()

	log.Printf("Listening on port %d...\n", port)

	buf := make([]byte, 512)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("read error:", err)
			continue
		}

		queryCopy := make([]byte, n)
		copy(queryCopy, buf[:n])
		go handleQuery(queryCopy, clientAddr, conn, records)

	}
}

package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/joho/godotenv"
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

func loadData(path string) (map[string]string, error) {
	_, err := os.Stat(path)
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
		fmt.Println("no config found — created empty file at", path)
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

func main() {
	godotenv.Load(".env")

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
	defer conn.Close()

	fmt.Printf("Listening on port %d...\n", port)

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

		ipstring, found := records[name]
		if !found {
			fmt.Println("no record for", name)
			continue
		}

		fmt.Printf("Query ID=%d from %s: name=%s type=%d class=%d\n",
			header.ID, clientAddr, name, qtype, qclass)

		response := buildResponseHeader(header.ID)
		questions, nameByte := buildResponseQuestions(buf, 12)
		response = append(response, questions...)
		answer := buildReponseAnswer(nameByte, net.ParseIP(ipstring))
		response = append(response, answer...)

		conn.WriteToUDP(response, clientAddr)
	}
}

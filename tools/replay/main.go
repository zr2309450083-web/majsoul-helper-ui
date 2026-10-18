// Command replay feeds the JSON messages from a helper log back into a
// running helper instance. It is intended for UI regression tests on a
// separate port (for example -p 12123 -ui-port 12124).
package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type logEntry struct {
	Message string `json:"message"`
}

func main() {
	logPath := flag.String("log", "", "helper log file (gamedata-*.log)")
	url := flag.String("url", "https://127.0.0.1:12121/", "helper POST endpoint")
	delay := flag.Duration("delay", 5*time.Millisecond, "delay between messages")
	flag.Parse()

	if *logPath == "" {
		fmt.Fprintln(os.Stderr, "usage: replay -log <gamedata.log> [-url https://127.0.0.1:12121/] [-delay 5ms]")
		os.Exit(2)
	}

	file, err := os.Open(*logPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open log:", err)
		os.Exit(1)
	}
	defer file.Close()

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	total, sent, failed := 0, 0, 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		message := strings.TrimSpace(entry.Message)
		if !strings.HasPrefix(message, "{") {
			continue
		}
		total++

		req, err := http.NewRequest(http.MethodPost, *url, bytes.NewBufferString(message))
		if err != nil {
			fmt.Fprintln(os.Stderr, "build request:", err)
			failed++
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			failed++
			continue
		}
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			failed++
		} else {
			sent++
		}
		time.Sleep(*delay)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "read log:", err)
		os.Exit(1)
	}
	fmt.Printf("sent=%d failed=%d total=%d\n", sent, failed, total)
	if failed > 0 {
		os.Exit(1)
	}
}

package common

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// BSMSRecord represents a parsed BSMS 1.0 key record (BIP-129).
type BSMSRecord struct {
	Version     string
	Token       string
	Key         string
	Description string
	Signature   string
}

// ParseBSMS parses a BSMS 1.0 key record from the reader.
func ParseBSMS(r io.Reader) (*BSMSRecord, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(lines) < 4 {
		return nil, errors.New("invalid BSMS record: too few lines")
	}
	if !strings.HasPrefix(lines[0], "BSMS") {
		return nil, errors.New("invalid BSMS record: missing version header")
	}
	rec := &BSMSRecord{
		Version:     lines[0],
		Token:       lines[1],
		Key:         lines[2],
		Description: lines[3],
	}
	if len(lines) >= 5 {
		rec.Signature = lines[4]
	}
	return rec, nil
}

// IsBSMS returns true if the content looks like a BSMS file.
func IsBSMS(r io.ReadSeeker) bool {
	buf := make([]byte, 7)
	n, _ := r.Read(buf)
	_, _ = r.Seek(0, io.SeekStart)
	return n >= 4 && strings.HasPrefix(string(buf[:n]), "BSMS")
}

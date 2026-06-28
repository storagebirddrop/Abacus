package common

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// BSMSRecord represents a parsed BSMS 1.0 key record (BIP-129).
// Example from real file:
//
//	BSMS 1.0
//	wsh(sortedmulti(3,[fp/path]xpub...,...))#checksum
//	No path restrictions
//	bc1q...  (first address for verification)
type BSMSRecord struct {
	Version     string // "BSMS 1.0"
	Descriptor  string // output descriptor with checksum
	PathRestrictions string // "No path restrictions" or path list
	FirstAddress string // verification address
}

// ParseBSMS parses a BSMS 1.0 file from the reader.
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
	if len(lines) < 3 {
		return nil, errors.New("invalid BSMS: too few lines")
	}
	if !strings.HasPrefix(lines[0], "BSMS") {
		return nil, errors.New("invalid BSMS: missing version header")
	}
	rec := &BSMSRecord{
		Version:          lines[0],
		Descriptor:       lines[1],
		PathRestrictions: lines[2],
	}
	if len(lines) >= 4 {
		rec.FirstAddress = lines[3]
	}
	return rec, nil
}

// IsBSMS returns true if the content starts with a BSMS version header.
func IsBSMS(r io.ReadSeeker) bool {
	buf := make([]byte, 7)
	n, _ := r.Read(buf)
	_, _ = r.Seek(0, io.SeekStart)
	return n >= 4 && strings.HasPrefix(string(buf[:n]), "BSMS")
}

// ExtractDescriptor returns the descriptor without the checksum suffix (#...).
func ExtractDescriptor(descriptor string) string {
	if idx := strings.LastIndex(descriptor, "#"); idx != -1 {
		return descriptor[:idx]
	}
	return descriptor
}

// ExtractFingerprints returns all master key fingerprints from a descriptor.
// e.g. [b1631dce/48'/0'/0'/2'] → "b1631dce"
func ExtractFingerprints(descriptor string) []string {
	var fps []string
	seen := map[string]bool{}
	parts := strings.Split(descriptor, "[")
	for _, p := range parts[1:] { // skip first empty
		if idx := strings.Index(p, "/"); idx != -1 {
			fp := p[:idx]
			if len(fp) == 8 && !seen[fp] {
				fps = append(fps, fp)
				seen[fp] = true
			}
		}
	}
	return fps
}

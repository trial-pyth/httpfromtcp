package headers

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

var rn = []byte("\r\n")

type Headers struct {
	headers map[string]string
}

var tokenChars = []byte{'!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~'}

func isValidToken(data []byte) bool {
	for _, token := range data {
		if (token < 'a' || token > 'z') &&
			(token < 'A' || token > 'Z') &&
			(token < '0' || token > '9') &&
			!slices.Contains(tokenChars, token) {
			return false
		}
	}

	return true
}

func NewHeaders() Headers {
	return Headers{
		headers: map[string]string{},
	}
}

func (h *Headers) Get(name string) (string, bool) {
	str, ok := h.headers[strings.ToLower(name)]
	return str, ok
}

func (h *Headers) Set(key, value string) {
	key = strings.ToLower(key)
	if existingValue, ok := h.headers[key]; ok {
		h.headers[key] = existingValue + ", " + value
	} else {
		h.headers[key] = value
	}
}

func (h *Headers) ForEach(cb func(k, v string)) {
	for k, v := range h.headers {
		cb(k, v)
	}
}

func (h *Headers) Len() int {
	return len(h.headers)
}

func parseHeader(fieldLine []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed field line")

	}
	name := parts[0]
	value := bytes.TrimSpace(parts[1])

	if bytes.HasSuffix(name, []byte(" ")) {
		return "", "", fmt.Errorf("malformed field name")
	}

	return string(name), string(value), nil
}

func (h Headers) Parse(data []byte) (int, bool, error) {
	read := 0
	done := false
	for {
		idx := bytes.Index(data[read:], rn)
		if idx == -1 {
			break
		}

		if idx == 0 {
			done = true
			read += len(rn)
			break
		}

		name, value, err := parseHeader(data[read : read+idx])
		if err != nil {
			return 0, false, err
		}

		if !isValidToken([]byte(name)) {
			return 0, false, fmt.Errorf("malformed header name")
		}

		read += idx + len(rn)
		h.Set(name, value)
	}

	return read, done, nil
}

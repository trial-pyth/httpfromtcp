package request

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/trial-pyth/httpfromtcp/internal/headers"
)

type parserState string

type Request struct {
	RequestLine RequestLine
	Headers     *headers.Headers
	Body        string

	state parserState
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

var ErrorMalformedRequestLine = fmt.Errorf("malformed request line")
var ErrorUnsupportedHttpVersion = fmt.Errorf("unsupported http version")
var ErrorRequestInErrorState = fmt.Errorf("request in error state")
var SEPARATOR = []byte("\r\n")

func newRequest() *Request {
	return &Request{
		state:   StateInit,
		Headers: headers.NewHeaders(),
		Body:    "",
	}
}

func (r *RequestLine) ValidHTTP() bool {
	return r.HttpVersion == "HTTP/1.1"
}

func getInt(headers *headers.Headers, name string, defaultValue int) int {
	valueStr, exists := headers.Get(name)
	if !exists {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func (r *Request) hasBody() bool {
	length := getInt(r.Headers, "content-length", 0)
	return length > 0
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
outer:
	for {
		currentData := data[read:]

		if len(currentData) == 0 {
			break outer
		}

		switch r.state {
		case StateError:
			return 0, ErrorRequestInErrorState
		case StateInit:
			rl, n, err := parseRequestLine(currentData)
			if err != nil {
				r.state = StateError
				return 0, err
			}
			if n == 0 {
				break outer
			}

			r.RequestLine = *rl
			read += n

			r.state = StateHeaders
		case StateHeaders:
			n, done, err := r.Headers.Parse(currentData)
			if err != nil {
				r.state = StateError
				return 0, err
			}

			if n == 0 {
				break outer
			}

			read += n
			if done {
				if r.hasBody() {
					r.state = StateBody
				} else {
					r.state = StateDone
				}

				break outer
			}

		case StateBody:
			length := getInt(r.Headers, "content-length", 0)
			if length == 0 {
				panic("chunk not implmented")
			}

			remaining := min(length-len(r.Body), len(currentData))
			r.Body += string(currentData[:remaining])
			read += remaining

			if len(r.Body) > length {
				return 0, fmt.Errorf("body length %d exceeds content-length %d", len(r.Body), length)
			}

			if len(r.Body) == length {
				r.state = StateDone
				break outer
			}

			// If we've processed all available data but don't have the full body yet, break to read more
			if remaining == 0 {
				break outer
			}

			// If we've processed all available data but don't have the full body yet, break to read more
			if remaining == 0 || len(currentData) == 0 {
				break outer
			}

		case StateDone:
			break outer
		default:
			panic("somehow we have programmed poorly")
		}
	}

	return read, nil
}

func (r *Request) done() bool {
	return r.state == StateDone || r.state == StateError
}

const (
	StateInit    parserState = "init"
	StateDone    parserState = "done"
	StateBody    parserState = "body"
	StateHeaders parserState = "headers"
	StateError   parserState = "error"
)

func parseRequestLine(b []byte) (*RequestLine, int, error) {
	idx := bytes.Index(b, SEPARATOR)
	if idx == -1 {
		return nil, 0, nil
	}
	startLine := b[:idx]
	read := idx + len(SEPARATOR)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ErrorMalformedRequestLine
	}

	httpParts := bytes.Split(parts[2], []byte("/"))
	if len(httpParts) != 2 || string(httpParts[0]) != "HTTP" || string(httpParts[1]) != "1.1" {
		return nil, 0, ErrorMalformedRequestLine
	}

	rl := &RequestLine{
		Method:        string(parts[0]),
		RequestTarget: string(parts[1]),
		HttpVersion:   string(httpParts[1]),
	}

	return rl, read, nil

}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := newRequest()
	buf := make([]byte, 4096)
	bufLen := 0
	for !request.done() {
		n, err := reader.Read(buf[bufLen:])
		if err != nil {
			if err == io.EOF {
				// EOF - try to parse what we have
				if bufLen > 0 {
					readN, parseErr := request.parse(buf[:bufLen])
					if parseErr != nil {
						return nil, parseErr
					}
					// If parsing consumed data and request is done, return it
					if readN > 0 && request.done() {
						return request, nil
					}
				}
				// EOF and no data or incomplete request - this is a connection that closed without sending
				return nil, err
			}
			return nil, err
		}

		if n == 0 {
			// No data read - this shouldn't happen normally, but handle it
			break
		}

		bufLen += n
		readN, err := request.parse(buf[:bufLen])
		if err != nil {
			return nil, err
		}

		// Shift remaining data to the beginning of the buffer
		if readN > 0 {
			copy(buf, buf[readN:bufLen])
			bufLen -= readN
		}
	}

	return request, nil
}

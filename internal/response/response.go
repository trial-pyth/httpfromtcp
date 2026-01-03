package response

import (
	"fmt"
	"io"

	"github.com/trial-pyth/httpfromtcp/internal/headers"
)

type Writer struct {
	writer io.Writer
	state  WriterState
}

func NewWriter(writer io.Writer) *Writer {
	return &Writer{writer: writer}
}
type WriterState string

const (
	WriteStateStatusLine WriterState = "StatusLine"
	WriteStateHeaders    WriterState = "Headers"
	WriteStateBody       WriterState = "Body"
	WriteStateTrailer    WriterState = "Trailer"
)

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	statusLine := []byte{}
	switch statusCode {
	case StatusOK:
		statusLine = []byte("HTTP/1.1 200 OK\r\n")
	case StatusBadRequest:
		statusLine = []byte("HTTP/1.1 400 Bad Request\r\n")
	case StatusInternalServerError:
		statusLine = []byte("HTTP/1.1 500 Internal Server Error\r\n")
	default:
		return fmt.Errorf("unrecognized error code")
	}

	_, err := w.writer.Write(statusLine)
	return err
}
func (w *Writer) WriteHeaders(headers headers.Headers) error {
	b := []byte{}
	headers.ForEach(func(k, v string) {
		b = fmt.Appendf(b, "%s: %s\r\n", k, v)
	})
	b = fmt.Appendf(b, "\r\n")
	_, err := w.writer.Write(b)

	return err
}
func (w *Writer) WriteBody(body []byte) (int, error) {
	n, err := w.writer.Write(body)
	return n, err
}

// func (w *Writer) WriteChunkedBody(p []byte) (int, error)


type Response struct {
}

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

func GetDefaultHeaders(contentLen int) *headers.Headers {
	h := headers.NewHeaders()
	h.Set("Content-Length", fmt.Sprintf("%d", contentLen))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")
	return h
}
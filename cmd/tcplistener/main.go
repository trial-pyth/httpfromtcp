package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/trial-pyth/httpfromtcp/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v\n", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Printf("New connection from %s\n", conn.RemoteAddr())

	// Set a read deadline to prevent hanging
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	r, err := request.RequestFromReader(conn)
	if err != nil {
		// Skip logging EOF errors - these are usually connection probes or clients closing early
		if !errors.Is(err, io.EOF) {
			log.Printf("Error parsing request from %s: %v\n", conn.RemoteAddr(), err)
		} else {
			log.Printf("EOF from %s (connection closed early)\n", conn.RemoteAddr())
		}
		return
	}

	log.Printf("Successfully parsed request: %s %s\n", r.RequestLine.Method, r.RequestLine.RequestTarget)

	fmt.Printf("Request line:\n")
	fmt.Printf("- Method: %s\n", r.RequestLine.Method)
	fmt.Printf("- Target: %s\n", r.RequestLine.RequestTarget)
	fmt.Printf("- Version: %s\n", r.RequestLine.HttpVersion)
	fmt.Printf("- Headers:\n")
	r.Headers.ForEach(func(k, v string) {
		fmt.Printf("- %s: %s\n", k, v)
	})
	fmt.Printf("Body:\n")
	fmt.Printf("%s\n", r.Body)
	fmt.Println("---")

	// Send HTTP response
	response := "HTTP/1.1 200 OK\r\n"
	response += "Content-Type: text/plain\r\n"
	response += "Content-Length: 13\r\n"
	response += "\r\n"
	response += "Request received"

	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Printf("Error sending response: %v\n", err)
	}
}

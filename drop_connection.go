// Package traefik_drop_connection plugin main package
package traefik_drop_connection

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Config the plugin configuration.
type Config struct {
	StatusCode string `yaml:"statusCode,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		StatusCode: "",
	}
}

type dropConnection struct {
	next            http.Handler
	name            string
	statusCodeStart int
	statusCodeEnd   int
}

// New instantiates and returns the required components used to handle a HTTP request
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {

	startCode := 0
	endCode := 0
	if len(config.StatusCode) != 0 {
		codes := strings.Split(config.StatusCode, "-")

		if len(codes) != 2 {
			return nil, fmt.Errorf("error compiling status code mapping")
		}

		start, err := strconv.Atoi(codes[0])

		if err != nil {
			return nil, fmt.Errorf("error converting first status code to integer, %w", err)
		}

		end, err := strconv.Atoi(codes[1])

		if err != nil {
			return nil, fmt.Errorf("error converting second status code to integer, %w", err)
		}

		startCode = start
		endCode = end
	}

	return &dropConnection{
		next:            next,
		name:            name,
		statusCodeStart: startCode,
		statusCodeEnd:   endCode,
	}, nil
}

// Check if the upstream's return is outside the given status code
// use hijacker to reset the connection when is inside
func (p *dropConnection) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// TODO: Check condition
	if p.statusCodeStart != 0 && p.statusCodeEnd != 0 {
		// Let's send a request to the next chain and wait for the feedback
		wrappedWriter := &responseWriter{ResponseWriter: rw}

		p.next.ServeHTTP(wrappedWriter, req)

		statusCode := wrappedWriter.statusCode

		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		bodyBytes := wrappedWriter.buffer.Bytes()

		if statusCode < p.statusCodeStart || p.statusCodeEnd < statusCode {
			rw.WriteHeader(statusCode)
			_, err := rw.Write(bodyBytes)
			if err != nil {
				log.Println("error when writing upstream data to writer", err)
			}
			return
		}
	}

	err := resetConn(rw)

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
	}
}

func resetConn(w http.ResponseWriter) error {
	if wr, ok := w.(http.Hijacker); ok {
		conn, _, err := wr.Hijack()
		if err != nil {
			return err
		}

		err = conn.Close()

		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("cannot reset connection, the hijacker is not existed")
	}

	return nil
}

type responseWriter struct {
	buffer     bytes.Buffer
	statusCode int

	http.ResponseWriter
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseWriter) Write(p []byte) (int, error) {
	return r.buffer.Write(p)
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}

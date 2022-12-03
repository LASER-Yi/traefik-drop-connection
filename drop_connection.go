package traefik_drop_connection

import (
	"bytes"
	"context"
	"fmt"
	"log"
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

	log.Printf("%s will read from upstream and waiting for the status code between %d and %d", name, startCode, endCode)

	return &dropConnection{
		next:            next,
		name:            name,
		statusCodeStart: startCode,
		statusCodeEnd:   endCode,
	}, nil
}

func (p *dropConnection) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// TODO: Check condition
	if p.statusCodeStart != 0 && p.statusCodeEnd != 0 {
		// Let's send a request to the next chain and wait for the feedback
		wrappedWriter := &responseWriter{ResponseWriter: rw}

		p.next.ServeHTTP(wrappedWriter, req)

		statusCode := wrappedWriter.statusCode

		if statusCode == 0 {
			statusCode = 200
		}

		bodyBytes := wrappedWriter.buffer.Bytes()

		if statusCode < p.statusCodeStart || p.statusCodeEnd < statusCode {
			rw.WriteHeader(statusCode)
			rw.Write(bodyBytes)
			return
		}
	}

	resetConn(rw)
}

func resetConn(w http.ResponseWriter) {
	if wr, ok := w.(http.Hijacker); ok {
		conn, _, err := wr.Hijack()
		if err != nil {
			log.Println(w, err)
			return
		}
		conn.Close()
	} else {
		log.Println("Cannot reset connection, the hijacker is not existed")
	}
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

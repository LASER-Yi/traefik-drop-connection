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
	StatusCode string `json:"status_code,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		StatusCode: "",
	}
}

type dropConnection struct {
	next              http.Handler
	name              string
	status_code_start int
	status_code_end   int
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {

	start_code := 0
	end_code := 0
	if len(config.StatusCode) != 0 {
		codes := strings.Split(config.StatusCode, "-")

		if len(codes) != 2 {
			return nil, fmt.Errorf("error compiling status code mapping")
		}

		start, err := strconv.Atoi(codes[0])

		if err != nil {
			return nil, fmt.Errorf("error converting first status code to integer, %s", err)
		}

		end, err := strconv.Atoi(codes[1])

		if err != nil {
			return nil, fmt.Errorf("error converting second status code to integer, %s", err)
		}

		start_code = start
		end_code = end
	}

	return &dropConnection{
		next:              next,
		name:              name,
		status_code_start: start_code,
		status_code_end:   end_code,
	}, nil
}

func (a *dropConnection) ResetConn(w http.ResponseWriter, req *http.Request) {
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

func (a *dropConnection) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// TODO: Check condition
	if a.status_code_start != 0 && a.status_code_end != 0 {
		// Let's send a request to the next chain and wait for the feedback
		wrappedWriter := &responseWriter{}

		log.Printf("read from upstream and waiting for the status code between %d and %d", a.status_code_start, a.status_code_end)

		a.next.ServeHTTP(wrappedWriter, req)

		statusCode := wrappedWriter.statusCode
		if statusCode == 0 {
			statusCode = 200
		}

		bodyBytes := wrappedWriter.buffer.Bytes()

		log.Printf("received status code %d", statusCode)

		if statusCode < a.status_code_start || a.status_code_end < statusCode {
			rw.Write(bodyBytes)
			return
		}
	}

	a.ResetConn(rw, req)
}

type responseWriter struct {
	buffer     bytes.Buffer
	statusCode int

	http.ResponseWriter
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseWriter) Write(p []byte) (int, error) {
	return r.buffer.Write(p)
}

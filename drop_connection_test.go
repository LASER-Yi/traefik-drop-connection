// Testing library for this plugin
package traefik_drop_connection_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	traefik_drop_connection "github.com/LASER-Yi/traefik-drop-connection"
)

const defaultCtx = "It works!"

func DefaultContextHandler(rw http.ResponseWriter, req *http.Request) {
	_, err := rw.Write([]byte(defaultCtx))
	if err != nil {
		log.Println("error when creating context handler", err)
	}
}

func TestDropConnection(t *testing.T) {
	cfg := traefik_drop_connection.CreateConfig()

	ctx := context.Background()
	next := http.HandlerFunc(DefaultContextHandler)

	handler, err := traefik_drop_connection.New(ctx, next, cfg, "drop-connection-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	result := recorder.Result()

	assertBody(t, result.Body, make([]byte, 0))
}

func TestDropConnectionOutsideRange(t *testing.T) {
	cfg := traefik_drop_connection.CreateConfig()
	cfg.StatusCode = "300-599"

	ctx := context.Background()
	next := http.HandlerFunc(DefaultContextHandler)

	handler, err := traefik_drop_connection.New(ctx, next, cfg, "drop-connection-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	result := recorder.Result()

	if result.StatusCode != http.StatusOK {
		t.Error("The context handler should return 200")
	}

	assertBody(t, result.Body, []byte(defaultCtx))
}

func TestDropConnectionInsideRange(t *testing.T) {
	cfg := traefik_drop_connection.CreateConfig()
	cfg.StatusCode = "100-599"

	ctx := context.Background()
	next := http.HandlerFunc(DefaultContextHandler)

	handler, err := traefik_drop_connection.New(ctx, next, cfg, "drop-connection-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	result := recorder.Result()

	if result.StatusCode != http.StatusOK {
		t.Error("The context handler should return 200")
	}

	assertBody(t, result.Body, make([]byte, 0))
}

func assertBody(t *testing.T, body io.Reader, expected []byte) {
	t.Helper()

	if resp, err := io.ReadAll(body); err == nil {
		if bytes.Equal(resp, expected) == false {
			t.Errorf("invalid body content \"%s\", should be \"%s\"", string(resp), string(expected))
		}
	}
}

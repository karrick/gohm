package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/karrick/gobp"
	"github.com/karrick/gohm/v2"
)

type ControlHandler struct {
}

func (m *ControlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "some error", http.StatusForbidden)
}

type NaiveHandler struct {
	Output io.Writer
}

func (n *NaiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	err := "some error"
	status := http.StatusForbidden
	http.Error(w, err, status)
	duration := time.Since(start)

	// "{client-ip} {http-client_ip} {http-CLIENT_IP} [{begin-iso8601}] \"{method} {uri} {proto}\" {status} {bytes} {duration} {error}"
	fmt.Fprintf(n.Output, "%s %s %s [%s] \"%s %s %s\" %d %d %s %s\"", r.RemoteAddr, r.Header.Get("client_ip"), r.Header.Get("CLIENT_IP"), start, r.Method, r.RequestURI, r.Proto, status, len(err), duration, err)
}

type MinimalHandler struct {
	Output io.Writer
}

func (m *MinimalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	err := "some error"
	status := http.StatusForbidden
	http.Error(w, err, status)
	duration := time.Since(start)

	// "{client-ip} {http-client_ip} {http-CLIENT_IP} [{begin-iso8601}] \"{method} {uri} {proto}\" {status} {bytes} {duration} {error}"
	m.Output.Write([]byte(r.RemoteAddr + " " + r.Header.Get("client_ip") + " " + r.Header.Get("CLIENT_IP") + " [" + start.Format(time.RFC3339) + "] \"" + r.Method + " " + r.RequestURI + " " + r.Proto + " " + strconv.Itoa(status) + " " + strconv.Itoa(len(err)) + " " + duration.String() + " " + err))
}

func benchmarkHandler(b *testing.B, handler http.Handler) {
	request := httptest.NewRequest("GET", "/some/url", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkControl(b *testing.B) {
	benchmarkHandler(b, &ControlHandler{})
}

func BenchmarkGohm(b *testing.B) {
	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{
		BufPool:   new(gobp.Pool),
		LogWriter: new(bytes.Buffer),
		LogFormat: "{client-ip} {http-client_ip} {http-CLIENT_IP} [{begin-iso8601}] \"{method} {uri} {proto}\" {status} {bytes} {duration} {error}",
	})

	benchmarkHandler(b, handler)
}

func BenchmarkNaive(b *testing.B) {
	benchmarkHandler(b, &NaiveHandler{Output: new(bytes.Buffer)})
}

func BenchmarkMinimal(b *testing.B) {
	benchmarkHandler(b, &MinimalHandler{Output: new(bytes.Buffer)})
}

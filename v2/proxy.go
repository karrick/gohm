package gohm

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// debugLogProxy will cause additional log lines to be output for every
	// proxied request.
	debugLogProxy = true

	// globalHTTPClientRequestTimeout is set pretty long because some proxied
	// calls take quite a bit of time to complete...
	globalHTTPClientRequestTimeout = 2 * time.Minute
)

var globalDebug = atomicBool(0)

type atomicBool int32

func (a *atomicBool) Get() bool {
	return atomic.LoadInt32((*int32)(a)) != 0
}

func (a *atomicBool) Set(flag bool) {
	var value int32
	if flag {
		value = 1
	}
	atomic.StoreInt32((*int32)(a), value)
}

// Do sends an HTTP request and returns an HTTP response.
//
// The provided ctx must be non-nil. If it is canceled or times out,
// ctx.Err() will be returned.
func Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// globalHTTPClient = &http.Client{
	// 	Timeout: globalHTTPClientRequestTimeout,
	// }
	//
	// globalHTTPSClient = &http.Client{
	// 	Timeout:   globalHTTPClientRequestTimeout,
	// 	Transport: &http.Transport{TLSClientConfig: globalTLSConfig},
	// }

	// spawn go-routine to perform requested operation
	var response *http.Response
	cerr := make(chan error, 1)

	go func() {
		var err error
		var client *http.Client
		if req.URL.Scheme == "https" {
			client = globalHTTPSClient
		} else {
			client = globalHTTPClient
		}
		if globalDebug.Get() {
			log.Printf("[DEBUG] sending %s %s", req.Method, req.URL)
		}
		// ??? concern that this overwrites context already on http.Request.
		response, err = client.Do(req.WithContext(ctx))
		cerr <- err
	}()

	// wait for response or context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-cerr:
		return response, err
	}
}

func proxyPrefix(mux *http.ServeMux, trimPrefix, newPrefix string, corsConfig CORSConfig) {
	mux.Handle(trimPrefix, CORSHandler(corsConfig, buildProxy(trimPrefix, newPrefix)))
}

func buildProxy(trimPrefix, newPrefix string) http.Handler {
	trimCount := len(trimPrefix)

	return http.HandlerFunc(func(outboundResponse http.ResponseWriter, inboundRequest *http.Request) {
		url := newPrefix + inboundRequest.RequestURI[trimCount:]
		if debugLogProxy && globalDebug.Get() {
			log.Printf("proxy url: %s", url)
		}

		// create new request, but pass incoming r.Body as outbound request body
		outboundRequest, err := http.NewRequest(inboundRequest.Method, url, inboundRequest.Body)
		if err != nil {
			Error(outboundResponse, fmt.Sprintf("cannot create HTTP %s request", inboundRequest.Method), http.StatusInternalServerError)
			return
		}

		// copy request headers from upstream client to downstream server
		outboundRequest.ContentLength = copyHeaders(inboundRequest.Header, outboundRequest.Header)

		if debugLogProxy && globalDebug.Get() {
			buf, err := httputil.DumpRequestOut(outboundRequest, true)
			if err != nil {
				Error(outboundResponse, fmt.Sprintf("cannot dump outbound request: %s", err), http.StatusBadGateway)
				return
			}
			log.Printf("[DEBUG] outbound request:\n%s", string(buf))
		}

		inboundResponse, err := Do(context.Background(), outboundRequest)
		if err != nil {
			Error(outboundResponse, fmt.Sprintf("cannot query proxied server: %s", err), http.StatusBadGateway)
			return
		}

		// copy response headers from downstream server to upstream client
		rhContentLength := copyHeaders(inboundResponse.Header, outboundResponse.Header())
		outboundResponse.WriteHeader(inboundResponse.StatusCode)

		// Ask Go runtime to copy response body directly from downstream back to
		// upstream, allowing runtime to buffer the data efficiently.
		actualResponseLength, err := io.Copy(outboundResponse, inboundResponse.Body)
		if err2 := inboundResponse.Body.Close(); err == nil {
			// If the copy returned an error, do not overwrite it; otherwise,
			// use whatever the error return value from the close.
			err = err2
		}
		if err != nil {
			log.Printf("[WARNING] cannot copy response body: %q; %s", url, err)
		}

		if rhContentLength > 0 && rhContentLength != actualResponseLength {
			// This is more informational message about a downstream server
			// returning an invalid Content-Length header in its response.
			log.Printf("[WARNING] response provided invalid Content-Length header: %q; %d; actual: %d", url, rhContentLength, actualResponseLength)
		}
	})
}

// copyHeaders copies end-to-end headers while omitting hop-by-hop headers.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers#hbh
func copyHeaders(from, to http.Header) int64 {
	var contentLength int64
	var err error

	for key, values := range map[string][]string(from) {
		switch key {
		case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding", "Upgrade":
			if debugLogProxy && globalDebug.Get() {
				log.Printf("[DEBUG] skipping hop-by-hop header: %q: %v", key, values)
			}
		default:
			if debugLogProxy && globalDebug.Get() {
				log.Printf("[DEBUG] copy header: %q: %v", key, values)
			}
			switch key {
			case "Content-Length":
				contentLength, err = strconv.ParseInt(values[0], 10, 64)
				if err != nil {
					log.Printf("[WARNING] invalid Content-Length header: %s; %q", err, values[0])
				}
			default:
				to.Set(key, strings.Join(values, ", "))
			}
		}
	}
	return contentLength
}

// proxyPrefix(mux, "/proxy/foo", ps.amapi+"/foo", gohm.CORSConfig{
// 	OriginsFilter: allowedOrigins,
// 	AllowHeaders:  []string{"Content-Type"},
// 	AllowMethods:  []string{"GET", "POST"},
// 	MaxAgeSeconds: 600,
// })

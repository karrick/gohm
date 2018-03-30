package gohm

import (
	"log"
	"net/http"
	"net/http/httputil"
	"sync/atomic"
)

// WithRequestDumper wraps http.Handler and optionally dumps the request when
// the specified flag is non-zero. It uses atomic.LoadUnit32 to read the
// flag. When 0, requests will not be dumped. When 1, all but the body will be
// dumped. When 2, the entire request including the body will be dumped.
func WithRequestDumper(flag *uint32, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if value := atomic.LoadUint32(flag); value > 0 {
			buf, err := httputil.DumpRequest(r, value == 2)
			if err != nil {
				log.Printf("cannot dump request: %s", err)
			}
			log.Printf("[DEBUG] inbound request:\n%s", string(buf))
		}
		next.ServeHTTP(w, r)
	})
}

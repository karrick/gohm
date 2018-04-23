package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/karrick/gobp"
	gohm "github.com/karrick/gohm/v2"
)

var latencyThreshold = 5 * time.Nanosecond

func main() {
	optPort := flag.Int("port", 8080, "HTTP server network port")
	flag.Parse()

	bp := new(gobp.Pool) // See https://github.com/karrick/gobp for heavy use examples.

	h := gohm.New(http.HandlerFunc(someHandler), gohm.Config{
		BufPool: bp,
		Callback: func(stats *gohm.Statistics) {
			if len(stats.RequestBody) > 0 {
				if bytes.ContainsAny(stats.RequestBody, "\r\n") {
					log.Printf("non-empty request body:\n%s\n", string(stats.RequestBody))
				} else {
					log.Printf("non-empty request body: %s\n", string(stats.RequestBody))
				}
			}
			if stats.ResponseEnd.Sub(stats.RequestBegin).Nanoseconds() > int64(latencyThreshold) {
				stats.Log()
			}
		},
		EscrowReader: true,
	})

	log.Print("[INFO] web service port: ", *optPort)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *optPort),
		Handler: h,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("[ERROR] ", err)
	}
}

func someHandler(w http.ResponseWriter, r *http.Request) {
	// do something interesting...
}

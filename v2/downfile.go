package gohm

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type downFileChecker struct {
	contents atomic.Value // string
	pathname string
}

func newDownFileChecker(pathname string) *downFileChecker {
	dfc := &downFileChecker{pathname: pathname}
	dfc.contents.Store("")
	go dfc.run()
	return dfc
}

var zeroTime time.Time // this time value will never be modified, but used solely to copy a zero time variable.

func (dfc *downFileChecker) run() {
	var fi os.FileInfo
	var prevModTime time.Time
	var err error

	for {
		time.Sleep(5 * time.Second)

		fi, err = os.Stat(dfc.pathname)
		if err == nil {
			// Service down file was found.
			newModTime := fi.ModTime()
			if newModTime.Equal(prevModTime) {
				continue // no need to read file contents again
			}
			prevModTime = newModTime

			if fi.Size() == 0 {
				// empty down file
				log.Printf("[DOWN] node down for maintenance: empty down file")
				dfc.contents.Store("node down for maintenance")
				continue
			}

			// When down file has content, copy to response.
			why, err := ioutil.ReadFile(dfc.pathname)
			if err != nil {
				why = []byte(err.Error()) // When cannot read the downfile content, copy error message.
			} else if l := len(why); l > 0 && why[l-1] == '\n' {
				why = why[:l-1] // strip trailing newline
			}

			message := fmt.Sprintf("node down for maintenance: %s", why)
			log.Printf("[DOWN] %s\n", message)
			dfc.contents.Store(message)
		} else {
			// There is no down file.
			if !prevModTime.IsZero() {
				log.Printf("[DOWN] node restored from maintenance") // but there was last iteration thru loop
				dfc.contents.Store("")
				prevModTime = zeroTime
			}
		}
	}
}

func (dfc *downFileChecker) Contents() string {
	return dfc.contents.Load().(string)
}

func (dfc *downFileChecker) NewHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contents := dfc.Contents()
		if len(contents) == 0 {
			// When no down file, pass request along to next handler.
			next.ServeHTTP(w, r)
			return
		}
		// Reject query when down file is present.
		w.Header().Set("Cache-Control", "no-cache, no-store")
		Error(w, string(contents), http.StatusServiceUnavailable)
	})
}

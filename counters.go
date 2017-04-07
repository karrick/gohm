package gohm

import (
	"sync/atomic"
)

// Counters structure store status counters used to track number of HTTP responses resulted in
// various status classes.
//
//	var counters gohm.Counters
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.New(someHandler, gohm.Config{Counters: &counters}))
//	// later on...
//	countOf1xx := counters.Get1xx()
//	countOf2xx := counters.Get2xx()
//	countOf3xx := counters.Get3xx()
//	countOf4xx := counters.Get4xx()
//	countOf5xx := counters.Get5xx()
//	countTotal := counters.GetAll()
type Counters struct {
	counters [6]uint64
}

// GetAll returns total number of HTTP responses, regardless of status code.
func (c Counters) GetAll() uint64 {
	return atomic.LoadUint64(&(c.counters[0]))
}

// Get1xx returns number of HTTP responses resulting in a 1xx status code.
func (c Counters) Get1xx() uint64 {
	return atomic.LoadUint64(&(c.counters[1]))
}

// Get2xx returns number of HTTP responses resulting in a 2xx status code.
func (c Counters) Get2xx() uint64 {
	return atomic.LoadUint64(&(c.counters[2]))
}

// Get3xx returns number of HTTP responses resulting in a 3xx status code.
func (c Counters) Get3xx() uint64 {
	return atomic.LoadUint64(&(c.counters[3]))
}

// Get4xx returns number of HTTP responses resulting in a 4xx status code.
func (c Counters) Get4xx() uint64 {
	return atomic.LoadUint64(&(c.counters[4]))
}

// Get5xx returns number of HTTP responses resulting in a 5xx status code.
func (c Counters) Get5xx() uint64 {
	return atomic.LoadUint64(&(c.counters[5]))
}

// GetAndResetAll returns number of HTTP responses resulting in a All status code, and resets the
// counter to 0.
func (c Counters) GetAndResetAll() uint64 {
	return atomic.SwapUint64(&(c.counters[0]), 0)
}

// GetAndReset1xx returns number of HTTP responses resulting in a 1xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset1xx() uint64 {
	return atomic.SwapUint64(&(c.counters[1]), 0)
}

// GetAndReset2xx returns number of HTTP responses resulting in a 2xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset2xx() uint64 {
	return atomic.SwapUint64(&(c.counters[2]), 0)
}

// GetAndReset3xx returns number of HTTP responses resulting in a 3xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset3xx() uint64 {
	return atomic.SwapUint64(&(c.counters[3]), 0)
}

// GetAndReset4xx returns number of HTTP responses resulting in a 4xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset4xx() uint64 {
	return atomic.SwapUint64(&(c.counters[4]), 0)
}

// GetAndReset5xx returns number of HTTP responses resulting in a 5xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset5xx() uint64 {
	return atomic.SwapUint64(&(c.counters[5]), 0)
}

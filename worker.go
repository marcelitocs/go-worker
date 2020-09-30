package worker

import (
	"errors"
	"sync"
)

type Entry interface{}
type Result interface{}

type Response struct {
	Result Result
	Err    error
}

type ResponseWithEntry struct {
	Entry    Entry
	Response Response
}

type Work struct {
	buffer      int
	concurrency int
	entries     chan Entry
	responses   chan ResponseWithEntry
	callback    func(Entry) (Result, error)
	done        chan bool
}

func NewWork(concurrency, buffer int) *Work {
	w := Work{
		buffer:      buffer,
		concurrency: concurrency,
		entries:     make(chan Entry, buffer),
		done:        make(chan bool),
	}
	return &w
}

func (w *Work) SetCallback(callback func(Entry) (Result, error)) *Work {
	w.callback = callback
	return w
}

func (w *Work) AddEntry(entry interface{}) *Work {
	w.entries <- entry
	return w
}

func (w *Work) OnResponse(responseCallback func(Entry, Response)) *Work {
	w.responses = make(chan ResponseWithEntry, w.buffer)
	go func() {
		for re := range w.responses {
			responseCallback(re.Entry, re.Response)
		}
		w.done <- true
	}()
	return w
}

func (w *Work) Start() *Work {
	if w.callback == nil {
		panic(errors.New("callback not set"))
	}
	go func() {
		run(w.callback, w.concurrency, w.entries, w.responses)
		if w.responses == nil {
			w.done <- true
		}
	}()
	return w
}

func (w *Work) Stop() *Work {
	close(w.entries)
	return w
}

func (w *Work) Wait() *Work {
	<-w.done
	return w
}

func run(callback func(Entry) (Result, error), concurrency int, entries <-chan Entry, responses chan<- ResponseWithEntry) {
	var wg sync.WaitGroup
	for work := 0; work < concurrency; work++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for entry := range entries {
				result, err := callback(entry)
				if responses != nil {
					responses <- ResponseWithEntry{entry, Response{result, err}}
				}
			}
		}()
	}
	wg.Wait()
	if responses != nil {
		close(responses)
	}
}

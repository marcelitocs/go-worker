package worker

import (
	"errors"
	"sync"
)

type Entry interface{}

type Response interface{}

type ResponseWithEntry struct {
	Entry    Entry
	Response Response
}

type Error struct {
	entry   Entry
	message string
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Entry() Entry {
	return e.entry
}

func NewError(entry Entry, message string) Error {
	return Error{entry, message}
}

type Work struct {
	buffer      int
	concurrency int
	entries     chan Entry
	responses   chan ResponseWithEntry
	callback    func(Entry) (Response, error)
	errors      chan Error
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

func (w *Work) SetCallback(callback func(Entry) (Response, error)) *Work {
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

func (w *Work) OnError(errorCallback func(err Error)) *Work {
	w.errors = make(chan Error, w.buffer)
	go func() {
		for err := range w.errors {
			errorCallback(err)
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
		run(w.callback, w.concurrency, w.entries, w.responses, w.errors)
		if w.responses == nil {
			w.done <- true
		}
		if w.errors == nil {
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
	<-w.done
	return w
}

func run(callback func(Entry) (Response, error), concurrency int, entries <-chan Entry, responses chan<- ResponseWithEntry, errors chan<- Error) {
	var wg sync.WaitGroup
	for work := 0; work < concurrency; work++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for entry := range entries {
				response, err := callback(entry)
				if err != nil {
					if errors != nil {
						errors <- NewError(entry, err.Error())
					}
					continue
				}
				if responses != nil {
					responses <- ResponseWithEntry{entry, response}
				}
			}
		}()
	}
	wg.Wait()
	if responses != nil {
		close(responses)
	}
	if errors != nil {
		close(errors)
	}
}

package worker

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func Example() {
	w := NewWork(1, 0)

	go func() {
		w.AddEntry(1)
		w.Stop()
	}()

	w.SetCallback(func(entry Entry) (Response, error) {
		// set the workload here
		// time.Sleep(time.Second)
		return entry.(int) * 10, nil
	})

	w.OnResponse(func(e Entry, r Response) {
		fmt.Printf("send %d and return %d", e.(int), r.(int))
	})

	w.OnError(func(e *Error) {
		// set here what happens when a error is returned
	})

	// you must set a callback before call start
	w.Start().Wait()

	// Output:
	// send 1 and return 10
}

func TestRun(t *testing.T) {
	entries := make(chan Entry)
	responsesWithEntries := make(chan ResponseWithEntry)
	chanErrors := make(chan error)
	mu := sync.Mutex{}

	table := map[int]int{
		1: 10,
		2: 20,
		3: 30,
		4: 40,
		5: 50,
	}
	errorOn3 := false
	processedCount := 0
	processedErrorCount := 0
	done := make(chan bool)

	go func() {
		for v := range table {
			entries <- v
		}
		close(entries)
	}()

	go func() {
		for re := range responsesWithEntries {
			e, ok := re.Entry.(int)
			if !ok {
				t.Errorf("run() re.Entry.(int) failed")
				continue
			}
			r, ok := re.Response.(int)
			if !ok {
				t.Errorf("run() re.Response.(int) failed")
				continue
			}
			if table[e] != r {
				t.Errorf("run() add %v as response, wants %v", r, table[e])
			}

			mu.Lock()
			processedCount++
			mu.Unlock()
		}
		done <- true
	}()

	go func() {
		for e := range chanErrors {
			we, ok := e.(*Error)
			if !ok {
				t.Errorf("run() e.(*Error) failed")
				continue
			}
			mu.Lock()
			if we.Entry() == 3 && we.Error() == "entry = 3" {
				errorOn3 = true
			}
			processedErrorCount++
			mu.Unlock()
		}
		done <- true
	}()

	run(func(entry Entry) (Response, error) {
		if entry == 3 {
			return nil, errors.New("entry = 3")
		}
		return entry.(int) * 10, nil
	}, 2, entries, responsesWithEntries, chanErrors)

	<-done
	<-done

	mu.Lock()

	if !errorOn3 {
		t.Error("error 3 is not processed")
	}

	if processedCount != 4 {
		t.Errorf("processed count is %d, wants 4", processedCount)
	}

	if processedErrorCount != 1 {
		t.Errorf("processed error count is %d, wants 1", processedCount)
	}

	mu.Unlock()
}

func TestWorkStartEnd(t *testing.T) {
	w := NewWork(2, 0)
	mu := sync.Mutex{}

	table := map[int]int{
		1: 10,
		2: 20,
		3: 30,
		4: 40,
		5: 50,
	}
	errorOn3 := false
	processedCount := 0
	processedErrorCount := 0

	go func() {
		for v := range table {
			w.AddEntry(v)
		}
		w.Stop()
	}()

	w.SetCallback(func(entry Entry) (Response, error) {
		if entry == 3 {
			return nil, errors.New("entry = 3")
		}
		return entry.(int) * 10, nil
	})

	w.OnResponse(func(e Entry, r Response) {
		if e == 3 {
			t.Errorf("return entry 3 as response")
		}
		if table[e.(int)] != r {
			t.Errorf("return %v as response, wants %v", r, table[e.(int)])
		}
		mu.Lock()
		processedCount++
		mu.Unlock()
	})

	w.OnError(func(e *Error) {
		mu.Lock()
		if e.Entry() == 3 && e.Error() == "entry = 3" {
			errorOn3 = true
		}
		processedErrorCount++
		mu.Unlock()
	})

	w.Start().Wait()

	mu.Lock()

	if !errorOn3 {
		t.Error("error 3 is not processed")
	}

	if processedCount != 4 {
		t.Errorf("processed count is %d, wants 4", processedCount)
	}

	if processedErrorCount != 1 {
		t.Errorf("processed error count is %d, wants 1", processedCount)
	}

	mu.Unlock()
}

func BenchmarkRun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entries := make(chan Entry)
		responses := make(chan ResponseWithEntry)

		go func() {
			entries <- 1
			close(entries)
		}()

		go func() {
			for response := range responses {
				_ = response
			}
		}()

		run(func(entry Entry) (Response, error) {
			return nil, nil
		}, 1, entries, responses, nil)
	}
}

func BenchmarkWorkStartEnd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		w := NewWork(1, 0)

		go func() {
			w.AddEntry(1).Stop()
		}()

		w.OnResponse(func(entry Entry, response Response) {

		})

		w.SetCallback(func(entry Entry) (response Response, err error) {
			return nil, nil
		})

		w.Start().Wait()
	}
}

func BenchmarkProcessing(b *testing.B) {
	w := NewWork(1, 0)

	w.OnResponse(func(entry Entry, response Response) {

	})

	w.SetCallback(func(entry Entry) (response Response, err error) {
		return nil, nil
	})

	w.Start()

	for i := 0; i < b.N; i++ {
		w.AddEntry(i)
	}
	w.Stop()

	w.Wait()
}

func BenchmarkProcessingWithoutResponse(b *testing.B) {
	w := NewWork(1, 0)

	w.SetCallback(func(entry Entry) (response Response, err error) {
		return nil, nil
	})

	w.Start()

	for i := 0; i < b.N; i++ {
		w.AddEntry(i)
	}
	w.Stop()

	w.Wait()
}

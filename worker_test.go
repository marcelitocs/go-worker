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

	w.SetCallback(func(entry Entry) (Result, error) {
		// set the workload here
		// time.Sleep(time.Second)
		return entry.(int) * 10, nil
	})

	w.OnResponse(func(e Entry, r Response) {
		if r.Err != nil {
			// set here what happens when a error is returned
			return
		}
		fmt.Printf("send %d and return %d", e.(int), r.Result.(int))
	})

	// you must set a callback before call start
	w.Start().Wait()

	// Output:
	// send 1 and return 10
}

func TestRun(t *testing.T) {
	entries := make(chan Entry)
	responsesWithEntries := make(chan ResponseWithEntry)
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
			if re.Response.Err != nil {
				if re.Entry == 3 && re.Response.Err.Error() == "entry = 3" {
					errorOn3 = true
				}
				processedErrorCount++
				continue
			}
			r, ok := re.Response.Result.(int)
			if !ok {
				t.Errorf("run() re.Response.Result.(int) failed")
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

	run(func(entry Entry) (Result, error) {
		if entry == 3 {
			return nil, errors.New("entry = 3")
		}
		return entry.(int) * 10, nil
	}, 2, entries, responsesWithEntries)

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

	w.SetCallback(func(entry Entry) (Result, error) {
		if entry == 3 {
			return nil, errors.New("entry = 3")
		}
		return entry.(int) * 10, nil
	})

	w.OnResponse(func(e Entry, r Response) {
		if r.Err != nil {
			mu.Lock()
			if e == 3 && r.Err.Error() == "entry = 3" {
				errorOn3 = true
			}
			processedErrorCount++
			mu.Unlock()
			return
		}
		if e == 3 {
			t.Errorf("return entry 3 without error")
		}
		if table[e.(int)] != r.Result {
			t.Errorf("return %v as response, wants %v", r, table[e.(int)])
		}
		mu.Lock()
		processedCount++
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

		run(func(entry Entry) (Result, error) {
			return nil, nil
		}, 1, entries, responses)
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

		w.SetCallback(func(entry Entry) (result Result, err error) {
			return nil, nil
		})

		w.Start().Wait()
	}
}

func BenchmarkProcessing(b *testing.B) {
	w := NewWork(1, 0)

	w.OnResponse(func(entry Entry, response Response) {

	})

	w.SetCallback(func(entry Entry) (result Result, err error) {
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

	w.SetCallback(func(entry Entry) (result Result, err error) {
		return nil, nil
	})

	w.Start()

	for i := 0; i < b.N; i++ {
		w.AddEntry(i)
	}
	w.Stop()

	w.Wait()
}

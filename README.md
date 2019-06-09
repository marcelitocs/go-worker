# go-worker

go-worker is a package to help you manage the flow of processing

Example of using the go-worker

```
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
```
# go-worker

go-worker is a package to help you manage the flow of processing

Example of using the go-worker

```go
package worker

import (
    "fmt"
    "github.com/marcelitocs/go-worker"
)

func main() {
    w := worker.NewWork(1, 0)

    go func() {
        w.AddEntry(1)
        w.Stop()
    }()

    w.SetCallback(func(entry worker.Entry) (worker.Result, error) {
        // set the workload here
        // time.Sleep(time.Second)
        return entry.(int) * 10, nil
    })

    w.OnResponse(func(e worker.Entry, r worker.Response) {
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
```
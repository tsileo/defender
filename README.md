# defender

[![Godoc Reference](https://godoc.org/github.com/tsileo/defender?status.png)](https://godoc.org/github.com/tsileo/defender)

B
Low-level package to help prevent brute force attacks, built on top of [golang.org/x/time/rate](https://golang.org/x/time/rate).

```go
package main

import (
	"time"

	"github.com/tsileo/defender"
)

func main() {
	// Ban client for 1 hour if moe than 50 events per seconds are performed
	d := defender.New(50, 1 * time.Second, 1 * time.Hour)
	if d.Banned(r.RemoteAddr) {
		// Do sth
	}
}
```

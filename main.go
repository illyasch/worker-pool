package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/illyasch/worker-pool/pool"
)

const (
	NumWorkers  = 10
	HTTPTimeout = 10
)

// summary keeps statistic values about the download.
type summary struct {
	num      int
	volume   int
	duration time.Duration
	mu       sync.Mutex
}

// download implements pool.Runner interface for a pool task.
type download struct {
	url     string
	timeout time.Duration
	total   *summary
}

func main() {
	num := flag.Int("w", NumWorkers, "Number of workers.")
	timeout := flag.Int("t", HTTPTimeout, "HTTP timeout.")
	flag.Parse()

	workers := pool.New(*num)
	workers.Run(context.Background())
	fmt.Printf("processing started with %d workers\n", *num)

	total := summary{}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		workers.Execute(download{
			url:     fmt.Sprintf("https://%s", scanner.Text()),
			timeout: time.Duration(*timeout) * time.Second,
			total:   &total,
		})
	}

	if scanner.Err() != nil {
		fmt.Printf("error: scanner: %v\n", scanner.Err())
	}
	workers.Stop()

	fmt.Printf("\ndownloaded %.2d files, average %.2d bytes, %v\n",
		total.num,
		total.volume/total.num,
		total.duration/time.Duration(total.num),
	)
}

// Job does a download of an index page from a domain and measures its size and duration of the download.
func (d download) Job(ctx context.Context) {
	ctx, _ = context.WithTimeout(ctx, d.timeout)

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.url, nil)
	if err != nil {
		fmt.Printf("error: get request %s: %v\n", d.url, err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		fmt.Printf("error: getting %s: %v\n", d.url, err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("error: getting %s: status %d %s\n", d.url, resp.StatusCode, http.StatusText(resp.StatusCode))
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error: reading %s: %v\n", d.url, err)
		return
	}

	d.total.Add(len(body), duration)
	fmt.Printf("success: %s, size %d, duration %s\n", d.url, len(body), duration)
}

// Add increments download statistics thread safely.
func (s *summary) Add(volume int, duration time.Duration) {
	s.mu.Lock()
	s.num++
	s.volume += volume
	s.duration += duration
	s.mu.Unlock()
}

package pool_test

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/illyasch/worker-pool/pool"
)

type count struct {
	cnt *counter
	wg  *sync.WaitGroup
}

func (a *count) Job(context.Context) pool.JobResponse[string] {
	defer a.wg.Done()

	a.cnt.mu.Lock()
	v := a.cnt.value
	a.cnt.value++
	a.cnt.mu.Unlock()

	time.Sleep(time.Duration(rand.Intn(10000)) * time.Microsecond)

	return pool.JobResponse[string]{
		Value: strconv.Itoa(v),
	}
}

func TestNonBlocking_Run(t *testing.T) {
	t.Run("Successful 99 tasks run", func(t *testing.T) {
		const total = 99

		var cnt counter
		var wg sync.WaitGroup

		workers := pool.NewNonBlocking[string](10)
		workers.Run(context.Background())

		executed := 0
		requests := workers.RequestChan()

		for i := 0; i < total; i++ {
			wg.Add(1)
			req := <-requests
			req.Request <- &count{&cnt, &wg}
			t.Logf("request %d\n", i)

			go func(n int) {
				r := <-req.Response
				t.Logf("response %d, value = %s\n", n, r.Value)
			}(i)

			executed++
		}
		wg.Wait()
		workers.Stop()

		assert.Equal(t, executed, cnt.value)
	})

	t.Run("Workers blocking", func(t *testing.T) {
		const total = 33

		workers := pool.NewNonBlocking[string](10)
		workers.Run(context.Background())

		received := 0
		requests := workers.RequestChan()

		for i := 0; i < total; i++ {
			select {
			case req := <-requests:
				t.Logf("worker %d received\n", i)
				received++
				req.Close()

			case <-time.After(100 * time.Millisecond):
				t.Logf("worker %d missed\n", i)
			}
		}

		assert.Equal(t, total, received)
	})
}

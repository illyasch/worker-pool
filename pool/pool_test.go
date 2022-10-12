package pool_test

import (
	"context"
	"sync"
	"testing"

	"github.com/illyasch/worker-pool/pool"
	"github.com/stretchr/testify/assert"
)

type counter struct {
	value int
	mu    sync.Mutex
}

type add struct {
	cnt *counter
	wg  *sync.WaitGroup
}

func (a *add) Job(context.Context) {
	a.cnt.mu.Lock()
	a.cnt.value++
	a.cnt.mu.Unlock()
	a.wg.Done()
}

func TestPool_Run(t *testing.T) {
	t.Run("Successful 99 tasks run", func(t *testing.T) {
		var cnt counter
		var wg sync.WaitGroup

		workers := pool.New(10)
		workers.Run(context.Background())

		for i := 0; i < 99; i++ {
			wg.Add(1)
			workers.Execute(&add{
				&cnt,
				&wg,
			})
		}
		wg.Wait()
		workers.Stop()

		assert.Equal(t, 99, cnt.value)
	})
}

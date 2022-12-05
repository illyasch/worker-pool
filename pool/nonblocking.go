package pool

import (
	"context"
	"sync"
)

// NonBlocking carries a worker tasks channel, a wait group, and other values.
type NonBlocking[T any] struct {
	cancel     context.CancelFunc
	requests   chan JobRequest[T]
	start      sync.WaitGroup
	finish     sync.WaitGroup
	workersCnt int
}

// JobResponse keeps a response from a task sent to a worker.
type JobResponse[T any] struct {
	Value T
	Err   error
}

// JobRequest keeps necessary channels for requesting the execution of a task in a worker.
type JobRequest[T any] struct {
	Request  chan NonBlockingRunner[T]
	Response chan JobResponse[T]
}

// NonBlockingRunner is an interface for a task that can be executed in non-blocking worker pool.
type NonBlockingRunner[T any] interface {
	Job(ctx context.Context) JobResponse[T]
}

// NewNonBlocking creates a new worker pool.
func NewNonBlocking[T any](workersCnt int) *NonBlocking[T] {
	return &NonBlocking[T]{
		requests:   make(chan JobRequest[T]),
		workersCnt: workersCnt,
	}
}

// Run starts workers in the pool.
func (p *NonBlocking[T]) Run(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)

	p.start.Add(p.workersCnt)
	p.finish.Add(p.workersCnt)
	for i := 0; i < p.workersCnt; i++ {
		go func() {
			defer p.finish.Done()

			p.start.Done()
			for {
				input := make(chan NonBlockingRunner[T])
				output := make(chan JobResponse[T])

				select {
				case <-ctx.Done():
					return

				case p.requests <- JobRequest[T]{Request: input, Response: output}:
					task := <-input
					output <- task.Job(ctx)
				}

				close(input)
				close(output)
			}
		}()
	}

	p.start.Wait()
}

// Stop stops workers in the pool.
func (p *NonBlocking[T]) Stop() {
	p.cancel()
	p.finish.Wait()
}

// RequestChan returns a request channel for executing a task in a worker.
// You will need to retrieve a JobRequest[T] struct from the channel for requesting
// the execution of a task in a worker.
// Usage example:
// Gets a channel to get a free worker
// requests := workers.RequestChan()
// Retrieves a first free worker for a task execution
// req := <-requests
// Sends a task to the worker then worker starts the task execution
// req.Request <- task{param1, param2}
// Waits until the task execution is finished and retrieves a response struct
// resp := <-req.Response
// After the task is finished, the worker is free and returns to the worker pool,
// waiting for another task.
func (p *NonBlocking[T]) RequestChan() chan JobRequest[T] {
	return p.requests
}

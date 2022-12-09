package pool

import (
	"context"
	"fmt"
	"sync"
)

var (
	ErrResponseClosed = fmt.Errorf("response channel closed")
)

// NonBlocking carries a worker tasks channel, a wait group, and other values.
type NonBlocking[T any] struct {
	cancel     context.CancelFunc
	requests   chan *JobRequest[T]
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
// Use it to send a task to the worker and get the response.
// You have to use JobRequest[T] Close() function after the task execution
// (or if you want to skip the task execution) to ensure that the worker is free again.
type JobRequest[T any] struct {
	Request  chan NonBlockingRunner[T]
	Response chan JobResponse[T]
	closed   bool
	mu       sync.Mutex
}

// NonBlockingRunner is an interface for a task that can be executed in non-blocking worker pool.
type NonBlockingRunner[T any] interface {
	Job(ctx context.Context) JobResponse[T]
}

// NewNonBlocking creates a new worker pool.
func NewNonBlocking[T any](workersCnt int) *NonBlocking[T] {
	return &NonBlocking[T]{
		requests:   make(chan *JobRequest[T]),
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
				req := NewJobRequest[T]()

				select {
				case <-ctx.Done():
					return

				case p.requests <- req:
					task := <-req.Request
					if task != nil {
						_ = req.SendResponse(task.Job(ctx))
					}
				}
				req.Close()
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
func (p *NonBlocking[T]) RequestChan() chan *JobRequest[T] {
	return p.requests
}

// NewJobRequest creates a new JobRequest[T] struct which is used to interact with a worker.
func NewJobRequest[T any]() *JobRequest[T] {
	return &JobRequest[T]{
		Request:  make(chan NonBlockingRunner[T]),
		Response: make(chan JobResponse[T]),
		closed:   false,
	}
}

// Close closes the request and response channels of the JobRequest[T] struct.
// Use it when you want to finish an interaction with the worker and make it
// available to get another task.
func (j *JobRequest[T]) Close() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.closed {
		j.closed = true
		close(j.Request)
		close(j.Response)
	}
}

// SendResponse sends a response to the JobRequest[T] consumer.
func (j *JobRequest[T]) SendResponse(resp JobResponse[T]) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.closed {
		j.Response <- resp
		return nil
	}

	return ErrResponseClosed
}

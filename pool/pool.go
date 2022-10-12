// Package pool implements a pool of workers for simultaneous execution of different tasks.
package pool

import (
	"context"
	"sync"
)

// Runner is an interface for a task that can be executed in worker pool.
type Runner interface {
	Job(ctx context.Context)
}

// Pool carries a worker tasks channel, a wait group, and other values.
type Pool struct {
	input      chan Runner
	wg         sync.WaitGroup
	workersCnt int
}

// New creates a new worker pool.
func New(workersCnt int) *Pool {
	return &Pool{
		input:      make(chan Runner),
		workersCnt: workersCnt,
	}
}

// Run starts workers in the pool.
func (p *Pool) Run(ctx context.Context) {
	for i := 0; i < p.workersCnt; i++ {
		p.wg.Add(1)

		go func() {
			for task := range p.input {
				task.Job(ctx)
			}
			p.wg.Done()
		}()
	}
}

// Stop stops workers in the pool.
func (p *Pool) Stop() {
	close(p.input)
	p.wg.Wait()
}

// Execute adds a new task in the tasks queue of a worker pool.
func (p *Pool) Execute(task Runner) {
	p.input <- task
}

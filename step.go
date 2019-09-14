package pipeline

import (
	"sync"

	tomb "gopkg.in/tomb.v2"
)

const (
	// MaxWorkerCount limits the number of WorkerCount to create for a step.
	MaxWorkerCount = 20
	// MaxBufferSize limits the size of the out buffered channel for a step.
	MaxBufferSize = 10
)

// StepFn is the signature of a step function
type StepFn func(*Context, <-chan interface{}, chan interface{}) error

// Step is the main type for processing in a pipeline.
type Step struct {
	tomb.Tomb
	// Name of the step.
	Name string
	// Buffered indicates if the process' out channel will be buffered.
	// to allow sending more data before blocking.
	// Defaults to false
	Buffered bool
	// FanOut indicates whether workers should perform redundant work, i.e.,
	// fan out input channels to each one as opposed to using a single one.
	// Defaults to false
	FanOut bool
	// The number of workers to spawn in go routines to handle this step.
	// Defaults to 1, set > 1 for concurrent processing
	WorkerCount int
	// fn is the actual func to execute
	fn StepFn
	// wg is a wait group to sync exit
	wg *sync.WaitGroup
	// f is a fan for multiplexing messages
	f *fan
}

// NewStep creates a new step, defaults to worker step.
func NewStep(name string, step func(*Context, <-chan interface{}, chan interface{}) error) *Step {
	return NewWorkerStep(name, 1, step)
}

// NewWorkerStep creates a new worker based step.
func NewWorkerStep(name string, workerCount int, step func(*Context, <-chan interface{}, chan interface{}) error) *Step {
	return newStep(name, false, false, workerCount, step)
}

// NewBufferedStep creates a new buffered step.
func NewBufferedStep(name string, step func(*Context, <-chan interface{}, chan interface{}) error) *Step {
	return newStep(name, true, false, 1, step)
}

// NewFanOutStep creates a new fan out step.
func NewFanOutStep(name string, workerCount int, step func(*Context, <-chan interface{}, chan interface{}) error) *Step {
	return newStep(name, false, true, workerCount, step)
}

func newStep(name string, buffered bool, fanout bool, workerCount int, fn func(*Context, <-chan interface{}, chan interface{}) error) *Step {
	s := &Step{}
	s.Name = name
	s.Buffered = buffered
	s.FanOut = fanout
	s.WorkerCount = workerCount
	s.fn = fn
	return s
}

// Process executes this step.
func (s *Step) Process(ctx *Context, in <-chan interface{}) chan interface{} {
	c := &Context{s.Context(ctx), ctx.pipeline}
	var out chan interface{}
	if s.Buffered {
		out = make(chan interface{}, MaxBufferSize)
	} else {
		out = make(chan interface{})
	}
	if s.FanOut {
		s.f = fanOut(in, s.WorkerCount)
	}
	s.wg = &sync.WaitGroup{}
	s.wg.Add(s.WorkerCount)
	for i := 0; i < s.WorkerCount; i++ {
		index := i
		if s.FanOut {
			s.Go(func() error {
				defer s.wg.Done()
				in := s.f.outs[index]
				return s.fn(c, in, out)
			})
		} else {
			s.Go(func() error {
				defer s.wg.Done()
				return s.fn(c, in, out)
			})
		}
	}
	s.Go(func() error {
		s.wg.Wait()
		// safe to kill fan
		if s.f != nil {
			s.f.Kill(nil)
			s.f.Wait()
		}
		if out != nil {
			close(out)
		}
		return nil
	})
	return out
}

// Replicated duplicates this step for number of times requested.
func (s *Step) Replicated(num int) []*Step {
	steps := []*Step{}
	for i := 0; i < num; i++ {
		steps = append(steps, NewStep(s.Name, s.fn))
	}
	return steps
}

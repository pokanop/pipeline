package pipeline

import (
	"fmt"
	"sync"

	tomb "gopkg.in/tomb.v2"
)

// Stage is a type to compose with steps.
type Stage struct {
	tomb.Tomb
	// Name is the name of the step
	Name string
	// Concurrent determines whether to process this stage serially or not.
	Concurrent bool
	// steps are the actual steps to run for this stage
	steps []*Step
	// ctx is the context for this stage
	ctx *Context
}

func newStage(name string, concurrent bool, steps ...*Step) *Stage {
	s := &Stage{}
	s.Name = name
	s.Concurrent = concurrent
	s.steps = steps
	return s
}

// NewStage creates a new stage, defaults to serial stage.
func NewStage(name string, steps ...*Step) *Stage {
	return NewSerialStage(name, steps...)
}

// NewSerialStage creates a new stage for given steps that will run sequentially.
func NewSerialStage(name string, steps ...*Step) *Stage {
	return newStage(name, false, steps...)
}

// NewConcurrentStage creates a new stage for given steps that will run sequentially.
func NewConcurrentStage(name string, steps ...*Step) *Stage {
	return newStage(name, true, steps...)
}

// NewSerialStageFunc creates a new stage for given steps that will run sequentially.
func NewSerialStageFunc(name string, stepFns ...StepFn) *Stage {
	return newStageFunc(name, false, stepFns...)
}

// NewConcurrentStageFunc creates a new stage for given steps that will run sequentially.
func NewConcurrentStageFunc(name string, stepFns ...StepFn) *Stage {
	return newStageFunc(name, true, stepFns...)
}

func newStageFunc(name string, concurrent bool, stepFns ...StepFn) *Stage {
	steps := []*Step{}
	for i, fn := range stepFns {
		steps = append(steps, NewStep(fmt.Sprintf("%s-%d", name, i), fn))
	}
	return newStage(name, concurrent, steps...)
}

// AddStep adds a step to the stage
func (s *Stage) AddStep(step *Step) {
	s.steps = append(s.steps, step)
}

// Process executes this stage.
func (s *Stage) Process(ctx *Context, in <-chan interface{}) chan interface{} {
	s.ctx = ctx
	c := &Context{s.Context(ctx), ctx.pipeline}
	if s.Concurrent {
		// Process steps concurrently
		ins := make([]chan interface{}, len(s.steps))
		for _, step := range s.steps {
			s.updateStatus(step.Name, StatusStepStarted)
			ins = append(ins, step.Process(c, in))
		}
		f := fanIn(ins...)
		s.trackSteps(f)
		return f.out
	}

	// Feed one step's output channel into the next one's input
	var out chan interface{}
	for _, step := range s.steps {
		// Can downgrade a rw channel to a read-only channel but cannot upgrade
		// therefore this must be done like this
		s.updateStatus(step.Name, StatusStepStarted)
		if out == nil {
			out = step.Process(c, in)
		} else {
			out = step.Process(c, out)
		}
	}
	s.trackSteps(nil)
	return out
}

func (s *Stage) trackSteps(f *fan) {
	s.Go(func() error {
		var firstErr error
		wg := &sync.WaitGroup{}
		for _, st := range s.steps {
			wg.Add(1)
			step := st
			go func() {
				defer wg.Done()
				err := step.Wait()
				if firstErr == nil {
					firstErr = err
				}
				s.updateStatus(step.Name, StatusStepFinished)
			}()
		}
		wg.Wait()
		if f != nil {
			f.Kill(nil)
			f.Wait()
		}
		return firstErr
	})
}

func (s *Stage) updateStatus(name string, status Status) {
	if s.ctx.pipeline != nil {
		s.ctx.pipeline.updateStatus(name, status)
	}
}

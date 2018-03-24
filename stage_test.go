package pipeline

import (
	"context"
	"testing"
)

var step1 = StepFn(func(ctx *Context, in <-chan interface{}, out chan interface{}) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case n := <-in:
			if n == nil {
				continue
			}
			out <- n.(int) * 2
		}
	}
})

var step2 = StepFn(func(ctx *Context, in <-chan interface{}, out chan interface{}) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case n := <-in:
			if n == nil {
				continue
			}
			out <- n.(int) + 2
		}
	}
})

var step3 = StepFn(func(ctx *Context, in <-chan interface{}, out chan interface{}) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case n := <-in:
			if n == nil {
				continue
			}
			out <- n.(int) - 2
		}
	}
})

var step4 = StepFn(func(ctx *Context, in <-chan interface{}, out chan interface{}) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case n := <-in:
			if n == nil {
				continue
			}
			out <- n.(int) / 2
		}
	}
})

func TestConcurrentStage(t *testing.T) {
	values := []int{2, 2, 2, 2, 2}
	stage := NewConcurrentStageFunc("toy", step2, step3, step1, step4)
	in := make(chan interface{})
	defer close(in)
	go func() {
		for _, i := range values {
			in <- i
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	out := stage.Process(&Context{ctx, nil}, in)
	results := []int{}
	for i := 0; i < len(values); i++ {
		results = append(results, (<-out).(int))
	}
	for i, v := range results {
		if v != 4 && v != 1 && v != 0 {
			t.Fatalf("replicated stage expected value %d, found %d", values[i], v)
		}
	}
	cancel()
	stage.Wait()
}

func TestSerialStage(t *testing.T) {
	values := []int{2, 4, 5, 8, 3}
	stage := NewSerialStageFunc("toy", step1, step2, step3, step4)
	in := make(chan interface{})
	defer close(in)
	go func() {
		for _, i := range values {
			in <- i
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	out := stage.Process(&Context{ctx, nil}, in)
	results := []int{}
	for i := 0; i < len(values); i++ {
		results = append(results, (<-out).(int))
	}
	for i, v := range results {
		if v != values[i] {
			t.Fatalf("replicated stage expected value %d, found %d", values[i], v)
		}
	}
	cancel()
	stage.Wait()
}

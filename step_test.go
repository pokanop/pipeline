package pipeline

import (
	"context"
	"testing"
)

func TestProcess(t *testing.T) {
	step := NewStep("adder", func(ctx *Context, in <-chan interface{}, out chan interface{}) error {
		// Add
		out <- (<-in).(int) + 3
		return nil
	})
	in := make(chan interface{})
	defer close(in)
	go func() {
		in <- 5
	}()
	ctx := &Context{context.Background(), nil}
	n := (<-step.Process(ctx, in)).(int)
	if n != 8 {
		t.Fatalf("step should have added to total 8, found %d", n)
	}
	step.Wait()
}

func TestReplicated(t *testing.T) {
	step := NewStep("multipler", func(ctx *Context, in <-chan interface{}, out chan interface{}) error {
		// Multiply
		out <- (<-in).(int) * 5
		return nil
	})

	iters := 5
	in := make(chan interface{})
	defer close(in)
	go func() {
		for i := 0; i < iters; i++ {
			in <- 5
		}
	}()
	for _, step := range step.Replicated(iters) {
		ctx := &Context{context.Background(), nil}
		n := (<-step.Process(ctx, in)).(int)
		if n != 25 {
			t.Fatalf("step should have multiplied num to 25, found %d", n)
		}
		step.Wait()
	}
}

func TestWorkerCount(t *testing.T) {
	step := NewStep("multipler", func(ctx *Context, in <-chan interface{}, out chan interface{}) error {
		// Multiply
		for {
			select {
			case <-ctx.Done():
				return nil
			case n := <-in:
				out <- n.(int) * 5
			}
		}
	})
	step.WorkerCount = 10

	iters := 10000
	in := make(chan interface{})
	defer close(in)
	go func() {
		for i := 0; i < iters; i++ {
			in <- 5
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	out := step.Process(&Context{ctx, nil}, in)
	for i := 0; i < iters; i++ {
		n := (<-out).(int)
		if n != 25 {
			t.Fatalf("step should have multiplied num to 25, found %d", n)
		}
	}
	cancel()
	step.Wait()
}

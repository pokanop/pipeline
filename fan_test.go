package pipeline

import (
	"sort"
	"sync"
	"testing"
)

func TestFanIn(t *testing.T) {
	values := []int{3, 2, 5, 6, 3}
	ins := make([]chan interface{}, len(values))
	for i := 0; i < len(values); i++ {
		ins[i] = make(chan interface{})
	}
	f := fanIn(ins...)

	for i := 0; i < len(values); i++ {
		go func(i int) {
			in := ins[i]
			in <- values[i]
		}(i)
	}

	results := []int{}
	for i := 0; i < len(values); i++ {
		results = append(results, (<-f.out).(int))
	}
	f.close()
	f.Kill(nil)
	f.Wait()
	sort.Ints(values)
	sort.Ints(results)
	for i := 0; i < len(values); i++ {
		if results[i] != values[i] {
			t.Fatalf("expected %d but found %d", values[i], results[i])
		}
	}
}

func TestFanOut(t *testing.T) {
	in := make(chan interface{})
	f := fanOut(in, 5)
	wg := &sync.WaitGroup{}
	wg.Add(5)
	mu := &sync.Mutex{}
	results := []int{}
	for i := 0; i < 5; i++ {
		go func(i int) {
			defer wg.Done()
			out := f.outs[i]
			for n := range out {
				if n == nil {
					return
				}
				mu.Lock()
				results = append(results, n.(int))
				mu.Unlock()
			}
		}(i)
	}

	values := []int{52, 325, 15, 32, 84}
	for i := 0; i < 5; i++ {
		in <- values[i]
	}
	close(in)
	f.Wait()
	wg.Wait()

	if len(results) != 25 {
		t.Fatalf("expected 25 results, found %d", len(results))
	}
}

package pipeline

import (
	tomb "gopkg.in/tomb.v2"
)

type fan struct {
	tomb.Tomb
	ins  []chan interface{}
	out  chan interface{}
	outs []chan interface{}
}

// fanIn combines multiple channels into a single output channel.
func fanIn(ins ...chan interface{}) *fan {
	f := &fan{}
	f.out = make(chan interface{})
	f.ins = ins
	defer f.teardown()
	for i := 0; i < len(ins); i++ {
		index := i
		f.Go(func() error {
			in := f.ins[index]
			for {
				select {
				case <-f.Dying():
					return nil
				case data := <-in:
					if data == nil {
						return nil
					}
					f.out <- data
				}
			}
		})
	}
	return f
}

// fanOut splits a single channel into a number of output channels.
func fanOut(in <-chan interface{}, num int) *fan {
	f := &fan{}
	f.outs = make([]chan interface{}, num)
	for i := 0; i < num; i++ {
		f.outs[i] = make(chan interface{})
	}
	defer f.teardown()
	f.Go(func() error {
		for data := range in {
			f.sendOut(data)
		}
		return nil
	})
	return f
}

func (f *fan) sendOut(input interface{}) {
	for _, out := range f.outs {
		select {
		case <-f.Dying():
			return
		case out <- input:
		}
	}
}

func (f *fan) close() {
	if f.out != nil {
		close(f.out)
		f.out = nil
	} else if f.outs != nil {
		for _, out := range f.outs {
			close(out)
		}
		f.outs = nil
	}
}

func (f *fan) teardown() {
	go func() {
		f.Wait()
		f.close()
	}()
}

package pipeline

import (
	"encoding/csv"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestWorkerPipeline(t *testing.T) {
	csvStep := NewWorkerStep("csv reader", 1, csvReader)
	printStep := NewWorkerStep("line printer", 5, linePrinter)
	makeStep := NewWorkerStep("object maker", 5, objectMaker)
	stage := NewSerialStage("csv consumer", csvStep, printStep, makeStep)
	pipeline := NewPipeline("csv consumer", stage)
	in := make(chan interface{})
	defer close(in)
	out := pipeline.Process(nil, in)
	users := []*user{}
	go func() {
		in <- &syncCSVReader{Reader: csv.NewReader(strings.NewReader(lines))}
		for {
			select {
			case <-printStep.Dead():
				pipeline.Kill(nil)
				return
			}
		}
	}()

OuterLoop:
	for {
		select {
		case <-pipeline.Dead():
			break OuterLoop
		case obj := <-out:
			if obj == nil {
				continue
			}
			user := obj.(*user)
			users = append(users, user)
		}
	}

	pipeline.Wait()

	if len(users) != 30 {
		t.Fatalf("expected 30 users, but pipeline produced %d", len(users))
	}
}

func TestFanOutPipeline(t *testing.T) {
	csvStep := NewFanOutStep("csv reader", 5, csvReader)
	printStep := NewFanOutStep("line printer", 5, linePrinter)
	makeStep := NewFanOutStep("object maker", 5, objectMaker)
	stage := NewSerialStage("csv consumer", csvStep, printStep, makeStep)
	pipeline := NewPipeline("csv consumer")
	pipeline.AddStage(stage)
	in := make(chan interface{})
	out := pipeline.Process(nil, in)
	users := []*user{}
	go func() {
		in <- &syncCSVReader{Reader: csv.NewReader(strings.NewReader(lines))}
		close(in)
		for {
			select {
			case <-csvStep.Dead():
				pipeline.Kill(nil)
				return
			}
		}
	}()

OuterLoop:
	for {
		select {
		case <-pipeline.Dead():
			break OuterLoop
		case obj := <-out:
			if obj == nil {
				continue
			}
			user := obj.(*user)
			users = append(users, user)
		}
	}
	if len(users) != 750 {
		t.Fatalf("expected 750 users, but pipeline produced %d", len(users))
	}
}

func TestBufferedPipeline(t *testing.T) {
	csvStep := NewBufferedStep("csv reader", csvReader)
	printStep := NewBufferedStep("line printer", linePrinter)
	printStep.WorkerCount = 10
	makeStep := NewBufferedStep("object maker", objectMaker)
	makeStep.WorkerCount = 5
	stage := NewSerialStage("csv consumer", csvStep, printStep, makeStep)
	pipeline := NewPipeline("csv consumer", stage)
	stateCh := pipeline.State()
	progressCh := pipeline.AltProgress()
	in := make(chan interface{})
	defer close(in)
	go func() {
		in <- &syncCSVReader{Reader: csv.NewReader(strings.NewReader(lines))}

		for {
			select {
			case state := <-stateCh:
				fmt.Println(state)
			case progress := <-progressCh:
				fmt.Println(progress)
			case <-pipeline.Dead():
				break
			}
		}
	}()
	out := pipeline.Process(nil, in)
	if d := pipeline.ElapsedTime(); d.Nanoseconds() == 0 {
		t.Fatalf("expected elapsed time > 0, found %d nanoseconds", d.Nanoseconds())
	}
	users := []*user{}
	for obj := range out {
		user := obj.(*user)
		users = append(users, user)
	}
	if len(users) != 30 {
		t.Fatalf("expected 30 users, but pipeline produced %d", len(users))
	}
	pipeline.Kill(nil)
	pipeline.Wait()
}

type user struct {
	firstName string
	lastName  string
	username  string
}

type syncCSVReader struct {
	sync.Mutex
	*csv.Reader
}

func csvReader(ctx *Context, in <-chan interface{}, out chan interface{}) error {
	r := (<-in).(*syncCSVReader)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			r.Lock()
			record, err := r.Read()
			r.Unlock()
			if err != nil {
				return nil
			}
			out <- record
		}
	}
}

func linePrinter(ctx *Context, in <-chan interface{}, out chan interface{}) error {
	for record := range in {
		if record == nil {
			return nil
		}
		fmt.Println(record)
		out <- record
	}
	return nil
}

func objectMaker(ctx *Context, in <-chan interface{}, out chan interface{}) error {
	for record := range in {
		if record == nil {
			return nil
		}
		s := record.([]string)
		if len(s) != 3 {
			continue
		}
		out <- &user{s[0], s[1], s[2]}
	}
	return nil
}

var lines = `"Rob","Pike",rob
Ken,Thompson,ken
"Robert","Griesemer","gri"
John,Doe,jd
Jane,Doe,jd
Foo,Bar,fb
"Rob","Pike",rob
Ken,Thompson,ken
"Robert","Griesemer","gri"
John,Doe,jd
Jane,Doe,jd
Foo,Bar,fb
"Rob","Pike",rob
Ken,Thompson,ken
"Robert","Griesemer","gri"
John,Doe,jd
Jane,Doe,jd
Foo,Bar,fb
"Rob","Pike",rob
Ken,Thompson,ken
"Robert","Griesemer","gri"
John,Doe,jd
Jane,Doe,jd
Foo,Bar,fb
"Rob","Pike",rob
Ken,Thompson,ken
"Robert","Griesemer","gri"
John,Doe,jd
Jane,Doe,jd
Foo,Bar,fb
`

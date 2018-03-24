package pipeline

import (
	"context"
	"sync"
	"time"

	"gopkg.in/tomb.v2"
)

// Status enum for pipeline states
type Status int

const (
	// StatusPipelineStarted signal when pipeline has started
	StatusPipelineStarted Status = iota
	// StatusPipelineFinished signal when pipeline has finished
	StatusPipelineFinished
	// StatusStageStarted signal when stage has started
	StatusStageStarted
	// StatusStageFinished signal when stage has finished
	StatusStageFinished
	// StatusStepStarted signal when step has started
	StatusStepStarted
	// StatusStepFinished signal when step has finished
	StatusStepFinished
)

func (s Status) String() string {
	switch s {
	case StatusPipelineStarted:
		return "pipeline started"
	case StatusPipelineFinished:
		return "pipeline finished"
	case StatusStageStarted:
		return "stage started"
	case StatusStageFinished:
		return "stage finished"
	case StatusStepStarted:
		return "step started"
	case StatusStepFinished:
		return "step finished"
	default:
		return ""
	}
}

// State type indicating pipeline state
type State struct {
	// Name of the entity reporting the status
	Name string
	// Status of the pipeline
	Status Status
	// Progress percent of the pipeline steps
	Progress float32
	// AltProgress percent of the pipeline, used for detailed progress
	AltProgress float32
}

// Pipeline is a type for composing stages and steps for processing.
type Pipeline struct {
	tomb.Tomb
	// Name is the name of the step
	Name string
	// stages list of all stages in pipeline
	stages []*Stage
	// stateCh is a channel that will send status changes
	stateCh chan *State
	// progressCh is a channel to listen for progress updates
	progressCh chan float32
	// previousProgressPct returns the last sent progress update value
	previousProgressPct float32
	// unitTotal for total units of work
	unitTotal int
	// unitCount for current units of work completed
	unitCount int
	// startTime of pipeline processing
	startTime time.Time
	// endTime of pipeline processing
	endTime time.Time
}

// NewPipeline creates a new pipeline with the provided stages.
func NewPipeline(name string, stages ...*Stage) *Pipeline {
	p := &Pipeline{}
	p.Name = name
	p.stages = stages
	p.stateCh = make(chan *State)
	p.progressCh = make(chan float32)
	return p
}

// AddStage adds a stage to the pipeline
func (p *Pipeline) AddStage(stage *Stage) {
	p.stages = append(p.stages, stage)
}

// State returns the state channel for status updates
func (p *Pipeline) State() <-chan *State {
	return p.stateCh
}

// Progress returns the progress channel for progress updates
func (p *Pipeline) Progress() <-chan float32 {
	return p.progressCh
}

// CurrentProgress returns the current progress percent of the pipeline
// by measuring the number of steps.
func (p *Pipeline) CurrentProgress() (int, int, float32) {
	var stepCount float32
	var finishedCount float32
	for _, stage := range p.stages {
		stepCount += float32(len(stage.steps))
		for _, step := range stage.steps {
			if !step.Alive() {
				finishedCount++
			}
		}
	}
	return int(finishedCount), int(stepCount), finishedCount / stepCount
}

// CurrentAltProgress returns the current alternate progress of the pipeline
// by measuring the units of work completed.
func (p *Pipeline) CurrentAltProgress() (int, int, float32) {
	return p.unitCount, p.unitTotal, float32(p.unitCount) / float32(p.unitTotal)
}

// Total sets the unit total
func (p *Pipeline) Total(value int) {
	p.unitTotal = value
}

// Inc increments unit count
func (p *Pipeline) Inc() {
	// Calculate progress and determine if an update is needed to be sent
	p.unitCount++
	_, _, currentProgress := p.CurrentAltProgress()

	// Updates are sent per 1% increase
	if (currentProgress-p.previousProgressPct) > 0.01 || p.unitCount == p.unitTotal {
		p.previousProgressPct = currentProgress
		select {
		case p.progressCh <- currentProgress:
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// ElapsedTime of the pipeline process
func (p *Pipeline) ElapsedTime() time.Duration {
	t := p.endTime
	if t.IsZero() {
		t = time.Now()
	}
	return t.Sub(p.startTime)
}

// Process executes the pipeline.
func (p *Pipeline) Process(ctx context.Context, in <-chan interface{}) chan interface{} {
	// Process stages serially
	p.updateStatus(p.Name, StatusPipelineStarted)
	if ctx == nil {
		ctx = context.Background()
	}
	c := &Context{p.Context(ctx), p}
	var out chan interface{}
	for _, stage := range p.stages {
		p.updateStatus(stage.Name, StatusStageStarted)
		if out == nil {
			out = stage.Process(c, in)
		} else {
			out = stage.Process(c, out)
		}
	}
	p.trackStages()
	return out
}

func (p *Pipeline) trackStages() {
	p.Go(func() error {
		var firstErr error
		wg := &sync.WaitGroup{}
		for _, s := range p.stages {
			wg.Add(1)
			stage := s
			go func() {
				defer wg.Done()
				err := stage.Wait()
				if firstErr == nil {
					firstErr = err
				}
				p.updateStatus(stage.Name, StatusStageFinished)
			}()
		}
		wg.Wait()
		p.updateStatus(p.Name, StatusPipelineFinished)
		return firstErr
	})
}

func (p *Pipeline) updateStatus(name string, status Status) {
	if status == StatusPipelineStarted {
		p.startTime = time.Now()
	} else if status == StatusPipelineFinished {
		p.endTime = time.Now()
	}

	_, _, currentProgress := p.CurrentProgress()
	_, _, currentAltProgress := p.CurrentAltProgress()
	select {
	case p.stateCh <- &State{name, status, currentProgress, currentAltProgress}:
	case <-time.After(10 * time.Millisecond):
	}
}

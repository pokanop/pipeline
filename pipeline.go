package pipeline

import (
	"context"
	"sync"
	"time"

	"gopkg.in/tomb.v2"
)

// Pipeline is a type for composing stages and steps for processing.
type Pipeline struct {
	tomb.Tomb
	// Name is the name of the step
	Name string
	// stages list of all stages in pipeline
	stages []*Stage
	// stateCh is a channel that will send status changes
	stateCh chan *State
	// altProgressCh is a channel to listen for progress updates
	altProgressCh chan float32
	// altProgressPct returns the last sent progress update value
	altProgressPct float32
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
	p.stateCh = make(chan *State, 100)
	p.altProgressCh = make(chan float32, 100)
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

// AltProgress returns the progress channel for progress updates
func (p *Pipeline) AltProgress() <-chan float32 {
	return p.altProgressCh
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

// Total sets the unit total used for alternate progress updates.
func (p *Pipeline) Total(value int) {
	p.unitTotal = value
}

// Inc increments unit count used for alternate progress updates.
func (p *Pipeline) Inc() {
	// Calculate progress and determine if an update is needed to be sent
	p.unitCount++
	_, _, currentProgress := p.CurrentAltProgress()

	// Updates are sent per 1% increase
	if (currentProgress-p.altProgressPct) > 0.01 || p.unitCount == p.unitTotal {
		p.altProgressPct = currentProgress
		p.altProgressCh <- currentProgress
	}
}

// ElapsedTime of the pipeline process.
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
	p.stateCh <- &State{name, status, currentProgress, currentAltProgress}
}

package pipeline

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
